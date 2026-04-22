package imagegen

import (
	"bytes"
	"image/color"
	"image/jpeg"
	"math"
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
	if c1 == c2 {
		t.Fatal("consecutive business numbers should rotate to different colors")
	}
}

func TestPalette_BrandAlignedAndContrast(t *testing.T) {
	if len(palette) < 3 || len(palette) > 5 {
		t.Fatalf("expected palette size between 3 and 5, got %d", len(palette))
	}

	brandBlue := colorHex(0x00, 0x69, 0xC7)
	if colorHex(palette[0].R, palette[0].G, palette[0].B) != brandBlue {
		t.Fatalf("expected first palette color to be brand blue %s", brandBlue)
	}

	for i, c := range palette {
		if contrastRatioWithWhite(c) < 4.5 {
			t.Fatalf("palette color %d (%s) has insufficient contrast with white text", i, colorHex(c.R, c.G, c.B))
		}
	}
}

func colorHex(r, g, b uint8) string {
	return string([]byte{
		'#',
		hexNibble(r >> 4), hexNibble(r & 0x0F),
		hexNibble(g >> 4), hexNibble(g & 0x0F),
		hexNibble(b >> 4), hexNibble(b & 0x0F),
	})
}

func hexNibble(v uint8) byte {
	if v < 10 {
		return '0' + v
	}
	return 'A' + (v - 10)
}

func contrastRatioWithWhite(c color.RGBA) float64 {
	l := relativeLuminance(c)
	return (1.0 + 0.05) / (l + 0.05)
}

func relativeLuminance(c color.RGBA) float64 {
	r := linearized(float64(c.R) / 255.0)
	g := linearized(float64(c.G) / 255.0)
	b := linearized(float64(c.B) / 255.0)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func linearized(v float64) float64 {
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}
