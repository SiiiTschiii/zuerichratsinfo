package votelog

import (
	"os"
	"path/filepath"
	"testing"
)

// testJurisdiction stands in for a real body, so the path layout is exercised
// alongside the log contents.
const testJurisdiction = "zurich-city"

// setupTempDir runs the test in a scratch working directory with a data/
// directory, since log paths are relative to the working directory.
// The returned function restores the previous directory.
func setupTempDir(t *testing.T) func() {
	t.Helper()
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.MkdirAll(filepath.Join(tmpDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	return func() { _ = os.Chdir(oldWd) }
}

func TestNewEmpty(t *testing.T) {
	log := NewEmpty(testJurisdiction, PlatformX)

	if log.Platform != PlatformX {
		t.Errorf("Expected platform X, got %s", log.Platform)
	}

	if len(log.Votes) != 0 {
		t.Errorf("Expected empty votes, got %d", len(log.Votes))
	}

	if log.index == nil {
		t.Error("Expected index to be initialized")
	}
}

func TestMarkAsPosted(t *testing.T) {
	log := NewEmpty(testJurisdiction, PlatformX)

	log.MarkAsPosted("vote1")
	log.MarkAsPosted("vote2")

	if len(log.Votes) != 2 {
		t.Errorf("Expected 2 votes, got %d", len(log.Votes))
	}

	if !log.IsPosted("vote1") {
		t.Error("Expected vote1 to be posted")
	}

	if !log.IsPosted("vote2") {
		t.Error("Expected vote2 to be posted")
	}
}

func TestMarkAsPosted_NoDuplicates(t *testing.T) {
	log := NewEmpty(testJurisdiction, PlatformX)

	log.MarkAsPosted("vote1")
	log.MarkAsPosted("vote1")
	log.MarkAsPosted("vote1")

	if len(log.Votes) != 1 {
		t.Errorf("Expected 1 vote (no duplicates), got %d", len(log.Votes))
	}

	if log.Votes[0].ID != "vote1" {
		t.Errorf("Expected vote1, got %s", log.Votes[0].ID)
	}
}

func TestIsPosted(t *testing.T) {
	log := NewEmpty(testJurisdiction, PlatformX)

	log.MarkAsPosted("vote1")

	if !log.IsPosted("vote1") {
		t.Error("Expected vote1 to be posted")
	}

	if log.IsPosted("vote2") {
		t.Error("Expected vote2 to NOT be posted")
	}
}

func TestMarkMultipleVotes_SimulatingGroupPost(t *testing.T) {
	log := NewEmpty(testJurisdiction, PlatformX)

	// Simulate posting a group of 3 votes (like a Geschäft with multiple votes)
	voteGroup := []string{"vote1", "vote2", "vote3"}

	for _, voteID := range voteGroup {
		log.MarkAsPosted(voteID)
	}

	// Verify all votes are marked as posted
	if len(log.Votes) != 3 {
		t.Errorf("Expected 3 votes logged, got %d", len(log.Votes))
	}

	for _, voteID := range voteGroup {
		if !log.IsPosted(voteID) {
			t.Errorf("Expected %s to be posted", voteID)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()

	// Create data directory in temp location
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create and save log
	log := NewEmpty(testJurisdiction, PlatformX)
	log.MarkAsPosted("vote1")
	log.MarkAsPosted("vote2")
	log.MarkAsPosted("vote3")

	if err := log.Save(); err != nil {
		t.Fatalf("Failed to save log: %v", err)
	}

	// Load log
	loadedLog, err := Load(testJurisdiction, PlatformX)
	if err != nil {
		t.Fatalf("Failed to load log: %v", err)
	}

	// Verify loaded data
	if len(loadedLog.Votes) != 3 {
		t.Errorf("Expected 3 votes after loading, got %d", len(loadedLog.Votes))
	}

	if !loadedLog.IsPosted("vote1") || !loadedLog.IsPosted("vote2") || !loadedLog.IsPosted("vote3") {
		t.Error("Expected all votes to be marked as posted after loading")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()

	// Change to temp directory (no data dir created)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Load should return empty log without error
	log, err := Load(testJurisdiction, PlatformX)
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}

	if len(log.Votes) != 0 {
		t.Errorf("Expected empty log, got %d votes", len(log.Votes))
	}

	if log.Platform != PlatformX {
		t.Errorf("Expected platform X, got %s", log.Platform)
	}
}

func TestPersistenceAcrossMultipleSaves(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()

	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// First save
	log1 := NewEmpty(testJurisdiction, PlatformX)
	log1.MarkAsPosted("vote1")
	log1.MarkAsPosted("vote2")
	if err := log1.Save(); err != nil {
		t.Fatal(err)
	}

	// Second save (simulating a new run adding more votes)
	log2, err := Load(testJurisdiction, PlatformX)
	if err != nil {
		t.Fatal(err)
	}
	log2.MarkAsPosted("vote3")
	log2.MarkAsPosted("vote4")
	if err := log2.Save(); err != nil {
		t.Fatal(err)
	}

	// Load and verify all votes are there
	log3, err := Load(testJurisdiction, PlatformX)
	if err != nil {
		t.Fatal(err)
	}

	if len(log3.Votes) != 4 {
		t.Errorf("Expected 4 votes total, got %d", len(log3.Votes))
	}

	expectedVotes := []string{"vote1", "vote2", "vote3", "vote4"}
	for _, voteID := range expectedVotes {
		if !log3.IsPosted(voteID) {
			t.Errorf("Expected %s to be posted", voteID)
		}
	}
}

func TestNewNoOp(t *testing.T) {
	log := NewNoOp(testJurisdiction, PlatformX)

	// IsPosted always returns false
	log.MarkAsPosted("vote1") // should be a no-op
	if log.IsPosted("vote1") {
		t.Error("NewNoOp: IsPosted should always return false")
	}

	// Votes slice should remain empty (MarkAsPosted is a no-op)
	if len(log.Votes) != 0 {
		t.Errorf("NewNoOp: expected 0 votes, got %d", len(log.Votes))
	}

	// Save should be a no-op (no error, no file written)
	if err := log.Save(); err != nil {
		t.Errorf("NewNoOp: Save should return nil, got %v", err)
	}
}

func TestLogFilePath(t *testing.T) {
	if got, want := LogFilePath("zurich-city", PlatformInstagram), "data/zurich-city/posted_votes_instagram.json"; got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
	if got, want := LogFilePath("zurich-canton", PlatformX), "data/zurich-canton/posted_votes_x.json"; got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// A run that happens before the stored logs are moved to the nested layout must
// still see the existing history. Reading an empty log instead would treat every
// vote inside the age guard as unposted and re-publish it.
func TestLoad_FallsBackToLegacyFlatPath(t *testing.T) {
	defer setupTempDir(t)()

	legacy := filepath.Join("data", "posted_votes_x.json")
	if err := os.WriteFile(legacy, []byte(`{"platform":"x","votes":[{"id":"old-vote","posted_at":"2025-11-06T10:00:00Z"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	log, err := Load("zurich-city", PlatformX)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !log.IsPosted("old-vote") {
		t.Error("legacy log entry should be visible after the fallback read")
	}

	// Saving completes the migration: the new path holds the history.
	if err := log.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(LogFilePath("zurich-city", PlatformX)); err != nil {
		t.Errorf("expected the log to be written to the nested path: %v", err)
	}
}

// Other jurisdictions have no flat history, and must not inherit the city's.
func TestLoad_NoLegacyFallbackForOtherJurisdictions(t *testing.T) {
	defer setupTempDir(t)()

	legacy := filepath.Join("data", "posted_votes_x.json")
	if err := os.WriteFile(legacy, []byte(`{"platform":"x","votes":[{"id":"city-vote","posted_at":"2025-11-06T10:00:00Z"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	log, err := Load("zurich-canton", PlatformX)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if log.Count() != 0 {
		t.Errorf("expected an empty log for a jurisdiction with no history, got %d entries", log.Count())
	}
}

// The migration to the per-jurisdiction layout has to complete on a run, not on
// a post. Save is otherwise only reached after something is published, so a
// council in recess would leave the logs in the old layout — and the
// transitional handling in the CI workflow with them — until it next sits.
func TestLoad_ReportsTheLegacyPathItMigratedFrom(t *testing.T) {
	defer setupTempDir(t)()

	legacy := filepath.Join("data", "posted_votes_x.json")
	if err := os.WriteFile(legacy, []byte(`{"platform":"x","votes":[{"id":"old","posted_at":"2025-11-06T10:00:00Z"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	log, err := Load("zurich-city", PlatformX)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := log.LoadedFromLegacyPath(); got != legacy {
		t.Errorf("LoadedFromLegacyPath = %q, want %q", got, legacy)
	}

	// Reading from its own path is not a migration.
	if err := log.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	again, err := Load("zurich-city", PlatformX)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := again.LoadedFromLegacyPath(); got != "" {
		t.Errorf("LoadedFromLegacyPath = %q after migrating, want empty", got)
	}
	if !again.IsPosted("old") {
		t.Error("the migrated log lost its history")
	}
}
