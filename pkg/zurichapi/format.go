package zurichapi

import (
	"fmt"
	"strings"
)

// FormatVotePost creates a formatted X post for a vote (Abstimmung)
// This is the main function to format vote posts for X/Twitter
func FormatVotePost(vote *Abstimmung) string {
	const maxLength = 280
	
	// Prepare fixed components
	date := formatVoteDate(vote.SitzungDatum)
	header := fmt.Sprintf("🗳️  Gemeinderat | Abstimmung vom %s\n\n", date)
	
	resultEmoji := getVoteResultEmoji(vote.Schlussresultat)
	result := getVoteResultText(vote.Schlussresultat)
	
	ja := formatVoteCount(vote.AnzahlJa)
	nein := formatVoteCount(vote.AnzahlNein)
	enthaltung := formatVoteCount(vote.AnzahlEnthaltung)
	abwesend := formatVoteCount(vote.AnzahlAbwesend)
	voteCounts := fmt.Sprintf("📊 %s Ja | %s Nein | %s Enthaltung | %s Abwesend\n\n", 
		ja, nein, enthaltung, abwesend)
	
	link := generateVoteLink(vote.OBJGUID)
	linkLine := fmt.Sprintf("🔗 %s", link)
	
	// Calculate available space for title
	title := cleanVoteTitle(vote.TraktandumTitel)
	resultPrefix := fmt.Sprintf("%s %s: ", resultEmoji, result)
	
	// Build post and check length
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(resultPrefix)
	sb.WriteString(title)
	sb.WriteString("\n\n")
	sb.WriteString(voteCounts)
	sb.WriteString(linkLine)
	
	post := sb.String()
	
	// If too long, shorten the title
	if len(post) > maxLength {
		// Calculate how much we need to cut
		fixedLength := len(header) + len(resultPrefix) + len("\n\n") + len(voteCounts) + len(linkLine)
		availableForTitle := maxLength - fixedLength - 3 // -3 for "..."
		
		if availableForTitle > 0 {
			// Truncate title at word boundary
			title = truncateAtWord(title, availableForTitle) + "..."
		}
		
		// Rebuild post with shortened title
		sb.Reset()
		sb.WriteString(header)
		sb.WriteString(resultPrefix)
		sb.WriteString(title)
		sb.WriteString("\n\n")
		sb.WriteString(voteCounts)
		sb.WriteString(linkLine)
		post = sb.String()
	}
	
	return post
}

// formatVoteDate formats the date from ISO format to DD.MM.YYYY
func formatVoteDate(isoDate string) string {
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

// getVoteResultEmoji returns the appropriate emoji for a vote result
func getVoteResultEmoji(result string) string {
	result = strings.TrimSpace(strings.ToLower(result))
	if strings.Contains(result, "angenommen") || result == "ja" {
		return "✅"
	}
	return "❌"
}

// getVoteResultText returns the text for a vote result
func getVoteResultText(result string) string {
	result = strings.TrimSpace(strings.ToLower(result))
	if strings.Contains(result, "angenommen") || result == "ja" {
		return "Angenommen"
	}
	return "Abgelehnt"
}

// cleanVoteTitle removes newlines, extra whitespace, and Geschäft number from titles
func cleanVoteTitle(title string) string {
	// Replace newlines and carriage returns with spaces
	title = strings.ReplaceAll(title, "\r\n", " ")
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")
	
	// Replace multiple spaces with single space
	parts := strings.Fields(title)
	title = strings.Join(parts, " ")
	
	// Strip Geschäft number from the beginning (e.g., "2024/431 ")
	// Pattern: YYYY/NNN at the start followed by space
	if len(title) > 8 && title[4] == '/' {
		// Find the first space after the number
		spaceIdx := strings.Index(title[8:], " ")
		if spaceIdx != -1 {
			title = title[8+spaceIdx+1:]
		}
	}
	
	return title
}

// truncateAtWord truncates a string to maxLen at the nearest word boundary
func truncateAtWord(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	
	// Find the last space before maxLen
	truncated := s[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	
	if lastSpace > 0 {
		return s[:lastSpace]
	}
	
	// If no space found, just truncate at maxLen
	return truncated
}

// formatVoteCount formats a nullable int pointer
func formatVoteCount(count *int) string {
	if count == nil {
		return "0"
	}
	return fmt.Sprintf("%d", *count)
}

// generateVoteLink creates the link to the vote detail page
func generateVoteLink(objGUID string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=%s", objGUID)
}