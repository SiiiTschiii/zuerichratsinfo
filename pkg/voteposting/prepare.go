package voteposting

import (
	"errors"
	"fmt"
	"log"
	"sort"
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

	applyCompletenessGate(groups)

	return groups, nil
}

// applyCompletenessGate drops the member list from any vote whose recorded
// members do not account for the totals the source reported.
//
// A partial member list still renders a plausible-looking Fraktion table — it
// just understates whichever factions are missing, silently and unfalsifiably.
// Dropping it degrades the post to totals only, which are reported
// independently and remain correct.
//
// This lives here rather than in the adapters so every source is held to the
// same bar, and here rather than in the formatters so the decision is made once
// instead of once per platform.
func applyCompletenessGate(groups [][]votes.Vote) {
	for _, group := range groups {
		for i := range group {
			v := &group[i]
			if votes.IsBreakdownComplete(*v) {
				continue
			}
			total, _ := v.TotalRecorded()
			log.Printf("⚠️  %s: %d member votes for reported totals of %d — posting totals only, without the Fraktion breakdown",
				v.SourceID, len(v.MemberVotes), total)
			v.MemberVotes = nil
		}
	}
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

// VoteLogs resolves the log a group belongs in, keyed by jurisdiction. A
// channel serving several jurisdictions posts them through one platform
// instance but must record each in its own log.
type VoteLogs map[string]*votelog.VoteLog

// SingleLog builds a VoteLogs for the common one-jurisdiction case.
func SingleLog(jurisdiction string, vl *votelog.VoteLog) VoteLogs {
	return VoteLogs{jurisdiction: vl}
}

func (l VoteLogs) forGroup(group []votes.Vote) (*votelog.VoteLog, error) {
	key := group[0].Jurisdiction
	vl, ok := l[key]
	if !ok {
		return nil, fmt.Errorf("no vote log configured for jurisdiction %q", key)
	}
	return vl, nil
}

// MergeOldestFirst interleaves groups from several jurisdictions into the order
// they should be posted: oldest vote first, ties broken by the order the
// jurisdictions were passed in.
//
// This is what stops a busy sitting in one chamber from starving the other when
// they share a per-run budget, and it preserves today's behaviour of draining a
// backlog chronologically.
func MergeOldestFirst(perJurisdiction ...[][]votes.Vote) [][]votes.Vote {
	var merged [][]votes.Vote
	for _, groups := range perJurisdiction {
		merged = append(merged, groups...)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return groupDate(merged[i]).Before(groupDate(merged[j]))
	})
	return merged
}

// groupDate is the earliest known vote date in a group. Groups whose dates are
// all unknown sort last, so a data problem cannot jump the queue.
func groupDate(group []votes.Vote) time.Time {
	var earliest time.Time
	for _, v := range group {
		if v.Date.IsZero() {
			continue
		}
		if earliest.IsZero() || v.Date.Before(earliest) {
			earliest = v.Date
		}
	}
	if earliest.IsZero() {
		return time.Unix(1<<62, 0)
	}
	return earliest
}

// PostToPlatform posts vote groups to a platform.
// If dryRun is true, only prints the content without posting.
// Returns the number of groups successfully posted.
//
// The platform instance carries the per-run budget, so callers must pass the
// same instance for every jurisdiction on a channel — constructing one per
// jurisdiction resets the counter and doubles what the account posts.
func PostToPlatform(
	groups [][]votes.Vote,
	platform platforms.Platform,
	voteLogs VoteLogs,
	dryRun bool,
) (int, error) {
	posted := 0

	var firstUnsupportedErr error

	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
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
			fmt.Printf("  📋 %s %s (%s) — %d vote(s) [not visible in post]\n",
				group[0].Jurisdiction,
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
			voteLog, err := voteLogs.forGroup(group)
			if err != nil {
				return posted, err
			}

			// Real posting — log which group for tracing
			fmt.Printf("📋 %s %s (%s) — %d vote(s):\n",
				group[0].Jurisdiction,
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
