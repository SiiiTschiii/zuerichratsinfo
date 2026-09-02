package x

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// DefaultMaxChars is the X post character limit.
// Free-tier accounts are limited to 280 characters.
// Premium accounts can post up to 2000 (override via <CHANNEL>_X_MAX_CHARS).
const DefaultMaxChars = 280

// XPost holds the formatted text for a single post in an X thread
type XPost struct {
	Text string
}

// FormatVoteThread creates an X thread for a group of related votes.
// Returns a slice of posts: [0] is the root post, [1:] are replies.
// charLimit sets the per-post character limit (e.g. 280 for free accounts, 2000 for Premium).
//
// Root post contains: header, title, result (single vote), thread hint
// Replies contain: vote details (counts per vote), link
func FormatVoteThread(group []votes.Vote, contactMapper *contacts.Mapper, charLimit int) []*XPost {
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
			return []*XPost{buildStilleWahlPost(group, sw, contactMapper, charLimit)}
		}
	}

	firstVote := group[0]

	// Common components
	title := voteformat.CleanVoteTitle(firstVote.Title)

	// Tag X handles in the title if contact mapper is provided
	if contactMapper != nil {
		title = contactMapper.TagXHandlesInText(title)
	}

	// --- Build root post ---
	root, deferred := buildRootPost(group, title, contactMapper, charLimit)

	// --- Build reply posts ---
	// Signatories the root had no room for lead the thread: naming them costs a
	// line here, where truncating the title to keep them would have cost the
	// subject of the vote.
	//
	// Tagged like the root's own line. X finds no mentions on its own, so a
	// handle only exists where this puts one — and a member shed from the root
	// for length has done nothing to deserve losing their tag.
	signatories := voteformat.SignatoryLine(deferred)
	if contactMapper != nil {
		signatories = contactMapper.TagXHandlesInText(signatories)
	}
	replies := buildReplyPosts(group, signatories, voteformat.LinkLine(group), charLimit)

	thread := make([]*XPost, 0, 1+len(replies))
	thread = append(thread, root)
	thread = append(thread, replies...)

	return thread
}

// buildStilleWahlPost builds the single post announcing a stille Wahl: the
// office and who was elected to it, and a link — no counts, no verdict emoji,
// no thread. See voteformat.AsStilleWahl/StilleWahlBody for why.
func buildStilleWahlPost(group []votes.Vote, sw voteformat.StilleWahl, contactMapper *contacts.Mapper, charLimit int) *XPost {
	if contactMapper != nil {
		sw.Name = contactMapper.TagXHandlesInText(sw.Name)
	}

	header := fmt.Sprintf("🗳️  %s\n\n", voteformat.PostHeadline(group))
	link := voteformat.LinkLine(group)
	body := voteformat.StilleWahlBody(sw)

	fullText := header + body + link
	if weightedLen(fullText) > charLimit {
		// Extremely unlikely (every Amt seen in practice is well under this
		// budget), but truncate rather than post something the API rejects.
		// truncateText appends its own "…", so that has to come out of the
		// budget too, or the truncated post still overruns by its weight.
		available := charLimit - weightedLen(header) - weightedLen(link) - weightedLen("…")
		if available > 0 {
			body = truncateText(body, available)
		}
		fullText = header + body + link
	}

	return &XPost{Text: fullText}
}

// buildRootPost creates the root post with header, title, result, and thread hint.
// If the title is too long, it is truncated with "…".
func buildRootPost(group []votes.Vote, title string, contactMapper *contacts.Mapper, charLimit int) (*XPost, []votes.Author) {
	header := fmt.Sprintf("🗳️  %s\n\n", voteformat.PostHeadline(group))
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
	available := charLimit - weightedLen(header) - weightedLen(threadHint) - weightedLen(body) - 1
	subtitlePrefix, deferred := voteformat.FitAuthorPrefix(group, available, weightedLen)

	// Tagged too, not just the title: for a body whose titles carry no names —
	// Kanton Zürich's never do — this line is the only place a politician is
	// named, so tagging the title alone would tag nobody.
	if contactMapper != nil {
		subtitlePrefix = contactMapper.TagXHandlesInText(subtitlePrefix)
	}

	if subtitlePrefix != "" {
		body = subtitlePrefix + "\n" + body
	}

	fullText := header + body + threadHint

	// Truncate title if root exceeds limit (rare, only for very long titles)
	if weightedLen(fullText) > charLimit {
		overhead := weightedLen(header) + weightedLen(threadHint) + weightedLen("…")
		if subtitlePrefix != "" {
			overhead += weightedLen(subtitlePrefix) + 1 // +1 for "\n"
		}
		available := charLimit - overhead
		if len(group) == 1 {
			vote := group[0]
			if !voteformat.HasVerdict(vote) {
				title = truncateText(title, available)
				body = title
			} else {
				resultEmoji := voteformat.GetVoteResultEmoji(vote.Decision)
				result := voteformat.GetVoteResultText(vote.Decision)
				prefix := fmt.Sprintf("%s %s: ", resultEmoji, result)
				titleAvailable := available - weightedLen(prefix)
				if titleAvailable > 0 {
					title = truncateText(title, titleAvailable)
				}
				body = prefix + title
			}
		} else {
			body = truncateText(title, available)
		}
		// Reattached for both shapes: the overhead above reserves room for it, so
		// dropping it here shortened the post and lost the line at once.
		if subtitlePrefix != "" {
			body = subtitlePrefix + "\n" + body
		}
		fullText = header + body + threadHint
	}

	return &XPost{Text: fullText}, deferred
}

// buildReplyPosts creates reply posts with vote details and link.
// Packs as many vote entries as fit into each reply (≤charLimit).
// The link is appended to the last reply.
func buildReplyPosts(group []votes.Vote, signatoryLine, linkLine string, charLimit int) []*XPost {

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
			entry.WriteString(voteformat.FormatVoteCountsLong(counts))
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
			entry.WriteString(voteformat.FormatVoteCountsLong(counts))
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

	// Pack entries into replies, respecting the character limit.
	// The last reply gets the link appended.
	var replies []*XPost
	var currentEntries []string
	currentLen := 0

	for i, entry := range entries {
		entryLen := weightedLen(entry)
		separatorLen := 0
		if len(currentEntries) > 0 {
			separatorLen = 2 // "\n\n" between entries
		}

		// Check if adding this entry would exceed the limit.
		// If this is the last entry, account for the link line too.
		extraLen := 0
		if i == len(entries)-1 {
			extraLen = weightedLen(linkLine)
		}

		if currentLen+separatorLen+entryLen+extraLen > charLimit && len(currentEntries) > 0 {
			// Flush current reply (without link — not the last entry yet)
			replyText := strings.Join(currentEntries, "\n\n")
			replies = append(replies, &XPost{Text: replyText})
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
		if weightedLen(body+linkLine) <= charLimit {
			replies = append(replies, &XPost{Text: body + linkLine})
		} else {
			replies = append(replies, &XPost{Text: body})
			replies = append(replies, &XPost{Text: strings.TrimLeft(linkLine, "\n")})
		}
	}

	return replies
}

// truncateText truncates a string to X's weighted length maxLen, adding "…".
// Callers subtract the ellipsis from maxLen themselves.
func truncateText(s string, maxLen int) string {
	truncated := strings.TrimRight(truncateToWeighted(s, maxLen), " \n")
	return truncated + "…"
}
