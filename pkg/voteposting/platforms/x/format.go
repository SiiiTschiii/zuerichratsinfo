package x

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

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
	date := voteformat.FormatVoteDate(firstVote.SitzungDatum)
	header := fmt.Sprintf("🗳️  Gemeinderat | Abstimmung vom %s\n\n", date)

	// Build the full title
	// Use GeschaeftTitel if TraktandumTitel is just a generic "Antrag XXX" pattern
	title := voteformat.SelectBestTitle(firstVote.TraktandumTitel, firstVote.GeschaeftTitel)
	title = voteformat.CleanVoteTitle(title)

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
		counts := voteformat.VoteCounts{
			Ja: vote.AnzahlJa, Nein: vote.AnzahlNein,
			Enthaltung: vote.AnzahlEnthaltung, Abwesend: vote.AnzahlAbwesend,
			A: vote.AnzahlA, B: vote.AnzahlB, C: vote.AnzahlC, D: vote.AnzahlD, E: vote.AnzahlE,
		}
		voteCounts := voteformat.FormatVoteCountsLong(counts) + "\n\n"

		if voteformat.IsAuswahlVote(counts) {
			// Auswahl votes have no accepted/rejected outcome — omit result prefix
			sb.WriteString(title)
		} else {
			resultEmoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
			result := voteformat.GetVoteResultText(vote.Schlussresultat)
			sb.WriteString(fmt.Sprintf("%s %s: ", resultEmoji, result))
			sb.WriteString(title)
		}
		sb.WriteString("\n\n")
		sb.WriteString(voteCounts)
	} else {
		// For multiple votes, show title once and list all votes
		// No overall result - just show the title and individual vote results
		sb.WriteString(title)
		sb.WriteString("\n\n")

		// List each vote with its details
		for i, vote := range votes {
			voteTitle := voteformat.CleanVoteSubtitle(vote.Abstimmungstitel)
			counts := voteformat.VoteCounts{
				Ja: vote.AnzahlJa, Nein: vote.AnzahlNein,
				Enthaltung: vote.AnzahlEnthaltung, Abwesend: vote.AnzahlAbwesend,
				A: vote.AnzahlA, B: vote.AnzahlB, C: vote.AnzahlC, D: vote.AnzahlD, E: vote.AnzahlE,
			}

			if voteformat.IsAuswahlVote(counts) {
				// Auswahl: no ✅/❌ prefix — outcome is "Auswahl A/B/…", not accepted/rejected
				if voteTitle != "" {
					sb.WriteString(fmt.Sprintf("%s\n", voteTitle))
				} else {
					sb.WriteString(fmt.Sprintf("Abstimmung %d\n", i+1))
				}
			} else {
				voteEmoji := voteformat.GetVoteResultEmoji(vote.Schlussresultat)
				if voteTitle != "" {
					sb.WriteString(fmt.Sprintf("%s %s\n", voteEmoji, voteTitle))
				} else {
					sb.WriteString(fmt.Sprintf("%s Abstimmung %d\n", voteEmoji, i+1))
				}
			}
			sb.WriteString(voteformat.FormatVoteCountsLong(counts) + "\n")

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
	if voteformat.IsGenericAntragTitle(firstVote.TraktandumTitel) {
		// Link to Geschäft for generic "Antrag" titles
		link = voteformat.GenerateGeschaeftLink(firstVote.GeschaeftGuid)
	} else if len(votes) > 1 {
		// For grouped votes, link to the Traktandum (shows all votes together)
		link = voteformat.GenerateTraktandumLink(firstVote.SitzungGuid, firstVote.TraktandumGuid)
	} else {
		// For single votes, link to the individual vote
		link = voteformat.GenerateVoteLink(firstVote.OBJGUID)
	}
	linkLine := fmt.Sprintf("🔗 %s", link)
	sb.WriteString(linkLine)

	return sb.String()
}
