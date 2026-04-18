---
date: 2026-04-17T23:30:00+02:00
planner: copilot
topic: "Image generation vertical spacing alignment"
tags: [plan, imagegen, layout, spacing, refactor]
status: draft
last_updated: 2026-04-17
---

# Image Generation Vertical Spacing Alignment — Implementation Plan

## Overview

Replace all hardcoded pixel offsets in image generation with a font-metric-based layout cursor, consolidate duplicated font loading into a single font set, and add vertical centering support. This produces consistent, predictable spacing derived from actual font measurements.

## Current State Analysis

- 4 render functions (`renderCombinedCard`, `renderResultCard`, `renderTitleCard`, `drawFraktionTable`) use ~15 different hardcoded `y +=` pixel offsets with no relationship to font sizes.
- Font loading is duplicated: `renderCombinedCard` loads 8 faces (`imagegen.go:265-298`), `GenerateCarousel` loads 6 for multi-vote (`imagegen.go:210-245`), `renderResultCard` loads 5 more (`imagegen.go:615-640`), with significant overlap (e.g., `gobold 48`, `goregular 22` loaded 3× each).
- `y` tracks the text baseline. `drawShadowedText` and `drawCenteredText` take explicit `y int` parameters and don't advance any layout state.
- `drawStatColumns` and `drawFraktionTable` accept and return `y` — they act as mini layout engines but still use magic numbers internally.

### Key Discoveries

- `imagegen.go:300-302`: `y += 80` after 26pt header — effective gap = 80 - descent ≈ 73px visible whitespace, disproportionate.
- `imagegen.go:441-444`: `y += 60` after 48pt stat numbers, `y += 35` after 22pt labels — stat-to-label gap is visually detached.
- `imagegen.go:547-548`: `y += 30` for 22pt party rows — too tight compared to generous stat spacing.
- Go `font.Face.Metrics()` provides `Height`, `Ascent`, `Descent` (as `fixed.Int26_6`) — `Height.Ceil()` gives baseline-to-baseline distance.

## Desired End State

- A `layoutCursor` struct tracks the **top of the next text region** (not baseline).
- All vertical spacing derived from `font.Face.Metrics()` — no magic pixel numbers.
- A `fontSet` struct preloads all font faces once in `GenerateCarousel`.
- Draw functions accept `*layoutCursor` and advance it internally.
- Optional vertical centering via dry-run mode.
- **Visual output is equivalent or improved** — no regression in existing tests.

### Verification Approach

- Existing `TestGenerateCarousel_ValidJPEG` must pass (1080×1080, valid JPEG, <500KB).
- Manual comparison of generated preview images before/after using `testfixtures.AllFixtures()`.

## What We're NOT Doing

- Changing the color palette, font choices, or overall card layout structure.
- Adding new card types or changing the carousel logic.
- Modifying horizontal positioning or text wrapping logic.
- Changing JPEG quality or image dimensions.

## Implementation Approach

Incremental, phase-by-phase refactor. Each phase produces working code that passes all existing tests. The layout cursor is introduced first as a new type, then font loading is consolidated, then render functions are migrated one at a time, and finally vertical centering is added.

---

## Phase 1: Layout Cursor Type

### Overview

Introduce the `layoutCursor` struct and font-metric helpers. No changes to existing render functions yet — this is purely additive.

### Changes Required

#### 1. New layout cursor in `pkg/imagegen/imagegen.go`

**File**: `pkg/imagegen/imagegen.go`
**Changes**: Add `layoutCursor` struct and methods after the existing helper functions (after `encodeJPEG`, before `renderCombinedCard`).

```go
// layoutCursor tracks the Y position as the top of the next text region.
// Unlike raw baseline tracking, gap calculations represent visible whitespace
// between the bottom of one glyph and the top of the next.
type layoutCursor struct {
	y         int
	imgHeight int
}

// newCursor creates a layout cursor starting at the given Y position.
func newCursor(startY, imgHeight int) *layoutCursor {
	return &layoutCursor{y: startY, imgHeight: imgHeight}
}

// baseline returns the baseline Y for drawing text with the given face.
// Since the cursor tracks the top of the text region, the baseline = y + ascent.
func (c *layoutCursor) baseline(face font.Face) int {
	return c.y + face.Metrics().Ascent.Ceil()
}

// lineHeight returns the recommended baseline-to-baseline distance for a face.
func lineHeight(face font.Face) int {
	return face.Metrics().Height.Ceil()
}

// advance moves the cursor down by one line at the given font's line height.
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

### Success Criteria

#### Automated Verification:

- [x] `go build ./...` passes
- [x] `go vet ./...` passes
- [x] Existing tests pass: `go test ./pkg/imagegen/...`

#### Manual Verification:

- [x] New types are visible in the package — no dead code warnings expected since Phase 2 will use them immediately.

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 2: Font Set Consolidation

### Overview

Introduce a `fontSet` struct that preloads all font faces once. Replace the scattered `loadFace()` calls in `GenerateCarousel`, `renderCombinedCard`, and `renderResultCard` with a single load point. This eliminates ~20 individual `loadFace` + `if err` blocks.

### Changes Required

#### 1. Font set struct and loader

**File**: `pkg/imagegen/imagegen.go`
**Changes**: Add `fontSet` struct and `loadFontSet()` function near the top of the file (after constants, before render functions).

```go
// fontSet holds all preloaded font faces needed for image generation.
type fontSet struct {
	header      font.Face // goregular 26
	verdict     font.Face // gobold 64
	verdictSm   font.Face // gobold 56 (result card)
	statNum     font.Face // gobold 48
	statLabel   font.Face // goregular 22
	partyBold   font.Face // gobold 22
	partyNum    font.Face // goregular 22
	regular     font.Face // goregular 36
	small       font.Face // goregular 28
	boldHeading font.Face // gobold 48 (= statNum, shared)

	emojiHeader  font.Face // notoEmoji 26
	emojiRegular font.Face // notoEmoji 36
	emojiSmall   font.Face // notoEmoji 28
	emojiLarge   font.Face // notoEmoji 48
}

func loadFontSet() (*fontSet, error) {
	var fs fontSet
	var err error

	load := func(data []byte, size float64) (font.Face, error) {
		return loadFace(data, size)
	}

	if fs.header, err = load(goregular.TTF, 26); err != nil {
		return nil, fmt.Errorf("header font: %w", err)
	}
	if fs.verdict, err = load(gobold.TTF, 64); err != nil {
		return nil, fmt.Errorf("verdict font: %w", err)
	}
	if fs.verdictSm, err = load(gobold.TTF, 56); err != nil {
		return nil, fmt.Errorf("verdict-sm font: %w", err)
	}
	if fs.statNum, err = load(gobold.TTF, 48); err != nil {
		return nil, fmt.Errorf("statNum font: %w", err)
	}
	if fs.statLabel, err = load(goregular.TTF, 22); err != nil {
		return nil, fmt.Errorf("statLabel font: %w", err)
	}
	if fs.partyBold, err = load(gobold.TTF, 22); err != nil {
		return nil, fmt.Errorf("partyBold font: %w", err)
	}
	if fs.partyNum, err = load(goregular.TTF, 22); err != nil {
		return nil, fmt.Errorf("partyNum font: %w", err)
	}
	if fs.regular, err = load(goregular.TTF, 36); err != nil {
		return nil, fmt.Errorf("regular font: %w", err)
	}
	if fs.small, err = load(goregular.TTF, 28); err != nil {
		return nil, fmt.Errorf("small font: %w", err)
	}
	fs.boldHeading = fs.statNum // same face, gobold 48

	if fs.emojiHeader, err = load(notoEmojiTTF, 26); err != nil {
		return nil, fmt.Errorf("emojiHeader font: %w", err)
	}
	if fs.emojiRegular, err = load(notoEmojiTTF, 36); err != nil {
		return nil, fmt.Errorf("emojiRegular font: %w", err)
	}
	if fs.emojiSmall, err = load(notoEmojiTTF, 28); err != nil {
		return nil, fmt.Errorf("emojiSmall font: %w", err)
	}
	if fs.emojiLarge, err = load(notoEmojiTTF, 48); err != nil {
		return nil, fmt.Errorf("emojiLarge font: %w", err)
	}

	return &fs, nil
}
```

#### 2. Update `GenerateCarousel`

**File**: `pkg/imagegen/imagegen.go`
**Changes**: Replace the 6 individual `loadFace` calls in the multi-vote branch with a single `loadFontSet()` call. Pass `*fontSet` to all render functions.

- `renderCombinedCard(v, bg)` → `renderCombinedCard(v, bg, fonts)`
- `renderTitleCard(votes, bg, boldFace, regularFace, emojiFace)` → `renderTitleCard(votes, bg, fonts)`
- `renderResultCard(v, bg, boldFace, regularFace, smallFace, ...)` → `renderResultCard(v, bg, fonts)`

#### 3. Update render function signatures

**File**: `pkg/imagegen/imagegen.go`
**Changes**: Each render function receives `*fontSet` instead of individual faces. Remove all internal `loadFace` calls from `renderCombinedCard` (8 calls) and `renderResultCard` (5 calls). Reference faces via `fonts.verdict`, `fonts.statNum`, etc.

### Success Criteria

#### Automated Verification:

- [x] `go build ./...` passes
- [x] `go vet ./...` passes
- [x] `go test ./pkg/imagegen/...` passes — images remain valid JPEG, 1080×1080, <500KB

#### Manual Verification:

- [x] Generated preview images are visually identical to pre-refactor (no spacing changes yet).

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 3: Migrate Render Functions to Layout Cursor

### Overview

Replace all `y += <magic>` offsets with `layoutCursor` methods in each render function. This is the core visual change — spacing will shift to be font-metric-based. Migrate one function at a time to keep diffs reviewable.

### Changes Required

#### 1. Update draw helper signatures

**File**: `pkg/imagegen/imagegen.go`
**Changes**: Update `drawShadowedText` and `drawCenteredText` to accept `*layoutCursor` and use `cur.baseline(face)` for Y positioning. The original `y int` parameter versions can be kept temporarily as `drawShadowedTextAt` / `drawCenteredTextAt` if needed during migration, or all callers can be updated at once.

New signatures:

```go
func drawShadowedTextCur(img *image.RGBA, face, emojiFace font.Face, x int, cur *layoutCursor, text string, bg color.RGBA)
func drawCenteredTextCur(img *image.RGBA, face, emojiFace font.Face, cur *layoutCursor, text string, bg color.RGBA)
```

These compute `y = cur.baseline(face)` internally, then call the existing drawing logic.

#### 2. Migrate `renderCombinedCard`

**File**: `pkg/imagegen/imagegen.go` (lines ~265-395)
**Changes**: Replace manual `y` tracking with `layoutCursor`.

Current → New mapping:
| Current | New |
| -------------------------------- | ------------------------------------------------ |
| `y = padding + 32` | `cur := newCursor(padding, imgHeight)` |
| `y += 80` (after 26pt header) | `cur.gap(fonts.header, 2.0)` |
| `y += 80` (after 64pt verdict) | `cur.advance(fonts.verdict)` |
| `y += int(titleFontSize * 1.4)` | `cur.advance(titleFace)` |
| `y += 25` (before separator) | `cur.gap(titleFace, 0.5)` |
| `y += 50` (after separator) | `cur.gap(fonts.statNum, 0.75)` |
| `y += 25` (before 2nd sep) | `cur.gap(fonts.statLabel, 0.5)` |
| `y += 50` (after 2nd sep) | `cur.gap(fonts.partyBold, 0.75)` |

#### 3. Migrate `drawStatColumns`

**File**: `pkg/imagegen/imagegen.go` (lines ~425-447)
**Changes**: Accept `*layoutCursor` instead of `y int`, use `cur.advance()`.

| Current          | New                      |
| ---------------- | ------------------------ |
| `y += 60` (48pt) | `cur.advance(numFace)`   |
| `y += 35` (22pt) | `cur.advance(labelFace)` |

#### 4. Migrate `drawFraktionTable`

**File**: `pkg/imagegen/imagegen.go` (lines ~460-540)
**Changes**: Accept `*layoutCursor`, use `cur.advance()` for rows.

| Current          | New                    |
| ---------------- | ---------------------- |
| `y += 32` (hdr)  | `cur.advance(numFace)` |
| `y += 30` (rows) | `cur.advance(numFace)` |

#### 5. Migrate `renderTitleCard`

**File**: `pkg/imagegen/imagegen.go` (lines ~565-600)
**Changes**:

| Current                      | New                                    |
| ---------------------------- | -------------------------------------- |
| `y = padding + 60`           | `cur := newCursor(padding, imgHeight)` |
| `y += 80` (after header)     | `cur.gap(fonts.regular, 2.0)`          |
| `y += 50` (per wrapped line) | `cur.advance(fonts.regular)`           |

#### 6. Migrate `renderResultCard`

**File**: `pkg/imagegen/imagegen.go` (lines ~602-672)
**Changes**:

| Current                     | New                                    |
| --------------------------- | -------------------------------------- |
| `y = padding + 50`          | `cur := newCursor(padding, imgHeight)` |
| `y += 60` (after subtitle)  | `cur.advance(fonts.boldHeading)`       |
| `y += 80` (after verdict)   | `cur.advance(fonts.verdictSm)`         |
| `y += 50` (after separator) | `cur.gap(fonts.statNum, 0.75)`         |
| `y += 25` (before 2nd sep)  | `cur.gap(fonts.statLabel, 0.5)`        |
| `y += 50` (after 2nd sep)   | `cur.gap(fonts.partyBold, 0.75)`       |

#### 7. Adaptive title sizing update

**File**: `pkg/imagegen/imagegen.go` (in `renderCombinedCard`)
**Changes**: The title font size loop currently uses `int(titleFontSize * 1.4)` for height estimation. Update to use `lineHeight(titleFace)` from the cursor's helper function for consistency. The `bottomReserved` calculation should also use font metrics instead of magic pixel estimates.

### Success Criteria

#### Automated Verification:

- [x] `go build ./...` passes
- [x] `go vet ./...` passes
- [x] `go test ./pkg/imagegen/...` passes

#### Manual Verification:

- [ ] Generate preview images using `testfixtures.AllFixtures()` and compare:
  - Stat number→label gap is tighter (was 60px, now ~67px line height)
  - Section gaps are visually consistent
  - Party table rows have even spacing
  - No content overflows 1080px height
  - No text overlaps

**Implementation Note**: Pause for manual verification before proceeding.

---

## Phase 4: Vertical Centering

### Overview

Add a dry-run mode to the layout cursor that calculates total content height without drawing. Use this to compute a Y offset that centers content vertically within the 1080px image. This eliminates the bottom-heavy whitespace noted in the research.

### Changes Required

#### 1. Dry-run support in `layoutCursor`

**File**: `pkg/imagegen/imagegen.go`
**Changes**: Add a `dryRun` field to `layoutCursor` that, when set, tracks Y advancement without triggering any draw calls. Add a `contentHeight()` method.

```go
type layoutCursor struct {
	y         int
	startY    int
	imgHeight int
	dryRun    bool
}

// contentHeight returns the total vertical space consumed since cursor creation.
func (c *layoutCursor) contentHeight() int {
	return c.y - c.startY
}
```

#### 2. Two-pass rendering in each render function

**File**: `pkg/imagegen/imagegen.go`
**Changes**: Each render function runs the layout logic twice:

1. **Dry run**: advance the cursor through all elements without drawing → get `contentHeight`.
2. **Real run**: compute `startY = (imgHeight - contentHeight) / 2`, create cursor at that offset, draw everything.

To avoid duplicating the layout logic, extract the layout sequence into a helper that accepts `*layoutCursor` and `*image.RGBA` (nil during dry run). Example pattern:

```go
func renderCombinedCard(v *zurichapi.Abstimmung, bg color.RGBA, fonts *fontSet) ([]byte, error) {
	// Dry run to measure content height
	dry := newCursor(0, imgHeight)
	dry.dryRun = true
	layoutCombinedCard(nil, dry, v, bg, fonts)

	// Real run with centered offset
	startY := (imgHeight - dry.contentHeight()) / 2
	if startY < padding {
		startY = padding
	}
	img := newImage(bg)
	cur := newCursor(startY, imgHeight)
	layoutCombinedCard(img, cur, v, bg, fonts)

	return encodeJPEG(img)
}
```

#### 3. Guard draw calls in layout functions

All draw calls (`drawShadowedText`, `drawCenteredText`, `drawHLine`, `drawStatColumns`, `drawFraktionTable`) must be skipped when `img == nil` (dry-run mode). This is the simplest guard — just wrap draw calls in `if img != nil { ... }`.

#### 4. Adaptive title sizing integration

With dry-run mode, the title font size selection can be done more accurately:

1. Run a dry-run of all fixed sections (header, verdict, stats, party table) to get their total height.
2. Remaining space = `imgHeight - 2*padding - fixedHeight`.
3. Try title font sizes until wrapped title fits in remaining space.

### Success Criteria

#### Automated Verification:

- [x] `go build ./...` passes
- [x] `go vet ./...` passes
- [x] `go test ./pkg/imagegen/...` passes — images remain valid 1080×1080 JPEG <500KB

#### Manual Verification:

- [ ] Content is visually centered vertically on all card types
- [ ] Cards with less content (e.g., few party rows) don't look top-heavy
- [ ] Cards with maximum content still fit within 1080px (startY clamped to padding)
- [ ] Multi-vote result cards have balanced spacing

**Implementation Note**: This is the final phase. Generate full preview images and compare against original output.

---

## Testing Strategy

### Unit Tests

- Existing `TestGenerateCarousel_ValidJPEG` covers all fixture types (single vote, multi-vote) and validates JPEG format, dimensions, and file size. No new unit tests needed unless behavior changes.
- Consider adding a test for `lineHeight()` returning reasonable values for known font sizes (e.g., 48pt → ~60-70px).

### Manual Testing Steps

1. Run preview generation with all fixtures:
   ```bash
   go run /tmp/gen_preview.go
   ```
2. Compare `/tmp/preview_*.jpg` before and after each phase.
3. Verify: consistent gap rhythm, no overlaps, vertical centering in Phase 4.

## References

- Research: `thoughts/shared/research/2026-04-17-imagegen-vertical-spacing.md`
- Current implementation: `pkg/imagegen/imagegen.go`
- Test fixtures: `pkg/voteposting/testfixtures/fixtures.go`
- Existing tests: `pkg/imagegen/imagegen_test.go`
- Go font metrics API: `golang.org/x/image/font` — `Face.Metrics()` returns `Height`, `Ascent`, `Descent`
