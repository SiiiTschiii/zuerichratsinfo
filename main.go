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
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/bluesky"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Load configuration from environment
	apiKey := os.Getenv("X_API_KEY")
	apiSecret := os.Getenv("X_API_SECRET")
	accessToken := os.Getenv("X_ACCESS_TOKEN")
	accessSecret := os.Getenv("X_ACCESS_SECRET")

	xEnabled := apiKey != "" && apiSecret != "" && accessToken != "" && accessSecret != ""

	// Bluesky credentials
	bskyHandle := os.Getenv("BLUESKY_HANDLE")
	bskyPassword := os.Getenv("BLUESKY_PASSWORD")

	bskyEnabled := bskyHandle != "" && bskyPassword != ""

	if !xEnabled && !bskyEnabled {
		log.Fatal("No platform credentials configured. Set X_API_KEY/X_API_SECRET/X_ACCESS_TOKEN/X_ACCESS_SECRET for X, or BLUESKY_HANDLE/BLUESKY_PASSWORD for Bluesky.")
	}

	// Load rate limit configuration from environment
	maxVotesToCheck := getEnvInt("MAX_VOTES_TO_CHECK", 50)
	maxXPostsPerRun := getEnvInt("X_MAX_POSTS_PER_RUN", 10)
	maxBskyPostsPerRun := getEnvInt("BLUESKY_MAX_POSTS_PER_RUN", 10)

	fmt.Printf("Configuration: Check last %d votes\n", maxVotesToCheck)

	// Load contacts for X handle tagging
	contactsPath := filepath.Join("data", "contacts.yaml")
	contactMapper, err := contacts.LoadContacts(contactsPath)
	if err != nil {
		log.Printf("Warning: Could not load contacts for tagging: %v", err)
		contactMapper = nil // Continue without tagging
	}

	// Create API client
	client := zurichapi.NewClient()

	// --- X Platform ---
	if xEnabled {
		fmt.Println("\n━━━ X/Twitter ━━━")

		voteLog, err := votelog.Load(votelog.PlatformX)
		if err != nil {
			log.Fatalf("Error loading X vote log: %v", err)
		}
		fmt.Printf("Loaded X vote log: %d votes already posted\n", voteLog.Count())

		groups, err := voteposting.PrepareVoteGroups(client, voteLog, maxVotesToCheck)
		if err != nil {
			log.Fatalf("Error preparing votes for X: %v", err)
		}

		if len(groups) == 0 {
			fmt.Println("No new votes to post on X!")
		} else {
			fmt.Printf("Found %d group(s) to post on X\n", len(groups))

			xPlatform := x.NewXPlatform(
				apiKey, apiSecret, accessToken, accessSecret,
				contactMapper,
				maxXPostsPerRun,
			)

			posted, err := voteposting.PostToPlatform(groups, xPlatform, voteLog, false)
			if err != nil {
				log.Printf("Error posting to X: %v", err)
			} else {
				fmt.Printf("🎉 Posted %d new group(s) to X!\n", posted)
			}
		}
	}

	// --- Bluesky Platform ---
	if bskyEnabled {
		fmt.Println("\n━━━ Bluesky ━━━")

		voteLog, err := votelog.Load(votelog.PlatformBluesky)
		if err != nil {
			log.Fatalf("Error loading Bluesky vote log: %v", err)
		}
		fmt.Printf("Loaded Bluesky vote log: %d votes already posted\n", voteLog.Count())

		groups, err := voteposting.PrepareVoteGroups(client, voteLog, maxVotesToCheck)
		if err != nil {
			log.Fatalf("Error preparing votes for Bluesky: %v", err)
		}

		if len(groups) == 0 {
			fmt.Println("No new votes to post on Bluesky!")
		} else {
			fmt.Printf("Found %d group(s) to post on Bluesky\n", len(groups))

			bskyPlatform := bluesky.NewBlueskyPlatform(
				bskyHandle, bskyPassword,
				maxBskyPostsPerRun,
			)

			posted, err := voteposting.PostToPlatform(groups, bskyPlatform, voteLog, false)
			if err != nil {
				log.Printf("Error posting to Bluesky: %v", err)
			} else {
				fmt.Printf("🎉 Posted %d new group(s) to Bluesky!\n", posted)
			}
		}
	}
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
