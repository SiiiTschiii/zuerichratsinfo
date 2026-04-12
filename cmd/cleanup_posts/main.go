package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/xapi"
)

func main() {
	platform := flag.String("platform", "all", "platform to clean: x, bluesky, or all")
	flag.Parse()

	if *platform != "all" && *platform != "x" && *platform != "bluesky" {
		log.Fatalf("Unknown platform %q — use x, bluesky, or all", *platform)
	}

	// X cleanup
	if *platform == "all" || *platform == "x" {
		apiKey := os.Getenv("X_API_KEY")
		apiSecret := os.Getenv("X_API_SECRET")
		accessToken := os.Getenv("X_ACCESS_TOKEN")
		accessSecret := os.Getenv("X_ACCESS_SECRET")

		if apiKey == "" || apiSecret == "" || accessToken == "" || accessSecret == "" {
			if *platform == "x" {
				log.Fatal("X credentials required but not set")
			}
			fmt.Println("Skipping X (no credentials)")
		} else {
			cleanupX(apiKey, apiSecret, accessToken, accessSecret)
		}
	}

	// Bluesky cleanup
	if *platform == "all" || *platform == "bluesky" {
		handle := os.Getenv("BLUESKY_HANDLE")
		password := os.Getenv("BLUESKY_PASSWORD")

		if handle == "" || password == "" {
			if *platform == "bluesky" {
				log.Fatal("Bluesky credentials required but not set")
			}
			fmt.Println("Skipping Bluesky (no credentials)")
		} else {
			cleanupBluesky(handle, password)
		}
	}
}

func cleanupX(apiKey, apiSecret, accessToken, accessSecret string) {
	fmt.Println("━━━ Cleaning up X ━━━")

	userID, err := xapi.GetAuthenticatedUserID(apiKey, apiSecret, accessToken, accessSecret)
	if err != nil {
		log.Printf("Failed to get X user ID: %v", err)
		return
	}
	fmt.Printf("Authenticated as user ID: %s\n", userID)

	totalDeleted := 0
	for {
		tweetIDs, err := xapi.GetUserTweets(apiKey, apiSecret, accessToken, accessSecret, userID)
		if err != nil {
			log.Printf("Failed to list tweets: %v", err)
			break
		}
		if len(tweetIDs) == 0 {
			break
		}

		fmt.Printf("Found %d tweets to delete...\n", len(tweetIDs))
		for _, id := range tweetIDs {
			if err := xapi.DeleteTweet(apiKey, apiSecret, accessToken, accessSecret, id); err != nil {
				log.Printf("Failed to delete tweet %s: %v", id, err)
				continue
			}
			totalDeleted++
			// Small delay to avoid rate limiting
			time.Sleep(200 * time.Millisecond)
		}
	}

	fmt.Printf("Deleted %d tweets from X\n", totalDeleted)
}

func cleanupBluesky(handle, password string) {
	fmt.Println("━━━ Cleaning up Bluesky ━━━")

	session, err := bskyapi.CreateSession(handle, password)
	if err != nil {
		log.Printf("Failed to create Bluesky session: %v", err)
		return
	}
	fmt.Printf("Authenticated as: %s (%s)\n", session.Handle, session.DID)

	totalDeleted := 0
	for {
		uris, err := bskyapi.ListRecords(session, "app.bsky.feed.post")
		if err != nil {
			log.Printf("Failed to list Bluesky posts: %v", err)
			break
		}
		if len(uris) == 0 {
			break
		}

		fmt.Printf("Found %d posts to delete...\n", len(uris))
		for _, uri := range uris {
			if err := bskyapi.DeleteRecord(session, uri); err != nil {
				log.Printf("Failed to delete %s: %v", uri, err)
				continue
			}
			totalDeleted++
		}
	}

	fmt.Printf("Deleted %d posts from Bluesky\n", totalDeleted)
}
