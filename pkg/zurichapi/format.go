package zurichapi

import (
	"fmt"
	"strings"
)

// FormatGeschaeftTweet formats a geschaeft into a tweet message
func FormatGeschaeftTweet(geschaeft *Geschaeft) string {
	dateStr := geschaeft.Beginn.Text
	if dateStr == "" {
		dateStr = "Datum unbekannt"
	}

	// Format: GR Nr, Type, Title, Author
	author := ""
	if geschaeft.Erstunterzeichner.KontaktGremium.Name != "" {
		author = fmt.Sprintf(" von %s", geschaeft.Erstunterzeichner.KontaktGremium.Name)
		if geschaeft.Erstunterzeichner.KontaktGremium.Partei != "" {
			author += fmt.Sprintf(" (%s)", geschaeft.Erstunterzeichner.KontaktGremium.Partei)
		}
	}

	message := fmt.Sprintf("🏛️ Neues Geschäft im Gemeinderat Zürich\n\n📋 %s: %s\n📅 %s%s\n\n%s",
		geschaeft.GRNr,
		geschaeft.Geschaeftsart,
		dateStr,
		author,
		geschaeft.Titel)

	return message
}

// FormatAbstimmungTweet formats an abstimmung into a tweet message
func FormatAbstimmungTweet(abstimmung *Abstimmung) string {
	// Extract date from SitzungDatum (format: 2025-10-22T00:00:00.000)
	dateStr := abstimmung.SitzungDatum
	if len(dateStr) >= 10 {
		dateStr = dateStr[:10] // Just the date part
	}
	
	// Clean up the title (remove newlines and special chars)
	title := cleanTitle(abstimmung.TraktandumTitel)
	
	// Use same structure as geschaeft tweet
	message := fmt.Sprintf("🏛️ Neues Geschäft im Gemeinderat Zürich\n\n📋 %s: Abstimmung\n📅 %s\n\n%s",
		abstimmung.GeschaeftGrNr,
		dateStr,
		title)

	return message
}

// cleanTitle removes newlines and extra whitespace from titles
func cleanTitle(title string) string {
	// Replace newlines and carriage returns with spaces
	result := ""
	for _, ch := range title {
		if ch == '\n' || ch == '\r' || ch == '\t' {
			result += " "
		} else {
			result += string(ch)
		}
	}
	
	// Replace problematic quote characters with standard ones
	result = strings.ReplaceAll(result, "«", "\"")
	result = strings.ReplaceAll(result, "»", "\"")
	result = strings.ReplaceAll(result, "‹", "'")
	result = strings.ReplaceAll(result, "›", "'")
	
	// Remove multiple consecutive spaces
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	
	// Remove variation selectors (U+FE0F and similar)
	result = strings.ReplaceAll(result, "\uFE0F", "")
	result = strings.ReplaceAll(result, "\uFE0E", "")
	
	return strings.TrimSpace(result)
}