package voteformat

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

var geschaeftNumberRegex = regexp.MustCompile(`^\d+/\d+\s+`)
var geschaeftNumberUnderscoreRegex = regexp.MustCompile(`^\d+_\d+\s+`)

// FormatVoteDate renders a sitting date as DD.MM.YYYY.
// An unknown (zero) date renders as "" rather than year 1.
func FormatVoteDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02.01.2006")
}

// GetVoteResultEmoji returns the appropriate emoji for a vote result
func GetVoteResultEmoji(result string) string {
	result = strings.TrimSpace(strings.ToLower(result))
	if strings.Contains(result, "angenommen") || result == "ja" {
		return "✅"
	}
	return "❌"
}

// GetVoteResultText returns the text for a vote result
func GetVoteResultText(result string) string {
	result = strings.TrimSpace(strings.ToLower(result))
	if strings.Contains(result, "angenommen") || result == "ja" {
		return "Angenommen"
	}
	return "Abgelehnt"
}

// IsDecisionConsistent reports whether a vote's stated decision is consistent
// with its raw counts. Returns false when the source declares "Ja"/"Angenommen"
// but the Nein count exceeds the Ja count, indicating stale or erroneous data
// that should block posting.
// Auswahl (A/B/C/D) results and nil counts are always considered consistent.
func IsDecisionConsistent(decision string, ja, nein *int) bool {
	if ja == nil || nein == nil {
		return true
	}
	lower := strings.TrimSpace(strings.ToLower(decision))
	if strings.HasPrefix(lower, "auswahl") {
		return true
	}
	statedAccepted := strings.Contains(lower, "angenommen") || lower == "ja"
	return !statedAccepted || *nein <= *ja
}

// CleanVoteTitle removes newlines, extra whitespace, and Geschäft number from titles
func CleanVoteTitle(title string) string {
	// Replace newlines and carriage returns with spaces
	title = strings.ReplaceAll(title, "\r\n", " ")
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")

	// Replace multiple spaces with single space
	parts := strings.Fields(title)
	title = strings.Join(parts, " ")

	// Strip Geschäft number from the beginning (e.g., "2024/431 " or "2025/84 ")
	// Pattern: number/number followed by space
	title = geschaeftNumberRegex.ReplaceAllString(title, "")

	return title
}

// CleanVoteSubtitle cleans up vote subtitles (Abstimmungstitel)
// Similar to CleanVoteTitle but keeps it shorter
func CleanVoteSubtitle(subtitle string) string {
	// Replace newlines and carriage returns with spaces
	subtitle = strings.ReplaceAll(subtitle, "\r\n", " ")
	subtitle = strings.ReplaceAll(subtitle, "\n", " ")
	subtitle = strings.ReplaceAll(subtitle, "\r", " ")

	// Replace multiple spaces with single space
	parts := strings.Fields(subtitle)
	subtitle = strings.Join(parts, " ")

	// Strip Geschäft number patterns:
	// Pattern 1: "2025/369 " with slash
	// Pattern 2: "2025_0369 " with underscore
	subtitle = geschaeftNumberRegex.ReplaceAllString(subtitle, "")
	subtitle = geschaeftNumberUnderscoreRegex.ReplaceAllString(subtitle, "")

	return subtitle
}

// FormatVoteCount formats a nullable int pointer
func FormatVoteCount(count *int) string {
	if count == nil {
		return "0"
	}
	return fmt.Sprintf("%d", *count)
}

// VoteCounts holds all possible vote count fields from the API.
// Standard Ja/Nein votes use Ja/Nein/Enthaltung/Abwesend.
// "Gleichgerichtete Anträge mit N Optionen" votes use A/B/C/D/E.
type VoteCounts struct {
	Ja            *int
	Nein          *int
	Enthaltung    *int
	Abwesend      *int
	A, B, C, D, E *int
}

// CountsOf reads the totals off a vote. Totals are always taken from the
// source's own count fields and never summed from MemberVotes: sources derive
// the two separately and are allowed to disagree.
func CountsOf(v votes.Vote) VoteCounts {
	return VoteCounts{
		Ja: v.Yes, Nein: v.No, Enthaltung: v.Abstention, Abwesend: v.Absent,
		A: v.ChoiceA, B: v.ChoiceB, C: v.ChoiceC, D: v.ChoiceD, E: v.ChoiceE,
	}
}

// IsAuswahlVote returns true when the vote used the A/B/C/D/E option format
// ("Gleichgerichtete Anträge mit N Optionen") rather than the standard Ja/Nein format.
// In this case result emojis (✅/❌) should be omitted because the outcome is
// "Auswahl A/B/…", not accepted/rejected.
func IsAuswahlVote(c VoteCounts) bool {
	for _, f := range []*int{c.A, c.B, c.C, c.D, c.E} {
		if f != nil && *f > 0 {
			return true
		}
	}
	return false
}

// IsUnsupportedVoteType returns true when all active voter count fields are nil or zero.
// This indicates a vote format we don't know how to render (neither standard Ja/Nein
// nor Auswahl A/B/C/D). Callers should log a warning and skip posting such votes.
func IsUnsupportedVoteType(c VoteCounts) bool {
	fields := []*int{c.Ja, c.Nein, c.Enthaltung, c.A, c.B, c.C, c.D, c.E}
	for _, f := range fields {
		if f != nil && *f > 0 {
			return false
		}
	}
	return true
}

// FormatVoteCounts returns the 📊 summary line for a vote.
// Detects Auswahl A/B/C/D votes vs standard Ja/Nein automatically.
// Call IsUnsupportedVoteType first if you need to guard against unknown formats.
func FormatVoteCounts(c VoteCounts) string {
	abwesend := FormatVoteCount(c.Abwesend)

	// Check if any Auswahl option has votes
	auswahlPtrs := []*int{c.A, c.B, c.C, c.D, c.E}
	letters := []string{"A", "B", "C", "D", "E"}
	var auswahlParts []string
	for i, f := range auswahlPtrs {
		if f != nil && *f > 0 {
			auswahlParts = append(auswahlParts, fmt.Sprintf("%s: %d", letters[i], *f))
		}
	}
	if len(auswahlParts) > 0 {
		return fmt.Sprintf("📊 %s | Abw. %s", strings.Join(auswahlParts, " | "), abwesend)
	}

	// Standard Ja/Nein vote (short labels for space-constrained platforms)
	return fmt.Sprintf("📊 %s Ja | %s Nein | %s Enth. | %s Abw.",
		FormatVoteCount(c.Ja),
		FormatVoteCount(c.Nein),
		FormatVoteCount(c.Enthaltung),
		abwesend)
}

// FormatVoteCountsLong is like FormatVoteCounts but uses full German label names
// ("Enthaltung", "Abwesend") suited for platforms without a tight character limit.
func FormatVoteCountsLong(c VoteCounts) string {
	abwesend := FormatVoteCount(c.Abwesend)

	auswahlPtrs := []*int{c.A, c.B, c.C, c.D, c.E}
	letters := []string{"A", "B", "C", "D", "E"}
	var auswahlParts []string
	for i, f := range auswahlPtrs {
		if f != nil && *f > 0 {
			auswahlParts = append(auswahlParts, fmt.Sprintf("%s: %d", letters[i], *f))
		}
	}
	if len(auswahlParts) > 0 {
		return fmt.Sprintf("📊 %s | Abwesend %s", strings.Join(auswahlParts, " | "), abwesend)
	}

	return fmt.Sprintf("📊 %s Ja | %s Nein | %s Enthaltung | %s Abwesend",
		FormatVoteCount(c.Ja),
		FormatVoteCount(c.Nein),
		FormatVoteCount(c.Enthaltung),
		abwesend)
}

// IsSchlussabstimmung returns true if the Abstimmungstitel contains "Schlussabstimmung" (case-insensitive).
func IsSchlussabstimmung(abstimmungstitel string) bool {
	return strings.Contains(strings.ToLower(abstimmungstitel), "schlussabstimmung")
}

// SingleVoteSubtitlePrefix returns the cleaned Abstimmungsgegenstand text to prepend
// to a single-vote post, or "" if no prefix should be added.
// A prefix is added only when: the Abstimmungstitel is non-empty AND does not
// contain "Schlussabstimmung" (case-insensitive).
func SingleVoteSubtitlePrefix(abstimmungstitel string) string {
	if abstimmungstitel == "" {
		return ""
	}
	if IsSchlussabstimmung(abstimmungstitel) {
		return ""
	}
	cleaned := CleanVoteSubtitle(abstimmungstitel)
	if cleaned == "" {
		return ""
	}
	return cleaned
}

// GroupLink returns the URL a post about this group should point at: the page
// covering the whole group when several votes are posted together, otherwise
// the single vote's own page. Sources decide what those two URLs are; this only
// picks between them.
func GroupLink(group []votes.Vote) string {
	if len(group) == 0 {
		return ""
	}
	if len(group) > 1 && group[0].GroupURL != "" {
		return group[0].GroupURL
	}
	return group[0].SourceURL
}

// defaultBodyLabel is used when a source did not name the body. It is the Stadt
// Zürich chamber because that is the only jurisdiction whose posts predate the
// Body field; a post with a wrong-but-plausible label would be worse than this
// only if some other body could reach here, and none can.
const defaultBodyLabel = "Gemeinderat"

// BodyLabel returns the chamber name to put in a post header.
//
// Several bodies post to one account, so this is what tells a reader whether
// they are looking at a city or a cantonal vote. It is not decoration.
func BodyLabel(group []votes.Vote) string {
	if len(group) > 0 && group[0].Body != "" {
		return group[0].Body
	}
	return defaultBodyLabel
}

// PostHeadline is the first line of a post: which chamber voted, and when.
//
// The date clause is dropped when the date is unknown. Votes with an
// unparseable date are deliberately kept rather than discarded — silently
// dropping a vote is worse than posting one whose date we could not read — so
// this case is reachable, and "Abstimmung vom " trailing into nothing is not an
// acceptable way to render it.
//
// Callers prepend their own emoji, because the platforms do not agree on the
// spacing after it and that difference is already baked into published posts.
func PostHeadline(group []votes.Vote) string {
	body := BodyLabel(group)
	if len(group) == 0 || group[0].Date.IsZero() {
		return body + " | Abstimmung"
	}
	return body + " | Abstimmung vom " + FormatVoteDate(group[0].Date)
}

// LinkLine returns the trailing block of a post: the link, plus the source
// credit when the data's licence requires one.
//
// The two are built together so that the platforms' character-budget
// arithmetic accounts for the credit. Appending it afterwards would make posts
// overflow exactly on the votes that need it.
func LinkLine(group []votes.Vote) string {
	link := GroupLink(group)
	if link == "" {
		return ""
	}
	out := "\n\n🔗 " + link
	if len(group) > 0 && group[0].Attribution != "" {
		out += "\n" + group[0].Attribution
	}
	return out
}

// SubVoteLabel names one vote inside a multi-vote group, so a reader can tell
// the entries of a thread apart.
//
// Sources vary in how much help they give. Stadt Zürich titles each vote
// ("Schlussabstimmung über die Dispositivziffer 1"). Kanton Zürich publishes no
// per-vote title at all — its title field repeats the business matter — so the
// time of day is the only thing distinguishing them, and it is worth showing
// because the official archive lists the sitting's votes chronologically: the
// time tells a reader which entry to open.
//
// Falling back to an ordinal is the last resort. It says nothing except that
// these are different votes.
func SubVoteLabel(v votes.Vote, index int) string {
	if title := CleanVoteSubtitle(v.Subtitle); title != "" {
		return title
	}
	if v.DateIsExact {
		return "Abstimmung " + v.Date.Format("15:04")
	}
	return fmt.Sprintf("Abstimmung %d", index+1)
}
