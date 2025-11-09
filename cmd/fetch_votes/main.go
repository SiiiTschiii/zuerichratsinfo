package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Get number of votes from command line argument, default to 10
	numVotes := 10
	if len(os.Args) > 1 {
		if n, err := strconv.Atoi(os.Args[1]); err == nil && n > 0 {
			numVotes = n
		} else {
			log.Fatalf("Invalid argument: please provide a positive number")
		}
	}

	// Create API client
	client := zurichapi.NewClient()

	// Fetch the most recent votes
	votes, err := client.FetchRecentAbstimmungen(numVotes)
	if err != nil {
		log.Fatalf("Error fetching votes: %v", err)
	}

	fmt.Printf("🗳️  Last %d Council Votes\n", numVotes)
	fmt.Printf("═══════════════════════════════════════════════════════════════════════════\n\n")

	for i, vote := range votes {
		fmt.Printf("%d. %s\n", i+1, vote.SitzungDatum[:10])
		
		// Print traktandum title
		title := cleanTitle(vote.TraktandumTitel)
		fmt.Printf("   📋 %s\n", title)
		
		// Print vote result
		fmt.Printf("   ✓ Result: %s\n", vote.Schlussresultat)
		
		// Print vote counts
		ja := formatCount(vote.AnzahlJa)
		nein := formatCount(vote.AnzahlNein)
		enthaltung := formatCount(vote.AnzahlEnthaltung)
		abwesend := formatCount(vote.AnzahlAbwesend)
		
		fmt.Printf("   📊 Votes: %s Ja | %s Nein | %s Enthaltung | %s Abwesend\n", 
			ja, nein, enthaltung, abwesend)
		fmt.Println()
	}
}

// cleanTitle removes newlines and extra whitespace from titles
func cleanTitle(title string) string {
	// Replace newlines and carriage returns with spaces
	title = strings.ReplaceAll(title, "\r\n", " ")
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")
	
	// Replace multiple spaces with single space
	parts := strings.Fields(title)
	return strings.Join(parts, " ")
}

// formatCount formats a nullable int pointer
func formatCount(count *int) string {
	if count == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *count)
}
