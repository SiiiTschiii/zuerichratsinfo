package bluesky

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/urlshorten"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// maxGraphemes is the Bluesky post character limit (graphemes)
const maxGraphemes = 300

// BlueskyPost holds the formatted text and rich text facets for a Bluesky post
type BlueskyPost struct {
	Text   string
	Facets []bskyapi.Facet
}

// FormatVoteGroupPost creates a formatted Bluesky post for a group of related votes.
// Returns the post text and any rich text facets (e.g. for clickable links).
func FormatVoteGroupPost(votes []zurichapi.Abstimmung) *BlueskyPost {
	if len(votes) == 0 {
		return &BlueskyPost{}
	}

	firstVote := votes[0]

	// Fixed components
	date := voteformat.FormatVoteDate(firstVote.SitzungDatum)
	header := fmt.Sprintf("🗳️ Gemeinderat | %s\n\n", date)

	// Title selection (same logic as X)
	title := voteformat.SelectBestTitle(firstVote.TraktandumTitel, firstVote.GeschaeftTitel)
	title = voteformat.CleanVoteTitle(title)

	// Generate and shorten the link
	var link string
	if voteformat.IsGenericAntragTitle(firstVote.TraktandumTitel) {
		link = voteformat.GenerateGeschaeftLink(firstVote.GeschaeftGuid)
	} else if len(votes) > 1 {
		link = voteformat.GenerateTraktandumLink(firstVote.SitzungGuid, firstVote.TraktandumGuid)
	} else {
		link = voteformat.GenerateVoteLink(firstVote.OBJGUID)
	}
	link = urlshorten.ShortenURL(link)

	// Build the vote body
	var body strings.Builder

	if len(votes) == 1 {
		vote := votes[0]
		resultEmoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
		result := voteformat.GetVoteResultText(vote.Schlussresultat)

		body.WriteString(fmt.Sprintf("%s %s: %s\n\n", resultEmoji, result, title))
		body.WriteString(fmt.Sprintf("📊 %s Ja | %s Nein | %s Enth. | %s Abw.\n\n",
			voteformat.FormatVoteCount(vote.AnzahlJa),
			voteformat.FormatVoteCount(vote.AnzahlNein),
			voteformat.FormatVoteCount(vote.AnzahlEnthaltung),
			voteformat.FormatVoteCount(vote.AnzahlAbwesend)))
	} else {
		body.WriteString(title)
		body.WriteString("\n\n")

		for i, vote := range votes {
			voteEmoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
			voteTitle := voteformat.CleanVoteSubtitle(vote.Abstimmungstitel)

			if voteTitle != "" {
				body.WriteString(fmt.Sprintf("%s %s\n", voteEmoji, voteTitle))
			} else {
				body.WriteString(fmt.Sprintf("%s Abstimmung %d\n", voteEmoji, i+1))
			}
			body.WriteString(fmt.Sprintf("📊 %s Ja | %s Nein | %s Enth. | %s Abw.\n",
				voteformat.FormatVoteCount(vote.AnzahlJa),
				voteformat.FormatVoteCount(vote.AnzahlNein),
				voteformat.FormatVoteCount(vote.AnzahlEnthaltung),
				voteformat.FormatVoteCount(vote.AnzahlAbwesend)))

			if i < len(votes)-1 {
				body.WriteString("\n")
			}
		}
		body.WriteString("\n")
	}

	// Assemble full text, potentially truncating the title/body to fit
	linkLine := fmt.Sprintf("🔗 %s", link)
	fullText := header + body.String() + linkLine

	// Truncate if over limit
	if graphemeLen(fullText) > maxGraphemes {
		fullText = truncatePost(header, body.String(), linkLine)
	}

	// Build facets: make the URL a clickable link
	facets := buildLinkFacets(fullText, link)

	return &BlueskyPost{
		Text:   fullText,
		Facets: facets,
	}
}

// buildLinkFacets finds the URL in the text and creates a link facet for it.
func buildLinkFacets(text, url string) []bskyapi.Facet {
	// Find the URL in the text by byte offset
	idx := strings.Index(text, url)
	if idx < 0 {
		return nil
	}

	byteStart := len(text[:idx])
	byteEnd := byteStart + len(url)

	return []bskyapi.Facet{
		bskyapi.LinkFacet(byteStart, byteEnd, url),
	}
}

// truncatePost truncates the body to fit within the grapheme limit
// while preserving the header and link line.
func truncatePost(header, body, linkLine string) string {
	available := maxGraphemes - graphemeLen(header) - graphemeLen(linkLine) - 1 // 1 for "…"

	if available <= 0 {
		// Extreme edge case: just header + link
		return header + linkLine
	}

	// Truncate body to fit
	runes := []rune(body)
	if len(runes) > available {
		// Trim trailing whitespace and add ellipsis
		truncated := strings.TrimRight(string(runes[:available]), " \n")
		body = truncated + "…"
	}

	return header + body + "\n" + linkLine
}

// graphemeLen returns the number of graphemes (runes) in a string.
// This is a simplified approximation — true grapheme clusters (e.g. combined emoji)
// would need a Unicode segmentation library, but rune count is close enough for
// typical Latin+emoji text and matches what Bluesky enforces in practice.
func graphemeLen(s string) int {
	return len([]rune(s))
}
