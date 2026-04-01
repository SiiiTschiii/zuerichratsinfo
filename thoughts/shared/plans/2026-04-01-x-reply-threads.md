---
date: 2026-04-01T12:00:00+02:00
planner: copilot
topic: "X Reply Threads for Vote Posts"
tags: [plan, x, twitter, threads, voteposting]
status: ready
last_updated: 2026-04-01
---

# X Reply Threads for Vote Posts — Implementation Plan

## Overview

Add reply thread support to the X/Twitter platform, mirroring the existing Bluesky thread architecture. Keep X Premium and use a relaxed ~2,000 char limit per post. The thread architecture enables future extensions (e.g. Fraktionsresultate) as additional replies.

## Current State Analysis

- X posts each vote group as a **single flat tweet** with no threading (`FormatVoteGroupPost()` → single string)
- `PostTweet()` returns only `error`, does not return tweet ID, does not accept reply-to param
- Bluesky already implements full thread model: `FormatVoteThread()` → `[]*BlueskyPost`, posted via root + reply chain
- `generate_vote_post` command uses the platform interface with dry-run, so changes flow through automatically

### Key Discoveries

- `PostTweet()` at [pkg/xapi/client.go:20](pkg/xapi/client.go#L20) — returns `error` only, no tweet ID, no reply support
- `FormatVoteGroupPost()` at [pkg/voteposting/platforms/x/format.go:20](pkg/voteposting/platforms/x/format.go#L20) — single string, no char limit
- `XContent` at [pkg/voteposting/platforms/x/platform.go:11](pkg/voteposting/platforms/x/platform.go#L11) — wraps `message string`
- Bluesky thread pattern at [pkg/voteposting/platforms/bluesky/format.go:28](pkg/voteposting/platforms/bluesky/format.go#L28) — `FormatVoteThread()` returns `[]*BlueskyPost`
- Bluesky posting loop at [pkg/voteposting/platforms/bluesky/platform.go:85](pkg/voteposting/platforms/bluesky/platform.go#L85) — root + reply chain with `parentRef` advancement
- `Platform`/`Content` interfaces at [pkg/voteposting/platforms/interface.go](pkg/voteposting/platforms/interface.go) — already flexible, no changes needed

## Desired End State

- X posts vote groups as threads: root post (header + title + thread hint) + reply posts (vote details + link)
- API client supports reply chaining via `reply.in_reply_to_tweet_id`
- Bin-packing algorithm packs vote entries into replies within `maxChars` (2000) limit
- Visual alignment with Bluesky: same content structure, different char limits

## What We're NOT Doing

- Changing `prepare.go` or the `Platform`/`Content` interfaces
- Switching to `FormatVoteCounts()` (keeping `FormatVoteCountsLong()` for X)
- Dropping X Premium — keeping it for blue checkmark + relaxed char limit
- Adding Fraktionsresultate (future extension, just enabling the architecture)

## Implementation Approach

Mirror the Bluesky thread pattern across 4 files. The platform interface and posting loop remain unchanged — thread management is entirely within the X platform layer.

---

## Phase 1: API Client — Return Tweet ID + Reply Support

### Overview

Modify `PostTweet()` to support reply threading and return the tweet ID for chaining.

### Changes Required

#### 1. X API Client

**File**: `pkg/xapi/client.go`

**Changes**:

1. Change `PostTweet()` signature: add `inReplyToTweetID string` param, change return to `(string, error)`
2. When `inReplyToTweetID != ""`, add `"reply": {"in_reply_to_tweet_id": "..."}` to the JSON payload
3. Parse response JSON to extract `data.id` (tweet ID string) and return it
4. Define response struct:

```go
type tweetResponse struct {
    Data struct {
        ID string `json:"id"`
    } `json:"data"`
}
```

#### 2. Update caller

**File**: `pkg/voteposting/platforms/x/platform.go`

**Changes**: Update `Post()` to handle new `(string, error)` return — pass `""` as `inReplyToTweetID` for now (full thread loop comes in Phase 3).

### Success Criteria

#### Automated Verification:

- [x] `go build ./pkg/xapi/...` — compiles
- [x] `go build ./pkg/voteposting/platforms/x/...` — compiles
- [x] `go vet ./...` — no issues

#### Manual Verification:

- [ ] `PostTweet(..., "")` behaves like before (root post), returns tweet ID
- [ ] `PostTweet(..., "123456")` includes `reply.in_reply_to_tweet_id` in payload

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 2: Thread Formatting

### Overview

Replace single-string formatting with thread-aware formatting that produces a root post + bin-packed reply posts.

### Changes Required

#### 1. Thread format types and functions

**File**: `pkg/voteposting/platforms/x/format.go`

**Changes**:

1. Add `const maxChars = 2000` (analogous to Bluesky's `maxGraphemes = 300`)
2. Add `XPost` struct:

```go
// XPost holds the formatted text for a single post in an X thread
type XPost struct {
    Text string
}
```

3. Replace `FormatVoteGroupPost()` with `FormatVoteThread()` returning `[]*XPost`
   - Follow Bluesky's pattern: `buildRootPost()` + `buildReplyPosts()`

4. **`buildRootPost()`**: header + title + result (single vote) or just title (multi-vote) + `"👇 Details im Thread"`. Truncate title if exceeding `maxChars`. Keep X handle tagging on title via `contactMapper.TagXHandlesInText()`

5. **`buildReplyPosts()`**: bin-pack vote entries within `maxChars` each. Use `FormatVoteCountsLong()`. Append link (`🔗 URL`) to last reply. If link doesn't fit, create standalone link-only reply (same as Bluesky pattern at [bluesky/format.go:226-232](pkg/voteposting/platforms/bluesky/format.go#L226-L232))

6. Update `FormatVotePost()` (single-vote convenience) to call `FormatVoteThread()` — or remove if unused outside tests

### Key reference

Bluesky's `buildReplyPosts()` at [pkg/voteposting/platforms/bluesky/format.go:143-240](pkg/voteposting/platforms/bluesky/format.go#L143-L240):

- Accumulate entries until adding next would exceed limit
- Flush current batch as a reply, start new batch
- Last batch gets the link appended (or link goes in its own reply if it doesn't fit)

### Character counting

- Bluesky uses `graphemeLen()` (grapheme clusters)
- X uses **weighted characters**: Latin=1, emoji=2, CJK=2, URL=23 fixed
- Implementation: use `len(text)` as approximation (all content is Latin + emoji). With `maxChars=2000`, precision isn't critical — it becomes important only if `maxChars` is lowered to 280.

### Success Criteria

#### Automated Verification:

- [ ] `go build ./pkg/voteposting/platforms/x/...` — compiles
- [ ] `go vet ./...` — no issues
- [ ] `act -W .github/workflows/go-ci.yml` passes all tests

#### Manual Verification:

- [ ] `FormatVoteThread()` returns `[0]` = root, `[1:]` = replies
- [ ] Root post contains header, title, thread hint
- [ ] Reply posts contain vote details bin-packed within `maxChars`
- [ ] Link appears on last reply
- [ ] Single-vote and multi-vote groups both produce correct threads

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 3: Thread Posting

### Overview

Update `XContent` and `XPlatform.Post()` to post threads as root + reply chain.

### Changes Required

#### 1. Content struct and posting loop

**File**: `pkg/voteposting/platforms/x/platform.go`

**Changes**:

1. Change `XContent` struct: replace `message string` with `thread []*XPost`
2. Update `String()` method: iterate thread posts with `↳ Reply N:` separators (same pattern as `BlueskyContent.String()` at [bluesky/platform.go:21-35](pkg/voteposting/platforms/bluesky/platform.go#L21-L35))
3. Update `Format()`: call `FormatVoteThread()` instead of `FormatVoteGroupPost()`, wrap in `XContent{thread: ...}`
4. Update `Post()`:
   - Post root (`thread[0]`) via `xapi.PostTweet(..., "")` → get `rootTweetID`
   - Iterate `thread[1:]`: call `xapi.PostTweet(..., parentTweetID)` → get new tweet ID → advance `parentTweetID`
   - Follow exact pattern from `BlueskyPlatform.Post()` at [bluesky/platform.go:85-124](pkg/voteposting/platforms/bluesky/platform.go#L85-L124)
   - `postsThisRun` still counts vote groups (not individual tweets)

### Success Criteria

#### Automated Verification:

- [ ] `go build ./...` — compiles cleanly
- [ ] `go vet ./...` — no issues
- [ ] `act -W .github/workflows/go-ci.yml` passes all tests

#### Manual Verification:

- [ ] `go run cmd/generate_vote_post/main.go -n 1 --platform x` — shows thread preview with root + replies
- [ ] `go run cmd/generate_vote_post/main.go -n 3` — X and Bluesky threads visually aligned
- [ ] `String()` shows readable preview with reply indicators

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 4: Test Updates

### Overview

Update existing tests and add new tests for the thread format.

### Changes Required

#### 1. Test file

**File**: `pkg/voteposting/platforms/x/format_test.go`

**Changes**:

1. Update `TestFormatVoteGroupPost_PreservesPostulatMotion` → rename to `TestFormatVoteThread_PreservesPostulatMotion`
   - Call `FormatVoteThread()` instead of `FormatVoteGroupPost()`
   - Assert `expectedParts` appear somewhere across the thread (root or replies)

2. Update `TestFormatVoteGroupPost_AuswahlVote` → rename to `TestFormatVoteThread_AuswahlVote`
   - Same approach: check `shouldContain`/`shouldNotContain` across all posts in thread

3. Add new tests:
   - **Thread structure**: single vote → root + ≥1 reply; multi-vote → root + ≥1 reply
   - **Bin-packing**: create votes that exceed `maxChars` in a single reply → verify split into multiple replies
   - **Link placement**: link appears in last reply's text
   - **Root content**: root contains header, title, `"👇 Details im Thread"`
   - **Root truncation**: very long title is truncated with `"…"`

4. Add `allThreadText(thread []*XPost) string` helper that joins all post texts for simple `strings.Contains` assertions

### Success Criteria

#### Automated Verification:

- [ ] `go test ./pkg/voteposting/platforms/x/...` — all tests pass
- [ ] `go test ./...` — no regressions across the project
- [ ] `go vet ./...` — no issues
- [ ] `go build ./...` — compiles cleanly
- [ ] `act -W .github/workflows/go-ci.yml` passes all tests

#### Manual Verification:

- [ ] Test output matches expected thread structure

---

## Testing Strategy

### Unit Tests

- Thread formatting: root structure, reply bin-packing, link placement, title truncation
- Auswahl vote handling (no ✅/❌ prefix)
- Postulat/Motion titles preserved across thread

### Integration Tests

- `generate_vote_post` preview shows correct X thread format
- Full dry-run posting cycle

### Manual Testing Steps

1. `go run cmd/generate_vote_post/main.go -n 3` — verify X threads show root + replies
2. `go run cmd/generate_vote_post/main.go -n 1 --platform x` — single vote group thread
3. Compare X and Bluesky output side-by-side for visual alignment

---

## Decisions

| Decision                | Choice                   | Rationale                                                 |
| ----------------------- | ------------------------ | --------------------------------------------------------- |
| `maxChars`              | 2000                     | Leverages X Premium; one-line change to 280 if dropped    |
| Single-vote threading   | Always thread            | Aligns with Bluesky for consistency                       |
| Vote counts format      | `FormatVoteCountsLong()` | More readable with higher char limit                      |
| `postsThisRun` counting | Vote groups              | Not individual tweets (same as Bluesky)                   |
| Character counting      | `len(text)` approx       | Sufficient at 2000; precise weighted counting only at 280 |

## References

- Research: `thoughts/shared/research/2026-04-01-x-reply-threads.md`
- Bluesky thread format: `pkg/voteposting/platforms/bluesky/format.go:28`
- Bluesky thread posting: `pkg/voteposting/platforms/bluesky/platform.go:85`
- X API v2 reply docs: https://docs.x.com/x-api/posts/creation-of-a-post
