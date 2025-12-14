package votelog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewEmpty(t *testing.T) {
	log := NewEmpty(PlatformX)

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
	log := NewEmpty(PlatformX)

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
	log := NewEmpty(PlatformX)

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
	log := NewEmpty(PlatformX)

	log.MarkAsPosted("vote1")

	if !log.IsPosted("vote1") {
		t.Error("Expected vote1 to be posted")
	}

	if log.IsPosted("vote2") {
		t.Error("Expected vote2 to NOT be posted")
	}
}

func TestMarkMultipleVotes_SimulatingGroupPost(t *testing.T) {
	log := NewEmpty(PlatformX)

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
	log := NewEmpty(PlatformX)
	log.MarkAsPosted("vote1")
	log.MarkAsPosted("vote2")
	log.MarkAsPosted("vote3")

	if err := log.Save(); err != nil {
		t.Fatalf("Failed to save log: %v", err)
	}

	// Load log
	loadedLog, err := Load(PlatformX)
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
	log, err := Load(PlatformX)
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
	log1 := NewEmpty(PlatformX)
	log1.MarkAsPosted("vote1")
	log1.MarkAsPosted("vote2")
	if err := log1.Save(); err != nil {
		t.Fatal(err)
	}

	// Second save (simulating a new run adding more votes)
	log2, err := Load(PlatformX)
	if err != nil {
		t.Fatal(err)
	}
	log2.MarkAsPosted("vote3")
	log2.MarkAsPosted("vote4")
	if err := log2.Save(); err != nil {
		t.Fatal(err)
	}

	// Load and verify all votes are there
	log3, err := Load(PlatformX)
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
