---
date: 2026-04-17T23:00:00+02:00
researcher: copilot
topic: "Image generation vertical spacing alignment"
tags: [research, codebase, imagegen, layout, spacing]
status: complete
last_updated: 2026-04-17
---

# Research: Image Generation Vertical Spacing Alignment

**Date**: 2026-04-17

## Research Question

The generated vote images have inconsistent vertical spacing between text elements of the same font size. Find a concept that simplifies the image generation and produces aligned, consistent top/bottom spacing for text at any given font size.

## Summary

The current layout uses ~15 different hardcoded pixel offsets (`y += 80`, `y += 60`, `y += 50`, etc.) scattered across 4 render functions. These magic numbers have no relationship to the font sizes being used, causing visual misalignment—especially between the stat numbers and their labels, and in the gap between sections. The Go `font.Face.Metrics()` API provides `Ascent`, `Descent`, and `Height` values that can derive consistent spacing from any font size. A simple **layout cursor** abstraction (a struct tracking the Y position with methods that advance based on font metrics) would replace all magic numbers with a single, predictable spacing model.

## Detailed Findings

### 1. Current Spacing Approach

All vertical positioning is done by manually incrementing a `y int` variable with hardcoded pixel values. The offsets used across the codebase:

**`renderCombinedCard`** (`pkg/imagegen/imagegen.go:265-395`):

- `y = padding + 32` — initial header position (92px from top)
- `y += 80` — after header line (to verdict)
- `y += 80` — after verdict (to title)
- `y += int(titleFontSize * 1.4)` — title lines (only place using font-relative spacing)
- `y += 25` — gap before separator
- `y += 50` — after separator line (to stats)
- Stats: `y += 60` (number row), `y += 35` (label row) via `drawStatColumns`
- `y += 25` — gap before second separator
- `y += 50` — after separator (to party table)
- Party table: `y += 32` (header row), `y += 30` (each party row)

**`renderResultCard`** (`pkg/imagegen/imagegen.go:602-672`):

- `y = padding + 50` — initial position (110px from top)
- `y += 60` — after subtitle
- `y += 80` — after verdict
- `y += 50` — after separator
- Same stat/party spacing as combined card

**`renderTitleCard`** (`pkg/imagegen/imagegen.go:565-600`):

- `y = padding + 60` — initial position (120px from top)
- `y += 80` — after header
- `y += 50` — per wrapped text line

**`drawStatColumns`** (`pkg/imagegen/imagegen.go:425-447`):

- `y += 60` — after drawing number row (font size 48)
- `y += 35` — after drawing label row (font size 22)

**`drawFraktionTable`** (`pkg/imagegen/imagegen.go:460-540`):

- `y += 32` — header row (font size 22)
- `y += 30` — each data row (font size 22)

### 2. Font Sizes in Use

| Purpose               | Font      | Size  | Used in                  |
| --------------------- | --------- | ----- | ------------------------ |
| Header date line      | goregular | 26    | combinedCard             |
| Verdict               | gobold    | 64    | combinedCard             |
| Verdict (result card) | gobold    | 56    | resultCard               |
| Title (adaptive)      | gobold    | 34→20 | combinedCard             |
| Stat numbers          | gobold    | 48    | combinedCard, resultCard |
| Stat labels           | goregular | 22    | combinedCard, resultCard |
| Party names           | gobold    | 22    | combinedCard, resultCard |
| Party numbers         | goregular | 22    | combinedCard, resultCard |
| Regular text          | goregular | 36    | titleCard, resultCard    |
| Small text            | goregular | 28    | resultCard               |
| Bold heading          | gobold    | 48    | titleCard, resultCard    |

### 3. Go `font.Face.Metrics()` API

The `golang.org/x/image/font` package provides font metrics via `face.Metrics()`:

```go
type Metrics struct {
    Height  fixed.Int26_6  // recommended line-to-line spacing
    Ascent  fixed.Int26_6  // distance from baseline to top of glyph
    Descent fixed.Int26_6  // distance from baseline to bottom of glyph
    // ...
}
```

- **Height** = recommended baseline-to-baseline distance (includes leading)
- **Ascent** = distance from baseline upward to tallest glyph
- **Descent** = distance from baseline downward to lowest glyph

For a font at size 48, approximate values (Go fonts):

- Ascent ≈ 46px, Descent ≈ 12px, Height ≈ 67px

These values provide a principled way to compute:

- **Line height** within same-font blocks: `Height.Ceil()`
- **Gap between blocks**: A fraction of the larger font's Height (e.g., `Height/2`)
- **Separator positioning**: `Descent + gap` below last text, `Ascent + gap` above next text

### 4. Visible Spacing Problems

From the generated preview images:

1. **Stat number→label gap is too large**: The "90" (48pt bold) to "Ja" (22pt regular) gap uses `y += 60` which is much larger than needed. The label appears detached from its number.

2. **Inconsistent section gaps**: Gap between title and separator (`25px`) differs from separator to stats (`50px`) and stats to separator (`25px + 50px`). There's no visual rhythm.

3. **Party table rows too tight**: `y += 30` for 22pt text gives barely any breathing room, while stat dashboard has generous spacing.

4. **Bottom-heavy whitespace**: Content finishes ~60-70% down the image on most cards, leaving a large empty area at the bottom. The content is not vertically centered or distributed.

5. **Result card subtitle misalignment**: In multi-vote cards, the subtitle ("Einleitungsartikel") at `y=110` and verdict ("ANGENOMMEN") at `y=170` overlap visually because they're at completely different X positions (left-aligned vs centered).

### 5. Layout Cursor Concept

A `layoutCursor` struct would encapsulate Y positioning:

```go
type layoutCursor struct {
    y         int
    imgHeight int
}

// lineHeight returns the baseline-to-baseline distance for a face.
func lineHeight(face font.Face) int {
    return face.Metrics().Height.Ceil()
}

// advance moves the cursor down by one line at the given font's height.
func (c *layoutCursor) advance(face font.Face) {
    c.y += lineHeight(face)
}

// gap adds vertical space equal to a fraction of the face's line height.
func (c *layoutCursor) gap(face font.Face, fraction float64) {
    c.y += int(float64(lineHeight(face)) * fraction)
}

// gapPx adds an explicit pixel gap (for separators, padding).
func (c *layoutCursor) gapPx(px int) {
    c.y += px
}
```

This replaces all `y += <magic>` calls with:

- `cur.advance(face)` — move down one line in the current font
- `cur.gap(face, 0.5)` — half-line gap using font metrics
- `cur.gapPx(padding)` — explicit pixel-based gap (for top/bottom padding and separators)

### 6. How Spacing Would Map

Current → Proposed replacement pattern:

| Current code                    | Replaces with                                      |
| ------------------------------- | -------------------------------------------------- |
| `y += 80` after header (26pt)   | `cur.gap(headerFace, 2.0)` — 2× line height ≈ 76px |
| `y += 80` after verdict (64pt)  | `cur.advance(verdictFace)` — 1× height ≈ 89px      |
| `y += int(titleFontSize * 1.4)` | `cur.advance(titleFace)` — already font-relative   |
| `y += 25` before separator      | `cur.gap(prevFace, 0.5)` — half-line gap           |
| `y += 50` after separator       | `cur.gap(nextFace, 0.75)` — ¾ line gap             |
| `y += 60` stat numbers (48pt)   | `cur.advance(statNumFace)` — 1× height ≈ 67px      |
| `y += 35` stat labels (22pt)    | `cur.advance(statLabelFace)` — 1× height ≈ 31px    |
| `y += 32` table header (22pt)   | `cur.advance(partyNumFace)` — 1× height ≈ 31px     |
| `y += 30` table rows (22pt)     | `cur.advance(partyNumFace)` — 1× height ≈ 31px     |

### 7. Font Loading Simplification

Currently, fonts are loaded separately in each render function with `loadFace()` calls repeated across `renderCombinedCard`, `renderResultCard`, and `renderTitleCard`. A **font set** struct could preload all needed faces once:

```go
type fontSet struct {
    header, verdict, title, statNum, statLabel, partyBold, partyNum font.Face
    emojiHeader, emojiRegular, emojiSmall, emojiLarge              font.Face
}
```

This is loaded once in `GenerateCarousel` and passed to all render functions, eliminating the ~20 individual `loadFace()` calls and `if err` checks scattered across the code.

## Code References

- `pkg/imagegen/imagegen.go:265-300` — `renderCombinedCard` font loading (8 loadFace calls)
- `pkg/imagegen/imagegen.go:300-395` — `renderCombinedCard` layout with hardcoded y offsets
- `pkg/imagegen/imagegen.go:425-447` — `drawStatColumns` with `y += 60` / `y += 35`
- `pkg/imagegen/imagegen.go:460-540` — `drawFraktionTable` with `y += 32` / `y += 30`
- `pkg/imagegen/imagegen.go:565-600` — `renderTitleCard` with `y += 80` / `y += 50`
- `pkg/imagegen/imagegen.go:602-672` — `renderResultCard` with mixed offsets
- `pkg/imagegen/imagegen.go:200-245` — `GenerateCarousel` multi-vote font loading (6 loadFace calls)

## Architecture Documentation

### Current Pattern

- Each render function loads its own fonts, creates an `*image.RGBA`, draws with manual Y tracking, encodes to JPEG.
- `drawShadowedText` / `drawCenteredText` take an explicit `y` parameter; they don't advance any cursor.
- `drawStatColumns` and `drawFraktionTable` accept and return `y` — they are the only functions that act as mini layout engines.

### Font Drawing Convention

- `y` represents the text **baseline** (Go's `font.Drawer.Dot.Y` is the baseline).
- Ascent extends above baseline, descent below.
- When `y += 60` is used after drawing 48pt text: the actual visible gap = 60 - descent_of_48pt ≈ 60 - 12 = 48px of whitespace below the glyph bottoms, then next text's ascent goes upward from the new baseline.

## Open Questions

1. **Vertical centering**: Should the layout cursor also support distributing content vertically (centering all content within the 1080px height), or should content remain top-aligned with the cursor just providing consistent spacing?

   > yes, vertical centering would be a nice-to-have. The cursor could track the total content height as it advances, and at the end of rendering, if the content height is less than the image height, it could apply a final offset to center everything. This would ensure that cards with less content don't look top-heavy and have better overall balance.

2. **Adaptive title sizing**: The current approach of trying font sizes 34→20 until the title fits works but interacts with the layout cursor—the available space for the title depends on how much space the fixed sections need. Should the cursor pre-calculate the fixed sections' total height first?

   > The cursor could have a "dry run" mode where it calculates the total content height without actually drawing, allowing the title font size to be chosen based on the remaining space. Alternatively, the title could be rendered first with a large font size, and if it overflows, the cursor could backtrack and reduce the font size until it fits within the available space. This would decouple title sizing from the fixed sections and allow for more dynamic layouts.

3. **drawShadowedText Y convention**: Currently `y` is the baseline. Should the cursor track the top of the next text region instead, adding ascent internally? This would make gap calculations more intuitive (gap = visible whitespace between bottom of glyph and top of next glyph).
   > It might be more intuitive for the cursor to track the top of the next text region. This way, when you call `cur.advance(face)`, it would internally add the ascent of the font to position the next text correctly. The gap functions could then simply add visible whitespace without needing to account for ascent/descent. This would simplify the mental model when laying out elements, as you would think in terms of "where does the next text start" rather than "where is the baseline".
