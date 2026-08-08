package votelog

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Platform represents a social media platform
type Platform string

const (
	PlatformX         Platform = "x"
	PlatformBluesky   Platform = "bluesky"
	PlatformInstagram Platform = "instagram"
	PlatformMastodon  Platform = "mastodon"
)

// dataDir is the root the logs live under, relative to the working directory.
const dataDir = "data"

// VoteEntry represents a single posted vote
type VoteEntry struct {
	ID       string    `json:"id"`
	PostedAt time.Time `json:"posted_at"`
}

// VoteLog tracks posted votes for one jurisdiction on one platform.
//
// The pairing matters: dedup is per account *and* per body, because a vote
// posted to X has not thereby been posted to Bluesky, and two jurisdictions
// sharing an account still keep separate histories.
type VoteLog struct {
	Jurisdiction string               `json:"jurisdiction,omitempty"`
	Platform     Platform             `json:"platform"`
	Votes        []VoteEntry          `json:"votes"`
	filepath     string               // not exported, internal use
	index        map[string]VoteEntry // for fast lookup
	noOp         bool                 // when true, all votes are treated as unposted
	legacyPath   string               // set when loaded from the pre-jurisdiction path
}

// Load reads a jurisdiction's log for one platform.
//
// A missing file yields an empty log with no error, which reads as "nothing was
// ever posted". That is the correct behaviour for a genuinely new jurisdiction
// and a dangerous one for an existing jurisdiction whose log went missing — see
// the workflow's restore step, which treats a missing state branch as fatal for
// exactly this reason.
func Load(jurisdiction string, platform Platform) (*VoteLog, error) {
	path := LogFilePath(jurisdiction, platform)

	vl := &VoteLog{
		Jurisdiction: jurisdiction,
		Platform:     platform,
		Votes:        []VoteEntry{},
		filepath:     path,
		index:        make(map[string]VoteEntry),
	}

	readFrom := path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Fall back to the pre-jurisdiction flat layout, so a run that happens
		// before the stored logs are moved does not read as an empty history
		// and re-post everything inside the age guard. Writes always go to the
		// new path, so the first save completes the migration.
		legacy := legacyLogFilePath(jurisdiction, platform)
		if legacy == "" {
			return vl, nil
		}
		if _, err := os.Stat(legacy); os.IsNotExist(err) {
			return vl, nil
		}
		log.Printf("ℹ️  %s/%s: reading legacy vote log %s; it will be written to %s",
			jurisdiction, platform, legacy, path)
		readFrom = legacy
		vl.legacyPath = legacy
	}

	data, err := os.ReadFile(readFrom)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	if err := json.Unmarshal(data, vl); err != nil {
		return nil, fmt.Errorf("failed to parse log file: %w", err)
	}

	// The stored file may predate the jurisdiction field, and its path is
	// authoritative either way.
	vl.Jurisdiction = jurisdiction
	vl.Platform = platform
	vl.filepath = path
	vl.legacyPath = legacyPathIfUsed(readFrom, path)

	for _, entry := range vl.Votes {
		vl.index[entry.ID] = entry
	}

	return vl, nil
}

// IsPosted checks if a vote has been posted
func (l *VoteLog) IsPosted(voteID string) bool {
	if l.noOp {
		return false
	}
	_, exists := l.index[voteID]
	return exists
}

// MarkAsPosted marks a vote as posted
func (l *VoteLog) MarkAsPosted(voteID string) {
	if l.noOp {
		return
	}

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
	if l.noOp {
		return nil
	}

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

// LoadedFromLegacyPath returns the pre-jurisdiction path this log was read
// from, or "" when it was read from its own.
//
// Callers use it to finish the migration on the next run rather than on the
// next post. Save is otherwise only reached after something is published, so a
// council in recess would leave the logs in the old layout indefinitely — and
// the transitional handling in the workflow with them.
func (l *VoteLog) LoadedFromLegacyPath() string {
	return l.legacyPath
}

// legacyPathIfUsed reports the source path when it differs from where the log
// will be written.
func legacyPathIfUsed(readFrom, writeTo string) string {
	if readFrom == writeTo {
		return ""
	}
	return readFrom
}

// NewEmpty creates an empty vote log (useful for testing or when we want to show all votes)
func NewEmpty(jurisdiction string, platform Platform) *VoteLog {
	return &VoteLog{
		Jurisdiction: jurisdiction,
		Platform:     platform,
		Votes:        []VoteEntry{},
		filepath:     LogFilePath(jurisdiction, platform),
		index:        make(map[string]VoteEntry),
	}
}

// NewNoOp creates a no-op vote log that treats all votes as unposted
// and discards all mark/save operations. Used for manual e2e testing.
func NewNoOp(jurisdiction string, platform Platform) *VoteLog {
	return &VoteLog{
		Jurisdiction: jurisdiction,
		Platform:     platform,
		Votes:        []VoteEntry{},
		index:        make(map[string]VoteEntry),
		noOp:         true,
	}
}

// LogFilePath returns where a jurisdiction's log for a platform is stored.
// The nesting is what lets the CI workflow round-trip every jurisdiction's
// logs with one glob instead of an entry per jurisdiction.
func LogFilePath(jurisdiction string, platform Platform) string {
	return filepath.Join(dataDir, jurisdiction, fmt.Sprintf("posted_votes_%s.json", platform))
}

// legacyJurisdiction is the one jurisdiction whose logs predate the
// per-jurisdiction layout and therefore have a flat path to fall back to.
const legacyJurisdiction = "zurich-city"

// legacyLogFilePath returns the pre-jurisdiction path for a log, or "" when the
// jurisdiction never had one.
func legacyLogFilePath(jurisdiction string, platform Platform) string {
	if jurisdiction != legacyJurisdiction {
		return ""
	}
	return filepath.Join(dataDir, fmt.Sprintf("posted_votes_%s.json", platform))
}
