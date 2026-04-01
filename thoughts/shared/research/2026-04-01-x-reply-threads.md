---
date: 2026-04-01T12:00:00+02:00
researcher: copilot
topic: "X Reply Threads for Vote Posts"
tags: [research, codebase, x, twitter, threads, voteposting]
status: complete
last_updated: 2026-04-01
---

# Research: X Reply Threads for Vote Posts

**Date**: 2026-04-01

## Research Question

How to implement reply threads on X (Twitter) for vote posting, mirroring the existing Bluesky thread approach, to:

1. Remove dependency on X Premium (280-char limit for free accounts)
2. Enable future extensions (e.g. per-Fraktion breakdowns) as reply posts
3. Align with the existing Bluesky thread architecture

## Summary

The X platform currently posts each vote group as a **single flat tweet** with no character limit enforcement and no reply threading. The Bluesky platform already implements a full thread model (root + replies, bin-packed within 300 graphemes). The X API v2 natively supports reply threading via `reply.in_reply_to_tweet_id` in the `POST /2/tweets` payload. The implementation requires changes to 4 files: `pkg/xapi/client.go` (return tweet ID, accept reply-to param), `pkg/voteposting/platforms/x/format.go` (thread formatting), `pkg/voteposting/platforms/x/platform.go` (thread posting loop), and `pkg/voteposting/platforms/x/format_test.go` (test updates).

## Detailed Findings

### 1. Current X Posting Pipeline

#### API Client (`pkg/xapi/client.go`)

- `PostTweet(apiKey, apiSecret, accessToken, accessSecret, message string) error` at [client.go:20](pkg/xapi/client.go#L20)
- Sends `POST https://api.x.com/2/tweets` with body `{"text": message}`
- Uses OAuth 1.0a (HMAC-SHA1) for authentication
- Returns only `error` — **does not return the tweet ID** from the response
- The X API v2 response includes `data.id` (the tweet ID string) which is needed for reply chaining
- **No `reply` field** is included in the request payload

#### X Format (`pkg/voteposting/platforms/x/format.go`)

- `FormatVoteGroupPost(votes, contactMapper) string` at [format.go:20](pkg/voteposting/platforms/x/format.go#L20)
- Returns a **single string** containing header, title, all vote results, counts, and link
- For multi-vote groups, all individual votes are listed inline with `✅`/`❌` emoji + counts
- No character limit enforcement — relies on X Premium's extended tweet limit
- Structure for a single vote: `header + resultEmoji + result + title + counts + link`
- Structure for multi-vote: `header + title + (emoji + subtitle + counts) × N + link`

#### X Platform (`pkg/voteposting/platforms/x/platform.go`)

- `XContent` struct at [platform.go:11](pkg/voteposting/platforms/x/platform.go#L11): wraps a single `message string`
- `XPlatform.Format()` at [platform.go:54](pkg/voteposting/platforms/x/platform.go#L54): calls `FormatVoteGroupPost`, wraps in `XContent`
- `XPlatform.Post()` at [platform.go:62](pkg/voteposting/platforms/x/platform.go#L62): calls `xapi.PostTweet()` with the single message string

#### Platform Interface (`pkg/voteposting/platforms/interface.go`)

- `Platform` interface at [interface.go:11](pkg/voteposting/platforms/interface.go#L11): `Format()`, `Post()`, `MaxPostsPerRun()`, `Name()`
- `Content` interface at [interface.go:6](pkg/voteposting/platforms/interface.go#L6): only requires `String() string`
- The interface is **already flexible enough** — `Content` is opaque, and each platform's `Post()` type-asserts to its concrete type (e.g. `*XContent`, `*BlueskyContent`)

#### Posting Loop (`pkg/voteposting/prepare.go`)

- `PostToPlatform()` at [prepare.go:68](pkg/voteposting/prepare.go#L68) iterates vote groups, calling `platform.Format()` then `platform.Post()` for each
- The loop is platform-agnostic — it does not know about threading
- **No changes needed** to `prepare.go` — thread management is entirely within the platform's `Post()` method (same as Bluesky)

### 2. Existing Bluesky Thread Implementation (Reference Pattern)

#### Thread Format (`pkg/voteposting/platforms/bluesky/format.go`)

- `FormatVoteThread(votes, contactMapper) []*BlueskyPost` at [format.go:28](pkg/voteposting/platforms/bluesky/format.go#L28)
- Returns a slice: `[0]` = root post, `[1:]` = reply posts
- **Root post** (via `buildRootPost` at [format.go:73](pkg/voteposting/platforms/bluesky/format.go#L73)):
  - Header + title + result (single vote) + `"👇 Details im Thread"`
  - Truncated with `"…"` if exceeding 300 graphemes
- **Reply posts** (via `buildReplyPosts` at [format.go:143](pkg/voteposting/platforms/bluesky/format.go#L143)):
  - Vote entries are **bin-packed**: entries accumulate until adding the next would exceed 300 graphemes
  - Link (`🔗 URL`) is appended to the last reply; if it doesn't fit, a standalone link-only reply is created
- **Mention scanning**: after all posts are assembled, `contactMapper.FindBlueskyMentions()` populates `post.Mentions`

#### Thread Posting (`pkg/voteposting/platforms/bluesky/platform.go`)

- `BlueskyContent` struct at [platform.go:14](pkg/voteposting/platforms/bluesky/platform.go#L14): `thread []*BlueskyPost`
- `Post()` at [platform.go:85](pkg/voteposting/platforms/bluesky/platform.go#L85):
  1. Posts root → gets `rootRef` (`PostRef{URI, CID}`)
  2. Iterates replies: constructs `ReplyRef{Root: rootRef, Parent: parentRef}`, posts, advances `parentRef`
- This is the **exact pattern** to replicate for X, substituting `PostRef` with tweet IDs

#### Bluesky API Client (`pkg/bskyapi/client.go`)

- `CreateRecord(session, text, facets, replyTo *ReplyRef) (*PostRef, error)` at [client.go:121](pkg/bskyapi/client.go#L121)
- When `replyTo != nil`, adds `reply: {root: {uri, cid}, parent: {uri, cid}}` to the record
- Returns `*PostRef` (URI + CID) for chaining

### 3. X API v2 Reply Thread Support

From the [X API v2 docs](https://docs.x.com/x-api/posts/creation-of-a-post):

**Request payload for a reply:**

```json
{
  "text": "reply content",
  "reply": {
    "in_reply_to_tweet_id": "1346889436626259968"
  }
}
```

**Response (201 Created):**

```json
{
  "data": {
    "id": "1346889436626259968",
    "text": "..."
  }
}
```

Key facts:

- The `reply.in_reply_to_tweet_id` field is all that's needed for threading
- The response `data.id` is the tweet ID string needed for chain continuation
- Unlike Bluesky, X does not require separate root/parent references — just the parent tweet ID
- The X thread is simpler: each reply just references its parent (the API handles root tracking)

### 4. Character Limits

- **X Free tier**: 280 characters per tweet ([X docs](https://docs.x.com/fundamentals/counting-characters))
- **X Premium**: 10,000+ characters (current account uses this)
- **Bluesky**: 300 graphemes (enforced in `maxGraphemes` constant)

The current X format produces posts that routinely exceed 280 characters for multi-vote groups (the example in the task description is ~1,400+ characters). Switching to threads with 280-char posts would allow dropping X Premium.

### 5. X Handle Tagging

- X uses inline `@handle` substitution via `contactMapper.TagXHandlesInText(title)` at [format.go:39-41](pkg/voteposting/platforms/x/format.go#L39-L41)
- This replaces politician names with their X handles directly in the text
- In a thread model, tagging would occur in the root post title (same as now)
- No structural change needed for tagging

## Code References

- [pkg/xapi/client.go:20](pkg/xapi/client.go#L20) — `PostTweet()` — needs to return tweet ID and accept `inReplyTo` param
- [pkg/voteposting/platforms/x/format.go:20](pkg/voteposting/platforms/x/format.go#L20) — `FormatVoteGroupPost()` — needs thread-aware rewrite
- [pkg/voteposting/platforms/x/platform.go:11-80](pkg/voteposting/platforms/x/platform.go#L11-L80) — `XContent`, `XPlatform.Format()`, `XPlatform.Post()` — needs thread loop
- [pkg/voteposting/platforms/x/format_test.go](pkg/voteposting/platforms/x/format_test.go) — tests need updating for thread output
- [pkg/voteposting/platforms/bluesky/format.go:28](pkg/voteposting/platforms/bluesky/format.go#L28) — `FormatVoteThread()` — reference pattern for thread formatting
- [pkg/voteposting/platforms/bluesky/platform.go:85](pkg/voteposting/platforms/bluesky/platform.go#L85) — `Post()` — reference pattern for thread posting loop
- [pkg/voteposting/platforms/interface.go](pkg/voteposting/platforms/interface.go) — `Platform`/`Content` interfaces — no changes needed
- [pkg/voteposting/prepare.go](pkg/voteposting/prepare.go) — `PostToPlatform()` — no changes needed
- [pkg/voteposting/voteformat/voteformat.go](pkg/voteposting/voteformat/voteformat.go) — shared formatting utilities — reusable as-is

## Architecture Documentation

### Current patterns

- Each platform has its own `format.go` (formatting), `platform.go` (posting + Content type), and the API client in `pkg/<platform>api/`
- The `platforms.Platform` interface decouples the posting loop from platform specifics
- `Content` is a minimal interface (`String() string`) — each platform type-asserts to its concrete struct in `Post()`
- Thread management is **entirely within the platform layer** — `prepare.go` calls `Format()` then `Post()` without knowing about threads
- Bluesky already established the thread pattern: format returns a slice of posts, Post() chains them via API

### Shared formatting utilities in `voteformat` package

These are used by both X and Bluesky formatters and are fully reusable:

- `FormatVoteDate()`, `SelectBestTitle()`, `CleanVoteTitle()`, `CleanVoteSubtitle()`
- `GetVoteResultEmoji()`, `GetVoteResultText()`
- `FormatVoteCountsLong()` (X-style: `📊 87 Ja | 25 Nein | 0 Enthaltung | 13 Abwesend`)
- `FormatVoteCounts()` (Bluesky-style, shorter: `📊 87 Ja | 25 Nein | 0 Enthaltung | 13 Abw.`)
- `IsAuswahlVote()`, `IsGenericAntragTitle()`, `IsUnsupportedVoteType()`
- `GenerateVoteLink()`, `GenerateTraktandumLink()`, `GenerateGeschaeftLink()`

## Implementation Scope (files to change)

### 1. `pkg/xapi/client.go` — Add reply support & return tweet ID

- Change `PostTweet()` signature to accept optional `inReplyToTweetID string` parameter
- Add `reply: {in_reply_to_tweet_id: ...}` to the JSON payload when the param is non-empty
- Parse the response JSON to extract `data.id` and return it (change return from `error` to `(string, error)`)

### 2. `pkg/voteposting/platforms/x/format.go` — Thread formatting

- Introduce `XPost` struct (text string, analogous to `BlueskyPost`)
- Rename/replace `FormatVoteGroupPost()` with `FormatVoteThread()` returning `[]*XPost`
- Root post: header + title + result (single vote) + `"👇 Details im Thread"` — must fit in 280 chars
- Reply posts: bin-pack vote entries within 280 chars each (same algorithm as Bluesky's `buildReplyPosts`)
- Link goes on last reply
- Keep X handle tagging on root post title
- Consider: for single-vote groups that fit in 280 chars, you could skip threading entirely (just one post) — optional optimization

### 3. `pkg/voteposting/platforms/x/platform.go` — Thread posting

- Change `XContent` to hold `thread []*XPost` instead of `message string`
- Update `XPlatform.Format()` to call the new `FormatVoteThread()`
- Update `XPlatform.Post()` to:
  1. Post root → get tweet ID from response
  2. Iterate replies: call `PostTweet(..., parentTweetID)` → get new tweet ID → advance parent
- Update `String()` for logging/preview (similar to `BlueskyContent.String()`)

### 4. `pkg/voteposting/platforms/x/format_test.go` — Test updates

- Update tests to work with new `[]*XPost` return type
- Add tests for: thread splitting, 280-char limit enforcement, single-vote-fits-in-one-post case, link placement on last reply

### Files NOT changed

- `pkg/voteposting/platforms/interface.go` — interfaces are already flexible enough
- `pkg/voteposting/prepare.go` — posting loop is platform-agnostic
- `main.go` — no changes needed
- `pkg/voteposting/voteformat/` — shared utilities reused as-is

## Open Questions

1. **280 vs 10,000 char limit**: Should the formatter target 280 chars (free tier) or make the limit configurable? Hardcoding 280 is simpler and achieves the goal of dropping Premium.
   > **Decision**: Use a configurable `maxChars` constant (like Bluesky's `maxGraphemes = 300`). Default to a relaxed limit (~2,000 chars) that leverages X Premium's longer posts. This keeps threads compact (fewer API calls) while still using the thread architecture. If Premium is ever dropped, just lower the constant to 280.
2. **Single-post optimization**: For vote groups that fit within 280 chars (e.g. single vote with short title), should they be posted as a single tweet without threading? The Bluesky implementation always creates a thread (root + replies), even for single votes. Matching this behavior is simpler but creates a 2-post thread even when unnecessary.
   > Align with Bluesky for consistency, even if it means some single-vote posts are threads. The user experience is similar, and it simplifies the code by always using the thread pattern.
3. **FormatVoteCountsLong vs FormatVoteCounts**: The current X format uses `FormatVoteCountsLong()` (e.g. `📊 87 Ja | 25 Nein | 0 Enthaltung | 13 Abwesend`). This is 40+ chars per vote line. Switching to `FormatVoteCounts()` (shorter, used by Bluesky) would save space in 280-char replies. Or keep the long format since replies have more room per vote than the old single-post approach.
   > **Decision**: Keep `FormatVoteCountsLong()` for X. With Premium's relaxed char limit, space is not a constraint, and the longer format is more readable. X and Bluesky don't need identical formatting — visual similarity is sufficient.
4. **Rate limits**: ~~X free API has 17 POST requests per 24 hours.~~ **UPDATED**: X has moved to a **pay-per-use credit model** (no more fixed tiers/free tier). See details below.
   > **Decision**: Not a concern. 10,000 posts/24h rate limit is ample. With Premium + relaxed char limit, threads will be compact (root + 1–2 replies), so API call count stays low. `X_MAX_POSTS_PER_RUN` remains a logical vote-group counter.

5. **Backwards compatibility of `FormatVotePost()`**: The single-vote convenience function at [format.go:14](pkg/voteposting/platforms/x/format.go#L14) is used by `cmd/generate_vote_post/main.go`. Decide whether to keep it or update that command too.
   > generate_vote_post is only used for manual testing and previewing, it should show the votes as they will appear in the actual posts. Therefore, it makes sense to update it to use the new thread format, even if it means it will show multiple posts for multi-vote groups. The main purpose is to reflect the real output as closely as possible. It should be clear what is the main post and what is the threat. And it should be aligned with bluesky for consistency. Also ok to update bluesky if needed.

---

## Follow-up Research: X API Pricing & Character Limits (2026-04-01)

### Q1: Do replies have the same 280-char limit as root posts?

**Yes.** The 280-character limit applies to **all posts**, including replies. The X API docs on [counting characters](https://docs.x.com/fundamentals/counting-characters) make no distinction between root posts and replies. The character counting rules are:

- Latin characters, punctuation: **1 weight** each
- Emojis: **2 weight** each (regardless of ZWJ complexity)
- CJK characters: **2 weight** each
- URLs: always **23 characters** (t.co shortening)
- **@mentions auto-populated at the start of replies don't count** (this is important — when replying to your own tweet, the `@yourhandle` mention is auto-populated and costs 0 characters)

So the idea of "putting all votes in one long reply to save API calls" **would not work** — replies are subject to the same 280-char limit. The thread approach with bin-packed 280-char replies remains the correct strategy.

### Q2: Can a thread (root + replies) be sent in one API call?

**No.** The X API v2 `POST /2/tweets` endpoint creates **exactly one post per request**. There is no batch/thread endpoint. Each reply in a thread requires a separate `POST /2/tweets` call with `reply.in_reply_to_tweet_id` set to the parent tweet's ID. A thread of N posts = N API calls.

### Q3: X API Pricing (as of April 2026)

X has **completely replaced** the old tiered subscription model (Free / Basic $100/mo / Pro $5,000/mo) with a **pay-per-use credit model**:

| Resource | Unit Cost | Description |
|---|---|---|
| Posts: Read | $0.005/resource | Per post fetched |
| User: Read | $0.010/resource | Per user fetched |
| DM Event: Read | $0.010/resource | Per DM event fetched |
| **Content: Create** | **$0.010/request** | **Per post/media creation request** |
| DM Interaction: Create | $0.015/request | Per DM sent |
| User Interaction: Create | $0.015/request | Per follow/like/retweet |

Key changes:
- **No fixed monthly fees** — pay only for what you use
- **No monthly caps** on objects
- **Less restrictive rate limits** compared to old tiers

### Rate limits for `POST /2/tweets` (new model)

| | App-level | User-level |
|---|---|---|
| POST /2/tweets | **10,000/24hrs** | **100/15min** |

This is **dramatically more generous** than the old model (which had 17 posts/24h on free tier). Rate limits are no longer a practical concern for this use case.

### X Premium Pricing (personal subscription, not API)

X Premium allows **longer posts up to 25,000 characters** (including replies). This is the feature currently used to post everything in a single long tweet.

| Tier | Switzerland (CHF/mo) | Switzerland (CHF/yr) | Longer posts? |
|---|---|---|---|
| **Basic** | 2.63 | 27.81 | ✅ Yes (up to 25,000 chars) |
| **Premium** | 7.00 | 73.00 | ✅ Yes + blue checkmark |
| **Premium+** | 36.00 | 356.00 | ✅ Yes + no ads |

Note: **Basic** tier (CHF 2.63/mo) already includes longer posts. The current Premium subscription (CHF 7/mo) adds the blue checkmark but isn't needed for long-post functionality alone.

### Cost Comparison: Keep Premium vs. Drop Premium (threads)

The old free API tier (17 posts/24h) no longer exists. Under the new pay-per-use model, every `POST /2/tweets` costs **$0.01** regardless of whether it's a root post or a reply.

**Option A: Keep X Premium (single long posts)**
- X Premium Basic subscription: **CHF 2.63/mo** (or CHF 7/mo for checkmark)
- API cost: 1 API call per vote group × $0.01 = **$0.01/group**
- Monthly (30 days, ~5 groups/day): ~150 calls → **~$1.50/mo API** + CHF 2.63–7/mo subscription

**Option B: Drop X Premium (280-char threads)**
- No subscription: **CHF 0/mo**
- API cost: ~3–6 API calls per vote group (root + replies) × $0.01
- Monthly (30 days, ~5 groups/day): ~600–900 calls → **~$6–9/mo API**

| | Subscription | API cost/mo | Total/mo |
|---|---|---|---|
| **A: Premium Basic + single post** | ~CHF 2.63 | ~$1.50 | **~CHF 4–5** |
| **A: Premium + checkmark + single post** | ~CHF 7.00 | ~$1.50 | **~CHF 8–9** |
| **B: No Premium + 280-char threads** | CHF 0 | ~$6–9 | **~CHF 6–9** |

**Conclusion**: The costs are roughly comparable (~CHF 5–9/mo either way). See final decision below.

---

## Final Decision: Keep X Premium + Implement Threads with Relaxed Limit

**Approach**: Keep X Premium subscription. Implement the thread architecture (root + replies), but use a relaxed per-post character limit (~2,000 chars) that leverages Premium's "longer posts" feature (up to 25,000 chars).

### Rationale

1. **X Premium is worth keeping**: Professional appearance (blue checkmark), flat predictable cost (CHF 7/mo), and longer posts reduce API call count significantly.
2. **Still implement threads**: The thread architecture is needed as the foundation for future extensions (Fraktionsresultate, additional context). Without threads, adding new content types means making the single post even longer.
3. **Relaxed limit keeps threads compact**: With ~2,000 chars per post, most vote groups fit in root + 1 reply (vs. root + 5–6 replies at 280 chars). This saves API calls ($0.01 each) and rate limit headroom.
4. **Visual alignment with Bluesky**: Same content, same order — root has header + title + thread hint, replies have vote details + link. Bluesky distributes across more replies (300-grapheme limit), X across fewer. Visually similar, not identical.
5. **Future-proof**: If Premium is ever dropped, lowering `maxChars` from ~2,000 to 280 is a one-line change. The bin-packing algorithm handles the rest automatically.

### Concrete thread structure on X (with Premium)

**Root post** (~200 chars):
```
🗳️  Gemeinderat | Abstimmung vom 25.03.2026

Weisung vom 17.04.2024: Amt für Städtebau, öffentlicher Gestaltungsplan «Marina Tiefenbrunnen», Zürich-Seefeld, Kreis 8

👇 Details im Thread
```

**Reply 1** (~all votes fit in one reply with Premium):
```
✅ Rückweisungsantrag
📊 87 Ja | 25 Nein | 0 Enthaltung | 13 Abwesend

✅ Änderungsantrag 4 zu Dispositivziffer 1
📊 102 Ja | 9 Nein | 0 Enthaltung | 14 Abwesend

... (all remaining votes)

🔗 https://...
```

**Future Reply 2** (Fraktionsresultate):
```
SP: 32 Ja / 0 Nein
FDP: 18 Ja / 5 Nein
...
```

### Comparison across platforms

| | Bluesky (300 graphemes) | X with Premium (~2,000 chars) |
|---|---|---|
| Root | Header + title + thread hint | Header + title + thread hint |
| Vote details | ~5–6 replies (bin-packed) | **1–2 replies** |
| Future: Fraktionen | Additional replies | Additional reply |
| API calls/group | ~6–7 | **~2–3** |
| Cost/group | n/a | ~$0.02–0.03 |

### Updated implementation scope

Same 4 files as identified in the original research, with these adjustments:

1. **`pkg/xapi/client.go`** — Add `inReplyToTweetID` param + return tweet ID (unchanged)
2. **`pkg/voteposting/platforms/x/format.go`** — New `FormatVoteThread()` with `maxChars` constant (~2,000). Keep `FormatVoteCountsLong()` (not switching to short format). Bin-packing algorithm same as Bluesky but with the higher limit.
3. **`pkg/voteposting/platforms/x/platform.go`** — Thread posting loop (unchanged)
4. **`pkg/voteposting/platforms/x/format_test.go`** — Tests for thread output (unchanged)
