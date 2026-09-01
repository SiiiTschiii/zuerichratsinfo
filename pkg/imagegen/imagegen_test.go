package imagegen

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

func TestGenerateCarousel_ValidJPEG(t *testing.T) {
	fixtures := testfixtures.AllFixtures()
	for name, group := range fixtures {
		t.Run(name, func(t *testing.T) {
			images, err := GenerateCarousel(group)
			if err != nil {
				t.Fatalf("GenerateCarousel failed: %v", err)
			}
			if len(images) == 0 {
				t.Fatal("expected at least one image")
			}
			// Single vote: 1 combined image. Multi-vote: 1 title + 1 result per vote.
			var expected int
			if len(group) == 1 {
				expected = 1
			} else {
				expected = 1 + len(group)
			}
			if len(images) != expected {
				t.Errorf("expected %d images, got %d", expected, len(images))
			}
			for i, imgData := range images {
				cfg, err := jpeg.DecodeConfig(bytes.NewReader(imgData))
				if err != nil {
					t.Errorf("image %d: not valid JPEG: %v", i, err)
					continue
				}
				if cfg.Width != 1080 || cfg.Height != 1350 {
					t.Errorf("image %d: expected 1080x1350, got %dx%d", i, cfg.Width, cfg.Height)
				}
				// Check file size < 500KB
				if len(imgData) > 500*1024 {
					t.Errorf("image %d: size %d bytes exceeds 500KB", i, len(imgData))
				}
			}
		})
	}
}

func TestGenerateCarousel_Empty(t *testing.T) {
	_, err := GenerateCarousel(nil)
	if err == nil {
		t.Fatal("expected error for empty group")
	}
}

func TestLayoutResultCard_WrapsLongSubtitle(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet failed: %v", err)
	}

	group := testfixtures.MultiVoteGroup()
	shortVote := group[0]
	shortVote.Subtitle = "Kurz"

	longVote := group[0]
	longVote.Subtitle = "Änderungsantrag zur Teilrevision der Gemeindeordnung mit zusätzlichen Bestimmungen zur Stadtentwicklung und Raumplanung"

	bg := SelectColor("zurich-city", shortVote.Affair.Number)

	shortCur := newCursor(0, imgHeight)
	layoutResultCard(nil, shortCur, &shortVote, bg, fonts, 1, 2)

	longCur := newCursor(0, imgHeight)
	layoutResultCard(nil, longCur, &longVote, bg, fonts, 2, 2)

	if longCur.contentHeight() <= shortCur.contentHeight() {
		t.Fatalf("expected wrapped long subtitle to use more vertical space (short=%d, long=%d)", shortCur.contentHeight(), longCur.contentHeight())
	}
}

func TestLayoutTitleCard_WrapsLongSummaryLine(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet failed: %v", err)
	}

	shortVotes := testfixtures.MultiVoteGroup()
	shortVotes[0].Subtitle = "Einleitung"
	shortVotes[1].Subtitle = "Schluss"

	longVotes := testfixtures.MultiVoteGroup()
	longVotes[0].Subtitle = "Einleitung"
	longVotes[1].Subtitle = "Schlussabstimmung mit zusätzlichen Bestimmungen zur Neuordnung der Kompetenzen im Bereich Stadtentwicklung und Raumplanung"

	bg := SelectColor("zurich-city", shortVotes[0].Affair.Number)

	shortCur := newCursor(0, imgHeight)
	layoutTitleCard(nil, shortCur, shortVotes, bg, fonts)

	longCur := newCursor(0, imgHeight)
	layoutTitleCard(nil, longCur, longVotes, bg, fonts)

	if longCur.contentHeight() <= shortCur.contentHeight() {
		t.Fatalf("expected wrapped long summary line to use more vertical space (short=%d, long=%d)", shortCur.contentHeight(), longCur.contentHeight())
	}
}

func TestLayoutCombinedCard_AbstimmungsgegenstandPrefix(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet failed: %v", err)
	}

	base := testfixtures.SingleVoteAngenommen()[0]
	bg := SelectColor("zurich-city", base.Affair.Number)

	// Non-Schlussabstimmung Abstimmungstitel: prepended inline in front of the
	// title, the same way the title card does it.
	withPrefix := base
	withPrefix.Subtitle = "Dringlicherklärung"
	prefixCur := newCursor(0, imgHeight)
	_, prefixLines, err := layoutCombinedCard(nil, prefixCur, &withPrefix, bg, fonts)
	if err != nil {
		t.Fatalf("layoutCombinedCard failed: %v", err)
	}
	if got := strings.Join(prefixLines, " "); !strings.HasPrefix(got, "Dringlicherklärung: ") {
		t.Fatalf("expected title to start with %q, got %q", "Dringlicherklärung: ", got)
	}

	// The ballot type stays out of it: the card would otherwise open
	// "Dringlicherklärung · Ausgabenbremse: …", gluing a fact about the ballot
	// onto a fact about the business. It gets a line with the counts instead.
	withType := withPrefix
	withType.Type = "Ausgabenbremse"
	typeCur := newCursor(0, imgHeight)
	_, typeLines, err := layoutCombinedCard(nil, typeCur, &withType, bg, fonts)
	if err != nil {
		t.Fatalf("layoutCombinedCard failed: %v", err)
	}
	if got := strings.Join(typeLines, " "); strings.Contains(got, "Ausgabenbremse") {
		t.Fatalf("expected the ballot type to stay out of the title, got %q", got)
	}
	if typeCur.contentHeight() <= prefixCur.contentHeight() {
		t.Fatalf("expected the ballot type to take a line above the counts (untyped=%d, typed=%d)",
			prefixCur.contentHeight(), typeCur.contentHeight())
	}

	// Schlussabstimmung Abstimmungstitel: no prefix added.
	schluss := base
	schluss.Subtitle = "Schlussabstimmung"
	schlussCur := newCursor(0, imgHeight)
	_, schlussLines, err := layoutCombinedCard(nil, schlussCur, &schluss, bg, fonts)
	if err != nil {
		t.Fatalf("layoutCombinedCard failed: %v", err)
	}
	if got := strings.Join(schlussLines, " "); strings.Contains(got, "Schlussabstimmung:") {
		t.Fatalf("expected no prefix for Schlussabstimmung, got %q", got)
	}

	// Empty Abstimmungstitel: no prefix added.
	none := base
	none.Subtitle = ""
	noneCur := newCursor(0, imgHeight)
	_, noneLines, err := layoutCombinedCard(nil, noneCur, &none, bg, fonts)
	if err != nil {
		t.Fatalf("layoutCombinedCard failed: %v", err)
	}
	if got := strings.Join(noneLines, " "); strings.Contains(got, ": ") && strings.HasPrefix(got, "Dringlicherklärung") {
		t.Fatalf("expected no prefix for empty Abstimmungstitel, got %q", got)
	}
}

func TestSelectColor_Deterministic(t *testing.T) {
	c1 := SelectColor("zurich-city", "2025/100")
	c2 := SelectColor("zurich-city", "2025/100")
	if c1 != c2 {
		t.Error("same input should produce same color")
	}
}

func TestSelectColor_DifferentInputs(t *testing.T) {
	c1 := SelectColor("zurich-city", "2025/100")
	c2 := SelectColor("zurich-city", "2025/101")
	// Different inputs should (usually) produce different colors
	// This is probabilistic but with our palette it's very likely
	if c1 == c2 {
		t.Log("warning: different inputs produced same color (possible but unlikely)")
	}
}

func TestFormatSummaryLine_NumberingAndTruncation(t *testing.T) {
	vote := votes.Vote{
		Subtitle: "Antrag SP sehr lange Beschreibung mit noch mehr Details für die Übersicht auf der Titelfolie und weiteren Erläuterungen zur Kompetenzordnung im Bereich Stadtentwicklung und Raumplanung",
		Decision: "angenommen",
	}

	line, ok := formatSummaryLine(2, vote, 3)
	if !ok {
		t.Fatal("expected summary line to be generated")
	}
	if !strings.HasPrefix(line, "2. ✅ ") {
		t.Fatalf("expected numbered line with emoji prefix, got %q", line)
	}
	if !strings.Contains(line, "…") {
		t.Fatalf("expected truncated subtitle with ellipsis, got %q", line)
	}
}

func TestFormatSummaryLine_AuswahlVote(t *testing.T) {
	intPtr := func(i int) *int { return &i }
	vote := votes.Vote{
		Subtitle: "Änderungsantrag 17",
		Decision: "Auswahl A",
		Absent:   intPtr(11),
		ChoiceA:  intPtr(50),
		ChoiceB:  intPtr(24),
		ChoiceC:  intPtr(40),
	}

	line, ok := formatSummaryLine(2, vote, 3)
	if !ok {
		t.Fatal("expected summary line to be generated")
	}
	if !strings.HasPrefix(line, "2. [A] ") {
		t.Fatalf("expected Auswahl summary line to show bracket label, got %q", line)
	}
	if strings.Contains(line, "❌") || strings.Contains(line, "✅") {
		t.Fatalf("Auswahl summary line should not contain a result emoji, got %q", line)
	}
}

func TestFormatProgressBadge(t *testing.T) {
	if got := formatProgressBadge(2, 3); got != "2/3" {
		t.Fatalf("expected 2/3, got %q", got)
	}
	if got := formatProgressBadge(0, 3); got != "" {
		t.Fatalf("expected empty badge for invalid index, got %q", got)
	}
	if got := formatProgressBadge(1, 1); got != "" {
		t.Fatalf("expected empty badge for single vote, got %q", got)
	}
}

func TestDrawFraktionTable_AddsRowSpacing(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet failed: %v", err)
	}

	fraktionCounts := map[string]*voteformat.FraktionCounts{
		"SP":  {Counts: map[string]int{"Ja": 20, "Nein": 5}},
		"FDP": {Counts: map[string]int{"Ja": 10, "Nein": 15}},
	}

	startY := 100
	cur := newCursor(startY, 600)
	drawFraktionTable(nil, cur, fraktionCounts, SelectColor("zurich-city", "2025/100"), fonts.partyBold, fonts.partyNum)

	rowHeight := lineHeight(fonts.partyNum)
	rowGap := int(float64(rowHeight) * fraktionRowGapFactor)
	expectedY := startY + 3*rowHeight + 2*rowGap // header + header-gap + 2 rows + 1 gap between rows
	if cur.y != expectedY {
		t.Fatalf("expected y=%d, got %d", expectedY, cur.y)
	}
}

func TestDrawFraktionTable_LimitsRowsWhenSpaceIsTight(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet failed: %v", err)
	}

	fraktionCounts := map[string]*voteformat.FraktionCounts{}
	for i := 0; i < 12; i++ {
		fraktionCounts[fmt.Sprintf("Fraktion-%d", i)] = &voteformat.FraktionCounts{
			Counts: map[string]int{"Ja": 1},
		}
	}

	rowHeight := lineHeight(fonts.partyNum)
	rowGap := int(float64(rowHeight) * fraktionRowGapFactor)
	rowStride := rowHeight + rowGap
	maxRows := 3

	// The clamp is a drawing-pass guard, not a measuring one: the measuring
	// pass has to report the card's true height so renderCombinedCard can
	// centre it without pushing rows out of the frame. So this needs a real
	// destination image to exercise it.
	customImgHeight := padding + rowHeight + rowGap + maxRows*rowStride
	bg := SelectColor("zurich-city", "2025/100")
	cur := newCursor(0, customImgHeight)
	drawFraktionTable(newImage(bg), cur, fraktionCounts, bg, fonts.partyBold, fonts.partyNum)

	expectedY := rowHeight + rowGap + maxRows*rowHeight + (maxRows-1)*rowGap
	if cur.y != expectedY {
		t.Fatalf("expected y=%d with max %d rows, got %d", expectedY, maxRows, cur.y)
	}
}

// TestMeasureMixedTextMatchesWhatIsDrawn pins the centring arithmetic to the
// faces actually used.
//
// The verdict emoji renders in NotoEmoji 72 while the text around it is gobold
// 64, so measuring an emoji with the text face reports a glyph that is never
// drawn — 48px against the real 91px. A card whose verdict was nothing but "❌"
// came out 21px right of centre, plainly off against the title beneath it.
func TestMeasureMixedTextMatchesWhatIsDrawn(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet: %v", err)
	}

	for _, emoji := range []string{"❌", "✅"} {
		mixed := measureMixedText(fonts.verdict, fonts.emojiVerdict, emoji)
		drawn := font.MeasureString(fonts.emojiVerdict, emoji).Ceil()
		if mixed != drawn {
			t.Errorf("measureMixedText(%q) = %d, but it is drawn %d wide", emoji, mixed, drawn)
		}
		if textOnly := font.MeasureString(fonts.verdict, emoji).Ceil(); textOnly == drawn {
			t.Skipf("faces now agree on %q; this test no longer proves anything", emoji)
		}
	}
}

// TestMeasureMixedTextLeavesPlainTextAlone checks the fix did not move every
// other string on the card.
func TestMeasureMixedTextLeavesPlainTextAlone(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet: %v", err)
	}

	const plain = "Mitteilungen"
	got := measureMixedText(fonts.verdict, fonts.emojiVerdict, plain)
	want := font.MeasureString(fonts.verdict, plain).Ceil()
	if got != want {
		t.Errorf("measureMixedText(%q) = %d, want %d — plain text must measure as before", plain, got, want)
	}
}

// TestCardsKeepEveryFraktion checks that no card drops rows from its party
// breakdown.
//
// drawFraktionTable stops at imgHeight-padding, so a card laid out too low
// simply loses its last Fraktionen — with nothing on the card to say any were
// dropped, and the breakdown is the one part of a vote a reader cannot
// reconstruct from the caption. The check mirrors what render*Card does: it
// measures the card, centres it the way the real render does, and asserts the
// drawing pass covers the same height the measuring pass reported. A dropped
// row makes the drawn card shorter than the measured one.
func TestCardsKeepEveryFraktion(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet failed: %v", err)
	}

	for name, group := range testfixtures.AllFixtures() {
		t.Run(name, func(t *testing.T) {
			bg := SelectColor(group[0].Jurisdiction, group[0].Affair.Number)
			inset := bandInset(group[0])

			check := func(label string, layout func(*image.RGBA, *layoutCursor)) {
				t.Helper()

				dry := newCursor(inset, imgHeight)
				layout(nil, dry)

				startY := centredStart(inset, dry.contentHeight())
				if bottom := startY + dry.contentHeight(); bottom > imgHeight-padding {
					t.Errorf("%s: content ends at y=%d, past the %dpx bottom limit", label, bottom, imgHeight-padding)
				}

				real := newCursor(startY, imgHeight)
				layout(newImage(bg), real)
				if real.contentHeight() != dry.contentHeight() {
					t.Errorf("%s: drawn card is %dpx tall against %dpx measured — the table dropped rows",
						label, real.contentHeight(), dry.contentHeight())
				}
			}

			if len(group) == 1 {
				v := group[0]
				check("combined card", func(img *image.RGBA, cur *layoutCursor) {
					if _, _, err := layoutCombinedCard(img, cur, &v, bg, fonts); err != nil {
						t.Fatalf("layoutCombinedCard failed: %v", err)
					}
				})
				return
			}

			check("title card", func(img *image.RGBA, cur *layoutCursor) {
				layoutTitleCard(img, cur, group, bg, fonts)
			})
			for i := range group {
				v := group[i]
				check(fmt.Sprintf("result card %d", i+1), func(img *image.RGBA, cur *layoutCursor) {
					layoutResultCard(img, cur, &v, bg, fonts, i+1, len(group))
				})
			}
		})
	}
}

// TestFitTitle_EllipsisesRatherThanOverflowing checks the last resort: a title
// too long to fit even at titleFontMin gives up its tail, not the results.
func TestFitTitle_EllipsisesRatherThanOverflowing(t *testing.T) {
	maxWidth := imgWidth - 2*padding

	short := "Postulat betreffend Anpassung der Mindestarealfläche"
	_, lines, err := fitTitle(short, maxWidth, 600)
	if err != nil {
		t.Fatalf("fitTitle failed: %v", err)
	}
	if got := strings.Join(lines, " "); got != short {
		t.Errorf("a title with room to spare was altered: got %q", got)
	}
	if strings.HasSuffix(strings.Join(lines, " "), "…") {
		t.Error("a title that fits must not be ellipsised")
	}

	long := strings.Repeat("Rahmenkredit für ein dreijähriges Pilotprojekt zur Schaffung einer Überbrückungshilfe ", 8)
	face, lines, err := fitTitle(long, maxWidth, 200)
	if err != nil {
		t.Fatalf("fitTitle failed: %v", err)
	}
	if h := titleBlockHeight(face, len(lines)); h > 200 {
		t.Errorf("title block is %dpx tall, over the 200px budget", h)
	}
	if !strings.HasSuffix(lines[len(lines)-1], "…") {
		t.Errorf("expected the cut title to end in an ellipsis, got %q", lines[len(lines)-1])
	}
	for i, line := range lines {
		if w := font.MeasureString(face, line).Ceil(); w > maxWidth {
			t.Errorf("line %d is %dpx wide, over the %dpx text column", i, w, maxWidth)
		}
	}
}

// TestFitTitle_BudgetTooSmallForOneLine checks the degenerate case: even a
// budget with no room for a single line yields one ellipsised line rather than
// an empty title.
func TestFitTitle_BudgetTooSmallForOneLine(t *testing.T) {
	long := strings.Repeat("Rahmenkredit für das dreijährige Pilotprojekt Überbrückungshilfe ", 4)
	_, lines, err := fitTitle(long, imgWidth-2*padding, 0)
	if err != nil {
		t.Fatalf("fitTitle failed: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected exactly one line, got %d: %q", len(lines), lines)
	}
	if !strings.HasSuffix(lines[0], "…") {
		t.Errorf("expected an ellipsis, got %q", lines[0])
	}
}

// TestAppendEllipsis_TrimsPunctuation checks the cut lands on a word, not on
// the comma that happened to follow it.
func TestAppendEllipsis_TrimsPunctuation(t *testing.T) {
	face, err := loadFace(goregular.TTF, titleFontMin)
	if err != nil {
		t.Fatalf("loadFace failed: %v", err)
	}
	if got := appendEllipsis(face, "gültigem Aufenthaltsstatus,", imgWidth-2*padding); got != "gültigem Aufenthaltsstatus…" {
		t.Errorf("got %q, want %q", got, "gültigem Aufenthaltsstatus…")
	}
}
