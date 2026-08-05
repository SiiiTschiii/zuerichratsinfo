// Package config is the composition root: it names the parliamentary bodies the
// bot covers, wires each to the API that serves it, and groups them into the
// accounts they post to.
//
// It is the only package that knows about every source, which keeps the sources
// unaware of each other and everything downstream unaware of all of them.
package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// Jurisdiction is a body the bot posts for, plus the operational settings that
// are not properties of the body itself.
type Jurisdiction struct {
	votes.Jurisdiction

	// MaxAgeDays bounds how old a vote may be and still be posted.
	//
	// This is not a freshness preference. Dedup works by fetching the most
	// recent N votes and subtracting whatever the vote log records, and the
	// logs do not reach back to the beginning of time — so any vote older than
	// its log's earliest entry is indistinguishable from an unposted one. When
	// the source re-indexes an old vote back into the fetch window, this guard
	// is the only thing standing between that and a second posting.
	//
	// For a jurisdiction whose log starts empty, *every* historical vote looks
	// unposted, and this value alone bounds the first run.
	MaxAgeDays int

	// NewSource builds the adapter that fetches this body's votes.
	NewSource func() votes.Source
}

// jurisdictions is the registry, keyed by Jurisdiction.Key.
var jurisdictions = map[string]Jurisdiction{
	zurichapi.JurisdictionKey: {
		Jurisdiction: zurichapi.Jurisdiction,
		MaxAgeDays:   90,
		NewSource:    func() votes.Source { return zurichapi.NewClient() },
	},
}

// LookupJurisdiction returns a registered jurisdiction, with MaxAgeDays
// overridden from the environment if configured.
func LookupJurisdiction(key string) (Jurisdiction, error) {
	j, ok := jurisdictions[key]
	if !ok {
		return Jurisdiction{}, fmt.Errorf("unknown jurisdiction %q", key)
	}
	j.MaxAgeDays = maxAgeDays(key, j.MaxAgeDays)
	return j, nil
}

// JurisdictionKeys lists every registered jurisdiction, sorted for stable output.
func JurisdictionKeys() []string {
	keys := make([]string, 0, len(jurisdictions))
	for k := range jurisdictions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// maxAgeDays resolves the age guard for a jurisdiction. A per-jurisdiction
// variable wins; the unprefixed legacy name is honoured for Stadt Zürich so the
// existing repository variable keeps working.
func maxAgeDays(key string, fallback int) int {
	if v, ok := envInt("MAX_VOTE_AGE_DAYS_" + envKey(key)); ok {
		return v
	}
	if key == zurichapi.JurisdictionKey {
		if v, ok := envInt("MAX_VOTE_AGE_DAYS"); ok {
			return v
		}
	}
	return fallback
}

// envKey turns a jurisdiction or channel key into the shape used in environment
// variable names: "zurich-city" becomes "ZURICH_CITY".
func envKey(key string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(key))
}

func envInt(name string) (int, bool) {
	raw := os.Getenv(name)
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return n, true
}
