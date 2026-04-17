# Visual Vote Post Generation — Implementation Plan

## Overview

Generate JPEG images for Instagram carousel posts showing vote results from the Zürich Gemeinderat. Each vote group produces a carousel of images: a title card, per-vote result + Fraktion breakdown images. The work is split into an exploration phase (local) followed by pipeline integration phases (GitHub issues).

## Current State Analysis

- All vote data and text formatting already exists in `pkg/voteposting/voteformat/`
- Platform abstraction via `platforms.Platform` interface (`pkg/voteposting/platforms/interface.go`)
- Test fixtures with realistic faction data in `pkg/voteposting/testfixtures/fixtures.go`
- CLI tool patterns established in `cmd/generate_vote_post/` (live data) and `cmd/post_fixture/` (fixture data)
- No image generation code exists yet
- `go.mod` has minimal deps (`gopkg.in/yaml.v3` only)

## Desired End State

A `pkg/imagegen/` package that generates carousel JPEG images from vote data, integrated into an Instagram `Platform` implementation that publishes carousels via the Content Publishing API with temporary GitHub Pages hosting.

## What We're NOT Doing

- Bar charts (numbers only for now)
- Custom fonts (start with Go bundled fonts)
- TikTok integration (reusable later)
- Facebook integration (separate effort)

## Implementation Approach

4 phases. Phase 1 is local exploration with no GitHub issue. Phases 2–4 each map to a GitHub issue.

---

## Phase 1: Image Generation Exploration (local)

### Overview

Build the `pkg/imagegen/` package and a `cmd/generate_vote_image/` CLI tool that renders carousel images from test fixtures to local JPEG files. Iterate on layout, colors, fonts, and shadows until the output looks good.

### Changes Required

#### 1. New package: `pkg/imagegen/`

**File**: `pkg/imagegen/imagegen.go`

Core rendering functions using Go stdlib (`image`, `image/color`, `image/draw`, `image/jpeg`) + `golang.org/x/image/font` + `golang.org/x/image/font/opentype` + `golang.org/x/image/font/gofont/gobold` / `goregular`.

```go
// GenerateCarousel produces carousel JPEG images for a vote group.
// Returns [][]byte (JPEG-encoded images).
func GenerateCarousel(votes []zurichapi.Abstimmung) ([][]byte, error)

// Color palette — hash-based on GeschaeftGrNr
func SelectColor(geschaeftGrNr string) color.RGBA
```

Image types to implement:

- **Title card** (1080×1080): header, result emoji/text, wrapped title
- **Result + Fraktion card** (1080×1080): overall counts at top, faction breakdown below (explore combining into one image per Decision #7)
- **Auswahl variant**: same layout, A/B/C columns instead of Ja/Nein

Rendering approach:

- Fill background with selected color
- Draw text lines at computed Y positions using `font.Drawer`
- Shadow: draw text at (x+2, y+2) in darkened color, then draw white text on top
- Use `voteformat` functions for all text content

#### 2. New CLI tool: `cmd/generate_vote_image/main.go`

Pattern: follows `cmd/post_fixture/main.go` — fixture-based, no credentials needed.

```go
func main() {
    fixture := flag.String("fixture", "all", "fixture name or 'all'")
    outDir  := flag.String("out", "out/images", "output directory for generated JPEGs")
    flag.Parse()

    // Load fixtures from testfixtures.AllFixtures()
    // For each fixture:
    //   images, _ := imagegen.GenerateCarousel(votes)
    //   Write each image to <outDir>/<fixture>_<n>.jpg
}
```

#### 3. Dependencies

Add to `go.mod`:

- `golang.org/x/image` (for `font`, `font/opentype`, `font/gofont/*`)

### Milestone

`cmd/generate_vote_image` produces JPEG files in `out/images/` that look good when opened locally.

### Manual Verification

1. **Run the tool:**
   ```bash
   go run cmd/generate_vote_image/main.go -fixture single-angenommen -out out/images
   ```
2. **Open generated JPEGs** — verify:
   - [ ] Title card: header, date, result emoji, wrapped title text, colored background
   - [ ] Result card: vote counts displayed clearly, Fraktion breakdown below
   - [ ] All images in a group share the same background color
   - [ ] Different fixtures get different colors
   - [ ] Text is readable at 1080×1080 (not too small, not clipped)
   - [ ] Shadow/outline effect looks good
3. **Test all fixture types:**

   ```bash
   go run cmd/generate_vote_image/main.go -fixture all -out out/images
   ```

   - [ ] Single vote angenommen
   - [ ] Single vote abgelehnt
   - [ ] Long title (truncation/wrapping works)
   - [ ] Multi-vote group (multiple result images)
   - [ ] Auswahl vote (A/B/C layout)

4. **Verify JPEG compliance:**
   - [ ] Files open in Preview/browser without errors
   - [ ] File size reasonable (< 500 KB per image at quality 90)

### Automated Verification

- [x] `go build ./...` passes
- [x] `go test ./pkg/imagegen/...` passes (basic tests: output is valid JPEG, correct dimensions, correct number of images per fixture)
- [x] `go vet ./...` passes

**⏸️ Pause here. Iterate on visuals locally until satisfied before proceeding.**

---

## Phase 2: Integrate imagegen into Instagram Platform (GitHub Issue)

> **GitHub Issue Title**: Implement Instagram platform with image carousel formatting
>
> **Labels**: `enhancement`, `instagram`

### Overview

Create `pkg/voteposting/platforms/instagram/` implementing the `Platform` interface. The `Format()` method calls `imagegen.GenerateCarousel()` and builds an `InstagramContent` struct containing JPEG images + caption text. The `Post()` method is a stub (prints preview, no-op) — real posting comes in Phase 3.

### Changes Required

#### 1. `pkg/voteposting/platforms/instagram/format.go`

- `FormatCarousel(votes []zurichapi.Abstimmung) (*InstagramContent, error)`
- Calls `imagegen.GenerateCarousel(votes)` for images
- Builds caption text: full vote text (like X/Bluesky thread text flattened) + vote page link
- Returns `InstagramContent{Images: [][]byte, Caption: string}`

#### 2. `pkg/voteposting/platforms/instagram/platform.go`

- `InstagramPlatform` struct implementing `platforms.Platform`
- `Format()` → delegates to `FormatCarousel()`
- `Post()` → stub: logs "would post N images" + caption preview, returns `true, nil`
- `MaxPostsPerRun()` → configurable (default 5)
- `Name()` → `"instagram"`

#### 3. Wire into `cmd/generate_vote_image/`

- Add `-platform instagram` flag to preview formatted caption alongside images

### Milestone

`cmd/generate_vote_image -fixture single-angenommen -platform instagram` outputs JPEG files + prints the caption text that would accompany the carousel.

### Manual Verification

1. **Preview Instagram format:**

   ```bash
   go run cmd/generate_vote_image/main.go -fixture single-angenommen -platform instagram
   ```

   - [ ] Caption text includes full vote details + link
   - [ ] Caption text does not exceed Instagram's 2,200 character limit
   - [ ] Number of images matches expected carousel structure (title + result/Fraktion per vote)
   - [ ] Multi-vote fixture respects 10-image carousel cap

2. **Dry-run via existing pipeline:**

   ```bash
   go run cmd/post_fixture/main.go -fixture single-angenommen -platform instagram
   ```

   - [ ] Preview output shows caption + "would post N images"
   - [ ] No errors, no real API calls

### Automated Verification

- [ ] `go test ./pkg/voteposting/platforms/instagram/...` — format tests: correct image count, caption content, character limits
- [ ] `go test ./...` — all existing tests still pass
- [ ] `go vet ./...` passes

---

## Phase 3: Instagram Client, Hosting & Publishing (GitHub Issue)

> **GitHub Issue Title**: Implement Instagram API client with GitHub Pages image hosting
>
> **Labels**: `enhancement`, `instagram`, `infrastructure`

### Overview

Implement the real `Post()` method: upload images to GitHub Pages, create Instagram media containers, publish the carousel, poll for PUBLISHED status, clean up hosted images.

### Changes Required

#### 1. `pkg/igapi/client.go`

Instagram Graph API client:

- `CreateMediaContainer(igUserID, imageURL, accessToken) (containerID, error)` — with `is_carousel_item=true`
- `CreateCarouselContainer(igUserID, childIDs, caption, accessToken) (containerID, error)`
- `PublishContainer(igUserID, containerID, accessToken) (mediaID, error)`
- `PollContainerStatus(containerID, accessToken) (status, error)` — poll until `PUBLISHED` or `ERROR`/`EXPIRED`

#### 2. Image hosting via GitHub Pages

Strategy: commit JPEGs to a `gh-pages` branch, wait for Pages deployment, use the public URL for Instagram container creation, delete after PUBLISHED.

- Helper in `pkg/igapi/hosting.go` or a small utility:
  - `UploadImages(images [][]byte, names []string) (urls []string, error)` — git commit to gh-pages branch
  - `CleanupImages(names []string) error` — git rm + commit + push

#### 3. Wire real posting in `platforms/instagram/platform.go`

- `Post()` calls hosting → igapi client → cleanup
- Poll container status in a loop (max 5 min, 1 poll/10s)
- On failure: log error, leave images hosted (manual cleanup)

#### 4. Environment variables

- `IG_USER_ID` — Instagram professional account ID
- `IG_ACCESS_TOKEN` — long-lived **Page** access token (never expires, see below)
- `GITHUB_TOKEN` — for pushing to gh-pages branch (already available in Actions)

#### 5. Access token setup (one-time, verify during implementation)

Meta's token hierarchy allows obtaining a **never-expiring Page token**:

1. Generate a short-lived User Token in [Graph API Explorer](https://developers.facebook.com/tools/explorer/) with required permissions
2. Exchange for a long-lived User Token (~60 days):
   ```bash
   curl "https://graph.facebook.com/v25.0/oauth/access_token \
     ?grant_type=fb_exchange_token \
     &client_id=<APP_ID> \
     &client_secret=<APP_SECRET> \
     &fb_exchange_token=<SHORT_LIVED_TOKEN>"
   ```
3. Request Page tokens using the long-lived User Token:
   ```bash
   curl "https://graph.facebook.com/v25.0/me/accounts?access_token=<LONG_LIVED_USER_TOKEN>"
   ```
   The `access_token` returned per Page is a **long-lived Page token that does not expire** as long as:
   - The user remains an admin of the Page
   - App permissions are not revoked
4. Store as `IG_ACCESS_TOKEN` in GitHub Actions secrets — no renewal needed

**Ref**: https://developers.facebook.com/docs/facebook-login/guides/access-tokens/get-long-lived

**TODO**: Once verified during implementation, update `pkg/igapi/README.md` with the full token setup flow (replace the current short Access Token section).

### Milestone

Running `cmd/post_fixture/main.go -fixture single-angenommen -platform instagram` with real credentials publishes a carousel to Instagram.

### Manual Verification

1. **Publish a test carousel:**

   ```bash
   IG_USER_ID=... IG_ACCESS_TOKEN=... \
   go run cmd/post_fixture/main.go -fixture single-angenommen -platform instagram
   ```

   - [ ] Images appear on GitHub Pages URL before Instagram fetch
   - [ ] Instagram carousel published successfully (visible in app)
   - [ ] All 3 images display correctly in carousel
   - [ ] Caption text renders correctly with link
   - [ ] Hosted images cleaned up from gh-pages after PUBLISHED status

2. **Error handling:**
   - [ ] Invalid token → clear error message, no partial state
   - [ ] Image hosting failure → error before any IG API calls

### Automated Verification

- [ ] `go test ./pkg/igapi/...` — client tests with mocked HTTP responses
- [ ] `go test ./pkg/voteposting/platforms/instagram/...` — post tests with injected mock client
- [ ] `go test ./...` — all tests pass
- [ ] `go vet ./...` passes

---

## Phase 4: End-to-End Pipeline & GitHub Actions (GitHub Issue)

> **GitHub Issue Title**: Add Instagram to automated vote posting pipeline
>
> **Labels**: `enhancement`, `instagram`, `github-actions`

### Overview

Wire Instagram into `main.go` alongside X and Bluesky. Add `posted_votes_instagram.json` tracking. Update GitHub Actions workflow.

### Changes Required

#### 1. `main.go`

- Add Instagram platform initialization (gated on `IG_USER_ID` + `IG_ACCESS_TOKEN` env vars)
- Load `votelog.Load("instagram")`
- Call `PostToPlatform(groups, instagramPlatform, igVoteLog, dryRun)`

#### 2. `data/posted_votes_instagram.json`

- New file, same schema as X/Bluesky: `{"platform": "instagram", "votes": []}`

#### 3. `.github/workflows/bot.yml`

- Add `IG_USER_ID` and `IG_ACCESS_TOKEN` secrets
- Add `data/posted_votes_instagram.json` to git add/commit step
- Ensure gh-pages push permissions if hosting is done within the same workflow

#### 4. `cmd/generate_vote_post/main.go`

- Add `-platform instagram` support for dry-run previews

### Milestone

The hourly GitHub Actions workflow posts new votes to Instagram automatically alongside X and Bluesky.

### Manual Verification

1. **Local dry-run with live data:**

   ```bash
   go run cmd/generate_vote_post/main.go -platform instagram -n 1
   ```

   - [ ] Preview shows caption + image count for real recent votes

2. **Full pipeline test (single run):**

   ```bash
   IG_USER_ID=... IG_ACCESS_TOKEN=... BLUESKY_HANDLE=... \
   go run main.go
   ```

   - [ ] Instagram posts appear alongside X/Bluesky posts
   - [ ] `data/posted_votes_instagram.json` updated with posted vote GUIDs
   - [ ] Same vote not re-posted on second run

3. **GitHub Actions:**
   - [ ] Trigger workflow manually via `workflow_dispatch`
   - [ ] Verify Instagram post appears
   - [ ] Verify `posted_votes_instagram.json` committed

### Automated Verification

- [ ] `go test ./...` — all tests pass
- [ ] `go vet ./...` passes
- [ ] GitHub Actions workflow runs without errors

---

## Testing Strategy

### Unit Tests

- `pkg/imagegen/` — valid JPEG output, correct dimensions (1080×1080), correct image count per fixture type
- `pkg/igapi/` — API client with mocked HTTP, container lifecycle
- `pkg/voteposting/platforms/instagram/` — format tests (caption, image count, carousel cap)

### Integration Tests

- `cmd/generate_vote_image/` — end-to-end: fixture → JPEG files on disk
- `cmd/post_fixture/` — fixture → Instagram post (manual, requires credentials)

### Manual Testing

- Visual inspection of generated images at each phase
- Real Instagram posts via `post_fixture` before enabling in production pipeline

## References

- Research: `thoughts/shared/research/2026-04-17-visual-vote-posts.md`
- Platform pattern: `pkg/voteposting/platforms/x/platform.go`, `pkg/voteposting/platforms/bluesky/platform.go`
- CLI tool pattern: `cmd/post_fixture/main.go`, `cmd/generate_vote_post/main.go`
- Fixture data: `pkg/voteposting/testfixtures/fixtures.go`
- Instagram API docs: `pkg/igapi/README.md`
