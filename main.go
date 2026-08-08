package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/siiitschiii/zuerichratsinfo/pkg/config"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/bluesky"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/instagram"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// channelPlatform pairs a constructed platform with the vote-log key it uses.
//
// The instance is built once per channel and shared across that channel's
// jurisdictions. The per-run post budget is a counter on the instance, so
// rebuilding it per jurisdiction would reset the counter and silently multiply
// what the account posts in an hour.
type channelPlatform struct {
	displayName string
	logPlatform votelog.Platform
	poster      platforms.Platform
}

func main() {
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	maxVotesToCheck := getEnvInt("MAX_VOTES_TO_CHECK", 50)
	skipVoteLog := os.Getenv("SKIP_VOTE_LOG") == "true"

	fmt.Printf("Configuration: Check last %d votes\n", maxVotesToCheck)

	hasErrors := false
	anyPlatformConfigured := false

	for _, channel := range config.Channels() {
		// Only jurisdictions explicitly cleared to post. A newly registered
		// body ships disabled: its vote log has to be seeded and its first
		// posts reviewed before it runs unattended, and merging code should not
		// substitute for either.
		jurisdictions, err := channel.EnabledJurisdictions()
		if err != nil {
			log.Fatalf("Error resolving channel %q: %v", channel.Key, err)
		}
		if len(jurisdictions) == 0 {
			log.Printf("⚠️  Channel %q: no enabled jurisdictions, skipping", channel.Key)
			continue
		}

		plats := buildPlatforms(channel, jurisdictions)
		if len(plats) == 0 {
			log.Printf("⚠️  Channel %q: no platform credentials configured, skipping", channel.Key)
			continue
		}
		anyPlatformConfigured = true

		for _, p := range plats {
			if runChannelPlatform(channel, jurisdictions, p, skipVoteLog, maxVotesToCheck) {
				hasErrors = true
			}
		}
	}

	if !anyPlatformConfigured {
		log.Fatal("No platform credentials configured for any channel. Set X_API_KEY/X_API_SECRET/X_ACCESS_TOKEN/X_ACCESS_SECRET for X, BLUESKY_HANDLE/BLUESKY_PASSWORD for Bluesky, or IG_USER_ID/IG_ACCESS_TOKEN/GITHUB_TOKEN/IG_REPO_OWNER/IG_REPO_NAME for Instagram.")
	}

	if hasErrors {
		log.Println("❌ Action failed: one or more platforms encountered errors. Check logs above.")
		os.Exit(1)
	}
}

// buildPlatforms constructs one instance per platform for a channel, using that
// channel's credentials. Platforms without complete credentials are skipped.
func buildPlatforms(channel config.Channel, jurisdictions []config.Jurisdiction) []channelPlatform {
	// Tagging is configured per jurisdiction but a platform instance is per
	// channel, so its mapper has to cover every jurisdiction the channel serves.
	contactMapper := loadContacts(jurisdictions)

	var plats []channelPlatform

	apiKey := channel.Env("X_API_KEY")
	apiSecret := channel.Env("X_API_SECRET")
	accessToken := channel.Env("X_ACCESS_TOKEN")
	accessSecret := channel.Env("X_ACCESS_SECRET")
	if apiKey != "" && apiSecret != "" && accessToken != "" && accessSecret != "" {
		xPlatform := x.NewXPlatform(
			apiKey, apiSecret, accessToken, accessSecret,
			contactMapper,
			channel.EnvInt("X_MAX_POSTS_PER_RUN", 10),
		)
		xPlatform.SetMaxChars(channel.EnvInt("X_MAX_CHARS", x.DefaultMaxChars))
		plats = append(plats, channelPlatform{"X/Twitter", votelog.PlatformX, xPlatform})
	} else {
		log.Printf("⚠️  Channel %q: X/Twitter not configured (missing X_API_KEY/X_API_SECRET/X_ACCESS_TOKEN/X_ACCESS_SECRET)", channel.Key)
	}

	bskyHandle := channel.Env("BLUESKY_HANDLE")
	bskyPassword := channel.Env("BLUESKY_PASSWORD")
	if bskyHandle != "" && bskyPassword != "" {
		plats = append(plats, channelPlatform{"Bluesky", votelog.PlatformBluesky, bluesky.NewBlueskyPlatform(
			bskyHandle, bskyPassword,
			channel.EnvInt("BLUESKY_MAX_POSTS_PER_RUN", 10),
			contactMapper,
		)})
	} else {
		log.Printf("⚠️  Channel %q: Bluesky not configured (missing BLUESKY_HANDLE/BLUESKY_PASSWORD)", channel.Key)
	}

	igUserID := channel.Env("IG_USER_ID")
	igAccessToken := channel.Env("IG_ACCESS_TOKEN")
	igGithubToken := channel.Env("GITHUB_TOKEN")
	igRepoOwner := channel.Env("IG_REPO_OWNER")
	igRepoName := channel.Env("IG_REPO_NAME")
	if igUserID != "" && igAccessToken != "" && igGithubToken != "" && igRepoOwner != "" && igRepoName != "" {
		igPlatform := instagram.NewInstagramPlatformWithCredentials(
			igUserID, igAccessToken, igGithubToken, igRepoOwner, igRepoName,
			channel.EnvInt("IG_MAX_POSTS_PER_RUN", 5),
		)
		igPlatform.SetContactMapper(contactMapper)
		plats = append(plats, channelPlatform{"Instagram", votelog.PlatformInstagram, igPlatform})
	} else {
		log.Printf("⚠️  Channel %q: Instagram not configured (missing IG_USER_ID/IG_ACCESS_TOKEN/GITHUB_TOKEN/IG_REPO_OWNER/IG_REPO_NAME)", channel.Key)
	}

	return plats
}

// loadContacts merges every configured jurisdiction's contacts file. A missing
// or empty file is normal — a jurisdiction ships before its contacts are
// curated — so this degrades to no tagging rather than failing the run.
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

// runChannelPlatform posts one channel's jurisdictions to one platform,
// sharing a single per-run budget across them. It returns true if anything
// went wrong.
func runChannelPlatform(
	channel config.Channel,
	jurisdictions []config.Jurisdiction,
	p channelPlatform,
	skipVoteLog bool,
	maxVotesToCheck int,
) bool {
	fmt.Printf("\n━━━ %s / %s ━━━\n", channel.Key, p.displayName)

	logs := make(voteposting.VoteLogs, len(jurisdictions))
	var perJurisdiction [][][]votes.Vote

	for _, j := range jurisdictions {
		var vl *votelog.VoteLog
		if skipVoteLog {
			vl = votelog.NewNoOp(j.Key, p.logPlatform)
			fmt.Printf("⚠️  SKIP_VOTE_LOG=true — treating all %s votes as unposted, not saving vote log\n", j.Key)
		} else {
			var err error
			vl, err = votelog.Load(j.Key, p.logPlatform)
			if err != nil {
				log.Fatalf("Error loading %s/%s vote log: %v", j.Key, p.displayName, err)
			}
			fmt.Printf("Loaded %s/%s vote log: %d votes already posted\n", j.Key, p.displayName, vl.Count())

			// Finish the move to the per-jurisdiction layout now rather than
			// when this platform next publishes something. Save is otherwise
			// only reached after a post, so a council in recess would leave
			// the logs — and the workflow's transitional handling — in the old
			// layout for as long as the recess lasts.
			if legacy := vl.LoadedFromLegacyPath(); legacy != "" {
				if err := vl.Save(); err != nil {
					log.Fatalf("Error migrating %s/%s vote log from %s: %v", j.Key, p.displayName, legacy, err)
				}
				fmt.Printf("Migrated %s/%s vote log from %s\n", j.Key, p.displayName, legacy)
			}
		}
		logs[j.Key] = vl

		groups, err := voteposting.PrepareVoteGroups(j.NewSource(), vl, maxVotesToCheck, j.MaxAgeDays)
		if err != nil {
			log.Fatalf("Error preparing %s votes for %s: %v", j.Key, p.displayName, err)
		}
		if len(groups) > 0 {
			fmt.Printf("Found %d group(s) from %s\n", len(groups), j.Key)
		}
		perJurisdiction = append(perJurisdiction, groups)
	}

	merged := voteposting.MergeOldestFirst(perJurisdiction...)
	if len(merged) == 0 {
		fmt.Printf("No new votes to post on %s!\n", p.displayName)
		return false
	}

	posted, err := voteposting.PostToPlatform(merged, p.poster, logs, false)
	if err != nil {
		if errors.Is(err, voteposting.ErrUnsupportedVoteType) && posted > 0 {
			fmt.Printf("Posted %d group(s) to %s (some skipped — see warnings above)\n", posted, p.displayName)
		}
		log.Printf("❌ Error posting to %s: %v", p.displayName, err)
		return true
	}

	fmt.Printf("🎉 Posted %d new group(s) to %s!\n", posted, p.displayName)
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
