package x

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/urlshorten"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

var geschaeftNumberRegex = regexp.MustCompile(`^\d+/\d+\s+`)
var geschaeftNumberUnderscoreRegex = regexp.MustCompile(`^\d+_\d+\s+`)

// antragOnlyRegex matches titles that are just "Antrag XXX" or "Anträge XXX bis YYY"
// These generic titles should be replaced with the GeschaeftTitel
var antragOnlyRegex = regexp.MustCompile(`^(\d+/\d+\s+)?Antr(a|ä)ge?\s+\d+\.(\s+(bis|–|-)\s+\d+\.)?$`)

// FormatVotePost creates a formatted X post for a vote (Abstimmung)
// This is the main function to format vote posts for X/Twitter
func FormatVotePost(vote *zurichapi.Abstimmung, contactMapper *contacts.Mapper) string {
	return FormatVoteGroupPost([]zurichapi.Abstimmung{*vote}, contactMapper)
}

// FormatVoteGroupPost creates a formatted X post for a group of related votes
// (multiple votes on the same business matter on the same day)
func FormatVoteGroupPost(votes []zurichapi.Abstimmung, contactMapper *contacts.Mapper) string {
	if len(votes) == 0 {
		return ""
	}

	// Use first vote for common metadata
	firstVote := votes[0]

	// Prepare fixed components
	date := formatVoteDate(firstVote.SitzungDatum)
	header := fmt.Sprintf("🗳️  Gemeinderat | Abstimmung vom %s\n\n", date)

	// Build the full title
	// Use GeschaeftTitel if TraktandumTitel is just a generic "Antrag XXX" pattern
	title := selectBestTitle(firstVote.TraktandumTitel, firstVote.GeschaeftTitel)
	title = cleanVoteTitle(title)

	// Tag X handles in the title if contact mapper is provided
	if contactMapper != nil {
		title = contactMapper.TagXHandlesInText(title)
	}

	// Build post
	var sb strings.Builder
	sb.WriteString(header)

	// For single vote, use original format
	if len(votes) == 1 {
		vote := votes[0]
		resultEmoji := getVoteResultEmoji(vote.Schlussresultat)
		result := getVoteResultText(vote.Schlussresultat)
		resultPrefix := fmt.Sprintf("%s %s: ", resultEmoji, result)

		ja := formatVoteCount(vote.AnzahlJa)
		nein := formatVoteCount(vote.AnzahlNein)
		enthaltung := formatVoteCount(vote.AnzahlEnthaltung)
		abwesend := formatVoteCount(vote.AnzahlAbwesend)
		voteCounts := fmt.Sprintf("📊 %s Ja | %s Nein | %s Enthaltung | %s Abwesend\n\n",
			ja, nein, enthaltung, abwesend)

		sb.WriteString(resultPrefix)
		sb.WriteString(title)
		sb.WriteString("\n\n")
		sb.WriteString(voteCounts)
	} else {
		// For multiple votes, show title once and list all votes
		// No overall result - just show the title and individual vote results
		sb.WriteString(title)
		sb.WriteString("\n\n")

		// List each vote with its details
		for i, vote := range votes {
			ja := formatVoteCount(vote.AnzahlJa)
			nein := formatVoteCount(vote.AnzahlNein)
			enthaltung := formatVoteCount(vote.AnzahlEnthaltung)
			abwesend := formatVoteCount(vote.AnzahlAbwesend)

			voteEmoji := getVoteResultEmoji(vote.Schlussresultat)
			voteTitle := cleanVoteSubtitle(vote.Abstimmungstitel)

			if voteTitle != "" {
				sb.WriteString(fmt.Sprintf("%s %s\n", voteEmoji, voteTitle))
			} else {
				sb.WriteString(fmt.Sprintf("%s Abstimmung %d\n", voteEmoji, i+1))
			}
			sb.WriteString(fmt.Sprintf("📊 %s Ja | %s Nein | %s Enthaltung | %s Abwesend\n",
				ja, nein, enthaltung, abwesend))

			if i < len(votes)-1 {
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	// Generate and shorten the link
	// Special case: if we're using GeschaeftTitel (because TraktandumTitel is generic "Antrag XXX"),
	// link to the Geschäft page instead of Traktandum
	var link string
	if isGenericAntragTitle(firstVote.TraktandumTitel) {
		// Link to Geschäft for generic "Antrag" titles
		link = generateGeschaeftLink(firstVote.GeschaeftGuid)
	} else if len(votes) > 1 {
		// For grouped votes, link to the Traktandum (shows all votes together)
		link = generateTraktandumLink(firstVote.SitzungGuid, firstVote.TraktandumGuid)
	} else {
		// For single votes, link to the individual vote
		link = generateVoteLink(firstVote.OBJGUID)
	}
	link = urlshorten.ShortenURL(link)
	linkLine := fmt.Sprintf("🔗 %s", link)
	sb.WriteString(linkLine)

	return sb.String()
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

// selectBestTitle chooses between TraktandumTitel and GeschaeftTitel
// If TraktandumTitel is just a generic "Antrag XXX" pattern, use GeschaeftTitel instead
func selectBestTitle(traktandumTitel, geschaeftTitel string) string {
	if isGenericAntragTitle(traktandumTitel) {
		return geschaeftTitel
	}
	return traktandumTitel
}

// isGenericAntragTitle checks if a title is just a generic "Antrag XXX" pattern
func isGenericAntragTitle(traktandumTitel string) bool {
	// Clean up the traktandum title for pattern matching
	cleaned := strings.TrimSpace(traktandumTitel)
	cleaned = strings.ReplaceAll(cleaned, "\r\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	// Check if it matches the generic "Antrag XXX" pattern
	return antragOnlyRegex.MatchString(cleaned)
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

	// Strip Geschäft number from the beginning (e.g., "2024/431 " or "2025/84 ")
	// Pattern: number/number followed by space
	title = geschaeftNumberRegex.ReplaceAllString(title, "")

	return title
}

// cleanVoteSubtitle cleans up vote subtitles (Abstimmungstitel)
// Similar to cleanVoteTitle but keeps it shorter
func cleanVoteSubtitle(subtitle string) string {
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

// generateTraktandumLink creates the link to the Traktandum (agenda item) page
// This shows all votes related to a specific business matter in a session
func generateTraktandumLink(sitzungGuid, traktandumGuid string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/sitzungen/sitzung/?gid=%s#%s", sitzungGuid, traktandumGuid)
}

// generateGeschaeftLink creates the link to the Geschäft (business matter) page
// This shows all information related to a specific Geschäft (e.g., budget 2025/391)
func generateGeschaeftLink(geschaeftGuid string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=%s", geschaeftGuid)
}
