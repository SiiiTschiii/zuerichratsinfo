package instagram

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/imagegen"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
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
func FormatCarousel(votes []zurichapi.Abstimmung) (*InstagramContent, error) {
	if len(votes) == 0 {
		return nil, fmt.Errorf("no votes provided")
	}

	// Generate carousel images
	images, err := imagegen.GenerateCarousel(votes)
	if err != nil {
		return nil, fmt.Errorf("generating carousel images: %w", err)
	}

	// Enforce Instagram's 10-image carousel cap
	if len(images) > maxCarouselImages {
		images = images[:maxCarouselImages]
	}

	// Build caption text
	caption := buildCaption(votes)

	return &InstagramContent{
		Images:  images,
		Caption: caption,
	}, nil
}

// buildCaption creates the caption text for an Instagram carousel post.
// Includes vote details (similar to X/Bluesky thread text flattened) + vote page link.
func buildCaption(votes []zurichapi.Abstimmung) string {
	firstVote := votes[0]

	// Header
	date := voteformat.FormatVoteDate(firstVote.SitzungDatum)
	title := voteformat.SelectBestTitle(firstVote.TraktandumTitel, firstVote.GeschaeftTitel)
	title = voteformat.CleanVoteTitle(title)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🗳️ Gemeinderat | Abstimmung vom %s\n\n", date))
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// Vote details for each vote
	for i, vote := range votes {
		counts := voteformat.VoteCounts{
			Ja: vote.AnzahlJa, Nein: vote.AnzahlNein,
			Enthaltung: vote.AnzahlEnthaltung, Abwesend: vote.AnzahlAbwesend,
			A: vote.AnzahlA, B: vote.AnzahlB, C: vote.AnzahlC, D: vote.AnzahlD, E: vote.AnzahlE,
		}

		if len(votes) > 1 {
			// Multi-vote: include subtitle
			voteTitle := voteformat.CleanVoteSubtitle(vote.Abstimmungstitel)
			if voteTitle == "" {
				voteTitle = fmt.Sprintf("Abstimmung %d", i+1)
			}
			if voteformat.IsAuswahlVote(counts) {
				sb.WriteString(voteTitle)
			} else {
				emoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
				result := voteformat.GetVoteResultText(vote.Schlussresultat)
				sb.WriteString(fmt.Sprintf("%s %s: %s", emoji, result, voteTitle))
			}
			sb.WriteString("\n")
		} else {
			// Single vote: result line
			if !voteformat.IsAuswahlVote(counts) {
				emoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
				result := voteformat.GetVoteResultText(vote.Schlussresultat)
				sb.WriteString(fmt.Sprintf("%s %s\n", emoji, result))
			}
		}

		sb.WriteString(voteformat.FormatVoteCountsLong(counts))
		sb.WriteString("\n")

		// Fraktion breakdown
		if stimmabgaben := vote.Stimmabgaben.Stimmabgabe; len(stimmabgaben) > 0 {
			fraktionCounts := voteformat.AggregateFraktionCounts(stimmabgaben)
			if breakdown := voteformat.FormatFraktionBreakdown(fraktionCounts); breakdown != "" {
				sb.WriteString("\n")
				sb.WriteString(breakdown)
				sb.WriteString("\n")
			}
		}

		if i < len(votes)-1 {
			sb.WriteString("\n")
		}
	}

	// Link
	var link string
	if voteformat.IsGenericAntragTitle(firstVote.TraktandumTitel) {
		link = voteformat.GenerateGeschaeftLink(firstVote.GeschaeftGuid)
	} else if len(votes) > 1 {
		link = voteformat.GenerateTraktandumLink(firstVote.SitzungGuid, firstVote.TraktandumGuid)
	} else {
		link = voteformat.GenerateVoteLink(firstVote.OBJGUID)
	}

	return buildCaptionWithPreservedLink(sb.String(), link)
}

func buildCaptionWithPreservedLink(body, link string) string {
	body = strings.TrimRight(body, "\n")
	linkLine := fmt.Sprintf("🔗 %s", link)
	caption := body + "\n" + linkLine

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
