---
date: 2026-08-05
author: claude
topic: "Source-neutral vote model + Kanton Zürich (Kantonsrat) expansion"
tags: [plan, expansion, kantonsrat, openparldata, refactor, votes]
status: ready
research: thoughts/shared/research/2026-08-05-openparldata-federal-kantonsrat-feasibility.md
---

# Source-Neutral Vote Model + Kanton Zürich — Implementation Plan

## Overview

Extract a source-neutral vote domain model from `pkg/zurichapi`, then add **Kanton Zürich (Kantonsrat)** as a second jurisdiction sourced from OpenParlData. The refactor lands first as a pure no-behaviour-change step while still serving only Stadt Zürich, so the existing test suite is the safety net.

The Nationalrat is explicitly **out of scope here** — it is blocked on an upstream data-quality bug (see [research §2.2](../research/2026-08-05-openparldata-federal-kantonsrat-feasibility.md)) and needs its own editorial policy. The refactor in Phases 1–3 is what it will build on.

## Current State Analysis

- `zurichapi.Abstimmung` is the de-facto domain type, referenced in **28 files**. Concentrations: `testfixtures/fixtures.go` (42 refs), `platforms/bluesky/format_test.go` (18), `platforms/x/format_test.go` (15), `voteposting_test.go` (13), `imagegen.go` (8), `prepare.go` (7).
- `PrepareVoteGroups()` ([prepare.go:28](pkg/voteposting/prepare.go#L28)) takes a concrete `*zurichapi.Client`, calls `FetchRecentAbstimmungen` → `filterUnpostedVotes` → `GroupAbstimmungenByGeschaeft`, then validates `Schlussresultat` against counts.
- `platforms.Platform.Format()` ([interface.go:14](pkg/voteposting/platforms/interface.go#L14)) takes `[]zurichapi.Abstimmung` — the type crosses the platform boundary.
- `voteformat` is already largely source-neutral: `VoteCounts` ([voteformat.go:139](pkg/voteposting/voteformat/voteformat.go#L139)) is a standalone struct of `*int`s. Only `AggregateFraktionCounts` ([fraktion.go:18](pkg/voteposting/voteformat/fraktion.go#L18)) takes `[]zurichapi.Stimmabgabe`.
- `votelog` keys purely on a vote ID string, but paths are hardcoded: `data/posted_votes_%s.json` ([votelog.go:155](pkg/votelog/votelog.go#L155)).
- Hardcoded Zurich-city URLs exist in `prepare.go:169` (`gemeinderat-zuerich.ch/abstimmungen/detail.php`) and `voteformat.go:248-260` (`GenerateVoteLink`/`GenerateTraktandumLink`/`GenerateGeschaeftLink`).
- `data/contacts.yaml` is 132 curated Stadt-Zürich entries, keyed by politician `name`.
- All packages currently pass `go test ./...`.

### Key Discoveries

- **Totals and member votes must be independent fields.** PARIS delivers them together; OpenParlData derives them from separate steps and can in principle disagree. Formatters must read totals directly, not sum `MemberVotes`.
- **Kanton ZH data is clean**: 40/40 sampled votings reconcile exactly at 180/180 members, ~1% unmapped Fraktion. The completeness gate is insurance, not the common path.
- **`lang_format=flat` is mandatory** on every OpenParlData call, or `*_de` fields silently return `null`.
- **`affair_number` requires a second call** to `/v1/votings/{id}/affairs` — it is null inline on the voting.
- **ZH has no Traktandum and no vote type.** `SelectBestTitle(traktandumTitel, geschaeftTitel)` and `Abstimmungstyp` handling must degrade rather than branch per source.
- **Harvest lag is ~1.4 d median**, so the `maxAgeDays` guard (currently trimming votes older than N days) needs a per-jurisdiction value or Kantonsrat votes will be filtered out before posting.

## Desired End State

- A `pkg/votes` package defines `Vote` / `MemberVote` / `Jurisdiction` with no import of any source package.
- `pkg/zurichapi` and a new `pkg/openparldata` both implement a `votes.Source` interface; neither is referenced from `voteposting`, `imagegen` or the platforms.
- `platforms.Platform.Format()` takes `[]votes.Vote`.
- Stadt Zürich behaviour is **byte-identical** to today — same posts, same images, same dedup.
- Kanton Zürich posts to the same platforms with the full per-Fraktion breakdown, its own vote log, and its own contacts file.
- Per-jurisdiction state lives at `data/<jurisdiction>/posted_votes_<platform>.json`, with existing city logs migrated in place.

## What We're NOT Doing

- **No Nationalrat / OData adapter.** Blocked on the French `MeaningYes`/`Subject` regression and an unresolved editorial policy.
- **No Ständerat.** Name lists are only published for a subset of votes by design.
- **No other cantons.** The adapter will support them at ~zero marginal cost, but only `ZH` gets wired up and curated here.
- **No Stadt Bern.** It has vote data but no Fraktion field, so it fails the parity bar.
- **No switch of Stadt Zürich to OpenParlData.** PARIS stays canonical — it is faster and OPD merely re-serves it.
- **No contacts curation for the 180 Kantonsrat members.** That is the dominant human cost and belongs in its own task; Phase 5 ships with tagging disabled for ZH.
- **No changes to post copy, image layout, or platform credentials.**

## Implementation Approach

Strangler-style. Phase 1 introduces `pkg/votes` and a PARIS→`Vote` mapper *without* removing anything. Phase 2 migrates consumers to the neutral type, deleting the `zurichapi` imports as it goes. Phase 3 makes state per-jurisdiction. Only then (Phase 4) does a second source appear, validated against PARIS on `261` — a known-good 39/40 diff — before Phase 5 points it at `ZH`.

Phases 1–3 must produce **no behaviour change**; the existing suite plus a golden-output check is the gate.

---

## Phase 1: Neutral Domain Model + PARIS Mapper

### Overview

Add `pkg/votes` and a `zurichapi.Abstimmung` → `votes.Vote` mapper. Nothing else changes yet.

### Changes Required

**New `pkg/votes/types.go`:**

```go
type Jurisdiction struct {
    Key       string // "zurich-city", "zurich-canton"
    Name      string // "Gemeinderat Stadt Zürich"
    Seats     int    // 125, 180 — for the completeness gate
}

type Vote struct {
    SourceID     string    // dedup key: OBJGUID / external_id
    Jurisdiction string
    Date         time.Time
    Title        string
    Subtitle     string    // Traktandum-derived; may be empty
    Type         string    // may be empty
    SourceURL    string

    // Totals are first-class — never derived from MemberVotes.
    Yes, No, Abstention, Absent *int
    ChoiceA, ChoiceB, ChoiceC, ChoiceD, ChoiceE *int  // Auswahl votes
    Decision     string    // "Ja"/"Nein"; derived when the source omits it

    Affair       Affair
    MemberVotes  []MemberVote  // optional enrichment
    MeaningYes, MeaningNo string // optional, federal only
}

type MemberVote struct { Name, Party, Fraktion, Choice string }
type Affair struct { Number, Title, ID, URL string }
```

**New `pkg/votes/source.go`:**

```go
type Source interface {
    FetchRecent(limit int) ([]Vote, error)
    GroupByAffair(votes []Vote) ([][]Vote, error)
    Jurisdiction() Jurisdiction
}
```

**New `pkg/zurichapi/mapper.go`** — `ToVote(Abstimmung) votes.Vote`, mapping `SelectBestTitle(TraktandumTitel, GeschaeftTitel)` into `Title`/`Subtitle` exactly as the formatters do today, and `Stimmabgaben.Stimmabgabe` → `MemberVotes`.

**New `pkg/votes/completeness.go`** — `IsBreakdownComplete(v Vote, seats int) bool`, comparing `len(v.MemberVotes)` against `Yes+No+Abstention+Absent`.

### Success Criteria

- `go test ./...` passes unchanged.
- New unit tests: mapper round-trip on every fixture in `testfixtures`; completeness gate true for full data, false when member votes are dropped.
- `pkg/votes` imports nothing from `zurichapi`, `voteposting` or `imagegen` (enforce with a test asserting the import graph, or `go list -deps`).

---

## Phase 2: Migrate Consumers to `votes.Vote`

### Overview

Flip every consumer to the neutral type and delete the `zurichapi` imports outside the adapter. This is the largest phase and the one the existing tests must carry.

### Changes Required

- `platforms/interface.go` — `Format(votes []votes.Vote) (Content, error)`.
- `platforms/{x,bluesky,instagram}/format.go` — swap field access: `Abstimmungstitel`→`Title`, `Schlussresultat`→`Decision`, `AnzahlJa`→`Yes`, etc. **Behaviour must not change**; `SelectBestTitle` already ran in the mapper, so formatters read `Title`/`Subtitle` directly.
- `voteformat/fraktion.go` — `AggregateFraktionCounts([]votes.MemberVote)`.
- `imagegen/imagegen.go` — `GenerateCarousel([]votes.Vote)`.
- `voteposting/prepare.go` — `PrepareVoteGroups(src votes.Source, …)`; the `Schlussresultat` consistency check becomes a `Decision`-vs-counts check; replace the hardcoded `gemeinderat-zuerich.ch` URL at line 169 with `v.SourceURL`.
- `voteformat.go:248-260` — `GenerateVoteLink`/`GenerateTraktandumLink`/`GenerateGeschaeftLink` move behind the mapper, which populates `SourceURL` and `Affair.URL`.
- `testfixtures/fixtures.go` — rebuild fixtures as `votes.Vote` (42 refs; largest single edit).
- `main.go` + `cmd/*` — construct a `votes.Source` instead of `*zurichapi.Client`.

### Success Criteria

- `go test ./...` passes.
- **Golden-output check**: `cmd/post_fixture` and `cmd/generate_vote_post` produce byte-identical output to a capture taken before Phase 1. This is the real gate — capture it first.
- `grep -rl "zurichapi" --include="*.go" .` returns only `pkg/zurichapi/*` and its tests.

---

## Phase 3: Per-Jurisdiction State

### Overview

Give vote logs and config a jurisdiction dimension, keeping city dedup semantics intact.

### Changes Required

- `votelog.getLogFilePath` → `data/<jurisdiction>/posted_votes_<platform>.json`; `Load`/`New`/`NewNoOp` take a jurisdiction key.
- **Migrate existing logs**: `git mv data/posted_votes_{x,bluesky,instagram}.json data/zurich-city/`. Dedup IDs are unchanged, so no re-posting.
- `contacts` — resolve `data/<jurisdiction>/contacts.yaml`, falling back to `data/contacts.yaml` for `zurich-city`; or `git mv` it for symmetry (preferred — one rule, no fallback).
- `main.go` — loop over configured jurisdictions × platforms rather than platforms alone.
- `maxAgeDays` becomes per-jurisdiction (city keeps its current value; ZH needs headroom for the ~1.4 d median, ~4.2 d worst-case lag).
- Update `.github/workflows/*` paths that reference `data/posted_votes_*.json`.

### Success Criteria

- `go test ./...` passes; `votelog` tests cover the new path scheme.
- A dry run for `zurich-city` reports **zero** unposted votes for already-posted history — proof the migration preserved dedup.
- CI workflow commits the log to the new path.

---

## Phase 4: OpenParlData Adapter, Validated on Stadt Zürich

### Overview

Build the second `votes.Source` and prove it against PARIS on `body_key=261` before trusting it on a jurisdiction we cannot cross-check.

### Changes Required

**New `pkg/openparldata/`:**

- `client.go` — JSON over HTTPS, **`lang_format=flat` on every request**, `limit`/`offset` paging, timeouts and retry with backoff (no published rate limit — be conservative).
- `types.go` — `votingDTO`, `voteDTO`, `affairDTO`.
- `api.go` — `FetchRecent(limit)` → `GET /v1/votings/?body_key=<key>&sort_by=-date&limit=N&lang_format=flat`; then per voting, `GET /v1/votings/{id}/votes?limit=500&lang_format=flat` and `GET /v1/votings/{id}/affairs`.
- `mapper.go` — `external_id`→`SourceID`, `results_*`→totals, `decision` derived (`yes > no`) when null, `person_parliamentary_group_name_de`→`Fraktion`, `url_external_de`→`SourceURL`, `/affairs` `number`→`Affair.Number`.
- `GroupByAffair` — group on `affair_id`, mirroring `GroupAbstimmungenByGeschaeft`.

**New `cmd/compare_sources/`** — fetch the same N votes from PARIS and OpenParlData `261`, diff totals/decision/title, report mismatches. This is the phase gate and is worth keeping as a regression tool.

### Success Criteria

- Unit tests against recorded JSON fixtures (no live calls in CI).
- `cmd/compare_sources -n 40` reports **0 conflicting totals or decisions** on `261`, reproducing the 39/40 research result.
- Fraktion aggregation over an OpenParlData vote produces the same shape as the PARIS path.

---

## Phase 5: Wire Up Kanton Zürich

### Overview

Register the jurisdiction and enable posting, with the completeness gate active and tagging off.

### Changes Required

- Register `zurich-canton` (`body_key=ZH`, 180 seats, `maxAgeDays` sized for the lag).
- `data/zurich-canton/posted_votes_*.json` — start empty.
- **Completeness gate**: when `IsBreakdownComplete` is false, post totals only and log a warning; never post a partial Fraktion breakdown.
- **Degradation**: no Traktandum → subtitle empty; no `Type` → omit the vote-type line.
- Tagging disabled for `zurich-canton` (no curated contacts yet) — `contacts.yaml` for the jurisdiction is an empty list, and the tagger must handle that without erroring.
- Attribution: `"Source: OpenParlData.ch"` (CC BY 4.0) wherever post copy credits the data source.
- Extend the GitHub Action to run the new jurisdiction.

### Success Criteria

- Dry run against live `ZH` produces well-formed posts for the most recent sitting, with a correct Fraktion breakdown matching the research figures for voting 104481 (SVP 44/0/3, SP 0/35/1, …).
- Completeness gate exercised by a fixture with dropped member votes.
- First real post reviewed manually before the Action is enabled unattended.

---

## Testing Strategy

- **Phases 1–3 are behaviour-preserving.** The gate is the existing suite plus a golden-output capture from `cmd/post_fixture` and `cmd/generate_vote_post` taken *before* Phase 1. Any diff is a bug, not an improvement.
- **No live API calls in CI.** OpenParlData responses are recorded as fixtures; `cmd/compare_sources` is run manually.
- **Fraktion aggregation** gets table-driven tests over both sources, including the ~1% unmapped-Fraktion case (must not crash, must not silently drop the member from totals).
- **Completeness gate** tested both ways: complete data posts a breakdown, truncated data falls back to totals.
- `go test ./...` green at the end of every phase — each phase is independently mergeable.

## Decisions

- **Totals are first-class fields, never summed from `MemberVotes`.** Sources derive them independently and can disagree.
- **PARIS stays canonical for Stadt Zürich.** OpenParlData re-serves it, so switching adds a dependency and latency for zero data gain.
- **Kanton ZH ships before the Nationalrat**, reversing the earlier doc's ordering — federal needs a French-text workaround, a classifier robust to a 37%-null `Subject`, and an editorial policy; ZH needs none of these.
- **The completeness gate stays despite never triggering on current ZH data.** It is cheap, and Stadt Bern's `Präsidium`→`abstention` off-by-one shows the failure mode is real elsewhere.
- **Ship ZH without tagging** rather than delaying the jurisdiction on 180 rows of contact curation.

## References

- Research: [2026-08-05-openparldata-federal-kantonsrat-feasibility.md](../research/2026-08-05-openparldata-federal-kantonsrat-feasibility.md) — schema, coverage, verification results
- Prior scoping: [2026-07-06-expand-beyond-zurich.md](../research/2026-07-06-expand-beyond-zurich.md) — other cities, cantons, international
- [OpenParlData API docs](https://api.openparldata.ch/documentation) · CC BY 4.0, attribution `"Source: OpenParlData.ch"`
- Existing pattern for a shared formatter extraction: [2026-04-12-fraktion-vote-breakdown.md](2026-04-12-fraktion-vote-breakdown.md)
