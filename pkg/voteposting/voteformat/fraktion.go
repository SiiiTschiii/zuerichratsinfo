package voteformat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// FraktionCounts holds vote counts per faction, keyed by Abstimmungsverhalten value.
type FraktionCounts struct {
	Counts map[string]int // e.g. {"Ja": 32, "Nein": 0, "Enthaltung": 0, "Abwesend": 5}
}

// AggregateFraktionCounts groups Stimmabgaben by Fraktion, counting each Abstimmungsverhalten.
// Empty Fraktion ("") is omitted. The returned map contains exactly the factions present in the data.
func AggregateFraktionCounts(stimmabgaben []zurichapi.Stimmabgabe) map[string]*FraktionCounts {
	result := make(map[string]*FraktionCounts)
	for _, s := range stimmabgaben {
		if s.Fraktion == "" {
			continue
		}
		fc, ok := result[s.Fraktion]
		if !ok {
			fc = &FraktionCounts{Counts: make(map[string]int)}
			result[s.Fraktion] = fc
		}
		fc.Counts[s.Abstimmungsverhalten]++
	}
	return result
}

// headerAbbrev maps Abstimmungsverhalten values to short header labels.
var headerAbbrev = map[string]string{
	"Enthaltung": "Enth",
	"Abwesend":   "Abw",
}

// metaValues are Abstimmungsverhalten values that always sort last in the header.
var metaValues = map[string]bool{
	"Enthaltung": true,
	"Abwesend":   true,
}

// FormatFraktionBreakdown formats the aggregated counts into the display string.
// Returns "" if the input is empty.
// Header legend is built dynamically from distinct Abstimmungsverhalten values.
// Fraktionen sorted by total members descending (sum of all counts); ties broken alphabetically.
func FormatFraktionBreakdown(counts map[string]*FraktionCounts) string {
	if len(counts) == 0 {
		return ""
	}

	// Collect all distinct Abstimmungsverhalten keys.
	keySet := make(map[string]bool)
	for _, fc := range counts {
		for k := range fc.Counts {
			keySet[k] = true
		}
	}

	// Sort keys: non-meta first (natural order), then Enthaltung, then Abwesend.
	var primary []string
	hasEnth := false
	hasAbw := false
	hasStandardVote := false
	for k := range keySet {
		if k == "Enthaltung" {
			hasEnth = true
		} else if k == "Abwesend" {
			hasAbw = true
		} else if !metaValues[k] {
			primary = append(primary, k)
			if k == "Ja" || k == "Nein" {
				hasStandardVote = true
			}
		}
	}

	// For standard Ja/Nein votes, always include all 4 columns
	// (Ja, Nein, Enthaltung, Abwesend) for consistency, even when
	// no one voted that way.
	if hasStandardVote {
		if !keySet["Ja"] {
			primary = append(primary, "Ja")
		}
		if !keySet["Nein"] {
			primary = append(primary, "Nein")
		}
		hasEnth = true
		hasAbw = true
	}

	sort.Strings(primary)
	var columns []string
	columns = append(columns, primary...)
	if hasEnth {
		columns = append(columns, "Enthaltung")
	}
	if hasAbw {
		columns = append(columns, "Abwesend")
	}

	// Build header legend with abbreviations.
	headerParts := make([]string, len(columns))
	for i, col := range columns {
		if abbr, ok := headerAbbrev[col]; ok {
			headerParts[i] = abbr
		} else {
			headerParts[i] = col
		}
	}
	header := fmt.Sprintf("🏛️ Fraktionen (%s):", strings.Join(headerParts, "/"))

	// Sort factions by total members descending, ties alphabetically.
	type fraktionEntry struct {
		name  string
		total int
	}
	var factions []fraktionEntry
	for name, fc := range counts {
		total := 0
		for _, v := range fc.Counts {
			total += v
		}
		factions = append(factions, fraktionEntry{name: name, total: total})
	}
	sort.Slice(factions, func(i, j int) bool {
		if factions[i].total != factions[j].total {
			return factions[i].total > factions[j].total
		}
		return factions[i].name < factions[j].name
	})

	// Build output lines.
	var lines []string
	lines = append(lines, header)
	for _, f := range factions {
		fc := counts[f.name]
		vals := make([]string, len(columns))
		for i, col := range columns {
			vals[i] = fmt.Sprintf("%d", fc.Counts[col])
		}
		lines = append(lines, fmt.Sprintf("%s %s", f.name, strings.Join(vals, "/")))
	}

	return strings.Join(lines, "\n")
}
