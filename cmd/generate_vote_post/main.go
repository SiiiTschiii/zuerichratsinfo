package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Create API client
	client := zurichapi.NewClient()

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
		post := zurichapi.FormatVotePost(&vote)
		fmt.Println(post)
	}
}
