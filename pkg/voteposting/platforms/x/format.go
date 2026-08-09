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

	firstVote := group[0]

	// Common components
	title := voteformat.CleanVoteTitle(firstVote.Title)

	// Tag X handles in the title if contact mapper is provided
	if contactMapper != nil {
		title = contactMapper.TagXHandlesInText(title)
	}

	// --- Build root post ---
	root := buildRootPost(group, title, charLimit)

	// --- Build reply posts ---
	replies := buildReplyPosts(group, voteformat.LinkLine(group), charLimit)

	thread := make([]*XPost, 0, 1+len(replies))
	thread = append(thread, root)
	thread = append(thread, replies...)

	return thread
}

// buildRootPost creates the root post with header, title, result, and thread hint.
// If the title is too long, it is truncated with "…".
func buildRootPost(group []votes.Vote, title string, charLimit int) *XPost {
	header := fmt.Sprintf("🗳️  %s\n\n", voteformat.PostHeadline(group))
	threadHint := "\n\n👇 Details im Thread"

	// For single-vote non-Schlussabstimmung, prepend the Abstimmungsgegenstand
	var subtitlePrefix string
	if len(group) == 1 {
		subtitlePrefix = voteformat.SingleVoteSubtitlePrefix(group[0].Subtitle)
	}

	var body string
	if len(group) == 1 {
		vote := group[0]
		counts := voteformat.CountsOf(vote)
		if voteformat.IsAuswahlVote(counts) {
			body = title
		} else {
			resultEmoji := voteformat.GetVoteResultEmoji(vote.Decision)
			result := voteformat.GetVoteResultText(vote.Decision)
			body = fmt.Sprintf("%s %s: %s", resultEmoji, result, title)
		}
		if subtitlePrefix != "" {
			body = subtitlePrefix + "\n" + body
		}
	} else {
		body = title
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
			counts := voteformat.CountsOf(vote)
			if voteformat.IsAuswahlVote(counts) {
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
			if subtitlePrefix != "" {
				body = subtitlePrefix + "\n" + body
			}
		} else {
			body = truncateText(title, available)
		}
		fullText = header + body + threadHint
	}

	return &XPost{Text: fullText}
}

// buildReplyPosts creates reply posts with vote details and link.
// Packs as many vote entries as fit into each reply (≤charLimit).
// The link is appended to the last reply.
func buildReplyPosts(group []votes.Vote, linkLine string, charLimit int) []*XPost {

	// Build individual vote entry strings
	var entries []string

	for i, vote := range group {
		var entry strings.Builder

		counts := voteformat.CountsOf(vote)
		if len(group) == 1 {
			// Single vote: just the counts
			entry.WriteString(voteformat.FormatVoteCountsLong(counts))
		} else {
			// Multi-vote: subtitle + counts
			voteTitle := voteformat.SubVoteLabel(vote, i, len(group))
			if voteformat.IsAuswahlVote(counts) {
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
			fraktionCounts := voteformat.AggregateFraktionCounts(vote.MemberVotes)
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
