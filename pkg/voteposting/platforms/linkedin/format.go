package linkedin

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/linkedinapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// maxChars is the LinkedIn commentary limit.
const maxChars = linkedinapi.MaxChars

// LinkedInPost is one LinkedIn post. Unlike X and Bluesky a vote group is never
// threaded: 3,000 characters hold every count and the Fraktion breakdown, so
// the whole story is one post.
type LinkedInPost struct {
	Text string
	// LinkURL is the page LinkedIn unfurls into a preview card.
	LinkURL string
}

// FormatVotePost renders a group of related votes as one LinkedIn post.
//
// No politician is tagged. A LinkedIn mention needs the member's URN, which the
// Consumer API cannot look up from a profile URL, so names stay plain text
// rather than half the tags being guesses.
func FormatVotePost(group []votes.Vote) *LinkedInPost {
	if len(group) == 0 {
		return nil
	}

	header := "🗳️ " + voteformat.PostHeadline(group)
	links := voteformat.LinkLine(group)
	tags := hashtags(group)
	post := &LinkedInPost{LinkURL: firstLink(group)}

	// A Wahlgeschäft reports no counts and no verdict, only that the business
	// was before the chamber; see voteformat.WahlgeschaeftBody.
	if votes.IsWahlgeschaeftGroup(group) {
		post.Text = fit(header, voteformat.WahlgeschaeftBody(group), nil, links, tags)
		return post
	}

	title := voteformat.CleanVoteTitle(group[0].Title)
	if len(group) == 1 && voteformat.HasVerdict(group[0]) {
		title = fmt.Sprintf("%s %s: %s", voteformat.GetVoteResultEmoji(group[0].Decision),
			voteformat.GetVoteResultText(group[0].Decision), title)
	}
	if prefix := voteformat.GroupPrefixLine(group); prefix != "" {
		title = prefix + "\n" + title
	}

	post.Text = fit(header, title, voteSections(group, true), links, tags)
	if runeLen(post.Text) > maxChars {
		// The Fraktion tables go first: they are the long part, and the totals
		// they break down are still there.
		sections := voteSections(group, false)
		post.Text = fit(header, title, sections, links, tags)
		// Then trailing votes, with a line saying so. The link block still
		// leads to all of them.
		for n := len(sections) - 1; runeLen(post.Text) > maxChars && n >= 1; n-- {
			shown := append(append([]string{}, sections[:n]...),
				fmt.Sprintf("… und %d weitere Abstimmungen (siehe Links)", len(sections)-n))
			post.Text = fit(header, title, shown, links, tags)
		}
	}
	return post
}

// voteSections is one block per vote: its heading (when several share the
// post), the long-form counts, and optionally the per-Fraktion breakdown.
func voteSections(group []votes.Vote, withBreakdown bool) []string {
	var out []string
	for i, v := range group {
		var b strings.Builder
		if len(group) > 1 {
			label := voteformat.SubVoteLabel(v, i, len(group))
			if voteformat.HasVerdict(v) {
				label = voteformat.GetVoteResultEmoji(v.Decision) + " " + label
			}
			b.WriteString(label + "\n")
		} else if label := voteformat.TypeLabel(v.Type); label != "" {
			b.WriteString(label + "\n")
		}
		b.WriteString(voteformat.FormatVoteCountsLong(voteformat.CountsOf(v)))
		if withBreakdown && len(v.MemberVotes) > 0 {
			if table := voteformat.FormatFraktionBreakdown(voteformat.AggregateFraktionCounts(v)); table != "" {
				b.WriteString("\n\n" + table)
			}
		}
		out = append(out, b.String())
	}
	return out
}

// fit joins the parts and, when that overruns, gives way in a fixed order: the
// title is truncated, since the counts and links are what the post is for. The
// links and hashtags are never cut — a URL cut in half is not a link.
func fit(header, title string, sections []string, links, tags string) string {
	tail := links
	if tags != "" {
		tail += "\n\n" + tags
	}
	middle := strings.Join(sections, "\n\n")
	build := func(title string) string {
		parts := []string{header, title}
		if middle != "" {
			parts = append(parts, middle)
		}
		return strings.Join(parts, "\n\n") + tail
	}

	text := build(title)
	over := runeLen(text) - maxChars
	if over <= 0 {
		return text
	}
	// Truncating the title also adds the "…".
	room := runeLen(title) - over - 1
	if room < 1 {
		return text // the title cannot absorb it; the caller drops detail instead
	}
	return build(truncate(title, room))
}

// firstLink is the page to preview: the first entry of the link block, which is
// the Geschäft permalink where the source has one.
func firstLink(group []votes.Vote) string {
	if links := voteformat.GroupLinks(group); len(links) > 0 {
		return links[0].URL
	}
	return ""
}

// hashtags names the topic and the chamber: "#Abstimmung #GemeinderatZürich".
// The chamber tag is built from the body label ("Gemeinderat ZH"), and omitted
// where a source supplies none rather than guessed.
func hashtags(group []votes.Vote) string {
	topic := "#Abstimmung"
	if votes.IsWahlgeschaeftGroup(group) {
		topic = "#Wahl"
	}
	body := strings.TrimSuffix(voteformat.BodyLabel(group), " ZH")
	if body == voteformat.BodyLabel(group) || body == "" {
		return topic
	}
	return topic + " #" + strings.ReplaceAll(body, " ", "") + "Zürich"
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return strings.TrimRight(string(runes[:maxRunes]), " \n") + "…"
}

func runeLen(s string) int { return len([]rune(s)) }
