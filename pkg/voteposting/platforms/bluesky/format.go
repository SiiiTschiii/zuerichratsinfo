package bluesky

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
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
func FormatVoteThread(group []votes.Vote, contactMapper *contacts.Mapper) []*BlueskyPost {
	if len(group) == 0 {
		return nil
	}

	// A stille Wahl gets its own single post, not a thread: there are no
	// per-vote counts to put in replies, and nothing here is misleading
	// enough to need the "Details im Thread" hint. This must run before
	// anything below touches voteformat.CountsOf/FormatVoteCounts* or a
	// verdict emoji, none of which mean anything for an uncontested election.
	if len(group) == 1 {
		if sw, ok := voteformat.AsStilleWahl(group[0]); ok {
			post := buildStilleWahlPost(group, sw)
			if contactMapper != nil {
				post.Mentions = contactMapper.FindBlueskyMentions(post.Text)
			}
			return []*BlueskyPost{post}
		}
	}

	firstVote := group[0]

	// Common components
	title := voteformat.CleanVoteTitle(firstVote.Title)

	// --- Build root post ---
	root, deferred := buildRootPost(group, title)

	// --- Build reply posts ---
	// Signatories the root had no room for lead the thread: naming them costs a
	// line here, where truncating the title to keep them would have cost the
	// subject of the vote.
	replies := buildReplyPosts(group, voteformat.SignatoryLine(deferred),
		voteformat.LinkLine(group), voteformat.GroupLink(group))

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

// buildStilleWahlPost builds the single post announcing a stille Wahl: the
// office and who was elected to it, and a link — no counts, no verdict emoji,
// no thread. See voteformat.AsStilleWahl/StilleWahlBody for why.
func buildStilleWahlPost(group []votes.Vote, sw voteformat.StilleWahl) *BlueskyPost {
	header := fmt.Sprintf("🗳️ %s\n\n", voteformat.PostHeadline(group))
	link := voteformat.LinkLine(group)
	body := voteformat.StilleWahlBody(sw)

	fullText := header + body + link
	if graphemeLen(fullText) > maxGraphemes {
		// Extremely unlikely (every Amt seen in practice is well under this
		// budget), but truncate rather than post something the API rejects.
		// truncateText appends its own "…", so that has to come out of the
		// budget too, or the truncated post still overruns by its length.
		available := maxGraphemes - graphemeLen(header) - graphemeLen(link) - graphemeLen("…")
		if available > 0 {
			body = truncateText(body, available)
		}
		fullText = header + body + link
	}

	return makePost(fullText, voteformat.GroupLink(group))
}

// buildRootPost creates the root post with header, title, result, and thread hint.
// If the title is too long, it is truncated with "…"; replies go straight to vote details.
func buildRootPost(group []votes.Vote, title string) (*BlueskyPost, []votes.Author) {
	header := fmt.Sprintf("🗳️ %s\n\n", voteformat.PostHeadline(group))
	threadHint := "\n\n👇 Details im Thread"

	// The label line: what kind of business this is, who filed it, plus the
	// Abstimmungsgegenstand when a lone vote makes that meaningful.
	//
	// Sized against what the title leaves, so a long signatory list sheds names
	// into the thread rather than eating the subject of the vote.
	body := title
	if len(group) == 1 && voteformat.HasVerdict(group[0]) {
		body = fmt.Sprintf("%s %s: %s", voteformat.GetVoteResultEmoji(group[0].Decision),
			voteformat.GetVoteResultText(group[0].Decision), title)
	}
	available := maxGraphemes - graphemeLen(header) - graphemeLen(threadHint) - graphemeLen(body) - 1
	subtitlePrefix, deferred := voteformat.FitAuthorPrefix(group, available, graphemeLen)

	if subtitlePrefix != "" {
		body = subtitlePrefix + "\n" + body
	}

	fullText := header + body + threadHint

	// Truncate title if root exceeds limit (rare, only for very long titles)
	if graphemeLen(fullText) > maxGraphemes {
		overhead := graphemeLen(header) + graphemeLen(threadHint) + 1 // 1 for "…"
		if subtitlePrefix != "" {
			overhead += graphemeLen(subtitlePrefix) + 1 // +1 for "\n"
		}
		titleRoom := maxGraphemes - overhead
		if len(group) == 1 {
			vote := group[0]
			if !voteformat.HasVerdict(vote) {
				title = truncateText(title, titleRoom)
				body = title
			} else {
				// Truncate after "✅ Angenommen: " prefix
				resultEmoji := voteformat.GetVoteResultEmoji(vote.Decision)
				result := voteformat.GetVoteResultText(vote.Decision)
				prefix := fmt.Sprintf("%s %s: ", resultEmoji, result)
				titleAvailable := titleRoom - graphemeLen(prefix)
				if titleAvailable > 0 {
					title = truncateText(title, titleAvailable)
				}
				body = prefix + title
			}
		} else {
			body = truncateText(title, titleRoom)
		}
		// Reattached for both shapes: the overhead above reserves room for it, so
		// dropping it here shortened the post and lost the line at once.
		if subtitlePrefix != "" {
			body = subtitlePrefix + "\n" + body
		}
		fullText = header + body + threadHint
	}

	return &BlueskyPost{Text: fullText}, deferred
}

// buildReplyPosts creates reply posts with vote details and link.
// Packs as many vote entries as fit into each reply (≤300 graphemes).
// The link is appended to the last reply.
func buildReplyPosts(group []votes.Vote, signatoryLine, linkLine, linkURL string) []*BlueskyPost {

	// Build individual vote entry strings
	var entries []string

	// The signatories the root could not hold come first, so a reader meets
	// them before the tallies rather than after them.
	if signatoryLine != "" {
		entries = append(entries, signatoryLine)
	}

	for i, vote := range group {
		var entry strings.Builder

		counts := voteformat.CountsOf(vote)
		if len(group) == 1 {
			// Single vote: the counts, headed by the ballot type when it is one
			// worth naming. A lone threshold vote needs that most — there is no
			// sibling beside it to make the lopsided tally look unusual.
			if label := voteformat.TypeLabel(vote.Type); label != "" {
				entry.WriteString(label + "\n")
			}
			entry.WriteString(voteformat.FormatVoteCounts(counts))
		} else {
			// Multi-vote: subtitle + counts
			voteTitle := voteformat.SubVoteLabel(vote, i, len(group))
			if !voteformat.HasVerdict(vote) {
				// Auswahl: no ✅/❌ prefix
				entry.WriteString(fmt.Sprintf("%s\n", voteTitle))
			} else {
				voteEmoji := voteformat.GetVoteResultEmoji(vote.Decision)
				entry.WriteString(fmt.Sprintf("%s %s\n", voteEmoji, voteTitle))
			}
			entry.WriteString(voteformat.FormatVoteCounts(counts))
		}

		entries = append(entries, entry.String())

		// Add Fraktion breakdown as separate entry
		if len(vote.MemberVotes) > 0 {
			fraktionCounts := voteformat.AggregateFraktionCounts(vote)
			if breakdown := voteformat.FormatFraktionBreakdown(fraktionCounts); breakdown != "" {
				entries = append(entries, breakdown)
			}
		}
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
			replies = append(replies, makePost(body+linkLine, linkURL))
		} else {
			replies = append(replies, makePost(body, ""))
			replies = append(replies, makePost(strings.TrimLeft(linkLine, "\n"), linkURL))
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
