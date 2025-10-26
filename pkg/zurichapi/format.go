package zurichapi

import "fmt"

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

	// X has a 280 character limit
	if len(message) > 280 {
		message = message[:277] + "..."
	}

	return message
}