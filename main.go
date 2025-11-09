package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/xapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Load configuration from environment
	apiKey := os.Getenv("X_API_KEY")
	apiSecret := os.Getenv("X_API_SECRET")
	accessToken := os.Getenv("X_ACCESS_TOKEN")
	accessSecret := os.Getenv("X_ACCESS_SECRET")

	if apiKey == "" || apiSecret == "" || accessToken == "" || accessSecret == "" {
		log.Fatal("Missing X API credentials. Please set X_API_KEY, X_API_SECRET, X_ACCESS_TOKEN, and X_ACCESS_SECRET environment variables.")
	}

	// Load rate limit configuration from environment
	maxVotesToCheck := getEnvInt("MAX_VOTES_TO_CHECK", 50)
	maxPostsPerRun := getEnvInt("X_MAX_POSTS_PER_RUN", 10)

	fmt.Printf("Configuration: Check last %d votes, post max %d per run\n", maxVotesToCheck, maxPostsPerRun)

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

	fmt.Printf("Found %d unposted votes out of %d recent votes\n", len(unpostedVotes), len(abstimmungen))

	if len(unpostedVotes) == 0 {
		fmt.Println("No new votes to post!")
		return
	}

	// Limit posts per run to avoid rate limits
	if len(unpostedVotes) > maxPostsPerRun {
		fmt.Printf("Limiting to %d posts per run (found %d unposted)\n", maxPostsPerRun, len(unpostedVotes))
		unpostedVotes = unpostedVotes[:maxPostsPerRun]
	}

	// Post each unposted vote
	successCount := 0
	for i, vote := range unpostedVotes {
		fmt.Printf("\n[%d/%d] Posting vote %s...\n", i+1, len(unpostedVotes), vote.OBJGUID)
		
		// Format message
		message := zurichapi.FormatVotePost(&vote, contactMapper)
		fmt.Printf("Message (%d chars):\n%s\n\n", len(message), message)

		// Post to X
		err = xapi.PostTweet(apiKey, apiSecret, accessToken, accessSecret, message)
		if err != nil {
			log.Printf("ERROR: Failed to post vote %s: %v", vote.OBJGUID, err)
			continue // Skip this one, try next
		}

		// Mark as posted and save immediately
		voteLog.MarkAsPosted(vote.OBJGUID)
		if err := voteLog.Save(); err != nil {
			log.Printf("WARNING: Failed to save vote log: %v", err)
		}

		successCount++
		fmt.Printf("✅ Successfully posted!\n")
	}

	fmt.Printf("\n🎉 Posted %d new votes successfully!\n", successCount)
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
