package imagegen

import (
	"bytes"
	"image/jpeg"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
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

func TestLayoutResultCard_WrapsLongSubtitle(t *testing.T) {
	fonts, err := loadFontSet()
	if err != nil {
		t.Fatalf("loadFontSet failed: %v", err)
	}

	votes := testfixtures.MultiVoteGroup()
	shortVote := votes[0]
	shortVote.Abstimmungstitel = "Kurz"

	longVote := votes[0]
	longVote.Abstimmungstitel = "Änderungsantrag zur Teilrevision der Gemeindeordnung mit zusätzlichen Bestimmungen zur Stadtentwicklung und Raumplanung"

	bg := SelectColor(shortVote.GeschaeftGrNr)

	shortCur := newCursor(0, imgHeight)
	layoutResultCard(nil, shortCur, &shortVote, bg, fonts)

	longCur := newCursor(0, imgHeight)
	layoutResultCard(nil, longCur, &longVote, bg, fonts)

	if longCur.contentHeight() <= shortCur.contentHeight() {
		t.Fatalf("expected wrapped long subtitle to use more vertical space (short=%d, long=%d)", shortCur.contentHeight(), longCur.contentHeight())
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
