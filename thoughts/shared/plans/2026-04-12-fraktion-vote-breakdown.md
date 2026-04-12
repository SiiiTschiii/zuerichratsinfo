---
date: 2026-04-12
author: copilot
topic: "Per-Fraktion vote breakdown in X/Bluesky reply threads"
tags: [plan, fraktion, voting, x, bluesky, voteformat]
status: ready
research: thoughts/shared/research/2026-04-12-fraktion-vote-breakdown.md
---

# Fraktion Vote Breakdown — Implementation Plan

## Overview

Add a per-Fraktion vote breakdown reply to X and Bluesky threads. After the vote counts line, a new entry shows how each parliamentary faction voted (e.g. `SP 32/0/0/5`). The Fraktion breakdown is a separate bin-packing entry — stays with vote counts if space permits, otherwise spills to the next reply.

## Current State Analysis

- `buildReplyPosts()` in X ([format.go:131-214](pkg/voteposting/platforms/x/format.go#L131)) and Bluesky ([format.go:137-221](pkg/voteposting/platforms/bluesky/format.go#L137)) build a list of entry strings, then bin-pack them into replies respecting char limits (X: `charLimit`, Bluesky: `maxGraphemes=300`).
- Each entry is currently: `[emoji subtitle\n]📊 counts_line`.
- `Stimmabgaben` data is already present on each `Abstimmung` struct flowing into the formatters — no threading changes needed. Currently unused.
- Fixtures in `testfixtures/fixtures.go` don't populate `Stimmabgaben`.

## Desired End State

- Every Ja/Nein or Auswahl vote in a thread gets a Fraktion breakdown entry appended after its counts entry.
- The breakdown is formatted as `🏛️ Fraktionen (Ja/Nein/Enth/Abw):\nSP 32/0/0/5\n...` — one faction per line.
- Header legend is built dynamically from distinct `Abstimmungsverhalten` values (adapts to Auswahl `A/B/C/Abw` automatically).
- Empty Fraktion (`""`) is omitted. Sorted by total members descending (biggest factions first).
- If `Stimmabgaben` is empty (older API data), no breakdown entry is added — graceful degradation.

### Key Discoveries

- `vote.Stimmabgaben.Stimmabgabe` is a `[]Stimmabgabe` with `Fraktion` and `Abstimmungsverhalten` fields (`types.go:119-130`).
- Currently 7 Fraktionen + 1 empty, but this can change across legislative periods. The set is always derived from data, never hardcoded. Sorted by total members descending (biggest factions first).
- Fraktion breakdown is ~175 chars (Ja/Nein) — fits in 280 (X) and 300 (Bluesky) on its own.
- Combined vote counts + Fraktion = ~270 chars — fits for short subtitles, may split for long ones.
- X measures `len(string)` (bytes), Bluesky measures `len([]rune(s))` (graphemes). Both formatters call `FormatFraktionBreakdown` identically — the measurement difference is in the bin-packer, not in the entry construction.

## What We're NOT Doing

- No per-Partei breakdown (using Fraktion grouping only).
- No Fraktion-colored formatting or rich text — plain text only.
- No abbreviations for Fraktion names.
- No API changes or additional fetches.
- No changes to root post format.

## Implementation Approach

New shared `fraktion.go` in `pkg/voteposting/voteformat/` with aggregation + formatting. Both platform formatters call it after building the counts entry, adding the result as a second bin-packing entry. Existing bin-packing algorithm handles the rest untouched.

---

## Phase 1: Core Fraktion Formatting Logic

### Overview

Create `fraktion.go` with aggregation and formatting functions, plus comprehensive unit tests.

### Changes Required

#### 1. New file: `pkg/voteposting/voteformat/fraktion.go`

**Functions:**

```go
// FraktionCounts holds vote counts per faction, keyed by Abstimmungsverhalten value.
type FraktionCounts struct {
    Counts map[string]int // e.g. {"Ja": 32, "Nein": 0, "Enthaltung": 0, "Abwesend": 5}
}

// AggregateFraktionCounts groups Stimmabgaben by Fraktion, counting each Abstimmungsverhalten.
// Empty Fraktion ("") is omitted. The returned map contains exactly the factions present in the data.
func AggregateFraktionCounts(stimmabgaben []zurichapi.Stimmabgabe) map[string]*FraktionCounts

// FormatFraktionBreakdown formats the aggregated counts into the display string.
// Returns "" if the input is empty.
// Header legend is built dynamically from distinct Abstimmungsverhalten values.
// Fraktionen sorted by total members descending (sum of all counts); ties broken alphabetically.
func FormatFraktionBreakdown(counts map[string]*FraktionCounts) string
```

**Header legend logic:**

- Collect all distinct `Abstimmungsverhalten` keys across all factions.
- Abbreviate in header: `Enthaltung` → `Enth`, `Abwesend` → `Abw`. Others as-is.
- Order: non-meta values first (Ja/Nein or A/B/C/D/E in natural order), then Enth, then Abw.
- Example: `(Ja/Nein/Enth/Abw)` or `(A/B/C/Abw)`.

**Per-faction line:** `{Fraktion} {count1}/{count2}/{count3}/{count4}` — same order as header.

#### 2. New file: `pkg/voteposting/voteformat/fraktion_test.go`

**Test cases:**

- Standard Ja/Nein vote (7 factions, realistic counts) → verify header + all lines + sorted by total members desc.
- Auswahl vote (A/B/C + Abwesend only) → verify dynamic header `(A/B/C/Abw)`.
- Empty Stimmabgaben → returns `""`.
- Empty Fraktion filtered out.
- Tie-breaking → factions with same total sorted alphabetically.
- Single faction → still works.

### Success Criteria

#### Automated Verification:

- [x] `go test ./pkg/voteposting/voteformat/...` passes
- [x] `go build ./...` passes
- [x] `go vet ./...` passes

#### Manual Verification:

Run `go test -v -run TestFormatFraktionBreakdown ./pkg/voteposting/voteformat/...` and inspect the `-v` output:

- [x] Ja/Nein test: header is `(Ja/Nein/Enth/Abw)`, factions sorted largest-first, counts match
- [x] Auswahl test: header is `(A/B/C/Abw)` (no Enth)
- [x] Empty input returns `""`

**Pause for manual verification before proceeding.**

---

## Phase 2: Wire Fraktion Breakdown into Formatters

### Overview

Add Fraktion breakdown entries to `buildReplyPosts()` in both X and Bluesky. Each vote's Fraktion breakdown becomes a separate bin-packing entry immediately after the vote's counts entry.

### Changes Required

#### 1. X formatter: `pkg/voteposting/platforms/x/format.go`

**In `buildReplyPosts()`**, after building each vote's counts entry and appending it to `entries`:

```go
// After: entries = append(entries, entry.String())
// Add Fraktion breakdown as separate entry
if stimmabgaben := vote.Stimmabgaben.Stimmabgabe; len(stimmabgaben) > 0 {
    fraktionCounts := voteformat.AggregateFraktionCounts(stimmabgaben)
    if breakdown := voteformat.FormatFraktionBreakdown(fraktionCounts); breakdown != "" {
        entries = append(entries, breakdown)
    }
}
```

No changes to the bin-packing algorithm — it already handles any number of entries.

#### 2. Bluesky formatter: `pkg/voteposting/platforms/bluesky/format.go`

**Same change** in `buildReplyPosts()` — identical code, different package.

#### 3. Add Stimmabgaben to test fixtures: `pkg/voteposting/testfixtures/fixtures.go`

Add a helper function and populate `Stimmabgaben` on existing fixtures that need it:

```go
// stimmabgabe creates a single Stimmabgabe with the given faction and vote behavior.
func stimmabgabe(fraktion, verhalten string) zurichapi.Stimmabgabe {
    return zurichapi.Stimmabgabe{
        Fraktion:             fraktion,
        Abstimmungsverhalten: verhalten,
    }
}

// makeStimmabgaben creates a Stimmabgaben slice from faction vote distributions.
// Each entry is (fraktion, ja, nein, enthaltung, abwesend) for Ja/Nein votes.
func makeStimmabgaben(factions []struct{ Name string; Ja, Nein, Enth, Abw int }) []zurichapi.Stimmabgabe
```

Add `Stimmabgaben` to:

- `SingleVoteAngenommen()` — realistic 7-faction data matching its counts (79/29/0/17).
- `MultiVoteGroup()` — different Fraktion splits for each of the 2 votes.
- `AuswahlVote()` — A/B/C Stimmabgaben with Abwesend.

Other fixtures can be left without Stimmabgaben — they'll gracefully show no breakdown.

### Success Criteria

#### Automated Verification:

- [x] `go test ./...` passes (all existing tests still pass)
- [x] `go build ./...` passes

#### Manual Verification:

No new tests yet (Phase 3 adds them). Verify by reading the changed code in `buildReplyPosts()` in both formatters:

- [x] The Fraktion entry is appended right after the vote counts entry
- [x] Guarded by `len(stimmabgaben) > 0` and `breakdown != ""`
- [x] Fixture Stimmabgaben totals match the fixture's aggregate vote counts

**Pause for manual verification before proceeding.**

---

## Phase 3: Format Tests for Fraktion in Threads

### Overview

Add test cases to X and Bluesky format tests verifying Fraktion breakdown appears correctly in formatted threads.

### Changes Required

#### 1. X format tests: `pkg/voteposting/platforms/x/format_test.go`

**New test cases:**

- `TestFormatVoteThread_SingleVoteWithFraktion` — single vote with Stimmabgaben → thread has root + counts reply + Fraktion reply. Verify `🏛️ Fraktionen` appears, all 7 factions present, correct counts.
- `TestFormatVoteThread_MultiVoteWithFraktion` — 2 votes with Stimmabgaben → each vote has its own Fraktion entry. Verify bin-packing keeps vote+breakdown together when they fit.
- `TestFormatVoteThread_NoStimmabgaben` — vote without Stimmabgaben → no Fraktion entry, thread unchanged from current behavior.
- `TestFormatVoteThread_AuswahlWithFraktion` — Auswahl vote with A/B/C Stimmabgaben → header shows `(A/B/C/Abw)`.

#### 2. Bluesky format tests: `pkg/voteposting/platforms/bluesky/format_test.go`

**Same test cases** adapted for Bluesky (uses `FormatVoteCounts` short labels, `[]*BlueskyPost`).

### Success Criteria

#### Automated Verification:

- [x] `go test ./pkg/voteposting/...` passes
- [x] `go vet ./...` passes

#### Manual Verification:

Run `go test -v -run WithFraktion\|NoStimmabgaben\|AuswahlWithFraktion ./pkg/voteposting/platforms/...` and inspect the `-v` output:

- [x] `SingleVoteWithFraktion`: thread has 3+ posts (root, counts+link, Fraktion breakdown). Fraktion post contains `🏛️ Fraktionen`
- [x] `MultiVoteWithFraktion`: each vote has its own Fraktion entry (look for two `🏛️ Fraktionen` occurrences)
- [x] `NoStimmabgaben`: thread has same structure as before (no `🏛️ Fraktionen` in any post)
- [x] `AuswahlWithFraktion`: header shows `(A/B/C/Abw)` not `(Ja/Nein/Enth/Abw)`

**Pause for manual verification before proceeding.**

---

## Phase 4: E2E Fixture Update

### Overview

Update `post_fixture` fixtures to include Stimmabgaben so e2e posting tests include Fraktion breakdowns.

### Changes Required

#### 1. Fixtures map: `testfixtures/fixtures.go`

The `AllFixtures()` map already returns fixtures. Since Phase 2 added `Stimmabgaben` to `SingleVoteAngenommen`, `MultiVoteGroup`, and `AuswahlVote`, the `post_fixture` command will automatically post Fraktion breakdowns for those fixtures.

Verify by running:

```bash
go run ./cmd/cleanup_posts --platform=all
go run ./cmd/post_fixture --fixture=single-vote-angenommen
go run ./cmd/post_fixture --fixture=multi-vote-group
go run ./cmd/post_fixture --fixture=auswahl-vote
```

### Success Criteria

#### Automated Verification:

- [x] `go build ./cmd/post_fixture` succeeds

#### Manual Verification:

Run the commands above and inspect the posts on X and Bluesky test accounts:

- [x] E2e post on X test account shows Fraktion breakdown reply.
- [x] E2e post on Bluesky test account shows Fraktion breakdown reply.
- [x] Auswahl vote shows `(A/B/C/Abw)` header.
- [x] Multi-vote group has per-vote Fraktion entries.

---

## Testing Strategy

### Unit Tests

- `fraktion_test.go` — aggregation + formatting (Ja/Nein, Auswahl, empty, edge cases)
- `x/format_test.go` — thread structure with Fraktion entries
- `bluesky/format_test.go` — thread structure with Fraktion entries

### Integration Tests

- `post_fixture` e2e against test accounts (manual)

### Manual Testing Steps

1. Run `go run ./cmd/cleanup_posts --platform=all`
2. Post `single-vote` fixture → verify Fraktion reply on both platforms
3. Post `multi-vote-group` fixture → verify per-vote Fraktion entries
4. Post `auswahl-vote` fixture → verify dynamic A/B/C header
5. Post fixture without Stimmabgaben → verify no Fraktion reply (graceful)

## References

- Research: `thoughts/shared/research/2026-04-12-fraktion-vote-breakdown.md`
- `pkg/zurichapi/types.go:119-130` — Stimmabgabe struct
- `pkg/voteposting/platforms/x/format.go:131-214` — X buildReplyPosts
- `pkg/voteposting/platforms/bluesky/format.go:137-221` — Bluesky buildReplyPosts
- `pkg/voteposting/voteformat/voteformat.go` — existing format functions
- `pkg/voteposting/testfixtures/fixtures.go` — shared fixtures
