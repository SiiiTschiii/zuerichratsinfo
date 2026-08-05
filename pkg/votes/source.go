package votes

import (
	"sort"
	"strconv"
)

// Source is a provider of votes for one jurisdiction.
//
// GroupByAffair is on the interface rather than a free function because
// completing a group can require further source calls: PARIS only returns the
// most recent N votes, so votes from earlier agenda items of the same Geschäft
// have to be fetched before a group can be considered whole.
type Source interface {
	// FetchRecent returns up to limit of the most recent votes, newest first.
	FetchRecent(limit int) ([]Vote, error)

	// GroupByAffair groups votes into the units that get posted together,
	// ordered oldest group first.
	GroupByAffair(votes []Vote) ([][]Vote, error)

	// Jurisdiction describes the body this source serves.
	Jurisdiction() Jurisdiction
}

// GroupByAffairAndDate is the shared grouping rule: one group per business
// matter per sitting day, votes within a group ordered by Sequence, groups
// ordered by date and then by the sequence of their last vote.
//
// Ordering matters beyond aesthetics — the posting pipeline works through
// groups in order and stops at the per-run budget, so this is what makes the
// bot drain a backlog chronologically.
func GroupByAffairAndDate(votes []Vote) [][]Vote {
	if len(votes) == 0 {
		return nil
	}

	key := func(v Vote) string { return v.Affair.Number + "|" + v.DateString() }

	groupMap := make(map[string][]Vote)
	for _, v := range votes {
		groupMap[key(v)] = append(groupMap[key(v)], v)
	}

	for k := range groupMap {
		g := groupMap[k]
		sort.SliceStable(g, func(i, j int) bool { return sequenceLess(g[i].Sequence, g[j].Sequence) })
		groupMap[k] = g
	}

	// Emit groups in order of first occurrence, then sort — matching the
	// original implementation, whose sort is stable with respect to that order.
	seen := make(map[string]bool)
	var groups [][]Vote
	for _, v := range votes {
		if k := key(v); !seen[k] {
			seen[k] = true
			groups = append(groups, groupMap[k])
		}
	}

	sort.SliceStable(groups, func(i, j int) bool {
		dateI, dateJ := groups[i][0].DateString(), groups[j][0].DateString()
		if dateI != dateJ {
			return dateI < dateJ
		}
		lastI := groups[i][len(groups[i])-1].Sequence
		lastJ := groups[j][len(groups[j])-1].Sequence
		return sequenceLess(lastI, lastJ)
	})

	return groups
}

// sequenceLess compares two Sequence values numerically. Unparseable values
// sort as 0, which is what strconv.Atoi's error path yielded before.
func sequenceLess(a, b string) bool {
	ai, _ := strconv.Atoi(a)
	bi, _ := strconv.Atoi(b)
	return ai < bi
}
