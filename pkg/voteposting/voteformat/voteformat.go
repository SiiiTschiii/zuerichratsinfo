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

// GroupPrefixLine returns the label line a post carries above its title: what
// kind of business this is, and — for a lone vote — which question within it was
// put and of what kind.
//
// The business type is worth the words because it changes what the vote means.
// "Keine Baubewilligung mehr für Pergolen in Gärten" reads as a decision taken;
// "Postulat" in front of it reads as what it is, a demand addressed to the
// Regierungsrat. The difference between a government bill, a parliamentary
// demand and an individual initiative is exactly what a reader outside the
// building cannot infer from a tally.
//
// It goes on its own line rather than into the title because the title already
// follows a verdict on single-vote posts: folding it in produced "✅ Angenommen:
// Vorlage: Rahmenkredit …", two colons deep before the subject appears.
//
// Bodies whose source reports no business type are unaffected — Stadt Zürich's
// PARIS adapter never fills Affair.Type, so its posts keep exactly the line they
// had.
func GroupPrefixLine(group []votes.Vote) string {
	if len(group) == 0 {
		return ""
	}

	prefix := strings.TrimSpace(group[0].Affair.Type)

	// The Abstimmungsgegenstand identifies a question within the business, which
	// is meaningless when several votes share the post; the per-vote headings
	// carry it there instead.
	//
	// The ballot type is deliberately absent from this line. It belongs with the
	// counts, because what it explains is why they read "Zustimmung/ohne"
	// instead of "Ja/Nein" — and a line reading "Vorlage · Ausgabenbremse" put
	// two different kinds of fact side by side as though they were one, the
	// business on the left and the ballot on the right.
	if len(group) == 1 {
		if sub := singleVoteSubtitle(group[0].Subtitle); sub != "" {
			if prefix == "" {
				return sub
			}
			prefix += " · " + sub
		}
	}
	return prefix
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

	// Type is the source's vote-type label, carried alongside the counts
	// because the same four numbers mean different things depending on it: a
	// quorum vote's Nein is structurally 0 and its Abwesend is "did not
	// support", not "was not there".
	Type string
}

// CountsOf reads the totals off a vote. Totals are always taken from the
// source's own count fields and never summed from MemberVotes: sources derive
// the two separately and are allowed to disagree.
func CountsOf(v votes.Vote) VoteCounts {
	return VoteCounts{
		Ja: v.Yes, Nein: v.No, Enthaltung: v.Abstention, Abwesend: v.Absent,
		A: v.ChoiceA, B: v.ChoiceB, C: v.ChoiceC, D: v.ChoiceD, E: v.ChoiceE,
		Type: v.Type,
	}
}

// HasVerdict reports whether a post may state an accepted/rejected outcome for
// this vote. When it is false the formatters print the title and the counts and
// claim nothing about what parliament decided.
//
// Two cases have no verdict to state. An Auswahl vote's outcome is "Auswahl
// A/B/…", not accepted or rejected. And a vote whose source reports no decision
// has none that we are entitled to publish: an outcome is never inferred from
// the counts, because inferring it is only safe when Ja and Nein are the two
// sides of the same question. In a quorum vote they are not — Nein is
// structurally always 0, so every such vote would come out "Angenommen",
// including the ones that failed to reach their threshold. That threshold is
// not published anywhere we can read, and it differs by which procedure the
// vote serves, which type_de does not say. So the honest rendering is silence.
//
// Kanton Zürich currently reports no decision at all, for any vote. If
// OpenParlData starts populating it, verdicts return on their own.
func HasVerdict(v votes.Vote) bool {
	if IsAuswahlVote(CountsOf(v)) {
		return false
	}
	return strings.TrimSpace(v.Decision) != ""
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

// voteTypeQuorum is the value both sources publish for a quorum vote.
const voteTypeQuorum = "Quorum"

// voteTypeAusgabenbremse is the spending brake, which the Kantonsrat's own
// archive names separately. OpenParlData folds it into "Quorum".
const voteTypeAusgabenbremse = "Ausgabenbremse"

// isThresholdVoteType reports whether a type is decided against a threshold
// rather than by comparing Ja to Nein. Both kinds are counted the same way; they
// differ only in what a post calls them.
func isThresholdVoteType(voteType string) bool {
	switch strings.TrimSpace(voteType) {
	case voteTypeQuorum, voteTypeAusgabenbremse:
		return true
	}
	return false
}

// IsQuorumVote reports whether these counts belong to a threshold vote, which is
// counted and rendered differently everywhere it appears.
func IsQuorumVote(c VoteCounts) bool {
	return isThresholdVoteType(c.Type) && !IsAuswahlVote(c)
}

// formatQuorumCounts renders the summary line for a quorum vote, or "" when the
// vote is not one.
//
// A quorum vote counts supporters against a threshold and has no Nein option,
// so the standard four-part line prints "0 Nein | 0 Enth." for positions nobody
// could have taken, and files every non-supporter under "Abwesend" — which the
// official record shows is wrong for most of them (6 truly absent against 46
// present and not voting, where the API reports 52).
//
// Two honest numbers instead: how many supported, and how many did not. Both
// stay true whether or not the source ever separates absence from abstention,
// and the line is shorter than the standard one, so neither platform needs a
// shortened variant.
func formatQuorumCounts(c VoteCounts) string {
	if !IsQuorumVote(c) {
		return ""
	}
	_, ohne := QuorumTally(c)
	return fmt.Sprintf("📊 %s Zustimmungen | %d ohne Zustimmung",
		FormatVoteCount(c.Ja), ohne)
}

// QuorumTally splits a quorum vote into the two numbers that are true of it:
// how many supported, and how many did not.
//
// Nein is folded into the second rather than read as a third column. It used to
// be 0 on every quorum vote the source served, but that is not a property of
// quorum votes — it was a property of which ones carried a type. The 17.08.2026
// Ausgabenbremse is a binary ballot with a threshold and reports 141 Ja to 1
// Nein, and OpenParlData is separately considering mapping the Kantonsrat's
// "Nicht abgestimmt" onto no instead of absent (upstream #179). Either way both
// buckets mean "did not support", which is exactly what the label says.
//
// It is exported and shared because the caption, the Fraktion table and the
// generated image all state this number, and they must not be able to disagree:
// while every quorum vote had Nein=0 the image's own reading — absent alone —
// looked correct, and it silently stopped being correct the moment one didn't.
func QuorumTally(c VoteCounts) (support, without int) {
	return deref(c.Ja), deref(c.Nein) + deref(c.Abwesend)
}

// deref reads a nullable count, treating "not reported" as zero.
func deref(n *int) int {
	if n == nil {
		return 0
	}
	return *n
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
	if line := formatQuorumCounts(c); line != "" {
		return line
	}
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
	if line := formatQuorumCounts(c); line != "" {
		return line
	}
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

func singleVoteSubtitle(abstimmungstitel string) string {
	if abstimmungstitel == "" {
		return ""
	}
	if IsSchlussabstimmung(abstimmungstitel) {
		return ""
	}
	return CleanVoteSubtitle(abstimmungstitel)
}

// typeLabels names the vote types worth surfacing to a reader, keyed by the
// value the sources publish. PARIS (Abstimmungstyp) and OpenParlData (type_de)
// share this vocabulary, so one table serves both bodies.
//
// It is an allowlist rather than a passthrough because most of the vocabulary
// is either uninformative or already visible in the counts: "Normal" describes
// the overwhelming majority and says nothing, and "Gleichgerichtete Anträge mit
// 4 Optionen" restates an Auswahl vote that already renders as A/B/C/D. Only
// types whose tally would otherwise be read wrongly earn a label.
var typeLabels = map[string]string{
	// A quorum vote counts supporters against a threshold and has no Nein
	// option at all, so everyone not supporting it lands in Abwesend. Unlabelled,
	// "129 Ja | 0 Nein | 51 Abw." reads as near-unanimous agreement rather than
	// as a procedural vote most of the opposition deliberately sits out.
	"Quorum": "Quorum",
	// The spending brake, which needs 91 of 180 votes whatever the opposition
	// does. It is counted like any other threshold vote but named, because
	// "Ausgabenbremse" tells a reader what the threshold was for where "Quorum"
	// only tells them that one existed.
	"Ausgabenbremse": "Ausgabenbremse",
	// A knockout round between more than two competing proposals. Kanton Zürich
	// reports these with no aggregate counts at all, so they are rejected before
	// posting today (see IsUnsupportedVoteType); the label is here so they read
	// correctly if that is ever fixed upstream.
	"Cup-Abstimmung": "Cup-Abstimmung",
}

// TypeLabel returns the reader-facing label for a vote type, or "" when the
// type does not warrant one.
func TypeLabel(voteType string) string {
	return typeLabels[strings.TrimSpace(voteType)]
}

// auswahlTypePrefix begins the type both sources use for multi-option votes,
// which carry an option count in their name ("… mit 3 Optionen", "… mit 4
// Optionen"), so the whole family has to be matched by prefix.
const auswahlTypePrefix = "Gleichgerichtete Anträge"

// handledVoteTypes are the types whose counts the formatters render correctly.
// It is deliberately narrower than typeLabels: rendering a type correctly and
// having a name for it are different things.
var handledVoteTypes = map[string]bool{
	// Ja/Nein/Enthaltung/Abwesend, the overwhelming majority.
	"Normal": true,
	// Supporters counted against a threshold, no Nein option.
	"Quorum": true,
	// A binary ballot that carries only on reaching 91 of 180 votes.
	"Ausgabenbremse": true,
}

// unpostableVoteTypes are types we recognise and have decided not to publish.
//
// They are named separately from the types we have simply never seen, because
// the two call for opposite responses. A type on this list is expected to turn
// up — every Kanton Zürich sitting opens with an attendance roll call — and
// skipping it is the system working, so it must not fail a run. An unrecognised
// type is the source doing something new, which is exactly what someone needs
// to be told about.
// Cup-Abstimmung is deliberately absent. It is also never published, but for a
// reason that is expected to go away: the source serves it with null aggregates
// and duplicated member rows, both reported upstream. Its run-failing rejection
// is the reminder that the report is still open, so it stays an
// ErrUnsupportedVoteType — see TestPostToPlatform_CantonCupVoteIsNotPosted.
var unpostableVoteTypes = map[string]bool{
	// A roll call establishing who is in the chamber. It is not a vote on
	// anything and reports as a lopsided Ja tally, so publishing one would
	// announce a near-unanimous decision on a question nobody was asked.
	// Nothing upstream is broken here and nothing is going to change: the
	// Kantonsrat takes one at the start of most sittings.
	"Anwesenheitsermittlung": true,
}

// IsKnownUnpostableType reports whether a type is one we recognise and
// deliberately do not publish, as opposed to one we do not recognise at all.
func IsKnownUnpostableType(voteType string) bool {
	return unpostableVoteTypes[strings.TrimSpace(voteType)]
}

// IsHandledVoteType reports whether the formatters know how to render a vote of
// this type. Callers skip anything else rather than publish it.
//
// This is an allowlist, and an empty type does not pass it. That is the point:
// a type we have never seen is exactly the case where a lopsided tally is most
// likely to be read wrongly, and refusing to post is recoverable in a way that
// a misleading post about how parliament voted is not.
//
// OpenParlData serves a null type often enough that this used to reject whole
// Kanton Zürich sittings — the 17.08.2026 sitting arrived with all five of its
// votes untyped, among them two Ausgabenbremse votes and an attendance roll
// call. Those votes now arrive typed from the Kantonsrat's own archive (see
// pkg/recapp), so an empty type here means both sources came up blank, which is
// exactly when refusing is right.
//
// Cup-Abstimmung is knowingly excluded. It is a knockout round between more
// than two proposals, and the source reports no aggregate counts for it, so
// there is nothing correct to render yet.
func IsHandledVoteType(voteType string) bool {
	t := strings.TrimSpace(voteType)
	if handledVoteTypes[t] {
		return true
	}
	return strings.HasPrefix(t, auswahlTypePrefix)
}

// joinTypeLabel appends the type label to base with a separator, handling the
// case where base is empty and the label has to stand alone.
func joinTypeLabel(base, voteType string) string {
	label := TypeLabel(voteType)
	switch {
	case label == "":
		return base
	case base == "":
		return label
	default:
		return base + " · " + label
	}
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

// BodyLabel returns the chamber name to put in a post header.
//
// Several bodies post to one account, so this is what tells a reader whether
// they are looking at a city or a cantonal vote. It is not decoration.
func BodyLabel(group []votes.Vote) string {
	if len(group) > 0 && group[0].Body != "" {
		return group[0].Body
	}
	return ""
}

// PostHeadline is the first line of a post: which chamber voted, and when.
//
// A single-vote post names the time as well, where the source knows it. The
// post is about one moment, and without it two votes taken under the same
// agenda item on the same day — "Mitteilungen" is the recurring case, and it
// carries no business number, so each such vote posts on its own — produce two
// posts a reader cannot tell apart. Multi-vote groups span several times and
// keep the date alone; their entries carry the clock (see SubVoteLabel).
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
	prefix := ""
	if body != "" {
		prefix = body + " | "
	}
	if len(group) == 0 || group[0].Date.IsZero() {
		return prefix + "Abstimmung"
	}
	headline := prefix + "Abstimmung vom " + FormatVoteDate(group[0].Date)
	if len(group) == 1 && hasClockTime(group[0].Date) {
		headline += fmt.Sprintf(" (%s)", group[0].Date.Format("15:04"))
	}
	return headline
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

// SubVoteLabel names one vote inside a group, so the entries of a thread or the
// cards of a carousel can be told apart. groupSize is how many votes the group
// holds; a group of one needs no such label beyond whatever title it has.
//
// Stadt Zürich titles each vote ("Schlussabstimmung über die Dispositivziffer
// 1"). Kanton Zürich publishes no per-vote title at all — its title field
// repeats the business matter — so those groups fall back to an ordinal plus
// the time the vote was taken: "Abstimmung 2 (14:32)".
//
// The time was tried once before and removed, on the grounds that the link
// already lands in the right place and a clock reading means no more to a
// reader than a number does. What that judgement did not account for is how the
// ordinals look in practice: five cards reading "Abstimmung 1" to "Abstimmung 5"
// above near-identical tallies give a reader nothing to hold on to, and no way
// to tell whether the near-identical numbers are five votes or one repeated.
// The clock is thin, but it is a fact about the sitting rather than an artefact
// of our own numbering, and it is the only distinguishing fact the source
// offers. It appears only where it can help: multi-vote groups from a source
// with second-precision timestamps.
//
// The vote type is appended after all of that, when it is one worth naming (see
// typeLabels). Unlike the clock it is not there to tell two votes apart, but to
// stop a quorum vote's lopsided tally reading as near-unanimous agreement.
func SubVoteLabel(v votes.Vote, index, groupSize int) string {
	base := CleanVoteSubtitle(v.Subtitle)
	if base == "" {
		base = fmt.Sprintf("Abstimmung %d", index+1)
		// The clock disambiguates our own ordinals, so it belongs only on them.
		// A source that titles its votes has already done the distinguishing.
		if groupSize > 1 && hasClockTime(v.Date) {
			base += fmt.Sprintf(" (%s)", v.Date.Format("15:04"))
		}
	}
	return joinTypeLabel(base, v.Type)
}

// hasClockTime reports whether a date carries a time of day worth showing.
// PARIS supplies sitting dates at midnight, so printing "(00:00)" there would
// invent a precision the source does not have.
func hasClockTime(t time.Time) bool {
	return !t.IsZero() && (t.Hour() != 0 || t.Minute() != 0)
}
