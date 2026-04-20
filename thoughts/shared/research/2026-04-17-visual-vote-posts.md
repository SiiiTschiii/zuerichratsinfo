---
date: 2026-04-17T12:00:00+02:00
researcher: copilot
topic: "Visual vote post generation for Instagram carousel"
tags: [research, codebase, image-generation, instagram, visual-posts]
status: complete
last_updated: 2026-04-17
---

# Research: Visual Vote Post Generation

**Date**: 2026-04-17

## Research Question

How to generate image-based vote posts (carousel of 3 images) for Instagram and other visual platforms. What Go libraries exist, what data is already available, how to handle color rotation state, and what fonts/colors/layout to use.

## Summary

The codebase already has all the data and formatting logic needed to populate image content. The `voteformat` package provides cleaned titles, vote counts, result emojis, and Fraktion breakdowns. The `Platform` interface cleanly separates formatting from posting. Go's standard `image` and `image/draw` packages plus `golang.org/x/image/font` can generate JPEG images without external dependencies. For color rotation, a simple modular index derived from the vote group's position in the batch (or a hash of GeschaeftGrNr) avoids the need for persistent color state.

## Detailed Findings

### 1. Data Available for Image Content

All fields needed for the 3 carousel images come from `zurichapi.Abstimmung` (`pkg/zurichapi/types.go:89-118`):

**Image 1 — Title card:**

- Result emoji: `voteformat.GetVoteResultEmoji(vote.Schlussresultat)` → `"✅"` / `"❌"`
- Result text: `voteformat.GetVoteResultText(vote.Schlussresultat)` → `"Angenommen"` / `"Abgelehnt"`
- Title: `voteformat.SelectBestTitle(vote.TraktandumTitel, vote.GeschaeftTitel)` → cleaned via `CleanVoteTitle`
- Date: `voteformat.FormatVoteDate(vote.SitzungDatum)` → `"DD.MM.YYYY"`
- Tagged accounts (from contacts.yaml via `contactMapper`)

**Image 2 — Overall result:**

- `vote.AnzahlJa`, `AnzahlNein`, `AnzahlEnthaltung`, `AnzahlAbwesend` (all `*int`)
- Formatted: `voteformat.FormatVoteCounts(counts)` → `"📊 101 Ja | 12 Nein | 0 Enth. | 5 Abw."`
- Also supports Auswahl votes (A/B/C/D/E)

**Image 3 — Fraktion breakdown:**

- `vote.Stimmabgaben.Stimmabgabe` → slice of `Stimmabgabe` with `Fraktion` and `Abstimmungsverhalten`
- Aggregated: `voteformat.AggregateFraktionCounts(stimmabgaben)` → `map[string]*FraktionCounts`
- Formatted: `voteformat.FormatFraktionBreakdown(counts)` — already produces the exact text layout needed:
  ```
  🏛️ Fraktionen (Ja/Nein/Enth/Abw):
  SP 34/0/0/3
  FDP 18/0/0/5
  ...
  ```
- Factions sorted by total member count descending

### 2. Existing Vote Formatting Pipeline

The text version of what each image should show already exists:

| Image          | Existing text source                                                                  | Location                 |
| -------------- | ------------------------------------------------------------------------------------- | ------------------------ |
| 1 — Title      | `buildRootPost()` in both `bluesky/format.go:67-143` and `x/format.go:65-135`         | Header + emoji + title   |
| 2 — Counts     | `FormatVoteCounts()` / `FormatVoteCountsLong()` in `voteformat/voteformat.go:153-200` | Short/long count line    |
| 3 — Fraktionen | `FormatFraktionBreakdown()` in `voteformat/fraktion.go:55-160`                        | Full formatted breakdown |

The image generator can call the same `voteformat` functions to get text content, then render it onto colored backgrounds.

### 3. Platform Interface & Integration Point

`platforms.Platform` (`pkg/voteposting/platforms/interface.go:1-27`) has 4 methods:

- `Format(votes []Abstimmung) (Content, error)` — generate platform-specific content
- `Post(content Content) (shouldContinue bool, err error)` — publish
- `MaxPostsPerRun() int`
- `Name() string`

An Instagram platform would implement this interface. Its `Format()` would generate images (3 per carousel) + caption text. Its `Post()` would upload to GitHub Pages, create IG containers, poll for PUBLISHED, then clean up.

### 4. Go Image Generation Options

**Standard library (no external deps):**

- `image`, `image/color`, `image/draw`, `image/jpeg` — create images, draw rectangles, encode JPEG
- `golang.org/x/image/font` + `golang.org/x/image/font/opentype` — render TrueType/OpenType text onto images
- `golang.org/x/image/font/gofont/goregular` / `gobold` — bundled Go fonts (no external font files needed)

**Third-party options (for reference, not recommended):**

- `github.com/fogleman/gg` — 2D drawing library, wraps the standard image package. Adds convenience for text wrapping (`DrawStringWrapped`), rounded rectangles, and context-style drawing.
- `github.com/golang/freetype` — FreeType font rendering for Go (lower level).

**Recommendation: use Go stdlib + `golang.org/x/image`.** The layout is simple enough (colored background, positioned text lines, shadow offset) that `gg`'s convenience wrappers aren't needed. This keeps dependencies minimal (`go.mod` currently has only `gopkg.in/yaml.v3`).

**Output format: JPEG.** Instagram only accepts JPEG for image posts (see `pkg/igapi/README.md`). Go's `image/jpeg` encodes directly from `image.Image` — no intermediate format or rasterization step needed. SVG would require a separate rasterization tool (librsvg, headless browser) adding unnecessary complexity.

### 5. Color Rotation Strategy

**Current state tracking:** `VoteEntry` in `pkg/votelog/votelog.go:21-24` stores only `{ID, PostedAt}`. No custom metadata fields exist.

**Options for deterministic color assignment:**

| Option                            | Pros                                                           | Cons                                                  |
| --------------------------------- | -------------------------------------------------------------- | ----------------------------------------------------- |
| **A. Hash-based (GeschaeftGrNr)** | Stateless, deterministic, same Geschäft always gets same color | No sequential round-robin                             |
| **B. Index from batch position**  | Simple `i % len(colors)`, no state needed                      | Color depends on batch composition; re-runs may shift |
| **C. Counter in vote log**        | True round-robin across all posts over time                    | Requires schema change to posted_votes JSON           |
| **D. Separate color state file**  | Clean separation                                               | Another file to commit via GitHub Actions             |

**Option A is simplest and most robust.** Example:

```go
colors := []color.RGBA{
    {0x00, 0x69, 0xC7, 0xFF}, // Blue (matches logo #0069C7)
    {0xE6, 0x3E, 0x31, 0xFF}, // Red
    {0x2E, 0x8B, 0x57, 0xFF}, // Green
    {0xF5, 0xA6, 0x23, 0xFF}, // Amber
}
hash := fnv.New32a()
hash.Write([]byte(votes[0].GeschaeftGrNr))
colorIndex := int(hash.Sum32()) % len(colors)
```

This way the same Geschäft always gets the same color, no state to persist, and re-runs are idempotent. The color set of ~4 deliberately avoids political party colors to stay neutral.

### 6. Font & Typography Considerations

**Bundled Go fonts** (`golang.org/x/image/font/gofont`):

- `goregular`, `gobold`, `goitalic`, `gomono` — clean sans-serif, Apache 2.0 licensed, embedded in Go module (no font files to ship)
- Adequate for a clean infographic look

**Alternative: system or custom font:**

- A `.ttf`/`.otf` file in `assets/fonts/` loaded via `opentype.Parse`
- Allows using a specific typeface (e.g., Inter, Source Sans, or a Swiss-style font)
- Adds a binary asset to the repo (~100-300 KB per font weight)

**Recommendation:** Start with Go bundled fonts (`gobold` for headings, `goregular` for body). Switch to a custom font later if the visual quality needs upgrading.

### 7. Image Layout Concept

Instagram carousel images should be **1080×1080 px** (square, recommended by Instagram).

**Image 1 — Title card:**

```
┌──────────────────────────────┐
│  (colored background)        │
│                              │
│  🗳️ Gemeinderat Zürich       │
│  Abstimmung vom DD.MM.YYYY   │
│                              │
│  ✅ Angenommen:              │
│  [Title text, wrapped,       │
│   large font]                │
│                              │
│                              │
└──────────────────────────────┘
```

**Image 2 — Overall result:**

```
┌──────────────────────────────┐
│  (same colored background)   │
│                              │
│  📊 Ergebnis                 │
│                              │
│    101 Ja                    │
│     12 Nein                  │
│      0 Enthaltung            │
│     12 Abwesend              │
│                              │
│  (optional: horizontal bar)  │
└──────────────────────────────┘
```

**Image 3 — Fraktion breakdown:**

```
┌──────────────────────────────┐
│  (same colored background)   │
│                              │
│  🏛️ Fraktionen               │
│                              │
│  SP          34/0/0/3        │
│  FDP         18/0/0/5        │
│  Grüne       18/0/0/0        │
│  GLP         14/0/0/1        │
│  SVP          0/12/0/1       │
│  Mitte/EVP    8/0/0/2        │
│  AL           8/0/0/0        │
└──────────────────────────────┘
```

**Shadow effect:** Draw the same text offset by (2,2) px in a darker shade of the background color, then draw the white text on top. `gg` makes this trivial with `dc.SetColor()` + `dc.DrawStringWrapped()`.

### 8. Instagram Carousel API Flow

From `pkg/igapi/README.md`, the carousel flow is:

1. Create individual media containers for each image (3x `POST /{ig-user-id}/media` with `is_carousel_item=true`)
2. Create a carousel container (`POST /{ig-user-id}/media` with `media_type=CAROUSEL`, `children=<id1>,<id2>,<id3>`)
3. Publish the carousel (`POST /{ig-user-id}/media_publish` with `creation_id=<carousel_id>`)
4. Poll `GET /<container_id>?fields=status_code` until `PUBLISHED`
5. Clean up hosted images

Carousel limits: max 10 images, all cropped based on first image aspect ratio (1:1 default).

### 9. Existing Logo

`assets/logo.svg` is a 1024×1024 SVG with:

- White background (`#FFFFFF`)
- Blue fill (`#0069C7`) — this is the project's brand color
- Generated by potrace (vector trace of a bitmap)

The blue `#0069C7` should be one of the 4 rotation colors to maintain brand consistency.

## Code References

- `pkg/zurichapi/types.go:89-118` — `Abstimmung` struct with all vote fields
- `pkg/zurichapi/types.go:120-132` — `Stimmabgabe` (individual member vote)
- `pkg/voteposting/voteformat/voteformat.go:36-51` — result emoji/text
- `pkg/voteposting/voteformat/voteformat.go:52-57` — title selection
- `pkg/voteposting/voteformat/voteformat.go:70-115` — title/subtitle cleaning
- `pkg/voteposting/voteformat/voteformat.go:119-200` — vote counts formatting
- `pkg/voteposting/voteformat/fraktion.go:20-35` — faction count aggregation
- `pkg/voteposting/voteformat/fraktion.go:55-160` — faction breakdown formatting
- `pkg/voteposting/platforms/interface.go:1-27` — Platform interface
- `pkg/voteposting/platforms/bluesky/format.go:67-143` — Bluesky root post builder
- `pkg/voteposting/platforms/x/format.go:65-135` — X root post builder
- `pkg/voteposting/prepare.go:24-87` — `PrepareVoteGroups` orchestration
- `pkg/voteposting/prepare.go:97-182` — `PostToPlatform` posting loop
- `pkg/votelog/votelog.go:21-24` — `VoteEntry` struct (ID + PostedAt only)
- `pkg/voteposting/testfixtures/fixtures.go` — test vote data with realistic faction distributions
- `assets/logo.svg` — brand logo (blue #0069C7 on white)
- `pkg/igapi/README.md` — Instagram API flow documentation
- `go.mod` — current deps: only `gopkg.in/yaml.v3`

## Architecture Documentation

### Current patterns

- **Platform abstraction**: `platforms.Platform` interface with `Format` + `Post` methods
- **Shared formatting**: `voteformat` package used by all platform formatters
- **State persistence**: JSON files in `data/`, committed via GitHub Actions after each run
- **Thread structure**: Both X and Bluesky build root + replies; Instagram would use carousel instead
- **Test fixtures**: `testfixtures` package provides realistic vote data for testing

### Where image generation fits

- New package: `pkg/voteposting/imagegen/` (or `pkg/imagegen/`)
- Called by an Instagram platform's `Format()` method
- Also reusable for TikTok photo posts later
- Outputs `[]image.Image` or `[][]byte` (JPEG-encoded via `image/jpeg`)

## Decisions

1. **Font choice**: Start with bundled Go fonts (`gobold` for headings, `goregular` for body). They are clean, free, and require no asset management. Upgrade to a custom `.ttf` later if needed.
2. **Image library**: Go stdlib `image` + `golang.org/x/image/font` — no third-party deps like `fogleman/gg`.
3. **Output format**: JPEG — the only format Instagram accepts for image posts. Generated directly via `image/jpeg`.
4. **Bar chart on Image 2**: Start with just large numbers for simplicity. A bar chart can be added later.
5. **Caption text**: Duplicate the full thread text in the Instagram caption plus the link to the Gemeinderat's official vote page. Ensures accessibility for users who can't see the images clearly.
6. **Multi-vote groups (5+ sub-votes)**: Create 2 images per sub-vote (1 for overall result, 1 for Fraktion breakdown) to maintain readability. Instagram carousels allow up to **10 images**, so a group with ≤5 sub-votes fits (title image + 2×5 = 11 would exceed, so cap at 4 sub-votes per carousel = title + 8 + link considerations). Groups exceeding 10 images need a strategy (e.g. combine smaller sub-votes or split into multiple posts).
7. **join overall result and Fraktion breakdown**: Explore Combining overall result and Fraktion breakdown into a single image to save carousel slots during image generation verification. This saves space for more sub-votes if needed and keeps the overall result + breakdown visually connected. The layout would have the overall result at the top, then a clear separation (line or spacing), then the Fraktion breakdown below.
8. **Auswahl votes (A/B/C)**: Same number-based layout as Ja/Nein. Example:
9. **Multi-vote overflow**: Cap at 4 sub-votes per carousel (1 title + 8 result/Fraktion images = 9 ≤ 10 limit). Log a warning if exceeded. Revisit if this becomes common.
10. **Shadow style**: Subtle drop shadow (dark offset) or text outline? To be explored during image generation prototyping.

**Auswahl vote — Image 2 layout:**

```
┌──────────────────────────────┐
│  (colored background)        │
│                              │
│  📊 Ergebnis                 │
│                              │
│     45 A                     │
│     38 B                     │
│     30 C                     │
│     12 Abwesend              │
│                              │
└──────────────────────────────┘
```

**Auswahl vote — Image 3 (Fraktion) layout:**

```
┌──────────────────────────────┐
│  (same colored background)   │
│                              │
│  🏛️ Fraktionen (A/B/C/Abw)   │
│                              │
│  SP          20/10/4/3       │
│  FDP          5/18/0/2       │
│  Grüne       10/0/8/0        │
│  GLP          5/5/4/1        │
│  SVP          0/5/12/2       │
│  Mitte/EVP    3/0/2/2        │
│  AL           2/0/0/1        │
└──────────────────────────────┘
```
