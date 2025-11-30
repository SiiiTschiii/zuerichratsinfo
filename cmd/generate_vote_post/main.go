package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
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

	// Determine how many votes to show (default: 1)
	numVotes := 1
	if len(os.Args) > 1 {
		var n int
		if _, err := fmt.Sscanf(os.Args[1], "%d", &n); err == nil && n > 0 {
			numVotes = n
		}
	}

	// Use empty vote log (show all votes, not just unposted)
	emptyLog := votelog.NewEmpty(votelog.PlatformX)

	// Prepare votes (same logic as main.go)
	groups, err := voteposting.PrepareVoteGroups(client, emptyLog, numVotes)
	if err != nil {
		log.Fatalf("Error preparing votes: %v", err)
	}

	if len(groups) == 0 {
		log.Fatal("No votes found")
	}

	// Create X platform (for formatting only, no posting)
	xPlatform := x.NewXPlatform("", "", "", "", contactMapper, numVotes)

	// Dry run - just print, don't post
	_, err = voteposting.PostToPlatform(groups, xPlatform, emptyLog, true)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
