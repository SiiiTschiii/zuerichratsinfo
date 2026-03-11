package voteposting

import (
	"errors"
	"fmt"
	"log"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// ErrUnsupportedVoteType is returned when a group contains a vote with an
// unrecognised count format. The group is skipped (not posted, not logged).
var ErrUnsupportedVoteType = errors.New("unsupported vote type")

// PrepareVoteGroups prepares vote groups for posting
// It fetches recent votes, filters out already posted ones, and groups them by Geschäft
// This is platform-agnostic - the same preparation for all platforms
func PrepareVoteGroups(
	client *zurichapi.Client,
	voteLog *votelog.VoteLog,
	maxVotesToFetch int,
) ([][]zurichapi.Abstimmung, error) {
	// Fetch recent votes
	votes, err := client.FetchRecentAbstimmungen(maxVotesToFetch)
	if err != nil {
		return nil, err
	}

	if len(votes) == 0 {
		return nil, nil
	}

	// Filter out already posted votes BEFORE grouping
	// This is more efficient than grouping first
	unpostedVotes := filterUnpostedVotes(votes, voteLog)

	if len(unpostedVotes) == 0 {
		return nil, nil
	}

	// Group votes by Geschäft and date
	// This includes ensuring the last vote's group is complete
	groups, err := client.GroupAbstimmungenByGeschaeft(unpostedVotes)
	if err != nil {
		return nil, err
	}

	return groups, nil
}

// filterUnpostedVotes filters out votes that have already been posted
func filterUnpostedVotes(votes []zurichapi.Abstimmung, voteLog *votelog.VoteLog) []zurichapi.Abstimmung {
	var unposted []zurichapi.Abstimmung
	for _, vote := range votes {
		if !voteLog.IsPosted(vote.OBJGUID) {
			unposted = append(unposted, vote)
		}
	}
	return unposted
}

// PostToPlatform posts vote groups to a platform
// If dryRun is true, only prints the content without posting
// Returns the number of groups successfully posted
func PostToPlatform(
	groups [][]zurichapi.Abstimmung,
	platform platforms.Platform,
	voteLog *votelog.VoteLog,
	dryRun bool,
) (int, error) {
	posted := 0

	var firstUnsupportedErr error

	for _, group := range groups {
		// Validate vote counts before formatting; skip groups with unknown formats
		if err := validateGroupCounts(group); err != nil {
			log.Printf("⚠️  Skipping group (unsupported vote type): %v", err)
			if firstUnsupportedErr == nil {
				firstUnsupportedErr = err
			}
			continue
		}

		// Format the content
		content, err := platform.Format(group)
		if err != nil {
			return posted, err
		}

		// Log which group is being posted (helps trace which Bluesky URIs map to which votes)
		fmt.Printf("📋 %s (%s) — %d vote(s):\n",
			group[0].GeschaeftGrNr,
			group[0].SitzungDatum[:10],
			len(group),
		)
		for _, v := range group {
			fmt.Printf("   https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=%s\n", v.OBJGUID)
		}

		if dryRun {
			// Dry run: print and respect the same per-run limit as real posting
			if posted > 0 {
				fmt.Println()
				fmt.Println("────────────────────────────────────────────────────────────────────────────────")
				fmt.Println()
			}
			fmt.Println(content.String())
			posted++
			if posted >= platform.MaxPostsPerRun() {
				break
			}
		} else {
			// Real posting
			shouldContinue, err := platform.Post(content)
			if err != nil {
				return posted, err
			}

			// Mark all votes in the group as posted
			for _, vote := range group {
				voteLog.MarkAsPosted(vote.OBJGUID)
			}

			// Save vote log after each successful post
			if err := voteLog.Save(); err != nil {
				return posted, err
			}

			posted++

			// Check if we should stop
			if !shouldContinue {
				break
			}
		}
	}

	if firstUnsupportedErr != nil {
		return posted, firstUnsupportedErr
	}
	return posted, nil
}

// validateGroupCounts checks that every vote in a group has a recognisable
// count format (standard Ja/Nein or Auswahl A-E). Returns ErrUnsupportedVoteType
// with details if any vote is unrecognisable.
func validateGroupCounts(group []zurichapi.Abstimmung) error {
	for _, vote := range group {
		c := voteformat.VoteCounts{
			Ja: vote.AnzahlJa, Nein: vote.AnzahlNein,
			Enthaltung: vote.AnzahlEnthaltung, Abwesend: vote.AnzahlAbwesend,
			A: vote.AnzahlA, B: vote.AnzahlB, C: vote.AnzahlC, D: vote.AnzahlD, E: vote.AnzahlE,
		}
		if voteformat.IsUnsupportedVoteType(c) {
			return fmt.Errorf("%w: vote %s (%q, Abstimmungstyp=%q) has all-zero counts",
				ErrUnsupportedVoteType, vote.OBJGUID, vote.Abstimmungstitel, vote.Abstimmungstyp)
		}
	}
	return nil
}
