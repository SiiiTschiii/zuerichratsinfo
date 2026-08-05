package voteposting

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// ErrUnsupportedVoteType is returned when a group contains a vote with an
// unrecognised count format. The group is skipped (not posted, not logged).
var ErrUnsupportedVoteType = errors.New("unsupported vote type")

// ErrInconsistentDecision is returned when a vote's Decision field contradicts
// its raw vote counts (e.g. the source says "Ja" but Nein > Ja). The entire run
// is aborted so no platform posts incorrect results.
var ErrInconsistentDecision = errors.New("decision contradicts vote counts")

// PrepareVoteGroups fetches recent votes from a source, drops the ones already
// posted, and groups the rest for posting. It is platform-agnostic — every
// platform gets the same preparation.
//
// maxAgeDays limits how old a vote's sitting date can be (0 = no limit). It is
// not a freshness knob: it is the backstop that stops a re-indexed old vote
// being posted a second time when it falls outside what the vote log covers.
func PrepareVoteGroups(
	src votes.Source,
	voteLog *votelog.VoteLog,
	maxVotesToFetch int,
	maxAgeDays int,
) ([][]votes.Vote, error) {
	fetched, err := src.FetchRecent(maxVotesToFetch)
	if err != nil {
		return nil, err
	}

	if len(fetched) == 0 {
		return nil, nil
	}

	// Filter out already posted votes BEFORE grouping — cheaper, and grouping
	// can trigger further source calls.
	unposted := filterUnpostedVotes(fetched, voteLog)
	unposted = filterOldVotes(unposted, maxAgeDays)

	if len(unposted) == 0 {
		return nil, nil
	}

	groups, err := src.GroupByAffair(unposted)
	if err != nil {
		return nil, err
	}

	// Validate every vote's Decision against its counts. If the source has
	// published wrong data (e.g. "Ja" when Nein > Ja), abort before posting
	// anything so the workflow fails and alerts the operator.
	for _, group := range groups {
		for _, v := range group {
			if !voteformat.IsDecisionConsistent(v.Decision, v.Yes, v.No) {
				return nil, fmt.Errorf("%w: %s (%s) has Decision=%q but Ja=%d Nein=%d",
					ErrInconsistentDecision,
					v.Affair.Number, v.Title,
					v.Decision, *v.Yes, *v.No,
				)
			}
		}
	}

	return groups, nil
}

// filterUnpostedVotes filters out votes that have already been posted.
func filterUnpostedVotes(vs []votes.Vote, voteLog *votelog.VoteLog) []votes.Vote {
	var unposted []votes.Vote
	for _, v := range vs {
		if !voteLog.IsPosted(v.SourceID) {
			unposted = append(unposted, v)
		}
	}
	return unposted
}

// filterOldVotes drops votes whose sitting date is more than maxAgeDays old.
// Votes with an unknown date are kept: an unparseable date is a data problem,
// and silently dropping a vote is worse than posting one that is a little old.
func filterOldVotes(vs []votes.Vote, maxAgeDays int) []votes.Vote {
	if maxAgeDays <= 0 {
		return vs
	}
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	var recent []votes.Vote
	for _, v := range vs {
		if !v.Date.IsZero() && v.Date.Before(cutoff) {
			log.Printf("⚠️  Skipping old vote %s (date %s, older than %d days)",
				v.SourceID, v.DateString(), maxAgeDays)
			continue
		}
		recent = append(recent, v)
	}
	return recent
}

// PostToPlatform posts vote groups to a platform.
// If dryRun is true, only prints the content without posting.
// Returns the number of groups successfully posted.
func PostToPlatform(
	groups [][]votes.Vote,
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

		if dryRun {
			// Dry run: print and respect the same per-run limit as real posting
			if posted > 0 {
				fmt.Println()
			}
			fmt.Println("────────────────────────────────────────────────────────────────────────────────")
			fmt.Printf("  📋 %s (%s) — %d vote(s) [not visible in post]\n",
				group[0].Affair.Number,
				group[0].DateString(),
				len(group),
			)
			fmt.Println("────────────────────────────────────────────────────────────────────────────────")
			fmt.Println()
			fmt.Println(content.String())
			posted++
			if posted >= platform.MaxPostsPerRun() {
				break
			}
		} else {
			// Real posting — log which group for tracing
			fmt.Printf("📋 %s (%s) — %d vote(s):\n",
				group[0].Affair.Number,
				group[0].DateString(),
				len(group),
			)
			for _, v := range group {
				fmt.Printf("   %s\n", v.SourceURL)
			}

			shouldContinue, err := platform.Post(content)
			if err != nil {
				return posted, err
			}

			// Mark all votes in the group as posted
			for _, v := range group {
				voteLog.MarkAsPosted(v.SourceID)
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
func validateGroupCounts(group []votes.Vote) error {
	for _, v := range group {
		if voteformat.IsUnsupportedVoteType(voteformat.CountsOf(v)) {
			return fmt.Errorf("%w: vote %s (%q, type=%q) has all-zero counts",
				ErrUnsupportedVoteType, v.SourceID, v.Subtitle, v.Type)
		}
	}
	return nil
}
