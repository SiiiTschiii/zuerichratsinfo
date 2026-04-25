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

const truncatedCaptionNotice = "ℹ️ Gekürzt – weitere Teilabstimmungen im Link."

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

	var body strings.Builder
	body.WriteString(fmt.Sprintf("🗳️ Gemeinderat | Abstimmung vom %s\n\n", date))
	body.WriteString(title)
	body.WriteString("\n\n")

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
				body.WriteString(voteTitle)
			} else {
				emoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
				result := voteformat.GetVoteResultText(vote.Schlussresultat)
				body.WriteString(fmt.Sprintf("%s %s: %s", emoji, result, voteTitle))
			}
			body.WriteString("\n")
		} else {
			// Single vote: result line
			if !voteformat.IsAuswahlVote(counts) {
				emoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
				result := voteformat.GetVoteResultText(vote.Schlussresultat)
				body.WriteString(fmt.Sprintf("%s %s\n", emoji, result))
			}
		}

		body.WriteString(voteformat.FormatVoteCountsLong(counts))
		body.WriteString("\n")

		// Fraktion breakdown
		if stimmabgaben := vote.Stimmabgaben.Stimmabgabe; len(stimmabgaben) > 0 {
			fraktionCounts := voteformat.AggregateFraktionCounts(stimmabgaben)
			if breakdown := voteformat.FormatFraktionBreakdown(fraktionCounts); breakdown != "" {
				body.WriteString("\n")
				body.WriteString(breakdown)
				body.WriteString("\n")
			}
		}

		if i < len(votes)-1 {
			body.WriteString("\n")
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
	link = stripURLFragment(link)

	bodyText := strings.TrimRight(body.String(), "\n")
	linkLine := fmt.Sprintf("\n🔗 %s", link)
	caption := bodyText + linkLine

	// Truncate if over Instagram's character limit
	if len([]rune(caption)) > maxCaptionChars {
		noticeLine := "\n\n" + truncatedCaptionNotice
		if runeLen(noticeLine)+runeLen(linkLine) < maxCaptionChars {
			spaceForBody := maxCaptionChars - runeLen(linkLine) - runeLen(noticeLine)
			caption = truncateWithEllipsis(bodyText, spaceForBody) + noticeLine + linkLine
		} else {
			caption = truncateWithEllipsis(bodyText, maxCaptionChars-runeLen(linkLine)) + linkLine
		}
	}

	return caption
}

func runeLen(s string) int {
	return len([]rune(s))
}

func truncateWithEllipsis(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if runeLen(s) <= maxRunes {
		return s
	}
	if maxRunes == 1 {
		return "…"
	}
	runes := []rune(s)
	return string(runes[:maxRunes-1]) + "…"
}

func stripURLFragment(rawURL string) string {
	withoutFragment, _, hasFragment := strings.Cut(rawURL, "#")
	if !hasFragment {
		return rawURL
	}
	return withoutFragment
}
