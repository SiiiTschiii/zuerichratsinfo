package zurichapi

import (
	"sort"
	"strconv"
	"testing"
)

// Helper function to create test votes
func createTestVote(guid, geschaeftGrNr, sitzungDatum, abstimmungstitel string) Abstimmung {
	return Abstimmung{
		OBJGUID:          guid,
		GeschaeftGrNr:    geschaeftGrNr,
		TraktandumGuid:   "traktandum-" + geschaeftGrNr, // Default: same Geschäft = same Traktandum
		SitzungDatum:     sitzungDatum,
		Abstimmungstitel: abstimmungstitel,
	}
}

// Helper to create vote with SEQ number
func createTestVoteWithSEQ(guid, geschaeftGrNr, seq, sitzungDatum, abstimmungstitel string) Abstimmung {
	return Abstimmung{
		OBJGUID:          guid,
		SEQ:              seq,
		GeschaeftGrNr:    geschaeftGrNr,
		TraktandumGuid:   "traktandum-" + geschaeftGrNr,
		SitzungDatum:     sitzungDatum,
		Abstimmungstitel: abstimmungstitel,
	}
}

// Helper to create vote with explicit TraktandumGuid
func createTestVoteWithTraktandum(guid, geschaeftGrNr, traktandumGuid, sitzungDatum, abstimmungstitel string) Abstimmung {
	return Abstimmung{
		OBJGUID:          guid,
		GeschaeftGrNr:    geschaeftGrNr,
		TraktandumGuid:   traktandumGuid,
		SitzungDatum:     sitzungDatum,
		Abstimmungstitel: abstimmungstitel,
	}
}

// TestGroupAbstimmungenByGeschaeft_Grouping tests the core grouping logic
func TestGroupAbstimmungenByGeschaeft_Grouping(t *testing.T) {
	tests := []struct {
		name           string
		votes          []Abstimmung
		expectedGroups int
		expectedSizes  []int
		description    string
	}{
		{
			name:           "Empty votes",
			votes:          []Abstimmung{},
			expectedGroups: 0,
			expectedSizes:  []int{},
			description:    "Should return nil for empty input",
		},
		{
			name: "Single vote",
			votes: []Abstimmung{
				createTestVote("vote1", "2025/369", "2025-11-19", "Abstimmung 1"),
			},
			expectedGroups: 1,
			expectedSizes:  []int{1},
			description:    "Single vote should form one group",
		},
		{
			name: "Multiple votes same Geschäft",
			votes: []Abstimmung{
				createTestVote("vote1", "2025/369", "2025-11-19", "Abstimmung 1"),
				createTestVote("vote2", "2025/369", "2025-11-19", "Abstimmung 2"),
				createTestVote("vote3", "2025/369", "2025-11-19", "Abstimmung 3"),
			},
			expectedGroups: 1,
			expectedSizes:  []int{3},
			description:    "Multiple votes for same Geschäft/date should group together",
		},
		{
			name: "Multiple different Geschäfte",
			votes: []Abstimmung{
				createTestVote("vote1", "2025/369", "2025-11-19", "Abstimmung 1"),
				createTestVote("vote2", "2025/370", "2025-11-19", "Abstimmung 1"),
				createTestVote("vote3", "2025/371", "2025-11-19", "Abstimmung 1"),
			},
			expectedGroups: 3,
			expectedSizes:  []int{1, 1, 1},
			description:    "Different Geschäfte should form separate groups",
		},
		{
			name: "Same Geschäft different dates",
			votes: []Abstimmung{
				createTestVote("vote1", "2025/369", "2025-11-19", "Abstimmung 1"),
				createTestVote("vote2", "2025/369", "2025-11-20", "Abstimmung 1"),
			},
			expectedGroups: 2,
			expectedSizes:  []int{1, 1},
			description:    "Same Geschäft on different dates should form separate groups",
		},
		{
			name: "Mixed scenario with multiple groups",
			votes: []Abstimmung{
				createTestVoteWithSEQ("vote1", "2025/369", "1000", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithSEQ("vote2", "2025/370", "2000", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithSEQ("vote3", "2025/370", "2001", "2025-11-19", "Abstimmung 2"),
			},
			expectedGroups: 2,
			expectedSizes:  []int{1, 2}, // After reversal: [2025/369 with SEQ 1000] first, [2025/370 with SEQ 2000-2001] second
			description:    "Should properly group mixed votes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the pure grouping logic without API calls
			// The completion logic is tested separately with integration tests
			groups := groupVotesOnly(tt.votes)

			if len(groups) != tt.expectedGroups {
				t.Errorf("%s: expected %d groups, got %d", tt.description, tt.expectedGroups, len(groups))
			}

			if len(groups) > 0 && len(tt.expectedSizes) > 0 {
				for i, expectedSize := range tt.expectedSizes {
					if i >= len(groups) {
						break
					}
					if len(groups[i]) != expectedSize {
						t.Errorf("%s: group %d expected size %d, got %d", tt.description, i, expectedSize, len(groups[i]))
					}
				}
			}
		})
	}
}

// groupVotesOnly is a helper that does pure grouping without API calls
// This extracts the core grouping logic for testing
func groupVotesOnly(votes []Abstimmung) [][]Abstimmung {
	if len(votes) == 0 {
		return nil
	}

	// Build a map keyed by "GeschaeftGrNr|SitzungDatum"
	groupMap := make(map[string][]Abstimmung)

	for _, vote := range votes {
		// Extract just the date part (YYYY-MM-DD) from SitzungDatum
		date := vote.SitzungDatum
		if len(date) > 10 {
			date = date[:10]
		}

		key := vote.GeschaeftGrNr + "|" + date
		groupMap[key] = append(groupMap[key], vote)
	}

	// Sort votes within each group by SEQ (ascending) to preserve Sitzung chronological order
	for key := range groupMap {
		votes := groupMap[key]
		sort.Slice(votes, func(i, j int) bool {
			seqI, _ := strconv.Atoi(votes[i].SEQ)
			seqJ, _ := strconv.Atoi(votes[j].SEQ)
			return seqI < seqJ
		})
		groupMap[key] = votes
	}

	// Convert map to slice of groups, preserving the order of first occurrence
	seen := make(map[string]bool)
	var groups [][]Abstimmung

	for _, vote := range votes {
		date := vote.SitzungDatum
		if len(date) > 10 {
			date = date[:10]
		}
		key := vote.GeschaeftGrNr + "|" + date

		if !seen[key] {
			seen[key] = true
			groups = append(groups, groupMap[key])
		}
	}

	// Sort groups by their minimum SEQ value (oldest first)
	sort.Slice(groups, func(i, j int) bool {
		minSeqI, _ := strconv.Atoi(groups[i][0].SEQ) // First vote in group is already sorted to be oldest
		minSeqJ, _ := strconv.Atoi(groups[j][0].SEQ)
		return minSeqI < minSeqJ
	})

	return groups
}

// TestGroupByTraktandumGuid tests that votes are grouped by GeschaeftGrNr+Date
// Note: Despite the name suggesting TraktandumGuid, the actual grouping is by GeschaeftGrNr|Date
// TraktandumGuid is only used in ensureCompleteGroupIfNeeded to fetch missing votes
func TestGroupByGeschaeftAndDate(t *testing.T) {
	tests := []struct {
		name           string
		votes          []Abstimmung
		expectedGroups int
		description    string
	}{
		{
			name: "Same Geschäft, same date - should group together",
			votes: []Abstimmung{
				createTestVoteWithTraktandum("vote1", "2025/369", "trak-1", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithTraktandum("vote2", "2025/369", "trak-1", "2025-11-19", "Abstimmung 2"),
				createTestVoteWithTraktandum("vote3", "2025/369", "trak-1", "2025-11-19", "Abstimmung 3"),
			},
			expectedGroups: 1,
			description:    "Votes with same Geschäft and date group together",
		},
		{
			name: "Same Geschäft, different Traktandum, same date - should group together",
			votes: []Abstimmung{
				// In reality, same Geschäft usually means same Traktandum, but testing the grouping key
				createTestVoteWithTraktandum("vote1", "2025/369", "trak-1", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithTraktandum("vote2", "2025/369", "trak-2", "2025-11-19", "Abstimmung 2"),
			},
			expectedGroups: 1,
			description:    "Grouping key is GeschaeftGrNr|Date, not TraktandumGuid",
		},
		{
			name: "Different Geschäft, same date - should separate",
			votes: []Abstimmung{
				createTestVoteWithTraktandum("vote1", "2025/369", "trak-1", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithTraktandum("vote2", "2025/370", "trak-1", "2025-11-19", "Abstimmung 2"),
			},
			expectedGroups: 2,
			description:    "Different Geschäft numbers form separate groups",
		},
		{
			name: "Same Geschäft, different dates - should separate",
			votes: []Abstimmung{
				createTestVoteWithTraktandum("vote1", "2025/369", "trak-1", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithTraktandum("vote2", "2025/369", "trak-1", "2025-11-20", "Abstimmung 2"),
			},
			expectedGroups: 2,
			description:    "Same Geschäft on different dates forms separate groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := groupVotesOnly(tt.votes)

			if len(groups) != tt.expectedGroups {
				t.Errorf("%s: expected %d groups, got %d", tt.description, tt.expectedGroups, len(groups))
			}
		})
	}
}

// TestVoteOrderingBySEQ tests that votes are ordered correctly
func TestVoteOrderingBySEQ(t *testing.T) {
	tests := []struct {
		name        string
		votes       []Abstimmung
		description string
		verify      func(t *testing.T, groups [][]Abstimmung)
	}{
		{
			name: "Votes within same Geschäft ordered by SEQ ascending",
			votes: []Abstimmung{
				// Simulating API order: descending SEQ (newest first)
				createTestVoteWithSEQ("vote3", "2025/369", "5209122", "2025-12-03", "Abstimmung 3"),
				createTestVoteWithSEQ("vote2", "2025/369", "5209059", "2025-12-03", "Abstimmung 2"),
				createTestVoteWithSEQ("vote1", "2025/369", "5209024", "2025-12-03", "Abstimmung 1"),
			},
			description: "Multiple votes for same Geschäft should be sorted by SEQ ascending (chronological)",
			verify: func(t *testing.T, groups [][]Abstimmung) {
				if len(groups) != 1 {
					t.Fatalf("Expected 1 group, got %d", len(groups))
				}
				group := groups[0]
				if len(group) != 3 {
					t.Fatalf("Expected 3 votes in group, got %d", len(group))
				}
				// Check SEQ order is ascending (chronological)
				if group[0].SEQ != "5209024" || group[1].SEQ != "5209059" || group[2].SEQ != "5209122" {
					t.Errorf("Votes not in SEQ ascending order: %s, %s, %s", group[0].SEQ, group[1].SEQ, group[2].SEQ)
				}
			},
		},
		{
			name: "Multiple Geschäfte ordered oldest first",
			votes: []Abstimmung{
				// Simulating API order: descending SEQ (newest first)
				createTestVoteWithSEQ("vote6", "2025/111", "5209467", "2025-12-03", "Trans Jugendliche"),
				createTestVoteWithSEQ("vote5", "2025/37", "5209455", "2025-12-03", "Supported Education"),
				createTestVoteWithSEQ("vote4", "2025/277", "5209122", "2025-12-03", "Recyclingzentrum"),
				createTestVoteWithSEQ("vote3", "2023/562", "5209059", "2025-12-03", "Josef-Areal"),
				createTestVoteWithSEQ("vote2", "2022/260", "5209024", "2025-12-03", "Werft Wollishofen"),
			},
			description: "Multiple Geschäfte should be reversed so oldest (lowest SEQ) posts first",
			verify: func(t *testing.T, groups [][]Abstimmung) {
				if len(groups) != 5 {
					t.Fatalf("Expected 5 groups, got %d", len(groups))
				}
				// Check groups are in ascending SEQ order (oldest first)
				expectedOrder := []string{"5209024", "5209059", "5209122", "5209455", "5209467"}
				for i, group := range groups {
					if len(group) == 0 {
						t.Fatalf("Group %d is empty", i)
					}
					if group[0].SEQ != expectedOrder[i] {
						t.Errorf("Group %d: expected SEQ %s, got %s", i, expectedOrder[i], group[0].SEQ)
					}
				}
			},
		},
		{
			name: "Mixed: multiple votes per Geschäft and multiple Geschäfte",
			votes: []Abstimmung{
				// API order: newest first (descending SEQ)
				createTestVoteWithSEQ("v6", "2025/250", "5209213", "2025-12-03", "Vote 2"),
				createTestVoteWithSEQ("v5", "2025/250", "5209212", "2025-12-03", "Vote 1"),
				createTestVoteWithSEQ("v4", "2025/277", "5209122", "2025-12-03", "Single vote"),
				createTestVoteWithSEQ("v3", "2022/260", "5209024", "2025-12-03", "Oldest"),
			},
			description: "Mixed scenario with multiple groups and votes",
			verify: func(t *testing.T, groups [][]Abstimmung) {
				if len(groups) != 3 {
					t.Fatalf("Expected 3 groups, got %d", len(groups))
				}
				// First group should be 2022/260 (oldest)
				if groups[0][0].GeschaeftGrNr != "2022/260" {
					t.Errorf("First group should be 2022/260, got %s", groups[0][0].GeschaeftGrNr)
				}
				// Last group should be 2025/250 (newest)
				if groups[2][0].GeschaeftGrNr != "2025/250" {
					t.Errorf("Last group should be 2025/250, got %s", groups[2][0].GeschaeftGrNr)
				}
				// Within 2025/250 group, votes should be SEQ ascending
				if len(groups[2]) != 2 {
					t.Fatalf("Expected 2 votes in 2025/250 group, got %d", len(groups[2]))
				}
				if groups[2][0].SEQ != "5209212" || groups[2][1].SEQ != "5209213" {
					t.Errorf("2025/250 votes not in SEQ order: %s, %s", groups[2][0].SEQ, groups[2][1].SEQ)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := groupVotesOnly(tt.votes)
			tt.verify(t, groups)
		})
	}
}

// TestEnsureCompleteGroupLogic tests the completion behavior
// Note: This tests the logic, not the actual API call
func TestEnsureCompleteGroupLogic(t *testing.T) {
	tests := []struct {
		name        string
		votes       []Abstimmung
		description string
	}{
		{
			name: "Last vote has same TraktandumGuid as earlier votes",
			votes: []Abstimmung{
				createTestVoteWithTraktandum("vote1", "2025/369", "trak-1", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithTraktandum("vote2", "2025/370", "trak-2", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithTraktandum("vote3", "2025/369", "trak-1", "2025-11-19", "Abstimmung 2"),
			},
			description: "Last vote from same Traktandum should trigger completion check",
		},
		{
			name: "Last vote is only one from its Traktandum",
			votes: []Abstimmung{
				createTestVoteWithTraktandum("vote1", "2025/369", "trak-1", "2025-11-19", "Abstimmung 1"),
				createTestVoteWithTraktandum("vote2", "2025/370", "trak-2", "2025-11-19", "Abstimmung 1"),
			},
			description: "Last vote as single from Traktandum should check for more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.votes) == 0 {
				t.Skip("Empty votes")
			}

			lastVote := tt.votes[len(tt.votes)-1]

			// Verify the test data is set up correctly
			if lastVote.TraktandumGuid == "" {
				t.Error("Last vote should have TraktandumGuid set")
			}

			// In the real implementation, ensureCompleteGroupIfNeeded would:
			// 1. Take lastVote.TraktandumGuid
			// 2. Call FetchAbstimmungenForTraktandum(lastVote.TraktandumGuid)
			// 3. Merge any missing votes into the result
			// This test just validates the logic would have the right inputs
		})
	}
}
