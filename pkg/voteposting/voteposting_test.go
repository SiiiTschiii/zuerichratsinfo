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
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
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
	// lastGroup records what the formatter was actually handed, so tests can
	// assert on votes dropped before formatting.
	lastGroup []votes.Vote
}

type MockContent struct {
	text string
}

func (c *MockContent) String() string {
	return c.text
}

func (p *MockPlatform) Format(group []votes.Vote) (platforms.Content, error) {
	p.formatCalls++
	p.lastGroup = group
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

// Test helper to create a test group that passes validateGroup: a handled vote
// type, and a non-zero Ja count (all-zero counts are treated as unsupported).
func createVote(guid, affair, date string) votes.Vote {
	ja := 100
	return votes.Vote{
		SourceID:     guid,
		Jurisdiction: testJurisdiction,
		Date:         testfixtures.MustDate(date),
		Type:         "Normal",
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

// TestPostToPlatform_UnhandledVoteTypeIsNotPosted covers the case that made this
// guard necessary: Kanton Zürich publishes Anwesenheitsermittlung (a roll call)
// and the occasional quorum vote with no type at all, and both render as a
// perfectly ordinary lopsided Ja/Nein tally. Nothing about the counts gives them
// away, so only the type can stop them.
//
// The unhandled group must be skipped rather than posted, the handled one must
// still go out, and the run must end in an error so the failure is noticed
// instead of passing silently.
func TestPostToPlatform_UnhandledVoteTypeIsNotPosted(t *testing.T) {
	defer setupTempDir(t)()

	unhandled := createVote("anwesenheit-1", "2026/1", "2026-06-15")
	unhandled.Type = "" // what the source actually serves for a roll call

	handled := createVote("normal-1", "2026/2", "2026-06-15")

	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	groups := [][]votes.Vote{{unhandled}, {handled}}

	posted, err := PostToPlatform(groups, mockPlatform, SingleLog(testJurisdiction, voteLog), false)

	if !errors.Is(err, ErrUnsupportedVoteType) {
		t.Fatalf("expected ErrUnsupportedVoteType so the run fails visibly, got %v", err)
	}
	if posted != 1 {
		t.Errorf("expected the handled group to still post, got posted=%d", posted)
	}
	// The skipped vote must not be logged, or it would never be retried once the
	// type becomes renderable.
	if voteLog.IsPosted("anwesenheit-1") {
		t.Error("skipped vote was marked as posted; it would never be revisited")
	}
	if !voteLog.IsPosted("normal-1") {
		t.Error("handled vote was not marked as posted")
	}
}

// TestPostToPlatform_CantonCupVoteIsNotPosted covers the canton's Auswahl
// equivalent, a Cup-Abstimmung: one knockout round between more than two
// competing proposals.
//
// The real records cannot be published truthfully — every aggregate count is
// null, and the per-member rows are duplicated 296-for-180 with 175 members
// carrying a "Präsidium" value harmonised to abstention. Either defect alone
// would make a post wrong; the type check catches it before the counts do.
//
// Both are reported upstream. This pins that until they are fixed the group is
// skipped rather than rendered, and that the run still fails visibly.
func TestPostToPlatform_CantonCupVoteIsNotPosted(t *testing.T) {
	defer setupTempDir(t)()

	group := testfixtures.KantonsratCupVote()

	// Guard the fixture itself: if the source ever starts populating these, the
	// reason this test passes would silently change.
	v := group[0]
	if v.Yes != nil || v.No != nil || v.Abstention != nil {
		t.Fatalf("fixture should carry the null aggregates the source serves, got %v/%v/%v",
			v.Yes, v.No, v.Abstention)
	}
	if voteformat.IsHandledVoteType(v.Type) {
		t.Fatalf("Cup-Abstimmung should not be on the handled list, got type %q", v.Type)
	}

	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	posted, err := PostToPlatform([][]votes.Vote{group}, mockPlatform,
		SingleLog(testJurisdiction, voteLog), false)

	if !errors.Is(err, ErrUnsupportedVoteType) {
		t.Fatalf("expected ErrUnsupportedVoteType so the run fails visibly, got %v", err)
	}
	if posted != 0 {
		t.Errorf("a knockout round was published as an ordinary vote: posted=%d", posted)
	}
	if mockPlatform.formatCalls != 0 {
		t.Errorf("the group reached the formatter: formatCalls=%d", mockPlatform.formatCalls)
	}
	// Not logging it is what lets it be revisited once upstream is fixed.
	if voteLog.IsPosted(v.SourceID) {
		t.Error("skipped vote was marked as posted; it would never be revisited")
	}
}

// TestPostToPlatform_UnpostableVoteDoesNotSuppressItsGroup is the reason
// rejection is per vote rather than per group.
//
// Votes are grouped by business matter and sitting day, and Kanton Zürich
// interleaves vote types freely inside one business. The real 15.12.2025
// Steuerfuss item carries five ordinary Ja/Nein votes alongside three
// Cup-Abstimmung rounds; the 19.01.2026 Lehrpersonalgesetz item five alongside
// one. Rejecting the whole group would have suppressed the tax-rate decision
// because three procedural rounds in the same business are unrenderable.
func TestPostToPlatform_UnpostableVoteDoesNotSuppressItsGroup(t *testing.T) {
	defer setupTempDir(t)()

	// One business matter, one sitting day, mixed types — as the source serves it.
	substantive := createVote("steuerfuss-normal", "250382", "2025-12-15")
	unpostable := testfixtures.KantonsratCupVote()[0]
	unpostable.Jurisdiction = testJurisdiction
	unpostable.Affair.Number = "250382"
	unpostable.Date = substantive.Date

	group := []votes.Vote{substantive, unpostable}

	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	posted, err := PostToPlatform([][]votes.Vote{group}, mockPlatform,
		SingleLog(testJurisdiction, voteLog), false)

	// The run still fails, so the gap is noticed...
	if !errors.Is(err, ErrUnsupportedVoteType) {
		t.Fatalf("expected ErrUnsupportedVoteType, got %v", err)
	}
	// ...but the publishable vote is published rather than lost with it.
	if posted != 1 {
		t.Errorf("the substantive vote was suppressed by its neighbour: posted=%d", posted)
	}
	if !voteLog.IsPosted("steuerfuss-normal") {
		t.Error("substantive vote was not marked as posted")
	}
	// The dropped one must stay unlogged so it returns once the source is fixed.
	if voteLog.IsPosted(unpostable.SourceID) {
		t.Error("dropped vote was marked as posted; it would never be revisited")
	}

	// It must also not reach the formatter as part of the group.
	if got := len(mockPlatform.lastGroup); got != 1 {
		t.Errorf("formatter received %d votes, want only the postable one", got)
	}
}

// TestPostToPlatform_AttendanceRollCallIsSkippedWithoutFailingTheRun covers the
// case that made the 17.08.2026 run red for a fortnight.
//
// An attendance roll call is not a political vote, so it must never be
// published — but it is also not a fault. The Kantonsrat takes one at the start
// of most sittings, and nothing upstream needs fixing, so treating it like an
// unrecognised type would leave the action failing on every sitting day while
// the bot did exactly what it should.
func TestPostToPlatform_AttendanceRollCallIsSkippedWithoutFailingTheRun(t *testing.T) {
	defer setupTempDir(t)()

	rollCall := createVote("anwesenheit-1", "250999", "2026-08-17")
	rollCall.Type = "Anwesenheitsermittlung"

	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	posted, err := PostToPlatform([][]votes.Vote{{rollCall}}, mockPlatform,
		SingleLog(testJurisdiction, voteLog), false)

	if err != nil {
		t.Fatalf("a roll call must not fail the run, got %v", err)
	}
	if posted != 0 {
		t.Errorf("a roll call was published as a vote: posted=%d", posted)
	}
	if mockPlatform.formatCalls != 0 {
		t.Errorf("the roll call reached the formatter: formatCalls=%d", mockPlatform.formatCalls)
	}
	// Not logging it keeps the behaviour identical to any other skipped vote.
	if voteLog.IsPosted(rollCall.SourceID) {
		t.Error("skipped roll call was marked as posted")
	}
}

// TestPostToPlatform_AttendanceRollCallDoesNotMaskAnUnknownType pins that the
// two rejection kinds stay distinguishable when both occur in one run. A roll
// call must not swallow the signal that the source served something new.
func TestPostToPlatform_AttendanceRollCallDoesNotMaskAnUnknownType(t *testing.T) {
	defer setupTempDir(t)()

	rollCall := createVote("anwesenheit-2", "250998", "2026-08-17")
	rollCall.Type = "Anwesenheitsermittlung"

	novel := createVote("etwas-neues", "250997", "2026-08-17")
	novel.Type = "Wahlgang"

	mockPlatform := &MockPlatform{maxPosts: 10}
	voteLog := votelog.NewEmpty(testJurisdiction, votelog.PlatformX)

	_, err := PostToPlatform([][]votes.Vote{{rollCall}, {novel}}, mockPlatform,
		SingleLog(testJurisdiction, voteLog), false)

	if !errors.Is(err, ErrUnsupportedVoteType) {
		t.Fatalf("expected the unknown type to still fail the run, got %v", err)
	}
}
