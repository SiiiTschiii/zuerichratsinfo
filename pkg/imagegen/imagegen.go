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
	imgHeight = 1080
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
			if seg.isEmoji && emojiFace != nil {
				f = emojiFace
			}
			d := &font.Drawer{Dst: img, Src: src, Face: f, Dot: dot}
			d.DrawString(seg.text)
			dot = d.Dot
		}
	}
}

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

	var images [][]byte

	if len(votes) == 1 {
		// Single vote: combine title + results into one image
		combinedImg, err := renderCombinedCard(&votes[0], bgColor)
		if err != nil {
			return nil, fmt.Errorf("rendering combined card: %w", err)
		}
		images = append(images, combinedImg)
	} else {
		// Multi-vote: title card + one result card per vote
		boldFace, err := loadFace(gobold.TTF, 48)
		if err != nil {
			return nil, fmt.Errorf("loading bold font: %w", err)
		}
		regularFace, err := loadFace(goregular.TTF, 36)
		if err != nil {
			return nil, fmt.Errorf("loading regular font: %w", err)
		}
		smallFace, err := loadFace(goregular.TTF, 28)
		if err != nil {
			return nil, fmt.Errorf("loading small font: %w", err)
		}
		emojiFace, err := loadFace(notoEmojiTTF, 36)
		if err != nil {
			return nil, fmt.Errorf("loading emoji font: %w", err)
		}
		emojiFaceSmall, err := loadFace(notoEmojiTTF, 28)
		if err != nil {
			return nil, fmt.Errorf("loading small emoji font: %w", err)
		}
		emojiFaceLarge, err := loadFace(notoEmojiTTF, 48)
		if err != nil {
			return nil, fmt.Errorf("loading large emoji font: %w", err)
		}

		titleImg, err := renderTitleCard(votes, bgColor, boldFace, regularFace, emojiFace)
		if err != nil {
			return nil, fmt.Errorf("rendering title card: %w", err)
		}
		images = append(images, titleImg)

		for i := range votes {
			resultImg, err := renderResultCard(&votes[i], bgColor, boldFace, regularFace, smallFace, emojiFaceLarge, emojiFace, emojiFaceSmall)
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

// renderCombinedCard renders a single image with title + results + Fraktion breakdown.
// If the title text is too long, the font size is reduced to fit.
func renderCombinedCard(v *zurichapi.Abstimmung, bg color.RGBA) ([]byte, error) {
	img := newImage(bg)
	maxTextWidth := imgWidth - 2*padding

	regularFace, err := loadFace(goregular.TTF, 36)
	if err != nil {
		return nil, err
	}
	emojiFace, err := loadFace(notoEmojiTTF, 36)
	if err != nil {
		return nil, err
	}
	smallFace, err := loadFace(goregular.TTF, 28)
	if err != nil {
		return nil, err
	}
	emojiFaceSmall, err := loadFace(notoEmojiTTF, 28)
	if err != nil {
		return nil, err
	}

	y := padding + 60

	// Header
	date := voteformat.FormatVoteDate(v.SitzungDatum)
	header := fmt.Sprintf("\U0001F5F3\uFE0F Gemeinderat | Abstimmung vom %s", date)
	drawShadowedText(img, regularFace, emojiFace, padding, y, header, bg)
	y += 70

	// Build result + title text
	counts := voteformat.VoteCounts{
		Ja: v.AnzahlJa, Nein: v.AnzahlNein, Enthaltung: v.AnzahlEnthaltung,
		Abwesend: v.AnzahlAbwesend, A: v.AnzahlA, B: v.AnzahlB, C: v.AnzahlC,
		D: v.AnzahlD, E: v.AnzahlE,
	}
	var resultPrefix string
	if !voteformat.IsAuswahlVote(counts) {
		resultPrefix = voteformat.GetVoteResultEmoji(v.Schlussresultat) + " " + voteformat.GetVoteResultText(v.Schlussresultat)
	} else {
		resultPrefix = v.Schlussresultat
	}
	title := voteformat.CleanVoteTitle(
		voteformat.SelectBestTitle(v.TraktandumTitel, v.GeschaeftTitel),
	)
	fullText := resultPrefix + ": " + title

	// Calculate vertical space needed for the bottom section
	fraktionCounts := voteformat.AggregateFraktionCounts(v.Stimmabgaben.Stimmabgabe)
	breakdown := voteformat.FormatFraktionBreakdown(fraktionCounts)
	bottomLines := 1 // count line
	if breakdown != "" {
		bottomLines += len(strings.Split(breakdown, "\n"))
	}
	bottomHeight := bottomLines*50 + 30
	availableForTitle := imgHeight - y - bottomHeight - padding

	// Try font sizes from 36 down to 20 until title fits
	titleFontSize := 36.0
	var titleFace font.Face
	var titleEmojiF font.Face
	var titleLines []string
	for titleFontSize >= 20 {
		titleFace, err = loadFace(goregular.TTF, titleFontSize)
		if err != nil {
			return nil, err
		}
		titleEmojiF, err = loadFace(notoEmojiTTF, titleFontSize)
		if err != nil {
			return nil, err
		}
		titleLines = wrapText(titleFace, fullText, maxTextWidth)
		lineHeight := int(titleFontSize * 1.4)
		totalTitleHeight := len(titleLines) * lineHeight
		if totalTitleHeight <= availableForTitle {
			break
		}
		titleFontSize -= 2
	}

	// Draw title lines
	lineHeight := int(titleFontSize * 1.4)
	for _, line := range titleLines {
		if y > imgHeight-padding-bottomHeight {
			break
		}
		drawShadowedText(img, titleFace, titleEmojiF, padding, y, line, bg)
		y += lineHeight
	}

	// Gap before results
	y += 20

	// Vote counts
	countLine := voteformat.FormatVoteCountsLong(counts)
	drawShadowedText(img, regularFace, emojiFace, padding, y, countLine, bg)
	y += 70

	// Fraktion breakdown
	if breakdown != "" {
		bdLines := strings.Split(breakdown, "\n")
		for _, line := range bdLines {
			if y > imgHeight-padding {
				break
			}
			if strings.HasPrefix(line, "\U0001F3DB") {
				drawShadowedText(img, regularFace, emojiFace, padding, y, line, bg)
			} else {
				drawShadowedText(img, smallFace, emojiFaceSmall, padding, y, line, bg)
			}
			y += 50
		}
	}

	return encodeJPEG(img)
}

func renderTitleCard(votes []zurichapi.Abstimmung, bg color.RGBA, boldFace, regularFace, emojiFace font.Face) ([]byte, error) {
	img := newImage(bg)
	v := votes[0]
	y := padding + 60

	// Header: "🗳️ Gemeinderat | Abstimmung vom DD.MM.YYYY"
	date := voteformat.FormatVoteDate(v.SitzungDatum)
	header := fmt.Sprintf("\U0001F5F3\uFE0F Gemeinderat | Abstimmung vom %s", date)
	drawShadowedText(img, regularFace, emojiFace, padding, y, header, bg)
	y += 80

	// Result + Title combined: "✅ Angenommen: Schlussabstimmung..."
	counts := voteformat.VoteCounts{
		Ja: v.AnzahlJa, Nein: v.AnzahlNein, Enthaltung: v.AnzahlEnthaltung,
		Abwesend: v.AnzahlAbwesend, A: v.AnzahlA, B: v.AnzahlB, C: v.AnzahlC,
		D: v.AnzahlD, E: v.AnzahlE,
	}
	var resultPrefix string
	if !voteformat.IsAuswahlVote(counts) {
		resultPrefix = voteformat.GetVoteResultEmoji(v.Schlussresultat) + " " + voteformat.GetVoteResultText(v.Schlussresultat)
	} else {
		resultPrefix = v.Schlussresultat
	}
	title := voteformat.CleanVoteTitle(
		voteformat.SelectBestTitle(v.TraktandumTitel, v.GeschaeftTitel),
	)
	fullText := resultPrefix + ": " + title

	maxTextWidth := imgWidth - 2*padding
	lines := wrapText(regularFace, fullText, maxTextWidth)
	for _, line := range lines {
		if y > imgHeight-padding {
			break
		}
		drawShadowedText(img, regularFace, emojiFace, padding, y, line, bg)
		y += 50
	}

	return encodeJPEG(img)
}

func renderResultCard(v *zurichapi.Abstimmung, bg color.RGBA, boldFace, regularFace, smallFace, emojiFaceLarge, emojiFace, emojiFaceSmall font.Face) ([]byte, error) {
	img := newImage(bg)
	y := padding + 50

	// Subtitle if present (for multi-vote groups)
	if v.Abstimmungstitel != "" {
		sub := voteformat.CleanVoteSubtitle(v.Abstimmungstitel)
		drawShadowedText(img, boldFace, emojiFaceLarge, padding, y, sub, bg)
		y += 70
	}

	// Vote counts with 📊 emoji
	counts := voteformat.VoteCounts{
		Ja: v.AnzahlJa, Nein: v.AnzahlNein, Enthaltung: v.AnzahlEnthaltung,
		Abwesend: v.AnzahlAbwesend, A: v.AnzahlA, B: v.AnzahlB, C: v.AnzahlC,
		D: v.AnzahlD, E: v.AnzahlE,
	}
	countLine := voteformat.FormatVoteCountsLong(counts)
	drawShadowedText(img, regularFace, emojiFace, padding, y, countLine, bg)
	y += 80

	// Separator line
	y += 10

	// Fraktion breakdown with 🏛️ emoji
	fraktionCounts := voteformat.AggregateFraktionCounts(v.Stimmabgaben.Stimmabgabe)
	breakdown := voteformat.FormatFraktionBreakdown(fraktionCounts)
	if breakdown != "" {
		bdLines := strings.Split(breakdown, "\n")
		for _, line := range bdLines {
			if y > imgHeight-padding {
				break
			}
			if strings.HasPrefix(line, "\U0001F3DB") {
				drawShadowedText(img, regularFace, emojiFace, padding, y, line, bg)
			} else {
				drawShadowedText(img, smallFace, emojiFaceSmall, padding, y, line, bg)
			}
			y += 50
		}
	}

	return encodeJPEG(img)
}
