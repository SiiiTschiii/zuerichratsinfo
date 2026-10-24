package facebook

import (
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
)

func TestFormatPost_EmptyVotes(t *testing.T) {
	if _, err := FormatPost(nil); err == nil {
		t.Fatal("expected error for an empty group")
	}
}

func TestFormatPost_AllFixtures(t *testing.T) {
	for name, group := range testfixtures.AllFixtures() {
		t.Run(name, func(t *testing.T) {
			content, err := FormatPost(group)
			if err != nil {
				t.Fatalf("FormatPost: %v", err)
			}
			if len(content.Images) == 0 || len(content.Images) > maxPhotos {
				t.Errorf("%d images, want 1 to %d", len(content.Images), maxPhotos)
			}
			if !strings.HasPrefix(content.Message, "🗳️ ") {
				t.Errorf("message should open with the headline, got %q", firstLine(content.Message))
			}
			if strings.HasSuffix(content.Message, "\n") {
				t.Errorf("message should not end in a newline")
			}
			// With photos attached there is no preview card, so the URLs are the
			// only way to the vote page: every one the other platforms carry has
			// to be in the text.
			for _, l := range voteformat.GroupLinks(group) {
				if !strings.Contains(content.Message, l.URL) {
					t.Errorf("message is missing link %s", l.URL)
				}
			}
		})
	}
}

// The body is the Instagram caption's body, untagged and untruncated.
func TestFormatPost_MatchesLongFormBody(t *testing.T) {
	group := testfixtures.MultiVoteGroup()
	content, err := FormatPost(group)
	if err != nil {
		t.Fatalf("FormatPost: %v", err)
	}
	body := strings.TrimRight(voteformat.LongFormBody(group, voteformat.Tags{}), "\n")
	if !strings.HasPrefix(content.Message, body+"\n\n") {
		t.Errorf("message does not start with the long-form body:\n%s", content.Message)
	}
}

// The Instagram fixture built to overflow 2,200 characters posts whole here.
func TestFormatPost_LongGroupIsNotTruncated(t *testing.T) {
	group := testfixtures.InstagramLongMultiVoteTruncation()
	content, err := FormatPost(group)
	if err != nil {
		t.Fatalf("FormatPost: %v", err)
	}
	if strings.Contains(content.Message, "Gekürzt") {
		t.Errorf("long group was truncated")
	}
	if n := len([]rune(content.Message)); n <= 2200 {
		t.Errorf("fixture expected to exceed Instagram's 2200 characters, got %d", n)
	}
	if len(content.Images) > maxPhotos {
		t.Errorf("%d images, over the cap of %d", len(content.Images), maxPhotos)
	}
}

func TestFormatPost_WahlgeschaeftHasNoCounts(t *testing.T) {
	content, err := FormatPost(testfixtures.KantonsratWahlgeschaeft())
	if err != nil {
		t.Fatalf("FormatPost: %v", err)
	}
	if !strings.Contains(content.Message, voteformat.WahlgeschaeftNoResult) {
		t.Errorf("Wahlgeschäft notice is missing its no-result sentence:\n%s", content.Message)
	}
	for _, marker := range []string{"✅", "❌", "Ja", "Nein"} {
		if strings.Contains(content.Message, marker) {
			t.Errorf("Wahlgeschäft notice contains %q:\n%s", marker, content.Message)
		}
	}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
