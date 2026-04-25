package imagegen

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
)

func TestGenerateCarousel_ValidJPEG(t *testing.T) {
	fixtures := testfixtures.AllFixtures()
	for name, votes := range fixtures {
		t.Run(name, func(t *testing.T) {
			images, err := GenerateCarousel(votes)
			if err != nil {
				t.Fatalf("GenerateCarousel failed: %v", err)
			}
			if len(images) == 0 {
				t.Fatal("expected at least one image")
			}
			// Single vote: 1 combined image. Multi-vote: 1 title + 1 result per vote.
			var expected int
			if len(votes) == 1 {
				expected = 1
			} else {
				expected = 1 + len(votes)
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
		t.Fatal("expected error for empty votes")
	}
}

func TestSelectColor_Deterministic(t *testing.T) {
	c1 := SelectColor("2025/100")
	c2 := SelectColor("2025/100")
	if c1 != c2 {
		t.Error("same input should produce same color")
	}
}

func TestSelectColor_DifferentInputs(t *testing.T) {
	c1 := SelectColor("2025/100")
	c2 := SelectColor("2025/101")
	// Different inputs should (usually) produce different colors
	// This is probabilistic but with our palette it's very likely
	if c1 == c2 {
		t.Log("warning: different inputs produced same color (possible but unlikely)")
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
	drawFraktionTable(nil, cur, fraktionCounts, SelectColor("2025/100"), fonts.partyBold, fonts.partyNum)

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

	customImgHeight := padding + rowHeight + rowGap + maxRows*rowStride
	cur := newCursor(0, customImgHeight)
	drawFraktionTable(nil, cur, fraktionCounts, SelectColor("2025/100"), fonts.partyBold, fonts.partyNum)

	expectedY := rowHeight + rowGap + maxRows*rowHeight + (maxRows-1)*rowGap
	if cur.y != expectedY {
		t.Fatalf("expected y=%d with max %d rows, got %d", expectedY, maxRows, cur.y)
	}
}
