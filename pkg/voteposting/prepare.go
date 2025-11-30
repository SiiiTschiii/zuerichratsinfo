package voteposting

import (
	"fmt"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

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

	for i, group := range groups {
		// Format the content
		content, err := platform.Format(group)
		if err != nil {
			return posted, err
		}

		if dryRun {
			// Dry run: just print
			if i > 0 {
				fmt.Println()
				fmt.Println("────────────────────────────────────────────────────────────────────────────────")
				fmt.Println()
			}
			fmt.Println(content.String())
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

	return posted, nil
}
