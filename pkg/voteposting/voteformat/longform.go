package voteformat

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// LongFormBody renders a group as one block of text, without its links: the
// headline, the label line, the title, then per vote the verdict, the counts and
// the Fraktion breakdown.
//
// It is what the platforms that publish a vote as a single post carry — the
// Instagram caption and the Facebook message — where X and Bluesky spread the
// same facts over a thread. Sharing it keeps the two from drifting apart line
// by line.
//
// tag, when non-nil, rewrites the title and the label line to mark up the names
// in them; nothing else in the body names anyone.
//
// A Wahlgeschäft is not rendered here: its only votes are attendance roll
// calls, and nothing below means anything for a headcount. Callers check
// votes.IsWahlgeschaeftGroup first.
func LongFormBody(group []votes.Vote, tag func(string) string) string {
	if tag == nil {
		tag = func(s string) string { return s }
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🗳️ %s\n\n", PostHeadline(group)))

	// The label line: what kind of business this is, who filed it, plus the
	// Abstimmungsgegenstand when a lone vote makes that meaningful.
	//
	// Tagged like the title, and for a body whose titles carry no names — Kanton
	// Zürich's never do — this is the only line that names anyone at all.
	if prefix := GroupPrefixLine(group); prefix != "" {
		sb.WriteString(tag(prefix))
		sb.WriteString("\n")
	}

	sb.WriteString(tag(CleanVoteTitle(group[0].Title)))
	sb.WriteString("\n\n")

	for i, vote := range group {
		counts := CountsOf(vote)

		if len(group) > 1 {
			// Multi-vote: include subtitle
			voteTitle := SubVoteLabel(vote, i, len(group))
			if !HasVerdict(vote) {
				sb.WriteString(voteTitle)
			} else {
				sb.WriteString(fmt.Sprintf("%s %s: %s",
					GetVoteResultEmoji(vote.Decision), GetVoteResultText(vote.Decision), voteTitle))
			}
			sb.WriteString("\n")
		} else {
			// Single vote: the ballot type when it is one worth naming, then the
			// result line. A lone threshold vote needs the label most — there is
			// no sibling beside it to make the lopsided tally look unusual.
			if label := TypeLabel(vote.Type); label != "" {
				sb.WriteString(label + "\n")
			}
			if HasVerdict(vote) {
				sb.WriteString(fmt.Sprintf("%s %s\n",
					GetVoteResultEmoji(vote.Decision), GetVoteResultText(vote.Decision)))
			}
		}

		sb.WriteString(FormatVoteCountsLong(counts))
		sb.WriteString("\n")

		if len(vote.MemberVotes) > 0 {
			if breakdown := FormatFraktionBreakdown(AggregateFraktionCounts(vote)); breakdown != "" {
				sb.WriteString("\n")
				sb.WriteString(breakdown)
				sb.WriteString("\n")
			}
		}

		if i < len(group)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
