---
date: 2026-04-05
author: copilot
topic: "E2E testing: function-variable mocking, SKIP_VOTE_LOG, shared fixtures, post_fixture command"
tags: [plan, testing, xapi, bskyapi, voteposting, e2e]
status: done
research: thoughts/shared/research/2026-04-05-dry-post-e2e-testing.md
---

# E2E Testing Infrastructure — Implementation Plan

## Overview

Add testable posting infrastructure so the full `Format → Post → API call` chain can be exercised in unit tests (with mocked HTTP) and in manual e2e runs (with test accounts). Currently, unit tests only cover formatting; the `Post` method and API call construction are untested due to hardcoded HTTP dependencies.

## Current State Analysis

- **X Platform**: `XPlatform.Post` (`pkg/voteposting/platforms/x/platform.go:71-102`) calls `xapi.PostTweet` directly — a package-level function with a hardcoded URL and inline `http.Client` (`pkg/xapi/client.go:30-78`). Cannot be mocked.
- **Bluesky Platform**: `BlueskyPlatform.Post` (`pkg/voteposting/platforms/bluesky/platform.go:93-137`) calls `bskyapi.CreateRecord` directly — same pattern. Also calls `bskyapi.CreateSession` (lazy auth) and `bskyapi.ResolveHandle` (DID resolution).
- **Existing dry-run**: `PostToPlatform` (`pkg/voteposting/prepare.go:76`) accepts `dryRun bool` but skips `Post` entirely — doesn't exercise the posting path.
- **Existing tests**: `client_test.go` duplicates JSON marshalling logic inline because `PostTweet` can't be called against a test server (`pkg/xapi/client_test.go:26-28`). No tests exist for `BlueskyPlatform.Post` or `bskyapi` functions.
- **Vote log**: `main.go` always loads the real vote log, so `PrepareVoteGroups` filters out already-posted votes — no way to re-post for testing.
- **Fixtures**: Vote data is duplicated across `bluesky/format_test.go` and `x/format_test.go` with no shared fixture library.

## Desired End State

1. `XPlatform.Post` and `BlueskyPlatform.Post` are testable via function-variable injection — unit tests capture JSON payloads and verify thread chaining without HTTP calls.
2. A `SKIP_VOTE_LOG=true` env var lets `main.go` treat all fetched votes as unposted (for manual e2e with test accounts).
3. A shared `pkg/voteposting/testfixtures/` package provides ~10 edge-case vote fixtures for both unit tests and manual e2e posting.
4. A `cmd/post_fixture/main.go` command posts fixtures through the real platform `Post` path to test accounts.

### Key Discoveries:

- `XPlatform` struct (`pkg/voteposting/platforms/x/platform.go:33-41`) has no function fields — adding `postTweetFunc` is straightforward.
- `BlueskyPlatform` struct (`pkg/voteposting/platforms/bluesky/platform.go:40-48`) needs three injectable functions: `createRecordFunc`, `createSessionFunc`, and `resolveHandleFunc`.
- `votelog.NewEmpty()` (`pkg/votelog/votelog.go:121-128`) already exists for tests — the no-op VoteLog can follow the same pattern.
- `sampleVote()` in `bluesky/format_test.go:19-35` fills all GUID fields and is the best template for shared fixtures.
- The `Abstimmung` struct (`pkg/zurichapi/types.go:88-122`) includes Auswahl fields (`AnzahlA`–`AnzahlE`) which need fixture coverage.

## What We're NOT Doing

- No env var for dry-post mode — function variables are wired internally in tests and `cmd/post_fixture` only.
- No automatic post cleanup on test accounts — posts are left as-is.
- No CI/CD integration for e2e tests — manual only.
- No refactoring of `xapi.PostTweet` or `bskyapi.CreateRecord` internals (the functions themselves stay unchanged).
- No `VOTE_LINK`/`VOTE_GUID` env var for fetching specific votes — `SKIP_VOTE_LOG` + `MAX_VOTES_TO_CHECK` is sufficient.

## Implementation Approach

Use function-variable indirection (research option A1): each platform struct gets a function field defaulting to the real API function. Tests and `cmd/post_fixture` can override it with a mock/dry-post function. This is the least invasive approach — only `platform.go` files change, API packages stay untouched.

---

## Phase 1: Shared Test Fixtures

### Overview

Create a `pkg/voteposting/testfixtures/` package with ~10 hardcoded `[]zurichapi.Abstimmung` fixtures covering known edge cases. These are imported by both `_test.go` files (unit tests in Phases 2–3) and `cmd/post_fixture/main.go` (manual e2e in Phase 5). Adopt the `sampleVote()` pattern from `bluesky/format_test.go` (fills all GUID fields).

This phase is first because Phases 2–3 use fixtures in their unit tests.

### Changes Required:

#### 1. Fixture Package

**File**: `pkg/voteposting/testfixtures/fixtures.go` (new)
**Changes**:

Helper function:

```go
func vote(guid, title, grNr, result string, ja, nein, enth, abw int) zurichapi.Abstimmung
```

Named fixture functions (each returns `[]zurichapi.Abstimmung`):

| Function                   | Source                   | Description                                             |
| -------------------------- | ------------------------ | ------------------------------------------------------- |
| `SingleVoteAngenommen()`   | `bluesky/format_test.go` | Postulat, accepted (90/30/0/5)                          |
| `SingleVoteAbgelehnt()`    | `bluesky/format_test.go` | Antrag, rejected (20/95/5/5)                            |
| `LongTitleTruncation()`    | `bluesky/format_test.go` | ~300-char title triggering `…`                          |
| `MultiVoteGroup()`         | `bluesky/format_test.go` | 2 votes: Einleitungsartikel + Schlussabstimmung         |
| `GenericAntragFallback()`  | `bluesky/format_test.go` | TraktandumTitel `"Antrag 1."` → GeschaeftTitel fallback |
| `TenVoteStressTest()`      | `bluesky/format_test.go` | 10 votes forcing multiple replies                       |
| `VoteWithMentions()`       | `bluesky/format_test.go` | Postulat with @mention-triggering name                  |
| `AuswahlVote()`            | `x/format_test.go`       | A/B/C counts, no ✅/❌ prefix                           |
| `MixedMultiVote()`         | `x/format_test.go`       | One Ja/Nein + one Auswahl in same group                 |
| `PostulatWithGrNrPrefix()` | `x/format_test.go`       | `"2025/100 Postulat von ..."` — tests GrNr stripping    |

`AllFixtures()` returns `map[string][]zurichapi.Abstimmung` keyed by kebab-case name.

Each fixture fills all GUID fields (`OBJGUID`, `SitzungGuid`, `TraktandumGuid`, `GeschaeftGuid`) with unique deterministic values so they work for both formatting and posting.

### Success Criteria:

#### Automated Verification:

- [x] `go build ./pkg/voteposting/testfixtures/...` compiles
- [x] `go vet ./...` passes
- [x] Existing tests still pass: `go test ./...`

#### Manual Verification:

None — fixtures are pure data. Correctness is validated when they are used in unit tests (Phases 2–3) and in `cmd/post_fixture` (Phase 5).

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 2: Function Variable Mocking for X Platform

### Overview

Add a `postTweetFunc` field to `XPlatform` so the `Post` method can be tested with a mock that captures payloads instead of making HTTP calls. Add unit tests that verify thread chaining and JSON payload structure through the real `Post` code path. Tests use shared fixtures from Phase 1.

### Changes Required:

#### 1. X Platform — Function Variable

**File**: `pkg/voteposting/platforms/x/platform.go`
**Changes**:

Add a `PostTweetFunc` type alias and a field to `XPlatform`:

```go
// PostTweetFunc is the signature for posting a tweet. Defaults to xapi.PostTweet.
type PostTweetFunc func(apiKey, apiSecret, accessToken, accessSecret, message, inReplyToTweetID string) (string, error)

type XPlatform struct {
    apiKey         string
    apiSecret      string
    accessToken    string
    accessSecret   string
    contactMapper  *contacts.Mapper
    postsThisRun   int
    maxPostsPerRun int
    postTweetFunc  PostTweetFunc // injectable for testing
}
```

Update `NewXPlatform` to set the default:

```go
func NewXPlatform(...) *XPlatform {
    return &XPlatform{
        ...
        postTweetFunc: xapi.PostTweet,
    }
}
```

Replace the two `xapi.PostTweet(...)` calls in `Post` with `p.postTweetFunc(...)`.

#### 2. X Platform — Unit Tests for Post Method

**File**: `pkg/voteposting/platforms/x/platform_test.go` (new)
**Changes**:

Create tests that use a mock `postTweetFunc` to capture payloads:

- `TestPost_SingleTweet` — one-post thread, verify no reply field
- `TestPost_ThreadChaining` — multi-post thread, verify `in_reply_to_tweet_id` chains correctly (each reply references the previous tweet ID)
- `TestPost_EmptyThread` — error case
- `TestPost_PostLimitReached` — verify `shouldContinue=false` after reaching `maxPostsPerRun`

The mock function should:

- Record each call's `message` and `inReplyToTweetID` arguments
- Return sequential fake tweet IDs (`"fake-tweet-1"`, `"fake-tweet-2"`, ...)

### Success Criteria:

#### Automated Verification:

- [x] `go test ./pkg/voteposting/platforms/x/...` passes with new tests
- [x] `go vet ./...` passes
- [x] `go build ./...` passes
- [x] Existing tests still pass: `go test ./...`

#### Manual Verification:

- [x] Run `go test ./pkg/voteposting/platforms/x/... -v -run TestPost` and verify the test output shows root tweet with empty `inReplyTo` and replies with chained tweet IDs
- [x] Run `go test ./pkg/voteposting/platforms/x/... -count=1` twice to confirm no flaky tests

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 3: Function Variable Mocking for Bluesky Platform

### Overview

Same pattern as Phase 2 but for Bluesky. Three functions need injection: `CreateRecord`, `CreateSession`, and `ResolveHandle`. The `Post` method, `ensureSession`, and `resolveMentionFacets` all make HTTP calls that need to be mockable. Tests use shared fixtures from Phase 1.

### Changes Required:

#### 1. Bluesky Platform — Function Variables

**File**: `pkg/voteposting/platforms/bluesky/platform.go`
**Changes**:

Add function type aliases and fields to `BlueskyPlatform`:

```go
type CreateRecordFunc func(session *bskyapi.Session, text string, facets []bskyapi.Facet, replyTo *bskyapi.ReplyRef) (*bskyapi.PostRef, error)
type CreateSessionFunc func(handle, password string) (*bskyapi.Session, error)
type ResolveHandleFunc func(handle string) (string, error)

type BlueskyPlatform struct {
    handle            string
    password          string
    session           *bskyapi.Session
    contactMapper     *contacts.Mapper
    postsThisRun      int
    maxPostsPerRun    int
    didCache          map[string]string
    createRecordFunc  CreateRecordFunc  // injectable for testing
    createSessionFunc CreateSessionFunc // injectable for testing
    resolveHandleFunc ResolveHandleFunc // injectable for testing
}
```

Update `NewBlueskyPlatform` to set defaults:

```go
createRecordFunc:  bskyapi.CreateRecord,
createSessionFunc: bskyapi.CreateSession,
resolveHandleFunc: bskyapi.ResolveHandle,
```

Replace direct calls:

- `bskyapi.CreateRecord(...)` → `p.createRecordFunc(...)` in `Post`
- `bskyapi.CreateSession(...)` → `p.createSessionFunc(...)` in `ensureSession`
- `bskyapi.ResolveHandle(...)` → `p.resolveHandleFunc(...)` in `resolveHandleCached`

#### 2. Bluesky Platform — Unit Tests for Post Method

**File**: `pkg/voteposting/platforms/bluesky/platform_test.go` (new)
**Changes**:

- `TestPost_SinglePost` — one-post thread, verify `replyTo` is nil for root
- `TestPost_ThreadChaining` — multi-post thread, verify `replyRef.Root` always points to root and `replyRef.Parent` chains correctly
- `TestPost_WithMentionFacets` — verify facets are passed through to `CreateRecord`
- `TestPost_SessionLazyAuth` — verify `ensureSession` calls `createSessionFunc` on first post only
- `TestPost_EmptyThread` — error case
- `TestPost_PostLimitReached` — verify `shouldContinue=false`

The mock `createRecordFunc` should return sequential fake `PostRef` values:

```go
&bskyapi.PostRef{URI: "at://did:fake/app.bsky.feed.post/1", CID: "cid-fake-1"}
```

The mock `createSessionFunc` should return a dummy session:

```go
&bskyapi.Session{DID: "did:plc:fake", Handle: "test.bsky.social", AccessJwt: "fake-jwt", ServiceEndpoint: "https://fake.bsky.social"}
```

The mock `resolveHandleFunc` should return a synthetic DID: `"did:plc:resolved-" + handle`.

### Success Criteria:

#### Automated Verification:

- [x] `go test ./pkg/voteposting/platforms/bluesky/...` passes with new tests
- [x] `go vet ./...` passes
- [x] `go build ./...` passes
- [x] Existing tests still pass: `go test ./...`

#### Manual Verification:

- [x] Run `go test ./pkg/voteposting/platforms/bluesky/... -v -run TestPost` and verify the test output shows root with nil replyTo, replies with correct root/parent refs, and facets passed through
- [x] Run `go test ./pkg/voteposting/platforms/bluesky/... -count=1` twice to confirm no flaky tests

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 4: SKIP_VOTE_LOG Env Var

### Overview

Add a no-op `VoteLog` implementation and wire `SKIP_VOTE_LOG=true` in `main.go` so all fetched votes are treated as unposted and no vote log is written. This enables manual e2e testing with test account credentials against real recent votes.

### Changes Required:

#### 1. No-Op VoteLog

**File**: `pkg/votelog/votelog.go`
**Changes**:

Add a `NewNoOp` constructor that returns a `*VoteLog` where:

- `IsPosted()` always returns `false`
- `MarkAsPosted()` and `Save()` are no-ops

Implementation: use a flag field `noOp bool` on the existing `VoteLog` struct. When true, `IsPosted` returns false, `MarkAsPosted` does nothing, and `Save` returns nil. This avoids a new interface layer.

```go
// NewNoOp creates a no-op vote log that treats all votes as unposted
// and discards all mark/save operations. Used for manual e2e testing.
func NewNoOp(platform Platform) *VoteLog {
    return &VoteLog{
        Platform: platform,
        Votes:    []VoteEntry{},
        index:    make(map[string]VoteEntry),
        noOp:     true,
    }
}
```

Guard the existing methods:

```go
func (l *VoteLog) IsPosted(voteID string) bool {
    if l.noOp {
        return false
    }
    _, exists := l.index[voteID]
    return exists
}

func (l *VoteLog) MarkAsPosted(voteID string) {
    if l.noOp {
        return
    }
    // ... existing logic
}

func (l *VoteLog) Save() error {
    if l.noOp {
        return nil
    }
    // ... existing logic
}
```

#### 2. Wire in main.go

**File**: `main.go`
**Changes**:

Read `SKIP_VOTE_LOG` env var near the top configuration section. When true, use `votelog.NewNoOp(platform)` instead of `votelog.Load(platform)` for both X and Bluesky blocks.

```go
skipVoteLog := os.Getenv("SKIP_VOTE_LOG") == "true"
```

In the X block (~line 78):

```go
var voteLog *votelog.VoteLog
if skipVoteLog {
    voteLog = votelog.NewNoOp(votelog.PlatformX)
    fmt.Println("⚠️  SKIP_VOTE_LOG=true — treating all votes as unposted, not saving vote log")
} else {
    voteLog, err = votelog.Load(votelog.PlatformX)
    if err != nil {
        log.Fatalf("Error loading X vote log: %v", err)
    }
    fmt.Printf("Loaded X vote log: %d votes already posted\n", voteLog.Count())
}
```

Same pattern for the Bluesky block (~line 112).

### Success Criteria:

#### Automated Verification:

- [x] `go test ./pkg/votelog/...` passes (add a test for `NewNoOp` behavior)
- [x] `go build ./...` passes
- [x] `go vet ./...` passes
- [x] Existing tests still pass: `go test ./...`

#### Manual Verification:

- [x] `SKIP_VOTE_LOG=true go run main.go` (with no platform credentials) prints the warning and attempts to fetch votes

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 5: Fixture E2E Command

### Overview

Create `cmd/post_fixture/main.go` — a thin command that imports shared fixtures and posts them through real platform `Post` paths. It bypasses `PrepareVoteGroups` and the vote log. Used for manual visual verification on test accounts.

### Changes Required:

#### 1. Fixture Command

**File**: `cmd/post_fixture/main.go` (new)
**Changes**:

CLI flags:

- `--fixture=<name>` — fixture name from `AllFixtures()` keys, or `all` (default: `all`)
- `--platform=<x|bluesky|all>` — which platform(s) to post to (default: `all`)

Logic:

1. Read platform credentials from env vars (same pattern as `main.go`)
2. Load contacts for tagging (same as `main.go`)
3. Get the requested fixture(s) from `testfixtures.AllFixtures()`
4. For each fixture × platform:
   - Create platform instance via `x.NewXPlatform(...)` or `bluesky.NewBlueskyPlatform(...)`
   - Call `platform.Format(fixture)` → `platform.Post(content)`
   - Print success/failure

No vote log involvement. No Zurich API calls. Just fixture → format → post.

Usage:

```bash
# Post all fixtures to both platforms
X_API_KEY=... BLUESKY_HANDLE=... go run cmd/post_fixture/main.go

# Post one fixture to Bluesky only
BLUESKY_HANDLE=... go run cmd/post_fixture/main.go --fixture=auswahl-vote --platform=bluesky
```

### Success Criteria:

#### Automated Verification:

- [x] `go build ./cmd/post_fixture/...` compiles
- [x] `go vet ./...` passes

#### Manual Verification:

- [x] Run with test account credentials and verify posts appear correctly on the test profiles
- [x] Thread structure correct (root + replies linked)
- [x] Text rendering correct (emoji, line breaks, truncation)
- [x] Bluesky facets work (links clickable, @mentions resolve)

**Implementation Note**: This phase requires test account credentials. Set up X dev app + throwaway Bluesky account before testing.

---

## Testing Strategy

### Unit Tests (CI/CD — `go test ./...`):

- **X `Post` method** (`pkg/voteposting/platforms/x/platform_test.go`): thread chaining, reply IDs, post limits — via mock `postTweetFunc`
- **Bluesky `Post` method** (`pkg/voteposting/platforms/bluesky/platform_test.go`): thread chaining, `replyRef` structure, facets, lazy auth — via mock functions
- **No-op VoteLog** (`pkg/votelog/votelog_test.go`): `IsPosted` always false, `MarkAsPosted`/`Save` are no-ops
- **Existing format tests**: unchanged, still pass

### Manual E2E (not in CI/CD):

1. **Live votes**: `SKIP_VOTE_LOG=true MAX_VOTES_TO_CHECK=5 go run main.go` with test account env vars
2. **Fixture posting**: `go run cmd/post_fixture/main.go --fixture=all` with test account env vars

### Regression Workflow:

After any formatting or posting change:

1. `go test ./...` — automated
2. `source .env.test && go run cmd/post_fixture/main.go --fixture=all` — manual fixture verification
3. `source .env.test && SKIP_VOTE_LOG=true MAX_VOTES_TO_CHECK=5 go run main.go` — manual live vote verification

## References

- Research: `thoughts/shared/research/2026-04-05-dry-post-e2e-testing.md`
- X platform Post: `pkg/voteposting/platforms/x/platform.go:71-102`
- Bluesky platform Post: `pkg/voteposting/platforms/bluesky/platform.go:93-137`
- PostTweet: `pkg/xapi/client.go:30-78`
- CreateRecord: `pkg/bskyapi/client.go:130-196`
- Existing mock: `pkg/voteposting/voteposting_test.go:38-62`
- Bluesky sampleVote: `pkg/voteposting/platforms/bluesky/format_test.go:19-35`
- X reply thread plan: `thoughts/shared/plans/2026-04-01-x-reply-threads.md`
