package bluesky

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// maxGraphemes is the Bluesky post character limit (graphemes)
const maxGraphemes = 300

// BlueskyPost holds the formatted text and rich text facets for a Bluesky post
type BlueskyPost struct {
	Text     string
	Facets   []bskyapi.Facet
	Mentions []contacts.BlueskyMention // unresolved mentions (handle + byte offsets)
}

// FormatVoteThread creates a Bluesky thread for a group of related votes.
// Returns a slice of posts: [0] is the root post, [1:] are replies.
//
// Root post contains: header, title, result (single vote), thread hint
// Replies contain: vote details (counts per vote), link
func FormatVoteThread(votes []zurichapi.Abstimmung, contactMapper *contacts.Mapper) []*BlueskyPost {
	if len(votes) == 0 {
		return nil
	}

	firstVote := votes[0]

	// Common components
	date := voteformat.FormatVoteDate(firstVote.SitzungDatum)
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

	// --- Build root post ---
	root, titleContinuation := buildRootPost(votes, date, title)

	// --- Build reply posts ---
	replies := buildReplyPosts(votes, link, titleContinuation)

	thread := make([]*BlueskyPost, 0, 1+len(replies))
	thread = append(thread, root)
	thread = append(thread, replies...)

	// Scan all posts for politician mentions with Bluesky accounts
	if contactMapper != nil {
		for _, post := range thread {
			post.Mentions = contactMapper.FindBlueskyMentions(post.Text)
		}
	}

	return thread
}

// buildRootPost creates the root post with header, title, result, and thread hint.
// Returns the post and a title continuation string (non-empty when title was truncated).
func buildRootPost(votes []zurichapi.Abstimmung, date, title string) (*BlueskyPost, string) {
	header := fmt.Sprintf("🗳️ Gemeinderat | Abstimmung vom %s\n\n", date)
	threadHint := "\n\n👇 Details im Thread"

	var body string
	var fullBody string // untruncated body for continuation
	if len(votes) == 1 {
		// Single vote: include result in root (unless it's an Auswahl A/B/C vote)
		vote := votes[0]
		counts := voteformat.VoteCounts{
			Ja: vote.AnzahlJa, Nein: vote.AnzahlNein,
			Enthaltung: vote.AnzahlEnthaltung, Abwesend: vote.AnzahlAbwesend,
			A: vote.AnzahlA, B: vote.AnzahlB, C: vote.AnzahlC, D: vote.AnzahlD, E: vote.AnzahlE,
		}
		if voteformat.IsAuswahlVote(counts) {
			body = title
		} else {
			resultEmoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
			result := voteformat.GetVoteResultText(vote.Schlussresultat)
			body = fmt.Sprintf("%s %s: %s", resultEmoji, result, title)
		}
	} else {
		// Multi-vote: just the title
		body = title
	}
	fullBody = body

	fullText := header + body + threadHint

	// Truncate title if root exceeds limit (rare, only for very long titles)
	var titleContinuation string
	if graphemeLen(fullText) > maxGraphemes {
		overhead := graphemeLen(header) + graphemeLen(threadHint) + 1 // 1 for "…"
		available := maxGraphemes - overhead
		if len(votes) == 1 {
			vote := votes[0]
			counts := voteformat.VoteCounts{
				Ja: vote.AnzahlJa, Nein: vote.AnzahlNein,
				Enthaltung: vote.AnzahlEnthaltung, Abwesend: vote.AnzahlAbwesend,
				A: vote.AnzahlA, B: vote.AnzahlB, C: vote.AnzahlC, D: vote.AnzahlD, E: vote.AnzahlE,
			}
			if voteformat.IsAuswahlVote(counts) {
				title = truncateText(title, available)
				body = title
			} else {
				// Truncate after "✅ Angenommen: " prefix
				resultEmoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
				result := voteformat.GetVoteResultText(vote.Schlussresultat)
				prefix := fmt.Sprintf("%s %s: ", resultEmoji, result)
				titleAvailable := available - graphemeLen(prefix)
				if titleAvailable > 0 {
					title = truncateText(title, titleAvailable)
				}
				body = prefix + title
			}
		} else {
			body = truncateText(title, available)
		}
		fullText = header + body + threadHint
		titleContinuation = fullBody // pass full untruncated body to replies
	}

	return &BlueskyPost{Text: fullText}, titleContinuation
}

// buildReplyPosts creates reply posts with vote details and link.
// Packs as many vote entries as fit into each reply (≤300 graphemes).
// The link is appended to the last reply.
// If titleContinuation is non-empty, it is prepended as the first entry
// (used when the root post had to truncate the title).
func buildReplyPosts(votes []zurichapi.Abstimmung, link string, titleContinuation string) []*BlueskyPost {
	linkLine := fmt.Sprintf("\n\n🔗 %s", link)

	// Build individual vote entry strings
	var entries []string

	// If the title was truncated in the root, start with the full title
	if titleContinuation != "" {
		entries = append(entries, titleContinuation)
	}
	for i, vote := range votes {
		var entry strings.Builder

		counts := voteformat.VoteCounts{
			Ja: vote.AnzahlJa, Nein: vote.AnzahlNein,
			Enthaltung: vote.AnzahlEnthaltung, Abwesend: vote.AnzahlAbwesend,
			A: vote.AnzahlA, B: vote.AnzahlB, C: vote.AnzahlC, D: vote.AnzahlD, E: vote.AnzahlE,
		}
		if len(votes) == 1 {
			// Single vote: just the counts
			entry.WriteString(voteformat.FormatVoteCounts(counts))
		} else {
			// Multi-vote: subtitle + counts
			voteTitle := voteformat.CleanVoteSubtitle(vote.Abstimmungstitel)
			if voteTitle == "" {
				voteTitle = fmt.Sprintf("Abstimmung %d", i+1)
			}
			if voteformat.IsAuswahlVote(counts) {
				// Auswahl: no ✅/❌ prefix
				entry.WriteString(fmt.Sprintf("%s\n", voteTitle))
			} else {
				voteEmoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
				entry.WriteString(fmt.Sprintf("%s %s\n", voteEmoji, voteTitle))
			}
			entry.WriteString(voteformat.FormatVoteCounts(counts))
		}

		entries = append(entries, entry.String())
	}

	// Pack entries into replies, respecting the grapheme limit.
	// The last reply gets the link appended.
	var replies []*BlueskyPost
	var currentEntries []string
	currentLen := 0

	for i, entry := range entries {
		entryLen := graphemeLen(entry)
		separatorLen := 0
		if len(currentEntries) > 0 {
			separatorLen = 2 // "\n\n" between entries
		}

		// Check if adding this entry would exceed the limit.
		// If this is the last entry, account for the link line too.
		extraLen := 0
		if i == len(entries)-1 {
			extraLen = graphemeLen(linkLine)
		}

		if currentLen+separatorLen+entryLen+extraLen > maxGraphemes && len(currentEntries) > 0 {
			// Flush current reply (without link — not the last entry yet)
			replyText := strings.Join(currentEntries, "\n\n")
			replies = append(replies, makePost(replyText, ""))
			currentEntries = nil
			currentLen = 0
		}

		if len(currentEntries) > 0 {
			currentLen += 2 // "\n\n"
		}
		currentEntries = append(currentEntries, entry)
		currentLen += entryLen
	}

	// Flush remaining entries with the link.
	// If the link doesn't fit together with the remaining entries, put it
	// in its own reply so the URL is never truncated.
	if len(currentEntries) > 0 {
		body := strings.Join(currentEntries, "\n\n")
		if graphemeLen(body+linkLine) <= maxGraphemes {
			replies = append(replies, makePost(body+linkLine, link))
		} else {
			replies = append(replies, makePost(body, ""))
			linkOnly := fmt.Sprintf("🔗 %s", link)
			replies = append(replies, makePost(linkOnly, link))
		}
	}

	return replies
}

// makePost creates a BlueskyPost with optional link facet.
func makePost(text, link string) *BlueskyPost {
	post := &BlueskyPost{Text: text}
	if link != "" {
		post.Facets = buildLinkFacets(text, link)
	}
	return post
}

// buildLinkFacets finds the URL in the text and creates a link facet for it.
func buildLinkFacets(text, url string) []bskyapi.Facet {
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

// truncateText truncates a string to fit within maxRunes graphemes, adding "…".
func truncateText(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	truncated := strings.TrimRight(string(runes[:maxRunes]), " \n")
	return truncated + "…"
}

// graphemeLen returns the number of graphemes (runes) in a string.
func graphemeLen(s string) int {
	return len([]rune(s))
}
