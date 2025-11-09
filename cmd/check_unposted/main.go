package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Get number of votes to check from command line argument, or environment, or default
	maxVotesToCheck := 50
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
	
	maxPostsPerRun := getEnvInt("X_MAX_POSTS_PER_RUN", 10)

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
	fmt.Printf("📊 Loaded vote log: %d votes already posted\n\n", voteLog.Count())

	// Fetch recent votes from Zurich council API
	client := zurichapi.NewClient()
	abstimmungen, err := client.FetchRecentAbstimmungen(maxVotesToCheck)
	if err != nil {
		log.Fatalf("Error fetching abstimmungen: %v", err)
	}
	
	if len(abstimmungen) == 0 {
		log.Fatal("No abstimmungen found")
	}

	// Filter out already posted votes
	var unpostedVotes []zurichapi.Abstimmung
	for _, vote := range abstimmungen {
		if !voteLog.IsPosted(vote.OBJGUID) {
			unpostedVotes = append(unpostedVotes, vote)
		}
	}

	fmt.Printf("📝 Found %d unposted votes out of %d recent votes\n\n", len(unpostedVotes), len(abstimmungen))

	if len(unpostedVotes) == 0 {
		fmt.Println("✨ No new votes to post!")
		return
	}

	// Show what would be posted
	votesToPost := unpostedVotes
	if len(unpostedVotes) > maxPostsPerRun {
		fmt.Printf("⚠️  Would limit to %d posts per run (found %d unposted)\n\n", maxPostsPerRun, len(unpostedVotes))
		votesToPost = unpostedVotes[:maxPostsPerRun]
	}

	fmt.Printf("🚀 Would post these %d votes:\n\n", len(votesToPost))
	
	for i, vote := range votesToPost {
		message := zurichapi.FormatVotePost(&vote, contactMapper)
		fmt.Printf("─────────────────────────────────────────────────────────────────\n")
		fmt.Printf("[%d/%d] Vote ID: %s\n", i+1, len(votesToPost), vote.OBJGUID[:8]+"...")
		fmt.Printf("Date: %s\n", vote.SitzungDatum[:10])
		fmt.Printf("\n%s\n", message)
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
