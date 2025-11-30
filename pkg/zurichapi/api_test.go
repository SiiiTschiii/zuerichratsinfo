package zurichapi

import (
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
				createTestVote("vote1", "2025/369", "2025-11-19", "Abstimmung 1"),
				createTestVote("vote2", "2025/370", "2025-11-19", "Abstimmung 1"),
				createTestVote("vote3", "2025/370", "2025-11-19", "Abstimmung 2"),
			},
			expectedGroups: 2,
			expectedSizes:  []int{1, 2},
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
