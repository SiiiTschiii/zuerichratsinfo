package voteposting

import (
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

func jurisdictionVote(jurisdiction, id, affair, date string) votes.Vote {
	ja := 100
	return votes.Vote{
		SourceID:     id,
		Jurisdiction: jurisdiction,
		Date:         testfixtures.MustDate(date),
		Yes:          &ja,
		Affair:       votes.Affair{Number: affair},
	}
}

// TestSharedBudget_TwoJurisdictionsOneChannel guards the single mistake in this
// design that would be visible to followers: building a platform instance per
// jurisdiction resets the per-run counter, so an account served by two
// jurisdictions would post twice its hourly allowance.
//
// The budget lives on the platform instance, so the correctness condition is
// "one instance, N posts total" — not "N per jurisdiction".
func TestSharedBudget_TwoJurisdictionsOneChannel(t *testing.T) {
	defer setupTempDir(t)()

	const budget = 3

	city := [][]votes.Vote{
		{jurisdictionVote("zurich-city", "city-1", "2026/1", "2026-06-01")},
		{jurisdictionVote("zurich-city", "city-2", "2026/2", "2026-06-03")},
		{jurisdictionVote("zurich-city", "city-3", "2026/3", "2026-06-05")},
	}
	canton := [][]votes.Vote{
		{jurisdictionVote("zurich-canton", "canton-1", "100/2026", "2026-06-02")},
		{jurisdictionVote("zurich-canton", "canton-2", "101/2026", "2026-06-04")},
		{jurisdictionVote("zurich-canton", "canton-3", "102/2026", "2026-06-06")},
	}

	merged := MergeOldestFirst(city, canton)

	// Oldest first, alternating because the two chambers sat on alternating days.
	wantOrder := []string{"city-1", "canton-1", "city-2", "canton-2", "city-3", "canton-3"}
	for i, want := range wantOrder {
		if got := merged[i][0].SourceID; got != want {
			t.Fatalf("merged[%d] = %s, want %s (groups must be posted oldest first)", i, got, want)
		}
	}

	platform := &MockPlatform{maxPosts: budget}
	logs := VoteLogs{
		"zurich-city":   votelog.NewEmpty("zurich-city", votelog.PlatformX),
		"zurich-canton": votelog.NewEmpty("zurich-canton", votelog.PlatformX),
	}

	posted, err := PostToPlatform(merged, platform, logs, false)
	if err != nil {
		t.Fatalf("PostToPlatform: %v", err)
	}

	if posted != budget {
		t.Errorf("posted %d groups, want %d — the budget must be shared across the channel, not spent per jurisdiction", posted, budget)
	}

	// Each vote must land in its own jurisdiction's log.
	if !logs["zurich-city"].IsPosted("city-1") {
		t.Error("city-1 should be recorded in the city log")
	}
	if !logs["zurich-canton"].IsPosted("canton-1") {
		t.Error("canton-1 should be recorded in the canton log")
	}
	if logs["zurich-city"].IsPosted("canton-1") {
		t.Error("a canton vote must not be recorded in the city log")
	}
}

// A group whose jurisdiction has no log is a configuration bug. Failing is the
// only safe response: posting without recording would re-post next run.
func TestPostToPlatform_UnknownJurisdictionIsAnError(t *testing.T) {
	defer setupTempDir(t)()

	groups := [][]votes.Vote{
		{jurisdictionVote("zurich-canton", "canton-1", "100/2026", "2026-06-02")},
	}
	logs := VoteLogs{"zurich-city": votelog.NewEmpty("zurich-city", votelog.PlatformX)}

	if _, err := PostToPlatform(groups, &MockPlatform{maxPosts: 5}, logs, false); err == nil {
		t.Fatal("expected an error for a group with no configured vote log")
	}
}

func TestMergeOldestFirst_TiesFollowArgumentOrder(t *testing.T) {
	first := [][]votes.Vote{{jurisdictionVote("a", "a-1", "1", "2026-06-01")}}
	second := [][]votes.Vote{{jurisdictionVote("b", "b-1", "2", "2026-06-01")}}

	merged := MergeOldestFirst(first, second)
	if merged[0][0].SourceID != "a-1" || merged[1][0].SourceID != "b-1" {
		t.Errorf("equal dates should keep configuration order, got %s then %s",
			merged[0][0].SourceID, merged[1][0].SourceID)
	}
}

// stubSource serves a fixed set of votes without touching a network.
type stubSource struct {
	jurisdiction votes.Jurisdiction
	fetched      []votes.Vote
}

func (s stubSource) FetchRecent(int) ([]votes.Vote, error) { return s.fetched, nil }
func (s stubSource) GroupByAffair(vs []votes.Vote) ([][]votes.Vote, error) {
	return votes.GroupByAffairAndDate(vs), nil
}
func (s stubSource) Jurisdiction() votes.Jurisdiction { return s.jurisdiction }

func members(n int, fraktion, choice string) []votes.MemberVote {
	out := make([]votes.MemberVote, n)
	for i := range out {
		out[i] = votes.MemberVote{Fraktion: fraktion, Choice: choice}
	}
	return out
}

// A truncated member list still renders a plausible-looking Fraktion table that
// understates whichever factions are missing — silently, and with no way for a
// reader to tell. The gate drops the breakdown so the post falls back to the
// totals, which the source reports independently and which stay correct.
func TestCompletenessGate(t *testing.T) {
	ja, nein, abw := 100, 20, 5
	base := votes.Vote{
		SourceID:     "gate-1",
		Jurisdiction: testJurisdiction,
		Date:         testfixtures.MustDate("2026-06-01"),
		Yes:          &ja, No: &nein, Absent: &abw,
		Affair: votes.Affair{Number: "2026/1"},
	}

	t.Run("complete breakdown is kept", func(t *testing.T) {
		v := base
		v.MemberVotes = members(125, "SP", "Ja")

		groups := prepare(t, v)
		if got := len(groups[0][0].MemberVotes); got != 125 {
			t.Errorf("kept %d member votes, want all 125", got)
		}
	})

	t.Run("truncated breakdown is dropped", func(t *testing.T) {
		v := base
		v.MemberVotes = members(90, "SP", "Ja") // totals say 125

		groups := prepare(t, v)
		if got := len(groups[0][0].MemberVotes); got != 0 {
			t.Errorf("kept %d member votes; a partial breakdown must not be posted", got)
		}
	})

	t.Run("a source with no member list at all is untouched", func(t *testing.T) {
		// Totals-only data is correct as far as it goes; there is nothing to gate.
		groups := prepare(t, base)
		if groups[0][0].MemberVotes != nil {
			t.Error("expected no member votes")
		}
	})
}

func prepare(t *testing.T, v votes.Vote) [][]votes.Vote {
	t.Helper()
	defer setupTempDir(t)()

	src := stubSource{
		jurisdiction: votes.Jurisdiction{Key: testJurisdiction},
		fetched:      []votes.Vote{v},
	}
	groups, err := PrepareVoteGroups(src, votelog.NewEmpty(testJurisdiction, votelog.PlatformX), 10, 0)
	if err != nil {
		t.Fatalf("PrepareVoteGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	return groups
}
