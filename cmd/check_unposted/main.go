package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Get number of votes to check from command line argument, or environment, or default
	maxVotesToCheck := 150
	if len(os.Args) > 1 {
		// Command line argument takes priority
		if n, err := strconv.Atoi(os.Args[1]); err == nil && n > 0 {
			maxVotesToCheck = n
		} else {
			log.Fatalf("Invalid argument: please provide a positive number")
		}
	} else {
		// Fall back to environment variable
		maxVotesToCheck = getEnvInt("MAX_VOTES_TO_CHECK", 50)
	}

	fmt.Printf("Configuration: Check last %d votes\n\n", maxVotesToCheck)

	// Load contacts for X handle tagging
	contactsPath := filepath.Join("data", "contacts.yaml")
	contactMapper, err := contacts.LoadContacts(contactsPath)
	if err != nil {
		log.Printf("Warning: Could not load contacts for tagging: %v", err)
		contactMapper = nil // Continue without tagging
	}

	// Load the vote log for X platform
	voteLog, err := votelog.Load(votelog.PlatformX)
	if err != nil {
		log.Fatalf("Error loading vote log: %v", err)
	}
	fmt.Printf("Loaded vote log: %d votes already posted\n", voteLog.Count())

	// Create API client
	client := zurichapi.NewClient()

	// Prepare votes for posting using the same logic as main.go
	groups, err := voteposting.PrepareVoteGroups(client, voteLog, maxVotesToCheck)
	if err != nil {
		log.Fatalf("Error preparing votes: %v", err)
	}

	if len(groups) == 0 {
		fmt.Println("\n✨ No new votes to post!")
		return
	}

	fmt.Printf("Found %d group(s) to post\n\n", len(groups))

	fmt.Printf("🚀 Would post these %d groups:\n\n", len(groups))

	for i, group := range groups {
		message := x.FormatVoteGroupPost(group, contactMapper)
		fmt.Printf("─────────────────────────────────────────────────────────────────\n")
		fmt.Printf("[%d/%d] Group with %d vote(s)\n", i+1, len(groups), len(group))
		fmt.Printf("Business: %s\n", group[0].GeschaeftGrNr)
		fmt.Printf("Date: %s\n", group[0].SitzungDatum[:10])
		fmt.Printf("Vote IDs: ")
		for j, vote := range group {
			if j > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", vote.OBJGUID[:8]+"...")
		}
		fmt.Printf("\n\n%s\n", message)
		fmt.Printf("\nCharacter count: %d\n", len(message))
	}
	fmt.Printf("─────────────────────────────────────────────────────────────────\n")
}

// getEnvInt gets an integer from environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}
