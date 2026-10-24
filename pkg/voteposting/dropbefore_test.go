package voteposting

import (
	"testing"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

func groupOn(id string, date time.Time) []votes.Vote {
	return []votes.Vote{{SourceID: id, Date: date}}
}

func ids(groups [][]votes.Vote) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g[0].SourceID)
	}
	return out
}

func TestDropBefore(t *testing.T) {
	from := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	zurich := time.FixedZone("CEST", 2*60*60)

	groups := [][]votes.Vote{
		groupOn("day-before", time.Date(2026, 10, 16, 23, 59, 0, 0, time.UTC)),
		// Midnight Zurich time on the start day is 22:00 UTC on the day before:
		// a UTC comparison would drop the very sitting the date names.
		groupOn("start-day-zurich-midnight", time.Date(2026, 10, 17, 0, 0, 0, 0, zurich)),
		groupOn("start-day", from),
		groupOn("after", time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)),
		groupOn("unknown-date", time.Time{}),
	}

	got := ids(DropBefore(groups, from))
	want := []string{"start-day-zurich-midnight", "start-day", "after", "unknown-date"}
	if len(got) != len(want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("kept %v, want %v", got, want)
			break
		}
	}
}

// A zero start date means no limit: platforms other than Facebook pass it.
func TestDropBefore_ZeroKeepsEverything(t *testing.T) {
	groups := [][]votes.Vote{groupOn("old", time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC))}
	if got := DropBefore(groups, time.Time{}); len(got) != 1 {
		t.Errorf("zero start date dropped groups: %v", ids(got))
	}
}
