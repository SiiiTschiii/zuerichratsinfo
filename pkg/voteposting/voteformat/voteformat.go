package voteformat

import (
	"fmt"
	"regexp"
	"strings"
)

var geschaeftNumberRegex = regexp.MustCompile(`^\d+/\d+\s+`)
var geschaeftNumberUnderscoreRegex = regexp.MustCompile(`^\d+_\d+\s+`)

// antragOnlyRegex matches titles that are just "Antrag XXX" or "Anträge XXX bis YYY"
// These generic titles should be replaced with the GeschaeftTitel
var antragOnlyRegex = regexp.MustCompile(`^(\d+/\d+\s+)?Antr(a|ä)ge?\s+\d+\.(\s+(bis|–|-)\s+\d+\.)?$`)


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
