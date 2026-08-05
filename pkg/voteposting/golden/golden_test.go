// Package golden holds a cross-platform snapshot test over every fixture.
//
// It exists to make refactors that are supposed to be behaviour-preserving
// provable: the file testdata/golden.txt contains the exact text every platform
// formatter produces for every fixture, plus a hash of every generated image.
// Any diff is a behaviour change and must be justified, not accepted.
//
// Regenerate with:
//
//	go test ./pkg/voteposting/golden -update
package golden

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/bluesky"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/instagram"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

var update = flag.Bool("update", false, "rewrite testdata/golden.txt from current output")

const goldenPath = "testdata/golden.txt"

// xCharLimits covers both the free-tier and Premium post limits, because the
// packing logic in buildReplyPosts behaves differently at each.
var xCharLimits = []int{x.DefaultMaxChars, 2000}

func TestGoldenOutput(t *testing.T) {
	mapper, err := contacts.LoadContacts(filepath.Join("..", "..", "..", "data", "contacts_test.yaml"))
	if err != nil {
		t.Fatalf("loading test contacts: %v", err)
	}

	var sb strings.Builder
	for _, name := range fixtureNames() {
		votes := fixtureByName(t, name)
		sb.WriteString(fmt.Sprintf("################ FIXTURE %s ################\n\n", name))

		for _, limit := range xCharLimits {
			sb.WriteString(fmt.Sprintf("---- X (limit %d) ----\n", limit))
			for i, post := range x.FormatVoteThread(votes, mapper, limit) {
				sb.WriteString(fmt.Sprintf("[post %d]\n%s\n\n", i, post.Text))
			}
		}

		sb.WriteString("---- Bluesky ----\n")
		for i, post := range bluesky.FormatVoteThread(votes, mapper) {
			sb.WriteString(fmt.Sprintf("[post %d]\n%s\n", i, post.Text))
			for _, f := range post.Facets {
				sb.WriteString(fmt.Sprintf("[facet] %+v\n", f))
			}
			for _, m := range post.Mentions {
				sb.WriteString(fmt.Sprintf("[mention] %s @ %d-%d\n", m.Handle, m.ByteStart, m.ByteEnd))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("---- Instagram ----\n")
		content, err := instagram.FormatCarouselWithContacts(votes, mapper)
		if err != nil {
			t.Fatalf("formatting Instagram content for %s: %v", name, err)
		}
		for i, img := range content.Images {
			sum := sha256.Sum256(img)
			sb.WriteString(fmt.Sprintf("[image %d] %d bytes sha256=%s\n", i, len(img), hex.EncodeToString(sum[:])))
		}
		sb.WriteString(fmt.Sprintf("[caption]\n%s\n\n", content.Caption))
	}

	got := sb.String()

	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("creating testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden file: %v", err)
		}
		t.Logf("wrote %s (%d bytes)", goldenPath, len(got))
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file (run with -update to create it): %v", err)
	}

	if want := string(wantBytes); got != want {
		t.Errorf("output differs from %s: %s", goldenPath, firstDiff(want, got))
	}
}

// fixtureNames returns every fixture in a stable order, including the
// Instagram-only truncation fixture that AllFixtures deliberately omits.
func fixtureNames() []string {
	return append(append([]string{}, testfixtures.FixtureNames...), "instagram-long-multi-vote-truncation")
}

func fixtureByName(t *testing.T, name string) []zurichapi.Abstimmung {
	t.Helper()
	if name == "instagram-long-multi-vote-truncation" {
		return testfixtures.InstagramLongMultiVoteTruncation()
	}
	votes, ok := testfixtures.AllFixtures()[name]
	if !ok {
		t.Fatalf("unknown fixture %q", name)
	}
	return votes
}

// firstDiff reports the line number and surrounding context of the first
// difference, which is far more useful than dumping two multi-KB blobs.
func firstDiff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		w, g := "<missing>", "<missing>"
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return fmt.Sprintf("first difference at line %d\n  want: %q\n  got:  %q", i+1, w, g)
		}
	}
	return "no line difference (trailing whitespace?)"
}
