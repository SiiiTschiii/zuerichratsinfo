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

**Kantonsrat votes post to the existing `@zuerichratsinfo` / `@zueriratsinfo` accounts**, alongside city votes. This requires a new config concept — a *channel*, being one set of platform credentials serving one or more jurisdictions — and a fix to the post budget, which is currently per-platform and would let two jurisdictions each spend the full allowance on the same account.

The Nationalrat is explicitly **out of scope here** — it is blocked on an upstream data-quality bug (see [research §2.2](../research/2026-08-05-openparldata-federal-kantonsrat-feasibility.md)) and needs its own editorial policy. The refactor in Phases 1–3 is what it will build on. The channel model is designed to support any jurisdiction→account mapping so that federal, or other cantons, can take separate accounts later without reworking config; **which accounts they use is deliberately left undecided** (see Open Questions).

## Current State Analysis

- `zurichapi.Abstimmung` is the de-facto domain type, referenced in **28 files**. Concentrations: `testfixtures/fixtures.go` (42 refs), `platforms/bluesky/format_test.go` (18), `platforms/x/format_test.go` (15), `voteposting_test.go` (13), `imagegen.go` (8), `prepare.go` (7).
- `PrepareVoteGroups()` ([prepare.go:28](pkg/voteposting/prepare.go#L28)) takes a concrete `*zurichapi.Client`, calls `FetchRecentAbstimmungen` → `filterUnpostedVotes` → `GroupAbstimmungenByGeschaeft`, then validates `Schlussresultat` against counts.
- `platforms.Platform.Format()` ([interface.go:14](pkg/voteposting/platforms/interface.go#L14)) takes `[]zurichapi.Abstimmung` — the type crosses the platform boundary.
- `voteformat` is already largely source-neutral: `VoteCounts` ([voteformat.go:139](pkg/voteposting/voteformat/voteformat.go#L139)) is a standalone struct of `*int`s. Only `AggregateFraktionCounts` ([fraktion.go:18](pkg/voteposting/voteformat/fraktion.go#L18)) takes `[]zurichapi.Stimmabgabe`.
- `votelog` keys purely on a vote ID string, but paths are hardcoded: `data/posted_votes_%s.json` ([votelog.go:155](pkg/votelog/votelog.go#L155)).
- Hardcoded Zurich-city URLs exist in `prepare.go:169` (`gemeinderat-zuerich.ch/abstimmungen/detail.php`) and `voteformat.go:248-260` (`GenerateVoteLink`/`GenerateTraktandumLink`/`GenerateGeschaeftLink`).
- `data/contacts.yaml` is 132 curated Stadt-Zürich entries, keyed by politician `name`.
- All packages currently pass `go test ./...`.

**Account / infrastructure state:**

- Platform credentials are **flat singletons** in [bot.yml](.github/workflows/bot.yml): `X_API_KEY`, `BLUESKY_HANDLE`, `IG_USER_ID`, … There is no notion of *which* account, so nothing today can express "post this jurisdiction to that account".
- **Post budgets are per-platform**: `X_MAX_POSTS_PER_RUN`, `BLUESKY_MAX_POSTS_PER_RUN`, `IG_MAX_POSTS_PER_RUN`, consumed by `platform.MaxPostsPerRun()` in `PostToPlatform` ([prepare.go:158](pkg/voteposting/prepare.go#L158)).
- The bot runs **hourly** (`cron: "36 * * * *"`), and the workflow commits the vote logs back to `main` by explicit path.
- **Instagram image hosting is this repo** (`IG_REPO_OWNER: SiiiTschiii`, `IG_REPO_NAME: zuerichratsinfo`). The repository is infrastructure, not just source.
- Live accounts: [@zueriratsinfo](https://www.instagram.com/zueriratsinfo) (Instagram), [@zuerichratsinfo](https://x.com/zuerichratsinfo) (X), [@zuerichratsinfo.bsky.social](https://bsky.app/profile/zuerichratsinfo.bsky.social) (Bluesky). Note the Instagram handle differs by one character. X Premium is a recurring cost, currently funded via Buy Me a Coffee.
- `README.md` describes the project as covering the *Zurich City Council*; it becomes inaccurate the moment Kantonsrat votes ship.

### Key Discoveries

- **Totals and member votes must be independent fields.** PARIS delivers them together; OpenParlData derives them from separate steps and can in principle disagree. Formatters must read totals directly, not sum `MemberVotes`.
- **Kanton ZH data is clean**: 40/40 sampled votings reconcile exactly at 180/180 members, ~1% unmapped Fraktion. The completeness gate is insurance, not the common path.
- **`lang_format=flat` is mandatory** on every OpenParlData call, or `*_de` fields silently return `null`.
- **`affair_number` requires a second call** to `/v1/votings/{id}/affairs` — it is null inline on the voting.
- **ZH has no Traktandum and no vote type.** `SelectBestTitle(traktandumTitel, geschaeftTitel)` and `Abstimmungstyp` handling must degrade rather than branch per source.
- **`maxAgeDays` is a backstop against re-posting history, not a freshness knob — and it is load-bearing for a new jurisdiction.** Dedup works by fetching the most recent N votes and subtracting whatever the vote log records. The logs do not go back to the beginning: `x` starts 2025-11-06 (637 entries), `bluesky` 2026-03-11 (282), `instagram` 2026-04-20 (126). Any vote older than a platform's log start is therefore *indistinguishable from unposted*, so when PARIS re-indexes an old vote and it re-enters the fetch window, only the 90-day age guard stops it being posted again.

  For **Kanton ZH the log starts empty**, which means *every one of the 2,626 historical votings* looks unposted. `maxAgeDays` is the only thing standing between first run and a mass-post. It must be set before the jurisdiction is enabled, not after — and the first run should be dry, then seeded. This is a bigger deal than the ~1.4 d harvest lag that originally motivated making the value per-jurisdiction (the lag only sets the *lower* bound; it must exceed ~4.2 d worst case).
- **The post budget is enforced by a counter on the platform *instance*.** `postsThisRun` increments in `Post()` and gates `shouldContinue` ([x/platform.go:114-115](pkg/voteposting/platforms/x/platform.go#L114)); `MaxPostsPerRun()` is only consulted directly on the dry-run path ([prepare.go:158](pkg/voteposting/prepare.go#L158)). So the budget is already per-instance, and the fix is **construction, not config**: build one platform instance per *channel* and reuse it across all that channel's jurisdictions, and the existing counter shares the allowance correctly. Build one per jurisdiction — the obvious shape for a per-jurisdiction loop — and the counter resets, silently doubling hourly volume on `@zuerichratsinfo`. This is the single easiest thing to get wrong in Phase 3.

## Desired End State

- A `pkg/votes` package defines `Vote` / `MemberVote` / `Jurisdiction` with no import of any source package.
- `pkg/zurichapi` and a new `pkg/openparldata` both implement a `votes.Source` interface; neither is referenced from `voteposting`, `imagegen` or the platforms.
- `platforms.Platform.Format()` takes `[]votes.Vote`.
- Stadt Zürich behaviour is **byte-identical** to today — same posts, same images, same dedup.
- Kanton Zürich posts to the **same accounts** as Stadt Zürich, with the full per-Fraktion breakdown, its own vote log, and its own contacts file.
- Per-jurisdiction state lives at `data/<jurisdiction>/posted_votes_<platform>.json`, with existing city logs migrated in place.
- A **channel** maps 1..N jurisdictions to one set of platform credentials. Exactly one channel (`zurich`) is configured, serving both jurisdictions — but the model supports N channels, so a future federal channel is config, not a refactor.
- The **post budget is per channel, not per platform**, and jurisdictions sharing a channel share the allowance under a defined order.
- Posts make clear which body voted, so a reader cannot mistake a Kantonsrat vote for a Gemeinderat one.

## What We're NOT Doing

- **No Nationalrat / OData adapter.** Blocked on the French `MeaningYes`/`Subject` regression and an unresolved editorial policy.
- **No Ständerat.** Name lists are only published for a subset of votes by design.
- **No other cantons.** The adapter will support them at ~zero marginal cost, but only `ZH` gets wired up and curated here.
- **No Stadt Bern.** It has vote data but no Fraktion field, so it fails the parity bar.
- **No switch of Stadt Zürich to OpenParlData.** PARIS stays canonical — it is faster and OPD merely re-serves it.
- **No contacts curation for the 180 Kantonsrat members.** That is the dominant human cost and belongs in its own task; Phase 5 ships with tagging disabled for ZH.
- **No new social accounts.** Kantonsrat uses the existing ones; no second credential set is provisioned here.
- **No federal-only fields in the domain model.** `MeaningYes`/`MeaningNo` are added when the Nationalrat adapter is built. Adding them now would mean shipping fields no source populates, no formatter reads, and no test covers — and the research shows their federal content is currently French anyway, so their eventual shape may not even be a plain string pair.
- **No repo split and no repo rename.** Splitting would duplicate the whole imagegen/platforms/voteposting stack, and Instagram hosting is pinned to this repository. Renaming would break the Go module path, clone URLs, badges and `IG_REPO_NAME` for no gain.
- **No decision on which accounts serve federal or other cantons.** The channel model supports separate accounts; the policy is deliberately deferred (see Open Questions).
- **No changes to image layout or platform credentials.** Post copy changes only to the minimum needed to label the body (see Phase 5).

## Prerequisite: land [PR #43](https://github.com/SiiiTschiii/zuerichratsinfo/pull/43) first

[#43](https://github.com/SiiiTschiii/zuerichratsinfo/pull/43) moves vote-log persistence off `main` onto an isolated `state-log` branch, because the new required-PR ruleset on `main` blocks the bot's push. It should merge **before Phase 3**, for two reasons beyond its own merit:

1. **It rewrites the exact workflow block Phase 3 edits.** Today's `git add data/posted_votes_*.json` + `git pull --rebase` + `git push` is replaced by `STATE_BRANCH` / `STATE_FILES` and a `git worktree`. Phase 3 changing those paths first would guarantee a conflict, and #43 is the smaller, already-reviewed change.
2. **Its copy step flattens paths.** `cp $STATE_FILES "$worktree/data/"` collapses any directory structure, so once logs live at `data/<jurisdiction>/posted_votes_<platform>.json`, two jurisdictions' logs land on top of each other in the state branch. Phase 3 must therefore update `STATE_FILES` **and** switch the copy to something path-preserving (`rsync -R`, or `cp --parents`, or per-file `install -D`). Sequencing them the other way risks silently merging vote logs — which would look like votes being re-posted.

Note the breakage #43 fixes is **latent, not active**: bot runs are currently succeeding, because the Gemeinderat has been in summer recess since 2026-07-08 and `git diff --staged --quiet ||` short-circuits when there is nothing to commit. The failure fires on the next real post — which, without #43, would be the first Kantonsrat post.

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

## Phase 3: Per-Jurisdiction State and Channel Configuration

### Overview

Give vote logs and config a jurisdiction dimension, introduce the channel concept, and move the post budget from platform to channel. Keeps city dedup semantics and city posting volume exactly as today.

### Changes Required

**State:**

- `votelog.getLogFilePath` → `data/<jurisdiction>/posted_votes_<platform>.json`; `Load`/`New`/`NewNoOp` take a jurisdiction key.
- **Migrate existing logs**: `git mv data/posted_votes_{x,bluesky,instagram}.json data/zurich-city/`. Dedup IDs are unchanged, so no re-posting.
- `contacts` — `git mv data/contacts.yaml data/zurich-city/contacts.yaml` and resolve `data/<jurisdiction>/contacts.yaml`. One rule, no fallback.
- `maxAgeDays` becomes per-jurisdiction. City keeps **90**. ZH needs a value above the ~4.2 d worst-case harvest lag but small enough to bound the first-run backlog — see the seeding step in Phase 5.
- Update `.github/workflows/bot.yml` (post-#43): widen `STATE_FILES` to `data/*/posted_votes_*.json` so it needs no edit per jurisdiction, and **replace the flattening `cp $STATE_FILES "$worktree/data/"` with a path-preserving copy** — otherwise both jurisdictions' logs collide in the state branch. Same for the restore step.

**Channels:**

```go
type Channel struct {
    Key           string   // "zurich"
    Jurisdictions []string // ["zurich-city", "zurich-canton"]
    // platform credentials + limits resolved from env, prefixed by channel key
}
```

- Credential lookup becomes channel-aware: `<CHANNEL>_X_API_KEY`, falling back to the current unprefixed names when the channel is `zurich`. That keeps every existing secret working untouched while making a second channel purely additive.
- **One platform instance per channel, reused across its jurisdictions.** This is the whole budget fix — the existing `postsThisRun` counter then enforces the allowance across both jurisdictions rather than resetting per jurisdiction. `MaxPostsPerRun` stays where it is; do **not** construct platforms inside the jurisdiction loop.
- `PostToPlatform` currently takes one jurisdiction's groups; it needs to accept groups drawn from all of the channel's jurisdictions, with the vote log resolved per group's jurisdiction (it marks votes as posted, so it can no longer assume a single log).
- **Selection order when jurisdictions share a budget**: merge candidate groups across the channel's jurisdictions and post **oldest vote date first**. Keeps today's "work through the backlog in order" behaviour and stops a busy Kantonsrat sitting from starving city votes. Ties broken by config order.
- `main.go` — loop over channels → platforms, constructing each platform once, then feeding it the merged group list.

### Success Criteria

- `go test ./...` passes; `votelog` tests cover the new path scheme.
- A dry run for `zurich-city` reports **zero** unposted votes for already-posted history — proof the migration preserved dedup.
- **Volume is unchanged for the city**: with only `zurich-city` active, a dry run posts exactly as many groups as before the change.
- A test with two jurisdictions in one channel and a budget of N posts **N groups total**, not 2N, ordered oldest-first.
- CI workflow round-trips logs to the `state-log` branch at the new nested paths, with **no collision** between jurisdictions — verified by a manual run with two jurisdictions configured.

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

- Register `zurich-canton` (`body_key=ZH`, 180 seats) and add it to the `zurich` channel alongside `zurich-city`.
- **Seed the vote log before the first live run.** The log starts empty and ZH has 2,626 historical votings, all of which read as unposted. Set `maxAgeDays` for ZH first, then do a dry run to confirm the candidate set is the handful of votes you actually intend to post, then mark everything older as posted (a small `cmd/seed_votelog`, or commit a pre-populated log). Do not let the first real run discover this.
- **Make the jurisdiction evident in every post, text and image.** With both bodies on one account, a reader must not mistake a Kantonsrat vote for a Gemeinderat one. The *requirement* is fixed; the *design* is explicitly open to iterate — candidates include a text prefix or emoji in the root post, a body name line, a distinct image accent colour, or a header band in the image. `imagegen.SelectColor` currently keys only on `GeschaeftGrNr` ([imagegen.go:55](pkg/imagegen/imagegen.go#L55)), so it needs a jurisdiction input for any colour-based option.
- **Local dry-run comparison is the design loop**, not a one-off check: render both jurisdictions side by side via `cmd/generate_vote_post` and `cmd/generate_vote_image`, review, adjust, repeat. Ensure both commands take a jurisdiction argument so this is a single command per side.
- Update `README.md` — it currently describes a *Zurich City Council* bot throughout: the one-line description, the "What It Does" section, the `pkg/zurichapi` data-source paragraph, and the platform table (whose per-platform counts are Stadt-Zürich contacts only). It should state both jurisdictions covered, and which data source backs each.
- Update the **account bios/descriptions** on X, Bluesky and Instagram once the first Kantonsrat posts are live, so the profile matches what the feed now contains.
- Consider whether `PRIVACY.md` needs to mention the second data source (OpenParlData) — it currently describes PARIS only.
- **Completeness gate**: when `IsBreakdownComplete` is false, post totals only and log a warning; never post a partial Fraktion breakdown.
- **Degradation**: no Traktandum → subtitle empty; no `Type` → omit the vote-type line.
- Tagging disabled for `zurich-canton` (no curated contacts yet) — `contacts.yaml` for the jurisdiction is an empty list, and the tagger must handle that without erroring.
- Attribution: `"Source: OpenParlData.ch"` (CC BY 4.0) wherever post copy credits the data source.
- Extend the GitHub Action to run the new jurisdiction.

### Success Criteria

- Dry run against live `ZH` produces well-formed posts for the most recent sitting, with a correct Fraktion breakdown matching the research figures for voting 104481 (SVP 44/0/3, SP 0/35/1, …).
- Completeness gate exercised by a fixture with dropped member votes.
- A Kantonsrat post and a Gemeinderat post rendered side by side locally are **unambiguously distinguishable** without reading the vote title — reviewed by the project owner, iterating on the design until it reads right. This gates enabling the jurisdiction, not merging the phase.
- Combined hourly volume on the shared account is checked against a busy week (a Monday Kantonsrat sitting plus a Gemeinderat sitting) before the Action runs unattended.
- First real post reviewed manually before the Action is enabled unattended.

---

## Testing Strategy

- **Phases 1–3 are behaviour-preserving.** The gate is the existing suite plus a golden-output capture from `cmd/post_fixture` and `cmd/generate_vote_post` taken *before* Phase 1. Any diff is a bug, not an improvement.
- **No live API calls in CI.** OpenParlData responses are recorded as fixtures; `cmd/compare_sources` is run manually.
- **Fraktion aggregation** gets table-driven tests over both sources, including the ~1% unmapped-Fraktion case (must not crash, must not silently drop the member from totals).
- **Shared-budget regression test.** Two jurisdictions in one channel with a budget of N must post N groups total, oldest-first — not N per jurisdiction. This guards the one mistake that would visibly spam the live account, so it belongs in CI rather than in a manual check.
- **Completeness gate** tested both ways: complete data posts a breakdown, truncated data falls back to totals.
- `go test ./...` green at the end of every phase — each phase is independently mergeable.

## Decisions

- **Totals are first-class fields, never summed from `MemberVotes`.** Sources derive them independently and can disagree.
- **PARIS stays canonical for Stadt Zürich.** OpenParlData re-serves it, so switching adds a dependency and latency for zero data gain.
- **Kanton ZH ships before the Nationalrat**, reversing the earlier doc's ordering — federal needs a French-text workaround, a classifier robust to a 37%-null `Subject`, and an editorial policy; ZH needs none of these.
- **The completeness gate stays despite never triggering on current ZH data.** It is cheap, and Stadt Bern's `Präsidium`→`abstention` off-by-one shows the failure mode is real elsewhere.
- **Ship ZH without tagging** rather than delaying the jurisdiction on 180 rows of contact curation.
- **Defer federal-only model fields.** Build for the two jurisdictions actually shipping; add `MeaningYes`/`MeaningNo` with the adapter that populates them.
- **Seed the ZH vote log before first run.** With an empty log, `maxAgeDays` is the only guard against posting 2,626 historical votings — too much to rest on one config value.
- **The jurisdiction-labelling design is deliberately unspecified.** The requirement is fixed; the visual treatment is settled by local dry-run iteration, not decided up front on paper.
- **Kantonsrat posts to the existing accounts.** "Zürich Ratsinfo" is geographically accurate for the Kantonsrat, the audiences overlap almost completely, volume is modest (8–13 votings per Monday sitting, fewer after grouping), and it costs no new X Premium subscription and no new credential set — while growing the existing following rather than starting from zero.
- **One platform instance per channel.** Rate limits and audience fatigue are properties of an account, not of a jurisdiction. The existing `postsThisRun` counter already enforces this correctly *provided* the instance is shared — so this is a construction rule, not a new config surface.
- **Shared budgets are spent oldest-first**, preserving today's backlog behaviour and preventing one jurisdiction from starving the other.
- **Monorepo, name unchanged.** One pipeline; splitting duplicates imagegen/platforms/voteposting, and Instagram hosting is pinned to this repo via `IG_REPO_OWNER`/`IG_REPO_NAME`. A rename would break the Go module path, clone URLs and badges for no gain.
- **Channel config is N-channel from day one, even though only one is configured.** The cost is a map lookup; the alternative is reworking credential resolution when federal ships.

## Open Questions

Deliberately **not** decided here — the channel model supports any mapping, so these stay cheap to answer later:

- **Which accounts serve the Nationalrat.** Arguments for separate accounts: a Zürich-named account posting Bundesbern is misleading, the audience is national, session volume arrives in 3-week bursts of 30–45 postable votes then goes silent for two months, and Liip's Swiss Parliament Bot already occupies that space. Arguments for reuse: no new cost, no cold start. Revisit when the federal blocker is resolved.
- **Whether other cantons launch at all, and under whose accounts.** The adapter scales to 21 further cantons at ~zero marginal *engineering* cost, but each needs ~180 curated contacts, an account, and ongoing moderation. That is a "someone must want to run it" problem. A plausible answer is to keep the codebase multi-tenant and let a local volunteer configure a fork, rather than launching proactively.
- **Stadt Bern**, the only other city with vote data — currently fails the parity bar (no Fraktion field, party only).
- **Other countries** — a separate product (accounts, language, contacts, competitors), not a config change.
- **Funding**, if account count grows. X Premium is per-account and currently covered by Buy Me a Coffee for one account.

## References

- Research: [2026-08-05-openparldata-federal-kantonsrat-feasibility.md](../research/2026-08-05-openparldata-federal-kantonsrat-feasibility.md) — schema, coverage, verification results
- Prior scoping: [2026-07-06-expand-beyond-zurich.md](../research/2026-07-06-expand-beyond-zurich.md) — other cities, cantons, international
- [OpenParlData API docs](https://api.openparldata.ch/documentation) · CC BY 4.0, attribution `"Source: OpenParlData.ch"`
- Existing pattern for a shared formatter extraction: [2026-04-12-fraktion-vote-breakdown.md](2026-04-12-fraktion-vote-breakdown.md)
