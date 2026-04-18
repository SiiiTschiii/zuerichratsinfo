package imagegen

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

//go:embed fonts/NotoEmoji-Regular.ttf
var notoEmojiTTF []byte

const (
	imgWidth  = 1080
	imgHeight = 1350
	padding   = 60
	shadowOff = 2
)

var palette = []color.RGBA{
	{0x1B, 0x4F, 0x72, 0xFF}, // dark blue
	{0x7B, 0x24, 0x1C, 0xFF}, // dark red
	{0x14, 0x5A, 0x32, 0xFF}, // dark green
	{0x4A, 0x23, 0x5A, 0xFF}, // dark purple
	{0x78, 0x4F, 0x0B, 0xFF}, // dark gold
	{0x1A, 0x5C, 0x5C, 0xFF}, // dark teal
	{0x6C, 0x3A, 0x0A, 0xFF}, // brown
	{0x2C, 0x3E, 0x6B, 0xFF}, // steel blue
}

// SelectColor returns a deterministic color based on GeschaeftGrNr.
func SelectColor(geschaeftGrNr string) color.RGBA {
	h := sha256.Sum256([]byte(geschaeftGrNr))
	idx := int(h[0]) % len(palette)
	return palette[idx]
}

func darken(c color.RGBA) color.RGBA {
	return color.RGBA{c.R / 3, c.G / 3, c.B / 3, c.A}
}

func loadFace(fontData []byte, size float64) (font.Face, error) {
	f, err := opentype.Parse(fontData)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// drawShadowedText draws text with a shadow offset for readability.
// It uses the emoji font face for emoji characters and the text font for everything else.
func drawShadowedText(img *image.RGBA, face font.Face, emojiFace font.Face, x, y int, text string, bg color.RGBA) {
	shadow := darken(bg)
	// Draw shadow pass then foreground pass
	for pass := 0; pass < 2; pass++ {
		var src *image.Uniform
		var ox, oy int
		if pass == 0 {
			src = image.NewUniform(shadow)
			ox, oy = x+shadowOff, y+shadowOff
		} else {
			src = image.NewUniform(color.White)
			ox, oy = x, y
		}
		dot := fixed.P(ox, oy)
		for _, seg := range splitEmojiText(text) {
			f := face
			segDot := dot
			if seg.isEmoji && emojiFace != nil {
				f = emojiFace
				// Shift emoji baseline so its visual center aligns with the text.
				// Text spans [Y-Ascent, Y+Descent]; center = Y - (Ascent-Descent)/2.
				// Same for emoji. Setting centers equal gives this offset:
				textM := face.Metrics()
				emojiM := emojiFace.Metrics()
				segDot.Y += ((emojiM.Ascent - emojiM.Descent) - (textM.Ascent - textM.Descent)) / 2
			}
			d := &font.Drawer{Dst: img, Src: src, Face: f, Dot: segDot}
			d.DrawString(seg.text)
			dot.X = d.Dot.X
		}
	}
}

// drawCenteredText draws text horizontally centered on the image.
func drawCenteredText(img *image.RGBA, face font.Face, emojiFace font.Face, y int, text string, bg color.RGBA) {
	w := font.MeasureString(face, text).Ceil()
	x := (imgWidth - w) / 2
	drawShadowedText(img, face, emojiFace, x, y, text, bg)
}

// drawHLine draws a thin horizontal separator line.
func drawHLine(img *image.RGBA, y, x1, x2 int, c color.Color) {
	for x := x1; x <= x2; x++ {
		img.Set(x, y, c)
		img.Set(x, y+1, c)
	}
}

// semiWhite is a semi-transparent white used for separator lines.
var semiWhite = color.RGBA{0xFF, 0xFF, 0xFF, 0x66}

// textSegment represents a run of text that is either all emoji or all non-emoji.
type textSegment struct {
	text    string
	isEmoji bool
}

// isEmojiRune returns true for Unicode codepoints that are emoji symbols.
func isEmojiRune(r rune) bool {
	// Variation selectors and zero-width joiners are part of emoji sequences
	if r == 0xFE0F || r == 0xFE0E || r == 0x200D {
		return true
	}
	// Common emoji ranges
	return (r >= 0x2600 && r <= 0x27BF) || // Misc Symbols, Dingbats
		(r >= 0x1F300 && r <= 0x1FAF8) || // Emoticons, Symbols, etc.
		(r >= 0x2300 && r <= 0x23FF) || // Misc Technical
		(r >= 0x2700 && r <= 0x27BF) || // Dingbats
		(r >= 0x1F100 && r <= 0x1F1FF) || // Enclosed Alphanumeric Supplement
		!unicode.IsGraphic(r)
}

// splitEmojiText splits text into alternating emoji and non-emoji segments.
func splitEmojiText(text string) []textSegment {
	if text == "" {
		return nil
	}
	var segments []textSegment
	runes := []rune(text)
	start := 0
	curIsEmoji := isEmojiRune(runes[0])

	for i := 1; i < len(runes); i++ {
		ie := isEmojiRune(runes[i])
		if ie != curIsEmoji {
			segments = append(segments, textSegment{text: string(runes[start:i]), isEmoji: curIsEmoji})
			start = i
			curIsEmoji = ie
		}
	}
	segments = append(segments, textSegment{text: string(runes[start:]), isEmoji: curIsEmoji})
	return segments
}

// wrapText breaks text into lines that fit within maxWidth pixels.
func wrapText(face font.Face, text string, maxWidth int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		test := line + " " + w
		adv := font.MeasureString(face, test)
		if adv.Ceil() > maxWidth {
			lines = append(lines, line)
			line = w
		} else {
			line = test
		}
	}
	lines = append(lines, line)
	return lines
}

// GenerateCarousel produces carousel JPEG images for a vote group.
// Returns [][]byte (JPEG-encoded images).
func GenerateCarousel(votes []zurichapi.Abstimmung) ([][]byte, error) {
	if len(votes) == 0 {
		return nil, fmt.Errorf("no votes provided")
	}

	bgColor := SelectColor(votes[0].GeschaeftGrNr)

	fonts, err := loadFontSet()
	if err != nil {
		return nil, fmt.Errorf("loading fonts: %w", err)
	}

	var images [][]byte

	if len(votes) == 1 {
		// Single vote: combine title + results into one image
		combinedImg, err := renderCombinedCard(&votes[0], bgColor, fonts)
		if err != nil {
			return nil, fmt.Errorf("rendering combined card: %w", err)
		}
		images = append(images, combinedImg)
	} else {
		// Multi-vote: title card + one result card per vote
		titleImg, err := renderTitleCard(votes, bgColor, fonts)
		if err != nil {
			return nil, fmt.Errorf("rendering title card: %w", err)
		}
		images = append(images, titleImg)

		for i := range votes {
			resultImg, err := renderResultCard(&votes[i], bgColor, fonts)
			if err != nil {
				return nil, fmt.Errorf("rendering result card %d: %w", i, err)
			}
			images = append(images, resultImg)
		}
	}

	return images, nil
}

func newImage(bg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
	return img
}

func encodeJPEG(img *image.RGBA) ([]byte, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// layoutCursor tracks the Y position as the top of the next text region.
// Unlike raw baseline tracking, gap calculations represent visible whitespace
// between the bottom of one glyph and the top of the next.
type layoutCursor struct {
	y         int
	startY    int
	imgHeight int
}

// newCursor creates a layout cursor starting at the given Y position.
func newCursor(startY, imgHeight int) *layoutCursor {
	return &layoutCursor{y: startY, startY: startY, imgHeight: imgHeight}
}

// contentHeight returns the total vertical space consumed since cursor creation.
func (c *layoutCursor) contentHeight() int {
	return c.y - c.startY
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

// fontSet holds all preloaded font faces needed for image generation.
type fontSet struct {
	verdict     font.Face // gobold 64
	verdictSm   font.Face // gobold 56 (result card)
	statNum     font.Face // gobold 48
	statLabel   font.Face // goregular 26
	partyBold   font.Face // gobold 26
	partyNum    font.Face // goregular 26
	regular     font.Face // goregular 36
	small       font.Face // goregular 28
	boldHeading font.Face // gobold 48 (= statNum, shared)

	emojiRegular font.Face // notoEmoji 36
	emojiSmall   font.Face // notoEmoji 28
	emojiLarge   font.Face // notoEmoji 48
	emojiVerdict font.Face // notoEmoji 72
}

func loadFontSet() (*fontSet, error) {
	var fs fontSet
	var err error

	load := func(data []byte, size float64) (font.Face, error) {
		return loadFace(data, size)
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
	if fs.statLabel, err = load(goregular.TTF, 26); err != nil {
		return nil, fmt.Errorf("statLabel font: %w", err)
	}
	if fs.partyBold, err = load(gobold.TTF, 26); err != nil {
		return nil, fmt.Errorf("partyBold font: %w", err)
	}
	if fs.partyNum, err = load(goregular.TTF, 26); err != nil {
		return nil, fmt.Errorf("partyNum font: %w", err)
	}
	if fs.regular, err = load(goregular.TTF, 36); err != nil {
		return nil, fmt.Errorf("regular font: %w", err)
	}
	if fs.small, err = load(goregular.TTF, 28); err != nil {
		return nil, fmt.Errorf("small font: %w", err)
	}
	fs.boldHeading = fs.statNum // same face, gobold 48

	if fs.emojiRegular, err = load(notoEmojiTTF, 36); err != nil {
		return nil, fmt.Errorf("emojiRegular font: %w", err)
	}
	if fs.emojiSmall, err = load(notoEmojiTTF, 28); err != nil {
		return nil, fmt.Errorf("emojiSmall font: %w", err)
	}
	if fs.emojiLarge, err = load(notoEmojiTTF, 48); err != nil {
		return nil, fmt.Errorf("emojiLarge font: %w", err)
	}
	if fs.emojiVerdict, err = load(notoEmojiTTF, 72); err != nil {
		return nil, fmt.Errorf("emojiVerdict font: %w", err)
	}

	return &fs, nil
}

// renderCombinedCard renders a single image with visual hierarchy:
// large verdict, bold title, dashboard stats, and grouped party breakdown.
func renderCombinedCard(v *zurichapi.Abstimmung, bg color.RGBA, fonts *fontSet) ([]byte, error) {
	// Dry run to measure content height
	dry := newCursor(0, imgHeight)
	titleFace, titleLines, err := layoutCombinedCard(nil, dry, v, bg, fonts)
	if err != nil {
		return nil, err
	}

	// Real run with centered offset
	startY := (imgHeight - dry.contentHeight()) / 2
	if startY < padding {
		startY = padding
	}
	img := newImage(bg)

	cur := newCursor(startY, imgHeight)
	titleFace, titleLines, err = layoutCombinedCard(img, cur, v, bg, fonts)
	_ = titleFace
	_ = titleLines
	if err != nil {
		return nil, err
	}

	return encodeJPEG(img)
}

func layoutCombinedCard(img *image.RGBA, cur *layoutCursor, v *zurichapi.Abstimmung, bg color.RGBA, fonts *fontSet) (font.Face, []string, error) {
	maxTextWidth := imgWidth - 2*padding

	// Build counts
	counts := voteformat.VoteCounts{
		Ja: v.AnzahlJa, Nein: v.AnzahlNein, Enthaltung: v.AnzahlEnthaltung,
		Abwesend: v.AnzahlAbwesend, A: v.AnzahlA, B: v.AnzahlB, C: v.AnzahlC,
		D: v.AnzahlD, E: v.AnzahlE,
	}
	isAuswahl := voteformat.IsAuswahlVote(counts)

	// Title first: bold, wrapped, centered
	title := voteformat.CleanVoteTitle(
		voteformat.SelectBestTitle(v.TraktandumTitel, v.GeschaeftTitel),
	)

	// Calculate available space for title: reserve space for verdict + stats + party breakdown
	fraktionCounts := voteformat.AggregateFraktionCounts(v.Stimmabgaben.Stimmabgabe)
	numParties := len(fraktionCounts)
	verdictHeight := lineHeight(fonts.verdict)
	statsHeight := lineHeight(fonts.statNum) + lineHeight(fonts.statLabel)
	separatorHeight := lineHeight(fonts.statLabel) + lineHeight(fonts.statNum)
	partyHeight := lineHeight(fonts.partyNum) + numParties*lineHeight(fonts.partyNum)
	bottomReserved := verdictHeight + statsHeight + separatorHeight + partyHeight + padding
	availableForTitle := imgHeight - cur.y - bottomReserved

	// Try font sizes from 42 down to 20 until title fits
	titleFontSize := 42.0
	var titleFace font.Face
	var titleLines []string
	var err error
	for titleFontSize >= 20 {
		titleFace, err = loadFace(goregular.TTF, titleFontSize)
		if err != nil {
			return nil, nil, err
		}
		titleLines = wrapText(titleFace, title, maxTextWidth)
		totalTitleHeight := len(titleLines) * lineHeight(titleFace)
		if totalTitleHeight <= availableForTitle {
			break
		}
		titleFontSize -= 2
	}

	for _, line := range titleLines {
		if img != nil {
			drawCenteredText(img, titleFace, nil, cur.baseline(titleFace), line, bg)
		}
		cur.advance(titleFace)
		cur.gap(titleFace, 0.15)
	}

	cur.gap(titleFace, 0.75)

	// Verdict: large centered emoji
	var verdictText string
	if !isAuswahl {
		verdictText = voteformat.GetVoteResultEmoji(v.Schlussresultat)
	} else {
		verdictText = strings.ToUpper(v.Schlussresultat)
	}
	if img != nil {
		drawCenteredText(img, fonts.verdict, fonts.emojiVerdict, cur.baseline(fonts.verdict), verdictText, bg)
	}
	cur.advance(fonts.verdict)

	cur.gap(fonts.verdict, 0.75)

	// Horizontal separator
	if img != nil {
		drawHLine(img, cur.y, padding, imgWidth-padding, semiWhite)
	}
	cur.gap(fonts.statNum, 0.75)

	// Stats dashboard: large numbers with small labels in columns
	if !isAuswahl {
		drawStandardStatsDashboard(img, cur, counts, bg, fonts.statNum, fonts.statLabel)
	} else {
		drawAuswahlStatsDashboard(img, cur, counts, bg, fonts.statNum, fonts.statLabel)
	}

	cur.gap(fonts.statNum, 0.75)

	// Horizontal separator
	if img != nil {
		drawHLine(img, cur.y, padding, imgWidth-padding, semiWhite)
	}
	cur.gap(fonts.partyBold, 1.25)

	// Party breakdown table
	drawFraktionTable(img, cur, fraktionCounts, bg, fonts.partyBold, fonts.partyNum)

	return titleFace, titleLines, nil
}

// statCol holds a value/label pair for dashboard-style stat columns.
type statCol struct {
	value string
	label string
}

// drawStandardStatsDashboard draws Ja/Nein/Enthaltung as large centered numbers.
func drawStandardStatsDashboard(img *image.RGBA, cur *layoutCursor, counts voteformat.VoteCounts, bg color.RGBA, numFace, labelFace font.Face) {
	cols := []statCol{
		{voteformat.FormatVoteCount(counts.Ja), "Ja"},
		{voteformat.FormatVoteCount(counts.Nein), "Nein"},
		{voteformat.FormatVoteCount(counts.Enthaltung), "Enth."},
	}
	drawStatColumns(img, cur, cols, bg, numFace, labelFace)
}

// drawAuswahlStatsDashboard draws A/B/C/D/E option counts as large centered numbers.
func drawAuswahlStatsDashboard(img *image.RGBA, cur *layoutCursor, counts voteformat.VoteCounts, bg color.RGBA, numFace, labelFace font.Face) {
	var cols []statCol
	options := []struct {
		ptr   *int
		label string
	}{
		{counts.A, "A"}, {counts.B, "B"}, {counts.C, "C"},
		{counts.D, "D"}, {counts.E, "E"},
	}
	for _, o := range options {
		if o.ptr != nil && *o.ptr > 0 {
			cols = append(cols, statCol{fmt.Sprintf("%d", *o.ptr), o.label})
		}
	}
	if len(cols) == 0 {
		return
	}
	drawStatColumns(img, cur, cols, bg, numFace, labelFace)
}

// drawStatColumns draws stat values in evenly-spaced centered columns.
func drawStatColumns(img *image.RGBA, cur *layoutCursor, cols []statCol, bg color.RGBA, numFace, labelFace font.Face) {
	if len(cols) == 0 {
		return
	}
	colWidth := (imgWidth - 2*padding) / len(cols)

	// Draw large numbers
	for i, col := range cols {
		cx := padding + colWidth*i + colWidth/2
		w := font.MeasureString(numFace, col.value).Ceil()
		if img != nil {
			drawShadowedText(img, numFace, nil, cx-w/2, cur.baseline(numFace), col.value, bg)
		}
	}
	cur.advance(numFace)

	// Draw small labels
	for i, col := range cols {
		cx := padding + colWidth*i + colWidth/2
		w := font.MeasureString(labelFace, col.label).Ceil()
		if img != nil {
			drawShadowedText(img, labelFace, nil, cx-w/2, cur.baseline(labelFace), col.label, bg)
		}
	}
	cur.advance(labelFace)
}

// fraktionEntry holds a faction name and its vote counts for sorting.
type fraktionEntry struct {
	name   string
	counts map[string]int
	total  int
}

// drawFraktionTable draws a simple party breakdown table sorted by faction size descending.
func drawFraktionTable(img *image.RGBA, cur *layoutCursor, fraktionCounts map[string]*voteformat.FraktionCounts, bg color.RGBA, nameFace, numFace font.Face) {
	if len(fraktionCounts) == 0 {
		return
	}

	// Build faction entries
	var entries []fraktionEntry
	for name, fc := range fraktionCounts {
		total := 0
		for _, v := range fc.Counts {
			total += v
		}
		entries = append(entries, fraktionEntry{
			name: name, counts: fc.Counts, total: total,
		})
	}

	// Sort by total members descending, ties alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].total != entries[j].total {
			return entries[i].total > entries[j].total
		}
		return entries[i].name < entries[j].name
	})

	// Determine vote columns from the data
	keySet := make(map[string]bool)
	for _, e := range entries {
		for k := range e.counts {
			keySet[k] = true
		}
	}

	// Build columns: primary vote keys first, then Enthaltung, then Abwesend
	var primary []string
	hasEnth, hasAbw := false, false
	for k := range keySet {
		switch k {
		case "Enthaltung":
			hasEnth = true
		case "Abwesend":
			hasAbw = true
		default:
			primary = append(primary, k)
		}
	}
	sort.Strings(primary)

	// Abbreviate column headers
	var allCols []string // display headers
	var colKeys []string // actual map keys
	for _, k := range primary {
		allCols = append(allCols, k)
		colKeys = append(colKeys, k)
	}
	if hasEnth {
		allCols = append(allCols, "Enth.")
		colKeys = append(colKeys, "Enthaltung")
	}
	if hasAbw {
		allCols = append(allCols, "Abw.")
		colKeys = append(colKeys, "Abwesend")
	}

	// Layout
	nameColWidth := 200
	numColWidth := (imgWidth - 2*padding - nameColWidth) / len(allCols) / 2
	totalTableWidth := nameColWidth + numColWidth*len(allCols)
	tableStartX := (imgWidth - totalTableWidth) / 2
	numStartX := tableStartX + nameColWidth

	// Draw column headers
	for i, col := range allCols {
		cx := numStartX + numColWidth*i + numColWidth/2
		w := font.MeasureString(numFace, col).Ceil()
		if img != nil {
			drawShadowedText(img, numFace, nil, cx-w/2, cur.baseline(numFace), col, bg)
		}
	}
	cur.advance(numFace)

	// Draw party rows
	for _, e := range entries {
		if cur.y > imgHeight-padding {
			break
		}
		if img != nil {
			// Bold party name
			drawShadowedText(img, nameFace, nil, tableStartX, cur.baseline(numFace), e.name, bg)
			// Numbers in columns
			for i, key := range colKeys {
				cx := numStartX + numColWidth*i + numColWidth/2
				numStr := fmt.Sprintf("%d", e.counts[key])
				w := font.MeasureString(numFace, numStr).Ceil()
				drawShadowedText(img, numFace, nil, cx-w/2, cur.baseline(numFace), numStr, bg)
			}
		}
		cur.advance(numFace)
	}
}

func renderTitleCard(votes []zurichapi.Abstimmung, bg color.RGBA, fonts *fontSet) ([]byte, error) {
	// Dry run to measure content height
	dry := newCursor(0, imgHeight)
	layoutTitleCard(nil, dry, votes, bg, fonts)

	startY := (imgHeight - dry.contentHeight()) / 2
	if startY < padding {
		startY = padding
	}
	img := newImage(bg)

	cur := newCursor(startY, imgHeight)
	layoutTitleCard(img, cur, votes, bg, fonts)

	return encodeJPEG(img)
}

func layoutTitleCard(img *image.RGBA, cur *layoutCursor, votes []zurichapi.Abstimmung, bg color.RGBA, fonts *fontSet) {
	v := votes[0]
	maxTextWidth := imgWidth - 2*padding

	// Title: bold, wrapped, centered (no verdict — ambiguous for multi-vote groups)
	title := voteformat.CleanVoteTitle(
		voteformat.SelectBestTitle(v.TraktandumTitel, v.GeschaeftTitel),
	)

	titleFontSize := 42.0
	var titleFace font.Face
	var titleLines []string
	for titleFontSize >= 20 {
		var err error
		titleFace, err = loadFace(goregular.TTF, titleFontSize)
		if err != nil {
			return
		}
		titleLines = wrapText(titleFace, title, maxTextWidth)
		totalTitleHeight := len(titleLines) * lineHeight(titleFace)
		if totalTitleHeight <= imgHeight-cur.y-padding {
			break
		}
		titleFontSize -= 2
	}

	for _, line := range titleLines {
		if img != nil {
			drawCenteredText(img, titleFace, nil, cur.baseline(titleFace), line, bg)
		}
		cur.advance(titleFace)
		cur.gap(titleFace, 0.15)
	}

	// Summary: list each sub-vote with emoji + subtitle + result (no numbers)
	cur.gap(fonts.regular, 0.75)
	var summaryLines []string
	for _, sv := range votes {
		if sv.Abstimmungstitel == "" {
			continue
		}
		sub := voteformat.CleanVoteSubtitle(sv.Abstimmungstitel)
		emoji := voteformat.GetVoteResultEmoji(sv.Schlussresultat)
		summaryLines = append(summaryLines, fmt.Sprintf("%s %s", emoji, sub))
	}
	// Find widest line and center the block, then left-align all lines within it
	maxW := 0
	for _, line := range summaryLines {
		w := font.MeasureString(fonts.small, line).Ceil()
		if w > maxW {
			maxW = w
		}
	}
	blockX := (imgWidth - maxW) / 2
	for _, line := range summaryLines {
		if img != nil {
			drawShadowedText(img, fonts.small, fonts.emojiSmall, blockX, cur.baseline(fonts.small), line, bg)
		}
		cur.advance(fonts.small)
	}
}

func renderResultCard(v *zurichapi.Abstimmung, bg color.RGBA, fonts *fontSet) ([]byte, error) {
	// Dry run to measure content height
	dry := newCursor(0, imgHeight)
	layoutResultCard(nil, dry, v, bg, fonts)

	// Real run with centered offset
	startY := (imgHeight - dry.contentHeight()) / 2
	if startY < padding {
		startY = padding
	}
	img := newImage(bg)
	cur := newCursor(startY, imgHeight)
	layoutResultCard(img, cur, v, bg, fonts)

	return encodeJPEG(img)
}

func layoutResultCard(img *image.RGBA, cur *layoutCursor, v *zurichapi.Abstimmung, bg color.RGBA, fonts *fontSet) {
	// Subtitle if present (for multi-vote groups)
	if v.Abstimmungstitel != "" {
		sub := voteformat.CleanVoteSubtitle(v.Abstimmungstitel)
		if img != nil {
			drawCenteredText(img, fonts.boldHeading, fonts.emojiLarge, cur.baseline(fonts.boldHeading), sub, bg)
		}
		cur.advance(fonts.boldHeading)
		cur.gap(fonts.boldHeading, 0.5)
	}

	// Vote counts
	counts := voteformat.VoteCounts{
		Ja: v.AnzahlJa, Nein: v.AnzahlNein, Enthaltung: v.AnzahlEnthaltung,
		Abwesend: v.AnzahlAbwesend, A: v.AnzahlA, B: v.AnzahlB, C: v.AnzahlC,
		D: v.AnzahlD, E: v.AnzahlE,
	}

	// Verdict: large centered
	isAuswahl := voteformat.IsAuswahlVote(counts)
	var verdictText string
	if !isAuswahl {
		verdictText = voteformat.GetVoteResultEmoji(v.Schlussresultat)
	} else {
		verdictText = strings.ToUpper(v.Schlussresultat)
	}
	if img != nil {
		drawCenteredText(img, fonts.verdictSm, fonts.emojiVerdict, cur.baseline(fonts.verdictSm), verdictText, bg)
	}
	cur.advance(fonts.verdictSm)
	cur.gap(fonts.verdictSm, 0.75)

	// Horizontal separator
	if img != nil {
		drawHLine(img, cur.y, padding, imgWidth-padding, semiWhite)
	}
	cur.gap(fonts.statNum, 0.75)

	// Stats dashboard
	if !isAuswahl {
		drawStandardStatsDashboard(img, cur, counts, bg, fonts.statNum, fonts.statLabel)
	} else {
		drawAuswahlStatsDashboard(img, cur, counts, bg, fonts.statNum, fonts.statLabel)
	}

	cur.gap(fonts.statNum, 0.75)

	// Horizontal separator
	if img != nil {
		drawHLine(img, cur.y, padding, imgWidth-padding, semiWhite)
	}
	cur.gap(fonts.partyBold, 1.25)

	// Party breakdown table
	fraktionCounts := voteformat.AggregateFraktionCounts(v.Stimmabgaben.Stimmabgabe)
	drawFraktionTable(img, cur, fraktionCounts, bg, fonts.partyBold, fonts.partyNum)
}
