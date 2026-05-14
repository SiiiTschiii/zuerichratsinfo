package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/bluesky"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/instagram"
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

	igUserID := os.Getenv("IG_USER_ID")
	igAccessToken := os.Getenv("IG_ACCESS_TOKEN")
	igGithubToken := os.Getenv("GITHUB_TOKEN")
	igRepoOwner := os.Getenv("IG_REPO_OWNER")
	igRepoName := os.Getenv("IG_REPO_NAME")

	igEnabled := igUserID != "" && igAccessToken != "" && igGithubToken != "" && igRepoOwner != "" && igRepoName != ""

	if !xEnabled && !bskyEnabled && !igEnabled {
		log.Fatal("No platform credentials configured. Set X_API_KEY/X_API_SECRET/X_ACCESS_TOKEN/X_ACCESS_SECRET for X, BLUESKY_HANDLE/BLUESKY_PASSWORD for Bluesky, or IG_USER_ID/IG_ACCESS_TOKEN/GITHUB_TOKEN/IG_REPO_OWNER/IG_REPO_NAME for Instagram.")
	}

	if !xEnabled {
		log.Println("⚠️  X/Twitter not configured (missing X_API_KEY/X_API_SECRET/X_ACCESS_TOKEN/X_ACCESS_SECRET)")
	}
	if !bskyEnabled {
		log.Println("⚠️  Bluesky not configured (missing BLUESKY_HANDLE/BLUESKY_PASSWORD)")
	}
	if !igEnabled {
		log.Println("⚠️  Instagram not configured (missing IG_USER_ID/IG_ACCESS_TOKEN/GITHUB_TOKEN/IG_REPO_OWNER/IG_REPO_NAME)")
	}

	// Load rate limit configuration from environment
	maxVotesToCheck := getEnvInt("MAX_VOTES_TO_CHECK", 50)
	maxVoteAgeDays := getEnvInt("MAX_VOTE_AGE_DAYS", 90)
	maxXPostsPerRun := getEnvInt("X_MAX_POSTS_PER_RUN", 10)
	maxBskyPostsPerRun := getEnvInt("BLUESKY_MAX_POSTS_PER_RUN", 10)
	maxIGPostsPerRun := getEnvInt("IG_MAX_POSTS_PER_RUN", 5)
	xMaxChars := getEnvInt("X_MAX_CHARS", x.DefaultMaxChars)

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

	skipVoteLog := os.Getenv("SKIP_VOTE_LOG") == "true"
	hasUnsupportedVotes := false

	// --- X Platform ---
	if xEnabled {
		xPlatform := x.NewXPlatform(
			apiKey, apiSecret, accessToken, accessSecret,
			contactMapper,
			maxXPostsPerRun,
		)
		xPlatform.SetMaxChars(xMaxChars)

		if unsupported := runPlatform("X/Twitter", votelog.PlatformX, xPlatform, client, skipVoteLog, maxVotesToCheck, maxVoteAgeDays); unsupported {
			hasUnsupportedVotes = true
		}
	}

	// --- Bluesky Platform ---
	if bskyEnabled {
		bskyPlatform := bluesky.NewBlueskyPlatform(
			bskyHandle, bskyPassword,
			maxBskyPostsPerRun,
			contactMapper,
		)

		if unsupported := runPlatform("Bluesky", votelog.PlatformBluesky, bskyPlatform, client, skipVoteLog, maxVotesToCheck, maxVoteAgeDays); unsupported {
			hasUnsupportedVotes = true
		}
	}

	// --- Instagram Platform ---
	if igEnabled {
		igPlatform := instagram.NewInstagramPlatformWithCredentials(
			igUserID, igAccessToken, igGithubToken, igRepoOwner, igRepoName, maxIGPostsPerRun,
		)
		igPlatform.SetContactMapper(contactMapper)

		if unsupported := runPlatform("Instagram", votelog.PlatformInstagram, igPlatform, client, skipVoteLog, maxVotesToCheck, maxVoteAgeDays); unsupported {
			hasUnsupportedVotes = true
		}
	}

	if hasUnsupportedVotes {
		log.Println("❌ Action failed: one or more votes have an unrecognised format. Check warnings above.")
		os.Exit(1)
	}
}

// runPlatform loads the vote log, prepares vote groups, and posts to the given
// platform. It returns true if any votes were skipped due to an unsupported
// format.
func runPlatform(
	displayName string,
	platform votelog.Platform,
	poster platforms.Platform,
	client *zurichapi.Client,
	skipVoteLog bool,
	maxVotesToCheck, maxVoteAgeDays int,
) bool {
	fmt.Printf("\n━━━ %s ━━━\n", displayName)

	var vl *votelog.VoteLog
	if skipVoteLog {
		vl = votelog.NewNoOp(platform)
		fmt.Println("⚠️  SKIP_VOTE_LOG=true — treating all votes as unposted, not saving vote log")
	} else {
		var err error
		vl, err = votelog.Load(platform)
		if err != nil {
			log.Fatalf("Error loading %s vote log: %v", displayName, err)
		}
		fmt.Printf("Loaded %s vote log: %d votes already posted\n", displayName, vl.Count())
	}

	groups, err := voteposting.PrepareVoteGroups(client, vl, maxVotesToCheck, maxVoteAgeDays)
	if err != nil {
		log.Fatalf("Error preparing votes for %s: %v", displayName, err)
	}

	if len(groups) == 0 {
		fmt.Printf("No new votes to post on %s!\n", displayName)
		return false
	}

	fmt.Printf("Found %d group(s) to post on %s\n", len(groups), displayName)

	posted, err := voteposting.PostToPlatform(groups, poster, vl, false)
	if err != nil {
		if errors.Is(err, voteposting.ErrUnsupportedVoteType) {
			if posted > 0 {
				fmt.Printf("Posted %d group(s) to %s (some skipped — see warnings above)\n", posted, displayName)
			}
			return true
		}
		log.Printf("Error posting to %s: %v", displayName, err)
		return false
	}

	fmt.Printf("🎉 Posted %d new group(s) to %s!\n", posted, displayName)
	return false
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
