package x

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// maxChars is the X post character limit (using X Premium's relaxed limit)
const maxChars = 2000

// XPost holds the formatted text for a single post in an X thread
type XPost struct {
	Text string
}

// FormatVoteThread creates an X thread for a group of related votes.
// Returns a slice of posts: [0] is the root post, [1:] are replies.
//
// Root post contains: header, title, result (single vote), thread hint
// Replies contain: vote details (counts per vote), link
func FormatVoteThread(votes []zurichapi.Abstimmung, contactMapper *contacts.Mapper) []*XPost {
	if len(votes) == 0 {
		return nil
	}

	firstVote := votes[0]

	// Common components
	date := voteformat.FormatVoteDate(firstVote.SitzungDatum)
	title := voteformat.SelectBestTitle(firstVote.TraktandumTitel, firstVote.GeschaeftTitel)
	title = voteformat.CleanVoteTitle(title)

	// Tag X handles in the title if contact mapper is provided
	if contactMapper != nil {
		title = contactMapper.TagXHandlesInText(title)
	}

	// Generate the link
	var link string
	if voteformat.IsGenericAntragTitle(firstVote.TraktandumTitel) {
		link = voteformat.GenerateGeschaeftLink(firstVote.GeschaeftGuid)
	} else if len(votes) > 1 {
		link = voteformat.GenerateTraktandumLink(firstVote.SitzungGuid, firstVote.TraktandumGuid)
	} else {
		link = voteformat.GenerateVoteLink(firstVote.OBJGUID)
	}

	// --- Build root post ---
	root := buildRootPost(votes, date, title)

	// --- Build reply posts ---
	replies := buildReplyPosts(votes, link)

	thread := make([]*XPost, 0, 1+len(replies))
	thread = append(thread, root)
	thread = append(thread, replies...)

	return thread
}

// buildRootPost creates the root post with header, title, result, and thread hint.
// If the title is too long, it is truncated with "…".
func buildRootPost(votes []zurichapi.Abstimmung, date, title string) *XPost {
	header := fmt.Sprintf("🗳️  Gemeinderat | Abstimmung vom %s\n\n", date)
	threadHint := "\n\n👇 Details im Thread"

	var body string
	if len(votes) == 1 {
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
		body = title
	}

	fullText := header + body + threadHint

	// Truncate title if root exceeds limit (rare, only for very long titles)
	if len(fullText) > maxChars {
		overhead := len(header) + len(threadHint) + len("…")
		available := maxChars - overhead
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
				resultEmoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
				result := voteformat.GetVoteResultText(vote.Schlussresultat)
				prefix := fmt.Sprintf("%s %s: ", resultEmoji, result)
				titleAvailable := available - len(prefix)
				if titleAvailable > 0 {
					title = truncateText(title, titleAvailable)
				}
				body = prefix + title
			}
		} else {
			body = truncateText(title, available)
		}
		fullText = header + body + threadHint
	}

	return &XPost{Text: fullText}
}

// buildReplyPosts creates reply posts with vote details and link.
// Packs as many vote entries as fit into each reply (≤maxChars).
// The link is appended to the last reply.
func buildReplyPosts(votes []zurichapi.Abstimmung, link string) []*XPost {
	linkLine := fmt.Sprintf("\n\n🔗 %s", link)

	// Build individual vote entry strings
	var entries []string

	for i, vote := range votes {
		var entry strings.Builder

		counts := voteformat.VoteCounts{
			Ja: vote.AnzahlJa, Nein: vote.AnzahlNein,
			Enthaltung: vote.AnzahlEnthaltung, Abwesend: vote.AnzahlAbwesend,
			A: vote.AnzahlA, B: vote.AnzahlB, C: vote.AnzahlC, D: vote.AnzahlD, E: vote.AnzahlE,
		}
		if len(votes) == 1 {
			// Single vote: just the counts
			entry.WriteString(voteformat.FormatVoteCountsLong(counts))
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
			entry.WriteString(voteformat.FormatVoteCountsLong(counts))
		}

		entries = append(entries, entry.String())
	}

	// Pack entries into replies, respecting the character limit.
	// The last reply gets the link appended.
	var replies []*XPost
	var currentEntries []string
	currentLen := 0

	for i, entry := range entries {
		entryLen := len(entry)
		separatorLen := 0
		if len(currentEntries) > 0 {
			separatorLen = 2 // "\n\n" between entries
		}

		// Check if adding this entry would exceed the limit.
		// If this is the last entry, account for the link line too.
		extraLen := 0
		if i == len(entries)-1 {
			extraLen = len(linkLine)
		}

		if currentLen+separatorLen+entryLen+extraLen > maxChars && len(currentEntries) > 0 {
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
		if len(body+linkLine) <= maxChars {
			replies = append(replies, &XPost{Text: body + linkLine})
		} else {
			replies = append(replies, &XPost{Text: body})
			replies = append(replies, &XPost{Text: fmt.Sprintf("🔗 %s", link)})
		}
	}

	return replies
}

// truncateText truncates a string to fit within maxLen bytes, adding "…".
func truncateText(s string, maxLen int) string {
	runes := []rune(s)
	// Approximate: trim runes until byte length fits
	for len(string(runes)) > maxLen && len(runes) > 0 {
		runes = runes[:len(runes)-1]
	}
	truncated := strings.TrimRight(string(runes), " \n")
	return truncated + "…"
}

// FormatVoteGroupPost creates a formatted X post for a group of related votes.
// Deprecated: use FormatVoteThread for thread-aware formatting.
// Kept for backward compatibility during migration.
func FormatVoteGroupPost(votes []zurichapi.Abstimmung, contactMapper *contacts.Mapper) string {
	thread := FormatVoteThread(votes, contactMapper)
	if len(thread) == 0 {
		return ""
	}
	var parts []string
	for _, post := range thread {
		parts = append(parts, post.Text)
	}
	return strings.Join(parts, "\n\n")
}

// FormatVotePost creates a formatted X post for a single vote.
// Deprecated: use FormatVoteThread for thread-aware formatting.
func FormatVotePost(vote *zurichapi.Abstimmung, contactMapper *contacts.Mapper) string {
	return FormatVoteGroupPost([]zurichapi.Abstimmung{*vote}, contactMapper)
}
