package instagram

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/imagegen"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// maxCaptionChars is Instagram's caption character limit.
const maxCaptionChars = 2200

// maxCarouselImages is the maximum number of images in an Instagram carousel.
const maxCarouselImages = 10

// Captions are intentionally German because post copy targets Zurich municipal council followers.
const captionTruncatedNoticeLine = "ℹ️ Gekürzt – weitere Teilabstimmungen im Link."

// InstagramContent implements platforms.Content for Instagram
type InstagramContent struct {
	Images  [][]byte // JPEG-encoded carousel images
	Caption string   // caption text accompanying the carousel
}

// String returns the text representation for logging/preview
func (c *InstagramContent) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📸 Instagram carousel: %d image(s)\n\n", len(c.Images)))
	sb.WriteString(c.Caption)
	return sb.String()
}

// FormatCarousel generates carousel images and builds the caption text for an Instagram post.
func FormatCarousel(group []votes.Vote) (*InstagramContent, error) {
	return FormatCarouselWithContacts(group, nil)
}

// FormatCarouselWithContacts generates carousel images and builds the caption text for an Instagram post,
// including mapped Instagram @mentions where supported by the title text.
func FormatCarouselWithContacts(group []votes.Vote, contactMapper *contacts.Mapper) (*InstagramContent, error) {
	if len(group) == 0 {
		return nil, fmt.Errorf("no votes provided")
	}

	// Generate carousel images
	images, err := imagegen.GenerateCarousel(group)
	if err != nil {
		return nil, fmt.Errorf("generating carousel images: %w", err)
	}

	// Enforce Instagram's 10-image carousel cap
	if len(images) > maxCarouselImages {
		images = images[:maxCarouselImages]
	}

	// Build caption text
	caption := buildCaption(group, contactMapper)

	return &InstagramContent{
		Images:  images,
		Caption: caption,
	}, nil
}

// buildCaption creates the caption text for an Instagram carousel post.
// Includes vote details (similar to X/Bluesky thread text flattened) + vote page link.
func buildCaption(group []votes.Vote, contactMapper *contacts.Mapper) string {
	firstVote := group[0]

	// Header
	title := voteformat.CleanVoteTitle(firstVote.Title)
	if contactMapper != nil {
		title = contactMapper.TagInstagramHandlesInText(title)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🗳️ %s\n\n", voteformat.PostHeadline(group)))

	// For single-vote non-Schlussabstimmung, prepend the Abstimmungsgegenstand
	if len(group) == 1 {
		if prefix := voteformat.SingleVoteSubtitlePrefix(group[0].Subtitle); prefix != "" {
			sb.WriteString(prefix)
			sb.WriteString("\n")
		}
	}

	sb.WriteString(title)
	sb.WriteString("\n\n")

	// Vote details for each vote
	for i, vote := range group {
		counts := voteformat.CountsOf(vote)

		if len(group) > 1 {
			// Multi-vote: include subtitle
			voteTitle := voteformat.SubVoteLabel(vote, i, len(group))
			if voteformat.IsAuswahlVote(counts) {
				sb.WriteString(voteTitle)
			} else {
				emoji := voteformat.GetVoteResultEmoji(vote.Decision)
				result := voteformat.GetVoteResultText(vote.Decision)
				sb.WriteString(fmt.Sprintf("%s %s: %s", emoji, result, voteTitle))
			}
			sb.WriteString("\n")
		} else {
			// Single vote: result line
			if !voteformat.IsAuswahlVote(counts) {
				emoji := voteformat.GetVoteResultEmoji(vote.Decision)
				result := voteformat.GetVoteResultText(vote.Decision)
				sb.WriteString(fmt.Sprintf("%s %s\n", emoji, result))
			}
		}

		sb.WriteString(voteformat.FormatVoteCountsLong(counts))
		sb.WriteString("\n")

		// Fraktion breakdown
		if len(vote.MemberVotes) > 0 {
			fraktionCounts := voteformat.AggregateFraktionCounts(vote.MemberVotes)
			if breakdown := voteformat.FormatFraktionBreakdown(fraktionCounts); breakdown != "" {
				sb.WriteString("\n")
				sb.WriteString(breakdown)
				sb.WriteString("\n")
			}
		}

		if i < len(group)-1 {
			sb.WriteString("\n")
		}
	}

	return buildCaptionWithPreservedLink(sb.String(), voteformat.LinkLine(group))
}

func buildCaptionWithPreservedLink(body, linkBlock string) string {
	body = strings.TrimRight(body, "\n")
	linkLine := strings.TrimLeft(linkBlock, "\n")
	caption := body + "\n\n" + linkLine

	// Truncate if over Instagram's character limit
	if len([]rune(caption)) > maxCaptionChars {
		tailWithNotice := captionTruncatedNoticeLine + "\n" + linkLine
		tailWithNoticeWithSeparator := "\n" + tailWithNotice

		if len([]rune(tailWithNotice)) > maxCaptionChars {
			// Extremely defensive fallback: keep at least the link if notice+link ever exceed the platform limit.
			linkRunes := []rune(linkLine)
			if len(linkRunes) <= maxCaptionChars {
				return linkLine
			}
			return string(linkRunes[:maxCaptionChars-1]) + "…"
		}

		availableBodyRunes := maxCaptionChars - len([]rune(tailWithNoticeWithSeparator))
		if availableBodyRunes <= 0 {
			// No room left for body text; publish only truncation notice + link.
			return tailWithNotice
		}

		body = truncateRunesWithEllipsis(body, availableBodyRunes)
		caption = body + tailWithNoticeWithSeparator
	}

	return caption
}

func truncateRunesWithEllipsis(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	if maxRunes == 1 {
		return "…"
	}
	return string(runes[:maxRunes-1]) + "…"
}
