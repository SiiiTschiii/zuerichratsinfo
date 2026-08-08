package votes

// TotalRecorded sums the reported totals across every count the source
// populated. Nil counts contribute nothing. It reports whether any total was
// reported at all, so callers can distinguish "0 voters" from "no data".
func (v Vote) TotalRecorded() (int, bool) {
	total := 0
	reported := false
	for _, c := range []*int{v.Yes, v.No, v.Abstention, v.Absent,
		v.ChoiceA, v.ChoiceB, v.ChoiceC, v.ChoiceD, v.ChoiceE} {
		if c != nil {
			total += *c
			reported = true
		}
	}
	return total, reported
}

// IsBreakdownComplete reports whether MemberVotes accounts for exactly the
// members the reported totals say took part.
//
// This gates the per-Fraktion breakdown. A partial member list still renders a
// plausible-looking table, so publishing one would silently understate a
// faction rather than fail visibly. When it returns false, post the totals only.
//
// seats is the chamber size and is used only as an upper bound: more recorded
// members than seats means the data is broken in a different way. It is
// deliberately *not* an equality check, because a chamber with a vacant seat
// legitimately records fewer members than it has seats.
//
// A vote with no MemberVotes at all is not "incomplete": some sources publish
// totals without name lists, and such posts are correct as far as they go.
// Callers skip the breakdown for those anyway — there is nothing to aggregate.
func IsBreakdownComplete(v Vote, seats int) bool {
	if len(v.MemberVotes) == 0 {
		return true
	}
	if seats > 0 && len(v.MemberVotes) > seats {
		return false
	}
	total, reported := v.TotalRecorded()
	return !reported || len(v.MemberVotes) == total
}
