// check_unposted is a dry-run mirror of main.go for local debugging.
// It reads the actual vote logs, respects MAX_VOTES_TO_CHECK and MAX_POSTS_PER_RUN,
// and prints what would be posted without making any API calls.
//
// Usage:
//
//	go run ./cmd/check_unposted [-platform x|bluesky] [-n N] [-max-posts M]
//
// Flags override the corresponding environment variables.
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"os"
	"strconv"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/bluesky"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	platformFlag := flag.String("platform", "", "platform to check: x, bluesky (default: all)")
	nFlag := flag.Int("n", 0, "override MAX_VOTES_TO_CHECK")
	maxPostsFlag := flag.Int("max-posts", 0, "override MAX_POSTS_PER_RUN for the chosen platform")
	flag.Parse()

	showX := *platformFlag == "" || strings.EqualFold(*platformFlag, "x")
	showBluesky := *platformFlag == "" || strings.EqualFold(*platformFlag, "bluesky") || strings.EqualFold(*platformFlag, "bsky")
	if !showX && !showBluesky {
		log.Fatalf("Unknown platform %q. Use: x, bluesky", *platformFlag)
	}

	maxVotesToCheck := getEnvInt("MAX_VOTES_TO_CHECK", 50)
	if *nFlag > 0 {
		maxVotesToCheck = *nFlag
	}

	maxXPostsPerRun := getEnvInt("X_MAX_POSTS_PER_RUN", 10)
	maxBskyPostsPerRun := getEnvInt("BLUESKY_MAX_POSTS_PER_RUN", 10)
	if *maxPostsFlag > 0 {
		maxXPostsPerRun = *maxPostsFlag
		maxBskyPostsPerRun = *maxPostsFlag
	}

	fmt.Printf("Configuration: check last %d votes", maxVotesToCheck)
	if *platformFlag != "" {
		fmt.Printf(", platform: %s", *platformFlag)
	}
	fmt.Println()

	contactsPath := filepath.Join("data", "contacts.yaml")
	contactMapper, err := contacts.LoadContacts(contactsPath)
	if err != nil {
		log.Printf("Warning: Could not load contacts for tagging: %v", err)
		contactMapper = nil
	}

	client := zurichapi.NewClient()

	if showX {
		fmt.Println("\n━━━ X/Twitter ━━━")
		voteLog, err := votelog.Load(votelog.PlatformX)
		if err != nil {
			log.Fatalf("Error loading X vote log: %v", err)
		}
		fmt.Printf("Loaded X vote log: %d votes already posted\n", voteLog.Count())

		groups, err := voteposting.PrepareVoteGroups(client, voteLog, maxVotesToCheck, 0)
		if err != nil {
			log.Fatalf("Error preparing votes for X: %v", err)
		}
		if len(groups) == 0 {
			fmt.Println("✨ No new votes to post on X!")
		} else {
			fmt.Printf("Found %d group(s) — would post up to %d per run\n\n", len(groups), maxXPostsPerRun)
			xPlatform := x.NewXPlatform("", "", "", "", contactMapper, maxXPostsPerRun)
			_, err = voteposting.PostToPlatform(groups, xPlatform, voteLog, true)
			if err != nil {
				log.Printf("Error: %v", err)
			}
		}
	}

	if showBluesky {
		if showX {
			fmt.Println()
		}
		fmt.Println("━━━ Bluesky ━━━")
		voteLog, err := votelog.Load(votelog.PlatformBluesky)
		if err != nil {
			log.Fatalf("Error loading Bluesky vote log: %v", err)
		}
		fmt.Printf("Loaded Bluesky vote log: %d votes already posted\n", voteLog.Count())

		groups, err := voteposting.PrepareVoteGroups(client, voteLog, maxVotesToCheck, 0)
		if err != nil {
			log.Fatalf("Error preparing votes for Bluesky: %v", err)
		}
		if len(groups) == 0 {
			fmt.Println("✨ No new votes to post on Bluesky!")
		} else {
			fmt.Printf("Found %d group(s) — would post up to %d per run\n\n", len(groups), maxBskyPostsPerRun)
			bskyPlatform := bluesky.NewBlueskyPlatform("", "", maxBskyPostsPerRun, contactMapper)
			_, err = voteposting.PostToPlatform(groups, bskyPlatform, voteLog, true)
			if err != nil {
				log.Printf("Error: %v", err)
			}
		}
	}
}

func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}
