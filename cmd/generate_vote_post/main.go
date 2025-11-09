package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Create API client
	client := zurichapi.NewClient()

	// Load contacts for X handle tagging
	contactsPath := filepath.Join("data", "contacts.yaml")
	contactMapper, err := contacts.LoadContacts(contactsPath)
	if err != nil {
		log.Printf("Warning: Could not load contacts for tagging: %v", err)
		contactMapper = nil // Continue without tagging
	}

	// Determine how many votes to fetch (default: 1)
	numVotes := 1
	if len(os.Args) > 1 {
		var n int
		if _, err := fmt.Sscanf(os.Args[1], "%d", &n); err == nil && n > 0 {
			numVotes = n
		}
	}

	// Fetch the most recent votes
	votes, err := client.FetchRecentAbstimmungen(numVotes)
	if err != nil {
		log.Fatalf("Error fetching votes: %v", err)
	}

	if len(votes) == 0 {
		log.Fatal("No votes found")
	}

	// Generate posts for each vote
	for i, vote := range votes {
		if i > 0 {
			fmt.Println("\n" + strings.Repeat("─", 80) + "\n")
		}
		post := zurichapi.FormatVotePost(&vote, contactMapper)
		fmt.Println(post)
	}
}
