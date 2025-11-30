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

	// Create API client
	client := zurichapi.NewClient()

	// Prepare votes for posting (platform-agnostic)
	groups, err := voteposting.PrepareVoteGroups(client, voteLog, maxVotesToCheck)
	if err != nil {
		log.Fatalf("Error preparing votes: %v", err)
	}

	if len(groups) == 0 {
		fmt.Println("No new votes to post!")
		return
	}

	fmt.Printf("Found %d group(s) to post\n", len(groups))

	// Create X platform poster
	xPlatform := x.NewXPlatform(
		apiKey, apiSecret, accessToken, accessSecret,
		contactMapper,
		maxPostsPerRun,
	)

	// Post to platform
	posted, err := voteposting.PostToPlatform(groups, xPlatform, voteLog, false)
	if err != nil {
		log.Fatalf("Error posting: %v", err)
	}

	fmt.Printf("\n🎉 Posted %d new group(s) successfully!\n", posted)
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
