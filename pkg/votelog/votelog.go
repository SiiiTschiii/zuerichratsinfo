package votelog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Platform represents a social media platform
type Platform string

const (
	PlatformX        Platform = "x"
	PlatformBluesky  Platform = "bluesky"
	PlatformMastodon Platform = "mastodon"
)

// VoteEntry represents a single posted vote
type VoteEntry struct {
	ID       string    `json:"id"`
	PostedAt time.Time `json:"posted_at"`
}

// VoteLog tracks posted votes for a specific platform
type VoteLog struct {
	Platform Platform               `json:"platform"`
	Votes    []VoteEntry            `json:"votes"`
	filepath string                 // not exported, internal use
	index    map[string]VoteEntry   // for fast lookup
}

// Load loads a vote log for the specified platform
// If the file doesn't exist, returns an empty log
func Load(platform Platform) (*VoteLog, error) {
	filepath := getLogFilePath(platform)
	
	log := &VoteLog{
		Platform: platform,
		Votes:    []VoteEntry{},
		filepath: filepath,
		index:    make(map[string]VoteEntry),
	}
	
	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		// File doesn't exist, return empty log
		return log, nil
	}
	
	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}
	
	// Parse JSON
	if err := json.Unmarshal(data, log); err != nil {
		return nil, fmt.Errorf("failed to parse log file: %w", err)
	}
	
	// Build index for fast lookup
	for _, entry := range log.Votes {
		log.index[entry.ID] = entry
	}
	
	return log, nil
}

// IsPosted checks if a vote has been posted
func (l *VoteLog) IsPosted(voteID string) bool {
	_, exists := l.index[voteID]
	return exists
}

// MarkAsPosted marks a vote as posted
func (l *VoteLog) MarkAsPosted(voteID string) {
	// Don't add duplicates
	if l.IsPosted(voteID) {
		return
	}
	
	entry := VoteEntry{
		ID:       voteID,
		PostedAt: time.Now(),
	}
	
	l.Votes = append(l.Votes, entry)
	l.index[voteID] = entry
}

// Save writes the log to disk
func (l *VoteLog) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(l.filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(l.filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}
	
	return nil
}

// Count returns the number of posted votes
func (l *VoteLog) Count() int {
	return len(l.Votes)
}

// getLogFilePath returns the file path for a platform's log
func getLogFilePath(platform Platform) string {
	return fmt.Sprintf("data/posted_votes_%s.json", platform)
}
