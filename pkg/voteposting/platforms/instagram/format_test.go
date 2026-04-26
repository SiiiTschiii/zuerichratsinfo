package instagram

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func TestFormatCarousel_EmptyVotes(t *testing.T) {
	_, err := FormatCarousel(nil)
	if err == nil {
		t.Error("expected error for nil votes")
	}

	_, err = FormatCarousel([]zurichapi.Abstimmung{})
	if err == nil {
		t.Error("expected error for empty votes")
	}
}

func TestFormatCarousel_SingleVote(t *testing.T) {
	votes := testfixtures.SingleVoteAngenommen()
	content, err := FormatCarousel(votes)
	if err != nil {
		t.Fatalf("FormatCarousel error: %v", err)
	}

	// Single vote: 1 combined image
	if len(content.Images) != 1 {
		t.Errorf("expected 1 image for single vote, got %d", len(content.Images))
	}

	// Caption must contain key parts
	for _, part := range []string{
		"🗳️ Gemeinderat",
		"Angenommen",
		"✅",
		"📊",
		"Ja",
		"Nein",
		"🔗",
	} {
		if !strings.Contains(content.Caption, part) {
			t.Errorf("caption missing %q\nFull caption:\n%s", part, content.Caption)
		}
	}
}

func TestFormatCarousel_MultiVote(t *testing.T) {
	votes := testfixtures.MultiVoteGroup()
	content, err := FormatCarousel(votes)
	if err != nil {
		t.Fatalf("FormatCarousel error: %v", err)
	}

	// Multi-vote: 1 title card + 1 per vote = 3
	expectedImages := 1 + len(votes)
	if len(content.Images) != expectedImages {
		t.Errorf("expected %d images for multi-vote, got %d", expectedImages, len(content.Images))
	}

	// Caption must contain vote subtitles
	for _, part := range []string{
		"Einleitungsartikel",
		"Schlussabstimmung",
		"🔗",
	} {
		if !strings.Contains(content.Caption, part) {
			t.Errorf("caption missing %q\nFull caption:\n%s", part, content.Caption)
		}
	}
}

func TestFormatCarousel_CaptionWithinLimit(t *testing.T) {
	// Test all fixtures to ensure no caption exceeds Instagram's 2200 char limit
	for name, votes := range testfixtures.AllFixtures() {
		t.Run(name, func(t *testing.T) {
			content, err := FormatCarousel(votes)
			if err != nil {
				t.Fatalf("FormatCarousel error: %v", err)
			}

			runeCount := len([]rune(content.Caption))
			if runeCount > maxCaptionChars {
				t.Errorf("caption exceeds %d chars: %d\nCaption:\n%s", maxCaptionChars, runeCount, content.Caption)
			}
		})
	}
}

func TestFormatCarousel_CarouselImageCap(t *testing.T) {
	// ten-vote-stress-test has 10 votes → would produce 11 images (1 title + 10 results)
	// Instagram carousel cap is 10, so it should be trimmed
	votes := testfixtures.TenVoteStressTest()
	content, err := FormatCarousel(votes)
	if err != nil {
		t.Fatalf("FormatCarousel error: %v", err)
	}

	if len(content.Images) > maxCarouselImages {
		t.Errorf("expected at most %d images (carousel cap), got %d", maxCarouselImages, len(content.Images))
	}
}

func TestFormatCarousel_AllFixtures(t *testing.T) {
	for name, votes := range testfixtures.AllFixtures() {
		t.Run(name, func(t *testing.T) {
			content, err := FormatCarousel(votes)
			if err != nil {
				t.Fatalf("FormatCarousel error: %v", err)
			}

			if len(content.Images) == 0 {
				t.Error("expected at least 1 image")
			}

			if len(content.Images) > maxCarouselImages {
				t.Errorf("images exceed carousel cap: %d > %d", len(content.Images), maxCarouselImages)
			}

			if content.Caption == "" {
				t.Error("caption should not be empty")
			}

			// Caption should always contain a link
			if !strings.Contains(content.Caption, "🔗") {
				t.Errorf("caption should contain a link\n%s", content.Caption)
			}

			// Verify images are valid JPEG (check first 2 bytes: FF D8)
			for i, img := range content.Images {
				if len(img) < 2 || img[0] != 0xFF || img[1] != 0xD8 {
					t.Errorf("image %d is not a valid JPEG", i)
				}
			}
		})
	}
}

func TestFormatCarousel_CaptionContainsVoteLink(t *testing.T) {
	votes := testfixtures.SingleVoteAngenommen()
	content, err := FormatCarousel(votes)
	if err != nil {
		t.Fatalf("FormatCarousel error: %v", err)
	}

	if !strings.Contains(content.Caption, "gemeinderat-zuerich.ch") {
		t.Errorf("caption should contain vote link\n%s", content.Caption)
	}
}

func TestFormatCarousel_LongMultiVoteCaptionPreservesLink(t *testing.T) {
	votes := testfixtures.InstagramLongMultiVoteTruncation()

	content, err := FormatCarousel(votes)
	if err != nil {
		t.Fatalf("FormatCarousel error: %v", err)
	}

	expectedLink := voteformat.GenerateTraktandumLink(votes[0].SitzungGuid, votes[0].TraktandumGuid)
	expectedLinkLine := "🔗 " + expectedLink

	if len([]rune(content.Caption)) > maxCaptionChars {
		t.Errorf("caption exceeds %d chars: %d", maxCaptionChars, len([]rune(content.Caption)))
	}

	if !strings.Contains(content.Caption, expectedLinkLine) {
		t.Errorf("caption should contain full link line %q\n%s", expectedLinkLine, content.Caption)
	}

	if !strings.Contains(content.Caption, captionTruncatedNoticeLine) {
		t.Errorf("caption should contain truncation notice %q\n%s", captionTruncatedNoticeLine, content.Caption)
	}

	if !strings.HasSuffix(content.Caption, expectedLinkLine) {
		t.Errorf("caption should end with link line %q\n%s", expectedLinkLine, content.Caption)
	}
}

func TestFormatCarousel_ShortCaptionHasNoTruncationNotice(t *testing.T) {
	votes := testfixtures.MultiVoteGroup()
	content, err := FormatCarousel(votes)
	if err != nil {
		t.Fatalf("FormatCarousel error: %v", err)
	}

	if strings.Contains(content.Caption, captionTruncatedNoticeLine) {
		t.Errorf("short caption should not contain truncation notice\n%s", content.Caption)
	}
}

func TestFormatCarousel_CaptionContainsFraktionBreakdown(t *testing.T) {
	votes := testfixtures.SingleVoteAngenommen()
	content, err := FormatCarousel(votes)
	if err != nil {
		t.Fatalf("FormatCarousel error: %v", err)
	}

	if !strings.Contains(content.Caption, "🏛️ Fraktionen") {
		t.Errorf("caption should contain Fraktion breakdown\n%s", content.Caption)
	}
}

func TestFormatCarousel_AuswahlVoteNoResultEmoji(t *testing.T) {
	votes := testfixtures.AuswahlVote()
	content, err := FormatCarousel(votes)
	if err != nil {
		t.Fatalf("FormatCarousel error: %v", err)
	}

	// Auswahl votes should not have ✅/❌ in caption
	if strings.Contains(content.Caption, "✅") || strings.Contains(content.Caption, "❌") {
		t.Errorf("Auswahl vote caption should not contain result emoji\n%s", content.Caption)
	}

	// But should contain A/B/C counts
	if !strings.Contains(content.Caption, "A:") || !strings.Contains(content.Caption, "B:") {
		t.Errorf("Auswahl vote caption should contain A/B counts\n%s", content.Caption)
	}
}

func TestContentString(t *testing.T) {
	content := &InstagramContent{
		Images:  make([][]byte, 3),
		Caption: "Test caption",
	}

	s := content.String()
	if !strings.Contains(s, "3 image(s)") {
		t.Errorf("String() should include image count, got: %s", s)
	}
	if !strings.Contains(s, "Test caption") {
		t.Errorf("String() should include caption, got: %s", s)
	}
}

func TestFormatCarouselWithContacts_TagsInstagramHandlesInTitle(t *testing.T) {
	votes := testfixtures.SingleVoteAngenommen()
	votes[0].TraktandumTitel = "Postulat von Anna Graff"

	contactsFile := filepath.Join(t.TempDir(), "contacts.yaml")
	err := os.WriteFile(contactsFile, []byte(`version: "1.0"
contacts:
  - name: Anna Graff
    instagram:
      - https://www.instagram.com/annagraff_/
`), 0o600)
	if err != nil {
		t.Fatalf("write contacts file: %v", err)
	}

	mapper, err := contacts.LoadContacts(contactsFile)
	if err != nil {
		t.Fatalf("load contacts: %v", err)
	}

	content, err := FormatCarouselWithContacts(votes, mapper)
	if err != nil {
		t.Fatalf("FormatCarouselWithContacts error: %v", err)
	}

	if !strings.Contains(content.Caption, "Anna Graff @annagraff_") {
		t.Errorf("expected caption to include tagged Instagram handle\n%s", content.Caption)
	}
}
