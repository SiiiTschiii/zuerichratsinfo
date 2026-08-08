// check_unposted is a dry-run mirror of main.go for local debugging.
// It reads the actual vote logs, respects MAX_VOTES_TO_CHECK and MAX_POSTS_PER_RUN,
// and prints what would be posted without making any API calls.
//
// Usage:
//
//	go run ./cmd/check_unposted [-platform x|bluesky] [-jurisdiction KEY] [-n N] [-max-posts M]
//
// Flags override the corresponding environment variables.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/config"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/bluesky"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

func main() {
	platformFlag := flag.String("platform", "", "platform to check: x, bluesky (default: all)")
	jurisdictionFlag := flag.String("jurisdiction", "", "limit to one jurisdiction (default: every jurisdiction on the channel)")
	nFlag := flag.Int("n", 0, "override MAX_VOTES_TO_CHECK")
	maxPostsFlag := flag.Int("max-posts", 0, "override MAX_POSTS_PER_RUN for the chosen platform")
	flag.Parse()

	showX := *platformFlag == "" || strings.EqualFold(*platformFlag, "x")
	showBluesky := *platformFlag == "" || strings.EqualFold(*platformFlag, "bluesky") || strings.EqualFold(*platformFlag, "bsky")
	if !showX && !showBluesky {
		log.Fatalf("Unknown platform %q. Use: x, bluesky", *platformFlag)
	}

	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	maxVotesToCheck := getEnvInt("MAX_VOTES_TO_CHECK", 50)
	if *nFlag > 0 {
		maxVotesToCheck = *nFlag
	}

	fmt.Printf("Configuration: check last %d votes", maxVotesToCheck)
	if *platformFlag != "" {
		fmt.Printf(", platform: %s", *platformFlag)
	}
	fmt.Println()

	for _, channel := range config.Channels() {
		jurisdictions, err := channel.ResolveJurisdictions()
		if err != nil {
			log.Fatalf("Error resolving channel %q: %v", channel.Key, err)
		}
		if *jurisdictionFlag != "" {
			jurisdictions = filterJurisdictions(jurisdictions, *jurisdictionFlag)
			if len(jurisdictions) == 0 {
				continue
			}
		}

		contactMapper := loadContacts(jurisdictions)

		if showX {
			maxPosts := pick(*maxPostsFlag, channel.EnvInt("X_MAX_POSTS_PER_RUN", 10))
			check(channel.Key, "X/Twitter", votelog.PlatformX, jurisdictions, maxVotesToCheck, maxPosts,
				x.NewXPlatform("", "", "", "", contactMapper, maxPosts))
		}

		if showBluesky {
			maxPosts := pick(*maxPostsFlag, channel.EnvInt("BLUESKY_MAX_POSTS_PER_RUN", 10))
			check(channel.Key, "Bluesky", votelog.PlatformBluesky, jurisdictions, maxVotesToCheck, maxPosts,
				bluesky.NewBlueskyPlatform("", "", maxPosts, contactMapper))
		}
	}
}

// check mirrors main.go's per-channel-platform run: one platform instance, one
// shared budget, jurisdictions merged oldest-first. Keeping the shapes identical
// is the point — a dry run that batched differently would not predict anything.
func check(
	channelKey, displayName string,
	logPlatform votelog.Platform,
	jurisdictions []config.Jurisdiction,
	maxVotesToCheck, maxPosts int,
	poster platforms.Platform,
) {
	fmt.Printf("\n━━━ %s / %s ━━━\n", channelKey, displayName)

	logs := make(voteposting.VoteLogs, len(jurisdictions))
	var perJurisdiction [][][]votes.Vote

	for _, j := range jurisdictions {
		voteLog, err := votelog.Load(j.Key, logPlatform)
		if err != nil {
			log.Fatalf("Error loading %s/%s vote log: %v", j.Key, displayName, err)
		}
		fmt.Printf("Loaded %s/%s vote log: %d votes already posted\n", j.Key, displayName, voteLog.Count())
		logs[j.Key] = voteLog

		groups, err := voteposting.PrepareVoteGroups(j.NewSource(), voteLog, maxVotesToCheck, j.MaxAgeDays)
		if err != nil {
			log.Fatalf("Error preparing %s votes for %s: %v", j.Key, displayName, err)
		}
		perJurisdiction = append(perJurisdiction, groups)
	}

	merged := voteposting.MergeOldestFirst(perJurisdiction...)
	if len(merged) == 0 {
		fmt.Printf("✨ No new votes to post on %s!\n", displayName)
		return
	}

	fmt.Printf("Found %d group(s) — would post up to %d per run\n\n", len(merged), maxPosts)
	if _, err := voteposting.PostToPlatform(merged, poster, logs, true); err != nil {
		log.Printf("Error: %v", err)
	}
}

func loadContacts(jurisdictions []config.Jurisdiction) *contacts.Mapper {
	var paths []string
	for _, j := range jurisdictions {
		paths = append(paths, contacts.PathFor(j.Key))
	}
	mapper, err := contacts.LoadContactFiles(paths...)
	if err != nil {
		log.Printf("Warning: Could not load contacts for tagging: %v", err)
		return nil
	}
	return mapper
}

func filterJurisdictions(js []config.Jurisdiction, key string) []config.Jurisdiction {
	var out []config.Jurisdiction
	for _, j := range js {
		if j.Key == key {
			out = append(out, j)
		}
	}
	return out
}

func pick(override, fallback int) int {
	if override > 0 {
		return override
	}
	return fallback
}

func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}
