package voteposting

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

const testJurisdiction = "zurich-city"

// setupTempDir creates a temp directory for tests and changes to it
// Returns a cleanup function that should be deferred
func setupTempDir(t *testing.T) func() {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()

	// Create data directory in temp location
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	return func() {
		_ = os.Chdir(oldWd)
	}
}

// MockPlatform is a test implementation of the Platform interface
type MockPlatform struct {
	formatCalls    int
	postCalls      int
	maxPosts       int
	shouldFailPost bool
}

type MockContent struct {
	text string
}

func (c *MockContent) String() string {
	return c.text
}

func (p *MockPlatform) Format(group []votes.Vote) (platforms.Content, error) {
	p.formatCalls++
	return &MockContent{text: "mock post"}, nil
}

func (p *MockPlatform) Post(content platforms.Content) (bool, error) {
	if p.shouldFailPost {
		return false, errors.New("mock posting error")
	}
	p.postCalls++
	shouldContinue := p.postCalls < p.maxPosts
	return shouldContinue, nil
}

func (p *MockPlatform) MaxPostsPerRun() int {
	return p.maxPosts
}

func (p *MockPlatform) Name() string {
	return "Mock"
}

// Test helper to create test group with a non-zero Ja count so they pass
// validateGroupCounts (all-zero counts are treated as unsupported vote types).
func createVote(guid, affair, date string) votes.Vote {
	ja := 100
	return votes.Vote{
		SourceID:     guid,
		Jurisdiction: testJurisdiction,
		Date:         testfixtures.MustDate(date),
		Yes:          &ja,
		Affair:       votes.Affair{Number: affair},
	}
}

func TestFilterUnpostedVotes(t *testing.T) {
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)
	voteLog.MarkAsPosted("vote1")
	voteLog.MarkAsPosted("vote3")

	group := []votes.Vote{
		createVote("vote1", "2025/369", "2025-11-19"),
		createVote("vote2", "2025/370", "2025-11-19"),
		createVote("vote3", "2025/371", "2025-11-19"),
		createVote("vote4", "2025/372", "2025-11-19"),
	}

	unposted := filterUnpostedVotes(group, voteLog)

	if len(unposted) != 2 {
		t.Errorf("Expected 2 unposted group, got %d", len(unposted))
	}

	if unposted[0].SourceID != "vote2" || unposted[1].SourceID != "vote4" {
		t.Errorf("Expected vote2 and vote4, got %v", unposted)
	}
}

func TestPostToPlatform_DryRun(t *testing.T) {
	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	groups := [][]votes.Vote{
		{createVote("vote1", "2025/369", "2025-11-19")},
		{createVote("vote2", "2025/370", "2025-11-19")},
	}

	posted, err := PostToPlatform(groups, mockPlatform, SingleLog(testJurisdiction, voteLog), true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// In dry run, posted counts groups printed (not real API calls)
	if posted != 2 {
		t.Errorf("Expected 2 printed in dry run, got %d", posted)
	}

	// Format should be called for each group
	if mockPlatform.formatCalls != 2 {
		t.Errorf("Expected 2 format calls, got %d", mockPlatform.formatCalls)
	}

	// Post should NOT be called in dry run
	if mockPlatform.postCalls != 0 {
		t.Errorf("Expected 0 post calls in dry run, got %d", mockPlatform.postCalls)
	}

	// Vote log should still be empty
	if voteLog.Count() != 0 {
		t.Errorf("Expected empty vote log in dry run, got %d", voteLog.Count())
	}
}

func TestPostToPlatform_RealPosting(t *testing.T) {
	defer setupTempDir(t)()

	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	groups := [][]votes.Vote{
		{createVote("vote1", "2025/369", "2025-11-19")},
		{createVote("vote2", "2025/370", "2025-11-19"), createVote("vote3", "2025/370", "2025-11-19")},
	}

	posted, err := PostToPlatform(groups, mockPlatform, SingleLog(testJurisdiction, voteLog), false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have posted 2 groups
	if posted != 2 {
		t.Errorf("Expected 2 posted, got %d", posted)
	}

	// Format and post should be called for each group
	if mockPlatform.formatCalls != 2 {
		t.Errorf("Expected 2 format calls, got %d", mockPlatform.formatCalls)
	}

	if mockPlatform.postCalls != 2 {
		t.Errorf("Expected 2 post calls, got %d", mockPlatform.postCalls)
	}

	// All 3 group should be marked as posted
	if voteLog.Count() != 3 {
		t.Errorf("Expected 3 group in log, got %d", voteLog.Count())
	}

	// Check specific group are logged
	if !voteLog.IsPosted("vote1") || !voteLog.IsPosted("vote2") || !voteLog.IsPosted("vote3") {
		t.Error("Not all group were marked as posted")
	}
}

func TestPostToPlatform_LimitRespected(t *testing.T) {
	defer setupTempDir(t)()

	// Platform that stops after 2 posts
	mockPlatform := &MockPlatform{maxPosts: 2}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	groups := [][]votes.Vote{
		{createVote("vote1", "2025/369", "2025-11-19")},
		{createVote("vote2", "2025/370", "2025-11-19")},
		{createVote("vote3", "2025/371", "2025-11-19")},
		{createVote("vote4", "2025/372", "2025-11-19")},
	}

	posted, err := PostToPlatform(groups, mockPlatform, SingleLog(testJurisdiction, voteLog), false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have stopped at 2 posts due to platform limit
	if posted != 2 {
		t.Errorf("Expected 2 posted (platform limit), got %d", posted)
	}

	if mockPlatform.postCalls != 2 {
		t.Errorf("Expected 2 post calls, got %d", mockPlatform.postCalls)
	}

	// Only first 2 group should be logged
	if voteLog.Count() != 2 {
		t.Errorf("Expected 2 group in log, got %d", voteLog.Count())
	}
}

func TestPostToPlatform_ErrorHandling(t *testing.T) {
	// Platform that fails on posting
	mockPlatform := &MockPlatform{maxPosts: 10, shouldFailPost: true}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	groups := [][]votes.Vote{
		{createVote("vote1", "2025/369", "2025-11-19")},
	}

	posted, err := PostToPlatform(groups, mockPlatform, SingleLog(testJurisdiction, voteLog), false)

	// Should return error
	if err == nil {
		t.Error("Expected error from failed posting")
	}

	// Should not have posted anything
	if posted != 0 {
		t.Errorf("Expected 0 posted on error, got %d", posted)
	}

	// Vote log should be empty
	if voteLog.Count() != 0 {
		t.Errorf("Expected empty vote log on error, got %d", voteLog.Count())
	}
}

func TestFilterUnpostedVotes_AllPosted(t *testing.T) {
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)
	voteLog.MarkAsPosted("vote1")
	voteLog.MarkAsPosted("vote2")

	group := []votes.Vote{
		createVote("vote1", "2025/369", "2025-11-19"),
		createVote("vote2", "2025/370", "2025-11-19"),
	}

	unposted := filterUnpostedVotes(group, voteLog)

	if len(unposted) != 0 {
		t.Errorf("Expected 0 unposted group when all are posted, got %d", len(unposted))
	}
}

func TestFilterUnpostedVotes_NonePosted(t *testing.T) {
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	group := []votes.Vote{
		createVote("vote1", "2025/369", "2025-11-19"),
		createVote("vote2", "2025/370", "2025-11-19"),
		createVote("vote3", "2025/371", "2025-11-19"),
	}

	unposted := filterUnpostedVotes(group, voteLog)

	if len(unposted) != 3 {
		t.Errorf("Expected 3 unposted group, got %d", len(unposted))
	}
}

func TestPostToPlatform_EmptyGroups(t *testing.T) {
	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	groups := [][]votes.Vote{}

	posted, err := PostToPlatform(groups, mockPlatform, SingleLog(testJurisdiction, voteLog), false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if posted != 0 {
		t.Errorf("Expected 0 posted for empty groups, got %d", posted)
	}

	if mockPlatform.formatCalls != 0 {
		t.Errorf("Expected 0 format calls for empty groups, got %d", mockPlatform.formatCalls)
	}
}

func TestPostToPlatform_AllVotesInGroupAreLogged(t *testing.T) {
	defer setupTempDir(t)()

	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	// Create a group with 5 group (simulating a complex Geschäft)
	groups := [][]votes.Vote{
		{
			createVote("vote1", "2025/179", "2025-11-19"),
			createVote("vote2", "2025/179", "2025-11-19"),
			createVote("vote3", "2025/179", "2025-11-19"),
			createVote("vote4", "2025/179", "2025-11-19"),
			createVote("vote5", "2025/179", "2025-11-19"),
		},
	}

	posted, err := PostToPlatform(groups, mockPlatform, SingleLog(testJurisdiction, voteLog), false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// One group should be posted
	if posted != 1 {
		t.Errorf("Expected 1 group posted, got %d", posted)
	}

	// All 5 group should be logged
	if voteLog.Count() != 5 {
		t.Errorf("Expected 5 group logged, got %d", voteLog.Count())
	}

	// Verify each specific vote is logged
	expectedVotes := []string{"vote1", "vote2", "vote3", "vote4", "vote5"}
	for _, voteID := range expectedVotes {
		if !voteLog.IsPosted(voteID) {
			t.Errorf("Expected %s to be logged as posted", voteID)
		}
	}
}

func TestPostToPlatform_MultipleGroupsAllVotesLogged(t *testing.T) {
	defer setupTempDir(t)()

	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	// Multiple groups with varying sizes
	groups := [][]votes.Vote{
		{createVote("vote1", "2025/369", "2025-11-19")}, // 1 vote
		{
			createVote("vote2", "2025/370", "2025-11-19"),
			createVote("vote3", "2025/370", "2025-11-19"),
		}, // 2 group
		{
			createVote("vote4", "2025/179", "2025-11-19"),
			createVote("vote5", "2025/179", "2025-11-19"),
			createVote("vote6", "2025/179", "2025-11-19"),
		}, // 3 group
	}

	posted, err := PostToPlatform(groups, mockPlatform, SingleLog(testJurisdiction, voteLog), false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 3 groups should be posted
	if posted != 3 {
		t.Errorf("Expected 3 groups posted, got %d", posted)
	}

	// All 6 group should be logged (1+2+3)
	if voteLog.Count() != 6 {
		t.Errorf("Expected 6 group logged, got %d", voteLog.Count())
	}

	// Verify each specific vote is logged
	for i := 1; i <= 6; i++ {
		voteID := fmt.Sprintf("vote%d", i)
		if !voteLog.IsPosted(voteID) {
			t.Errorf("Expected %s to be logged as posted", voteID)
		}
	}
}
