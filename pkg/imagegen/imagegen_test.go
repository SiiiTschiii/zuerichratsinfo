package imagegen

import (
	"bytes"
	"image/jpeg"
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
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

func TestFormatSummaryLine_NumberingAndTruncation(t *testing.T) {
	vote := zurichapi.Abstimmung{
		Abstimmungstitel: "Antrag SP sehr lange Beschreibung mit noch mehr Details für die Übersicht auf der Titelfolie",
		Schlussresultat:  "angenommen",
	}

	line, ok := formatSummaryLine(2, vote)
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
