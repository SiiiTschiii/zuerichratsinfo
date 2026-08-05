package votes

import (
	"reflect"
	"testing"
	"time"
)

func intPtr(i int) *int { return &i }

func day(d int) time.Time { return time.Date(2025, 6, d, 0, 0, 0, 0, time.UTC) }

func vote(id, affair, seq string, date time.Time) Vote {
	return Vote{SourceID: id, Sequence: seq, Date: date, Affair: Affair{Number: affair}}
}

func TestGroupByAffairAndDate(t *testing.T) {
	// Deliberately shuffled: grouping must not depend on input order.
	in := []Vote{
		vote("c", "2025/2", "30", day(15)),
		vote("a", "2025/1", "10", day(15)),
		vote("e", "2025/1", "5", day(16)),
		vote("b", "2025/1", "20", day(15)),
		vote("d", "2025/2", "40", day(15)),
	}

	groups := GroupByAffairAndDate(in)

	var got [][]string
	for _, g := range groups {
		var ids []string
		for _, v := range g {
			ids = append(ids, v.SourceID)
		}
		got = append(got, ids)
	}

	// Groups sort by date, then by the sequence of their last vote; votes
	// within a group sort by sequence ascending.
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("groups = %v, want %v", got, want)
	}
}

func TestGroupByAffairAndDate_SameAffairDifferentDaysStaySeparate(t *testing.T) {
	groups := GroupByAffairAndDate([]Vote{
		vote("a", "2025/1", "1", day(15)),
		vote("b", "2025/1", "2", day(16)),
	})
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2 (one per sitting day)", len(groups))
	}
}

func TestGroupByAffairAndDate_Empty(t *testing.T) {
	if got := GroupByAffairAndDate(nil); got != nil {
		t.Errorf("GroupByAffairAndDate(nil) = %v, want nil", got)
	}
}

func TestIsBreakdownComplete(t *testing.T) {
	members := func(n int) []MemberVote {
		out := make([]MemberVote, n)
		for i := range out {
			out[i] = MemberVote{Fraktion: "SP", Choice: "Ja"}
		}
		return out
	}

	tests := []struct {
		name  string
		vote  Vote
		seats int
		want  bool
	}{
		{
			name:  "member votes match totals",
			vote:  Vote{Yes: intPtr(90), No: intPtr(30), Abstention: intPtr(0), Absent: intPtr(5), MemberVotes: members(125)},
			seats: 125,
			want:  true,
		},
		{
			name:  "member votes truncated",
			vote:  Vote{Yes: intPtr(90), No: intPtr(30), Abstention: intPtr(0), Absent: intPtr(5), MemberVotes: members(100)},
			seats: 125,
			want:  false,
		},
		{
			name:  "no member votes at all is not incomplete",
			vote:  Vote{Yes: intPtr(90), No: intPtr(30)},
			seats: 125,
			want:  true,
		},
		{
			name:  "no totals reported, nothing to contradict",
			vote:  Vote{MemberVotes: members(120)},
			seats: 125,
			want:  true,
		},
		{
			// A vacant seat is normal; it must not suppress the breakdown.
			name:  "fewer members than seats but consistent with totals",
			vote:  Vote{Yes: intPtr(80), No: intPtr(44), MemberVotes: members(124)},
			seats: 125,
			want:  true,
		},
		{
			name:  "more members than seats",
			vote:  Vote{MemberVotes: members(200)},
			seats: 125,
			want:  false,
		},
		{
			name:  "auswahl totals",
			vote:  Vote{ChoiceA: intPtr(74), ChoiceB: intPtr(28), ChoiceC: intPtr(13), Absent: intPtr(10), MemberVotes: members(125)},
			seats: 125,
			want:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsBreakdownComplete(tc.vote, tc.seats); got != tc.want {
				t.Errorf("IsBreakdownComplete = %v, want %v", got, tc.want)
			}
		})
	}
}
