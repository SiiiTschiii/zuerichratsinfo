package voteformat

import (
	"fmt"
	"regexp"
	"strings"
)

var geschaeftNumberRegex = regexp.MustCompile(`^\d+/\d+\s+`)
var geschaeftNumberUnderscoreRegex = regexp.MustCompile(`^\d+_\d+\s+`)

// antragOnlyRegex matches titles that are just "Antrag XXX" or "Anträge XXX bis YYY",
// optionally followed by "zu Dispositivziffer X" (e.g. "Antrag 1 zu Dispositivziffer 1a").
// These generic titles should be replaced with the GeschaeftTitel.
// The dot after the number is optional because the API sometimes omits it (e.g. "Antrag 1" vs "Antrag 007.")
var antragOnlyRegex = regexp.MustCompile(`^(\d+/\d+\s+)?Antr(a|ä)ge?\s+\d+\.?(\s*(bis|–|-)\s*\d+\.?)?(\s+zu\s+Dispositivziffer\s+\S+)?$`)


// FormatVoteDate formats the date from ISO format to DD.MM.YYYY
func FormatVoteDate(isoDate string) string {
	if len(isoDate) < 10 {
		return isoDate
	}
	// isoDate is in format YYYY-MM-DD...
	parts := strings.Split(isoDate[:10], "-")
	if len(parts) == 3 {
		return fmt.Sprintf("%s.%s.%s", parts[2], parts[1], parts[0])
	}
	return isoDate[:10]
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

// SelectBestTitle chooses between TraktandumTitel and GeschaeftTitel
// If TraktandumTitel is just a generic "Antrag XXX" pattern, use GeschaeftTitel instead
func SelectBestTitle(traktandumTitel, geschaeftTitel string) string {
	if IsGenericAntragTitle(traktandumTitel) {
		return geschaeftTitel
	}
	return traktandumTitel
}

// IsGenericAntragTitle checks if a title is just a generic "Antrag XXX" pattern
func IsGenericAntragTitle(traktandumTitel string) bool {
	// Clean up the traktandum title for pattern matching
	cleaned := strings.TrimSpace(traktandumTitel)
	cleaned = strings.ReplaceAll(cleaned, "\r\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	// Check if it matches the generic "Antrag XXX" pattern
	return antragOnlyRegex.MatchString(cleaned)
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
	Ja         *int
	Nein       *int
	Enthaltung *int
	Abwesend   *int
	A, B, C, D, E *int
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

// GenerateVoteLink creates the link to the vote detail page
func GenerateVoteLink(objGUID string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=%s", objGUID)
}

// GenerateTraktandumLink creates the link to the Traktandum (agenda item) page
// This shows all votes related to a specific business matter in a session
func GenerateTraktandumLink(sitzungGuid, traktandumGuid string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/sitzungen/sitzung/?gid=%s#%s", sitzungGuid, traktandumGuid)
}

// GenerateGeschaeftLink creates the link to the Geschäft (business matter) page
// This shows all information related to a specific Geschäft (e.g., budget 2025/391)
func GenerateGeschaeftLink(geschaeftGuid string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=%s", geschaeftGuid)
}
