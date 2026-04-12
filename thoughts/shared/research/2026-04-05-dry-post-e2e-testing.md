---
date: 2026-04-05T12:00:00+02:00
researcher: copilot
topic: "E2E testing strategies: --dry-post flag for X and Bluesky"
tags: [research, codebase, testing, xapi, bskyapi, voteposting, e2e]
status: complete
last_updated: 2026-04-05
---

# Research: E2E Testing Strategies for X Posting

**Date**: 2026-04-05

## Research Question

How to add better e2e testing before going live with new features:

1. A `--dry-post` flag that intercepts `PostTweet` and prints the raw HTTP payload (JSON body with `text` and `reply` fields) instead of sending it.
2. A second "test" X account approach.

## Summary

The codebase already has a `dryRun` mode in `PostToPlatform` that calls `Format` but skips `Post`, printing `content.String()` instead. However, this **does not** exercise the `Post` → `xapi.PostTweet` path, so it can't verify the JSON payload actually sent to X.

A `--dry-post` flag would sit deeper — inside `XPlatform.Post` or `xapi.PostTweet` — intercepting the HTTP call and printing the exact JSON body including the `reply.in_reply_to_tweet_id` threading field. This would exercise the **full chain**: `Format → Content → Post loop → API call construction`.

## Detailed Findings

### 1. Existing Dry-Run Mode

`PostToPlatform` in `pkg/voteposting/prepare.go:74` accepts a `dryRun bool` parameter. When true (`prepare.go:100-114`), it:

- Calls `platform.Format(group)` — exercised
- Prints `content.String()` — the human-readable preview
- **Does NOT** call `platform.Post(content)` — the real posting path is skipped entirely

`main.go:93` always passes `false` for dryRun. There is no CLI flag to set it.

### 2. `PostTweet` — The HTTP Call

`pkg/xapi/client.go:30-78` — `PostTweet` is a **package-level function** (not a method on a struct). It:

- Hardcodes the URL: `https://api.x.com/2/tweets` (line 33)
- Builds a `map[string]interface{}` payload with `"text"` and optionally `"reply": {"in_reply_to_tweet_id": id}` (lines 36-42)
- Marshals to JSON, constructs an `http.Request`, adds OAuth headers, and sends via `http.Client{}` (lines 44-62)
- Parses the response for the tweet ID (lines 70-77)

Because `PostTweet` is a package-level function with a hardcoded URL and an inline `http.Client`, it cannot currently be intercepted without one of:

- Refactoring to accept an HTTP client or URL override
- Wrapping it behind an interface
- Using a variable-based function indirection

### 3. `XPlatform.Post` — The Thread Loop

`pkg/voteposting/platforms/x/platform.go:68-102` — `Post` calls `xapi.PostTweet` directly:

- Root tweet: `xapi.PostTweet(..., root.Text, "")` (line 82)
- Each reply: `xapi.PostTweet(..., reply.Text, parentTweetID)` (line 91)
- Chains via `parentTweetID = tweetID` (line 95)

This is the code path a `--dry-post` flag needs to exercise — it's where the `in_reply_to_tweet_id` threading happens.

### 4. Test Infrastructure

**`pkg/xapi/client_test.go`**: Tests verify JSON payload construction by **duplicating the marshalling logic** inline (not by calling `PostTweet`). An `httptest.Server` is created but unused because the URL is hardcoded. The test notes this limitation explicitly in comments (lines 26-28).

**`pkg/voteposting/voteposting_test.go`**: Uses a `MockPlatform` implementing the `Platform` interface. Tests cover dry-run, real posting, and rate limiting — but at the `Platform` interface level, not at the HTTP level.

**`pkg/voteposting/platforms/x/format_test.go`**: Tests `FormatVoteThread` directly. No tests for `XPlatform.Post`.

### 5. Platform Interface

`pkg/voteposting/platforms/interface.go:10-15`:

```go
type Platform interface {
    Format(votes []zurichapi.Abstimmung) (Content, error)
    Post(content Content) (shouldContinue bool, err error)
    MaxPostsPerRun() int
    Name() string
}
```

`Content` interface (`interface.go:5-8`) only requires `String() string`.

### 6. Credential Handling

`main.go:22-27` reads four env vars: `X_API_KEY`, `X_API_SECRET`, `X_ACCESS_TOKEN`, `X_ACCESS_SECRET`. These are passed down through `NewXPlatform` → stored in `XPlatform` struct fields → forwarded to every `xapi.PostTweet` call.

For a second test account, the same env vars would be set to different values — no code changes needed.

## Strategy Analysis

### Option A: `--dry-post` Flag (Mock HTTP Locally)

**What it verifies**: Format → Content → Post loop → JSON payload construction (including reply threading)

**Implementation approaches** (from least to most invasive):

#### A1. Function variable indirection in `xapi`

Replace the direct `PostTweet` function call in `XPlatform.Post` with a function variable on the `XPlatform` struct:

```go
type XPlatform struct {
    ...
    postTweetFunc func(apiKey, apiSecret, accessToken, accessSecret, message, inReplyToTweetID string) (string, error)
}
```

- `NewXPlatform` sets it to `xapi.PostTweet` by default
- A `--dry-post` mode replaces it with a function that marshals + prints the JSON payload and returns a fake tweet ID
- Minimal changes: `platform.go` only, plus a new constructor option or setter

#### A2. HTTP client injection in `PostTweet`

Refactor `PostTweet` to accept an `*http.Client` (or make it a method on a client struct). In dry-post mode, inject a client whose transport prints the request and returns a canned response. This also fixes the test limitation noted in `client_test.go`.

#### A3. URL override

Add an optional `baseURL` to the xapi package. In dry-post mode, spin up an `httptest.Server` that logs payloads. Heaviest approach.

**Recommendation**: A1 is the simplest. It exercises the full `Post` loop including reply chaining logic without touching `xapi` internals.

### Option B: Test Accounts (Manual E2E)

**What it verifies**: The entire real pipeline including auth, rate limits, actual API responses, and visual correctness of posts.

**Code changes needed**: None — just set different env vars pointing to test account credentials.

**Platform notes**:

- **X/Twitter**: Create a throwaway X account and register a new (free) developer app under it in the X Developer Portal. Generate access tokens directly from the portal — no OAuth callback flow needed. The developer app itself is free; API calls are billed under X's pay-what-you-use model, negligible for occasional manual test runs. This keeps credentials fully separate from the production app.
- **Bluesky**: No restrictions — any account can post via the AT Protocol API. Create a throwaway `@zuerichratsinfo-test.bsky.social` (or similar) account.

**Downsides**:

- Each test run creates real posts (must clean up manually or accept clutter on the test account)
- Slower feedback loop (network calls)
- API rate limits apply (but unlikely to be hit in manual testing)
- X developer app setup takes time (OAuth credentials)

## Code References

### X Platform

- `main.go:22-27` — X credential env vars
- `main.go:93` — `dryRun: false` hardcoded
- `pkg/xapi/client.go:30-78` — `PostTweet` function (hardcoded URL, inline HTTP client)
- `pkg/xapi/client.go:36-42` — JSON payload construction with `text` and `reply` fields
- `pkg/xapi/client_test.go:26-28` — Comment noting URL is hardcoded, can't test through server
- `pkg/voteposting/platforms/x/platform.go:34-60` — `XPlatform` struct and `NewXPlatform`
- `pkg/voteposting/platforms/x/platform.go:68-102` — `Post` method (thread loop with `PostTweet` calls)

### Bluesky Platform

- `main.go:30-33` — Bluesky credential env vars (`BLUESKY_HANDLE`, `BLUESKY_PASSWORD`)
- `pkg/bskyapi/client.go:66-114` — `CreateSession` (authentication)
- `pkg/bskyapi/client.go:130-196` — `CreateRecord` (post creation with facets and reply threading)
- `pkg/voteposting/platforms/bluesky/platform.go:40-60` — `BlueskyPlatform` struct and `NewBlueskyPlatform`
- `pkg/voteposting/platforms/bluesky/platform.go:65-77` — `ensureSession` (lazy auth)
- `pkg/voteposting/platforms/bluesky/platform.go:90-137` — `Post` method (thread loop with `CreateRecord` calls)
- `pkg/voteposting/platforms/bluesky/platform.go:148+` — `resolveMentionFacets` (DID resolution for @mentions)

### Shared

- `pkg/voteposting/platforms/interface.go:5-15` — `Content` and `Platform` interfaces
- `pkg/voteposting/prepare.go:74-142` — `PostToPlatform` with existing `dryRun` path
- `pkg/voteposting/voteposting_test.go:38-62` — Existing `MockPlatform` test infrastructure

## Architecture Documentation

- **Pattern**: Platform interface abstraction with per-platform Format/Post implementations
- **Config**: All credentials via environment variables, no CLI flags currently
- **Dry-run**: Exists at the `PostToPlatform` level (skips `Post`), not at the HTTP level
- **Testing**: Format functions are well-tested; `Post`/`PostTweet` are not tested due to hardcoded HTTP dependency

## Bluesky Platform — Dry-Post Applicability

### Bluesky Posting Pipeline

`BlueskyPlatform.Post` (`pkg/voteposting/platforms/bluesky/platform.go:90-137`) follows the same pattern as X:

1. Authenticates lazily via `bskyapi.CreateSession` (line 99) — returns a `*Session` with `AccessJwt`, `DID`, and `ServiceEndpoint`
2. Posts root: `bskyapi.CreateRecord(session, root.Text, root.Facets, nil)` (line 104)
3. Posts replies in chain: `bskyapi.CreateRecord(session, reply.Text, reply.Facets, replyRef)` (line 118) where `replyRef` contains `Root` and `Parent` `PostRef` (URI + CID)

### `bskyapi.CreateRecord` — The HTTP Call

`pkg/bskyapi/client.go:130-196` — Package-level function, same pattern as `xapi.PostTweet`:

- URL: `session.ServiceEndpoint + "/xrpc/com.atproto.repo.createRecord"` (line 131)
- JSON payload structure (`client.go:134-162`):
  ```json
  {
    "repo": "<DID>",
    "collection": "app.bsky.feed.post",
    "record": {
      "$type": "app.bsky.feed.post",
      "text": "...",
      "createdAt": "...",
      "facets": [...],
      "reply": {
        "root": {"uri": "...", "cid": "..."},
        "parent": {"uri": "...", "cid": "..."}
      }
    }
  }
  ```
- Auth: `Bearer` token from `session.AccessJwt` (line 174)
- Returns `*PostRef` with `URI` and `CID` (line 190)
- Inline `http.Client` with 10s timeout — same interception challenge as X

### Key Differences from X

| Aspect         | X (`xapi`)                                | Bluesky (`bskyapi`)                                              |
| -------------- | ----------------------------------------- | ---------------------------------------------------------------- |
| Auth           | OAuth 1.0a per-request                    | Session-based JWT (lazy auth on first post)                      |
| Reply model    | `{"reply": {"in_reply_to_tweet_id": id}}` | `{"reply": {"root": {uri, cid}, "parent": {uri, cid}}}`          |
| Return value   | Tweet ID (string)                         | `*PostRef` (URI + CID)                                           |
| API function   | `PostTweet(6 string args)`                | `CreateRecord(session, text, facets, replyRef)`                  |
| Extra features | —                                         | Rich text facets (links, mentions with DID resolution)           |
| DID resolution | N/A                                       | `resolveMentionFacets` calls `bskyapi.ResolveHandle` per mention |

### Dry-Post Strategy for Bluesky

The same function-variable approach (A1) applies. Add a `createRecordFunc` field to `BlueskyPlatform`:

```go
type BlueskyPlatform struct {
    ...
    createRecordFunc func(session *bskyapi.Session, text string, facets []bskyapi.Facet, replyTo *bskyapi.ReplyRef) (*bskyapi.PostRef, error)
}
```

- Default: `bskyapi.CreateRecord`
- Dry-post mode: print the full JSON payload (including `record.reply`, `record.facets`, and `repo`/`collection` wrapper) and return a fake `*PostRef` with synthetic URI/CID

**Additional consideration**: Bluesky's `ensureSession` (`platform.go:65-77`) also makes a real HTTP call. In dry-post mode, this should be skipped or faked — either by pre-populating `p.session` with a dummy session, or by also making `CreateSession` injectable.

Similarly, `resolveMentionFacets` (`platform.go:148+`) calls `bskyapi.ResolveHandle` for each unresolved mention. In dry-post mode, this would either need to be skipped (facets left unresolved) or the resolve function also made injectable. The simplest approach: if `p.session == nil` in dry-post mode, skip DID resolution and print mentions as-is.

### Bluesky Test Infrastructure

There are currently **no tests** for `bskyapi/client.go` or `bluesky/platform.go`. The only Bluesky test coverage is `bluesky/format_test.go` (if it exists — format functions only). This makes the dry-post approach even more valuable for Bluesky since there's no existing test safety net for the posting path.

## Feature Development Workflow (Proposed)

Once the client mocks (A1) and test accounts (B) are in place, new features follow a two-stage verification process:

### Stage 1: Develop + Unit Tests

Standard development loop, all local, no network calls:

1. Write / modify formatting logic (`format.go`) and post logic (`platform.go`)
2. Run unit tests with mock client functions — verify:
   - Correct JSON payload structure (text, reply fields, facets)
   - Thread chaining (reply IDs propagate correctly)
   - Rate limit / post-count behaviour
   - Edge cases (long titles, multi-vote groups, Auswahl votes, etc.)
3. Use the existing dry-run mode (`PostToPlatform(..., true)`) for quick human-readable previews

### Stage 2: Manual E2E with Test Accounts

Use the **same production binary** (`main.go`) — no separate test command needed. The only difference is which credentials are in the environment.

#### The Vote Log Problem

Running `main.go` as-is has a practical friction point:

- The **production vote log** (`data/posted_votes_x.json`, `data/posted_votes_bluesky.json`) is likely up-to-date, so `PrepareVoteGroups` filters everything out → nothing to post.
- Using a separate test vote log is cumbersome — it's never up-to-date, and maintaining it is overhead.
- Altering the prod vote log is risky.

#### Solution: `SKIP_VOTE_LOG=true` env var

Add a single env var that makes `main.go` treat all fetched votes as unposted (bypasses `filterUnpostedVotes` and skips `voteLog.MarkAsPosted` / `voteLog.Save`). Combined with existing `MAX_VOTES_TO_CHECK`, this gives full control:

```bash
# Post the most recent vote group to test accounts
SKIP_VOTE_LOG=true MAX_VOTES_TO_CHECK=5 \
  X_API_KEY=... X_API_SECRET=... X_ACCESS_TOKEN=... X_ACCESS_SECRET=... \
  BLUESKY_HANDLE=test.bsky.social BLUESKY_PASSWORD=... \
  go run main.go
```

This is minimal code change (a few lines in `main.go` to pass a nil or no-op vote log) and keeps the binary identical to production otherwise.

#### For posting a specific vote

For maximum control, a `VOTE_LINK` or `VOTE_GUID` env var could fetch and post a single specific vote by its GUID, bypassing `FetchRecentAbstimmungen` entirely. The Zurich API already supports fetching by session (`FetchAbstimmungenForSitzung`), so a similar fetch-by-GUID could be added. But `SKIP_VOTE_LOG=true` + `MAX_VOTES_TO_CHECK=N` covers most cases with zero new API code.

#### Workflow

1. Set env vars to test account credentials + `SKIP_VOTE_LOG=true`
2. Run `go run main.go` — same binary as the GitHub Actions bot
3. Open the test account profiles in a browser and verify:
   - Thread structure (root + replies linked correctly)
   - Text rendering (emoji, line breaks, truncation)
   - Bluesky facets (links clickable, @mentions resolve to correct profiles)
   - Character/grapheme limits respected (no truncation by the platform itself)

This ensures there is zero divergence between what is tested and what runs in production. The binary fetches real votes from the Zurich API, formats them, and posts to whichever accounts the env vars point to.

### Test Separation Principle

| Type                 | What                                                                      | Where                                                    | Runs in CI/CD            |
| -------------------- | ------------------------------------------------------------------------- | -------------------------------------------------------- | ------------------------ |
| Unit tests           | Format logic, payload structure, thread chaining, rate limits, edge cases | `*_test.go` files with mock client functions             | ✅ Yes (`go test ./...`) |
| Manual E2E (live)    | Visual correctness with real recent votes                                 | Same `main.go` + `SKIP_VOTE_LOG=true` + test account env | ❌ No — manual only      |
| Manual E2E (fixture) | Visual regression with hardcoded edge cases                               | `cmd/post_fixture/main.go` + test account env            | ❌ No — manual only      |

All tests that use client mocks (function variable approach A1) are **unit tests** in `_test.go` files and run as part of CI/CD. They exercise the full `Post` method logic including thread chaining, but with a mock that captures the payload instead of making HTTP calls.

The manual e2e steps are **intentionally excluded from CI/CD** (GitHub Actions) because they require human visual inspection and would otherwise just create post clutter with real API costs.

### Shared Test Fixtures

Fixtures are hardcoded `zurichapi.Abstimmung` structs representing edge cases discovered over time. They serve two purposes:

1. **Unit tests** — assert on formatting output, payload structure, thread chaining
2. **Fixture-based e2e** — post the exact same edge cases to test accounts for visual verification

This ensures the edge cases seen and validated by a human on the real platforms are the same ones that unit tests guard against regression.

**Fixture library** (`pkg/voteposting/testfixtures/`):

```go
package testfixtures

// Each fixture is a named []zurichapi.Abstimmung representing a vote group.

func SingleVoteAngenommen() []zurichapi.Abstimmung { ... }
func SingleVoteAbgelehnt() []zurichapi.Abstimmung { ... }
func MultiVoteGroup() []zurichapi.Abstimmung { ... }
func AuswahlVote() []zurichapi.Abstimmung { ... }
func LongTitleTruncation() []zurichapi.Abstimmung { ... }
func PostulatWithMentions() []zurichapi.Abstimmung { ... }

// AllFixtures returns all fixtures keyed by name.
func AllFixtures() map[string][]zurichapi.Abstimmung { ... }
```

**Existing fixtures** to seed from:

- `pkg/voteposting/platforms/bluesky/format_test.go` — `sampleVote()` helper + test cases
- `pkg/voteposting/platforms/x/format_test.go` — `intPtr()` helper, Postulat/Motion cases, multi-vote groups

#### Existing Test Cases Inventory

Both test files contain vote data that can be extracted. The Bluesky `sampleVote()` helper is the better template since it fills all GUID fields; the X tests are sparser (often missing `SitzungGuid`, `TraktandumGuid`, etc.) which is fine for unit tests but incomplete for e2e posting.

**From `bluesky/format_test.go`:**

| Fixture                                  | Description                                                                   |
| ---------------------------------------- | ----------------------------------------------------------------------------- |
| Single vote angenommen                   | Postulat, accepted (90/30/0/5)                                                |
| Single vote abgelehnt                    | Antrag, rejected (20/95/5/5)                                                  |
| Very long title                          | ~300-char title triggering truncation with `…`                                |
| Multi-vote group (2 votes)               | "Gesamtrevision der Gemeindeordnung" — Einleitungsartikel + Schlussabstimmung |
| Generic Antrag → GeschaeftTitel fallback | TraktandumTitel is `"Antrag 1."`, falls back to GeschaeftTitel                |
| 10-vote stress test                      | Forces multiple reply posts from packing logic                                |
| Vote with @mentions                      | "Postulat von Anna Graff (SP)" — tests mention facets                         |

**From `x/format_test.go`** (unique cases not in Bluesky):

| Fixture                              | Description                                                         |
| ------------------------------------ | ------------------------------------------------------------------- |
| Postulat with GrNr prefix            | `"2025/100 Postulat von Reto Brüesch (SVP)"` — tests GrNr stripping |
| Motion with GrNr prefix              | `"2025/200 Motion von Liv Mahrer (SP)"`                             |
| Single Auswahl vote                  | A/B/C counts, no ✅/❌ prefix                                       |
| Mixed multi-vote (Ja/Nein + Auswahl) | One standard + one Auswahl in same group                            |

**Starting set**: ~8-10 fixtures. The Bluesky `sampleVote()` pattern should be adopted as the shared helper (fills all fields). X-specific edge cases (Auswahl, mixed multi-vote, GrNr-prefixed titles) add coverage not present in Bluesky tests.

New edge cases are added to the library as they're discovered, and both unit tests and the fixture e2e command pick them up automatically.

### Fixture E2E Command (`cmd/post_fixture/main.go`)

A small command that imports fixtures from the shared library and posts them through the real platform `Post` path. It bypasses `PrepareVoteGroups` and the vote log entirely — it just formats and posts hardcoded data.

```bash
# Post a single fixture
go run cmd/post_fixture/main.go --fixture=long-title-truncation

# Post all fixtures
go run cmd/post_fixture/main.go --fixture=all

# Post only to Bluesky
go run cmd/post_fixture/main.go --fixture=auswahl-vote --platform=bluesky
```

This command:

- Imports `testfixtures.AllFixtures()` (or a named one via `--fixture`)
- Creates `XPlatform` / `BlueskyPlatform` from env vars (same credential pattern as `main.go`)
- Calls `platform.Format(fixture)` then `platform.Post(content)` — the real posting path
- No vote log, no Zurich API fetch — just fixture → format → post

It's intentionally thin — all the logic lives in the same platform packages that `main.go` uses. The only difference is where the vote data comes from (hardcoded fixture vs live API).

### Regression Testing

After any formatting or posting change:

1. `go test ./...` — automated (CI/CD): catches structural regressions via unit tests with mocked clients
2. `SKIP_VOTE_LOG=true go run main.go` with test account env vars — manual: verifies real recent votes look correct
3. `go run cmd/post_fixture/main.go --fixture=all` with test account env vars — manual: verifies all known edge cases still render correctly

### X API Cost Considerations

A separate free developer app is created under the throwaway test X account. API calls are billed under X's pay-what-you-use pricing — the cost of occasional manual test runs is negligible. This avoids sharing credentials with the production app entirely.

## Historical Context (from thoughts/)

`thoughts/shared/plans/2026-04-01-x-reply-threads.md` — Recent plan for X reply thread support. The threading logic (`in_reply_to_tweet_id` chaining) is now implemented in `XPlatform.Post`. This is exactly the code path that `--dry-post` would help validate.

## Open Questions

1. Should `--dry-post` be a CLI flag or an env var (consistent with current config pattern)?
   > consistencty with current env var approach suggests an env var like `X_DRY_POST=true` would be more appropriate than a CLI flag.
2. Should the dry-post output be machine-readable JSON (for scripted validation) or human-readable (for manual review)?
   > the goal would be to have unit tests that call the `--dry-post` code path and validate the JSON output, so it should be machine-readable. For manual review, the raw JSON can be printed as-is or pretty-printed.
3. Should the function variable approach (A1) also be leveraged in unit tests to replace the current duplicated-logic tests in `client_test.go`?
   > Yes, refactoring to use a function variable would allow the existing tests to call `PostTweet` directly and verify the JSON payload without duplication. This would be a cleaner test design.
4. For Bluesky dry-post: should DID resolution be skipped entirely, or should a mock resolver return synthetic DIDs?
   > Skipping is simpler; the facet byte offsets are already correct from formatting. DID resolution only affects the `did` field in mention facets — acceptable to leave empty in dry-post output.
5. Should both platforms share a single `DRY_POST=true` env var, or have per-platform flags (`X_DRY_POST`, `BLUESKY_DRY_POST`)?
   > Per-platform is more flexible (test one while posting to the other live), but a shared flag is simpler for the common case of testing everything.
6. ~~For the test account approach: does the project already have a second X developer app, or would one need to be created from scratch?~~
   > Resolved: Create a separate free developer app under the throwaway test X account. Simpler than sharing the production app's credentials via 3-legged OAuth. Cost is negligible under pay-what-you-use.
7. ~~Should the shared test fixtures live in a dedicated `pkg/voteposting/testfixtures/` package, or just be extracted into a shared `_test.go` file?~~
   > Resolved: Dedicated `pkg/voteposting/testfixtures/` package. It's not a `_test.go`-only package because `cmd/post_fixture/main.go` (non-test code) needs to import it. Fixtures grow over time as new edge cases are discovered.
8. Should the e2e test command clean up posts after visual inspection (delete via API), or leave them on the test accounts for later reference?
   > For simplicity, leave them. The test accounts are throwaways and won't be monitored, so post clutter is not a concern. If cleanup is desired in the future, API calls to delete posts could be added.
