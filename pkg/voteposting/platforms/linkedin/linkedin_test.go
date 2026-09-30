package linkedin

import (
	"errors"
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
)

// TestFormatVotePost_AllFixturesFit pins the one hard rule: whatever a fixture
// holds, the post stays inside LinkedIn's limit and keeps its links whole.
func TestFormatVotePost_AllFixturesFit(t *testing.T) {
	for name, group := range testfixtures.AllFixtures() {
		t.Run(name, func(t *testing.T) {
			post := FormatVotePost(group)
			if n := runeLen(post.Text); n > maxChars {
				t.Fatalf("%d characters, limit %d\n%s", n, maxChars, post.Text)
			}
			if !strings.Contains(post.Text, "#Abstimmung") && !strings.Contains(post.Text, "#Wahl") {
				t.Errorf("missing topic hashtag\n%s", post.Text)
			}
			if post.LinkURL != "" && !strings.Contains(post.Text, post.LinkURL) {
				t.Errorf("preview URL %q is not in the text", post.LinkURL)
			}
		})
	}
}

func TestFormatVotePost_SingleVote(t *testing.T) {
	post := FormatVotePost(testfixtures.SingleVoteAngenommen())
	for _, want := range []string{"🗳️ Gemeinderat ZH | Abstimmung vom", "Angenommen", "📊", "#GemeinderatZürich"} {
		if !strings.Contains(post.Text, want) {
			t.Errorf("missing %q\n%s", want, post.Text)
		}
	}
	if post.LinkURL == "" {
		t.Error("expected a link to preview")
	}
}

func TestFormatVotePost_KantonsratHashtag(t *testing.T) {
	post := FormatVotePost(testfixtures.KantonsratVote())
	if !strings.Contains(post.Text, "#KantonsratZürich") {
		t.Errorf("missing chamber hashtag\n%s", post.Text)
	}
}

func TestFormatVotePost_LongTitleTruncatesTitleNotLinks(t *testing.T) {
	group := testfixtures.SingleVoteAngenommen()
	group[0].Title = strings.Repeat("Sehr langer Titel ", 300)
	post := FormatVotePost(group)
	if runeLen(post.Text) > maxChars {
		t.Fatalf("over limit: %d", runeLen(post.Text))
	}
	if !strings.Contains(post.Text, "…") || !strings.Contains(post.Text, post.LinkURL) || !strings.Contains(post.Text, "📊") {
		t.Errorf("title should give way, not counts or links\n%s", post.Text)
	}
}

func TestFormatVotePost_EmptyGroup(t *testing.T) {
	if FormatVotePost(nil) != nil {
		t.Error("expected nil for an empty group")
	}
}

func TestPlatform_PostCountsAgainstBudget(t *testing.T) {
	var texts, urls []string
	p := NewLinkedInPlatform("token", "urn:li:person:x", 2)
	p.sharePostFunc = func(text, url string) (string, error) {
		texts, urls = append(texts, text), append(urls, url)
		return "urn:li:share:1", nil
	}
	content, err := p.Format(testfixtures.SingleVoteAngenommen())
	if err != nil {
		t.Fatal(err)
	}
	if cont, err := p.Post(content); err != nil || !cont {
		t.Fatalf("first post: cont=%v err=%v", cont, err)
	}
	if cont, err := p.Post(content); err != nil || cont {
		t.Fatalf("second post should reach the limit: cont=%v err=%v", cont, err)
	}
	if len(texts) != 2 || texts[0] == "" || urls[0] == "" {
		t.Errorf("unexpected calls: %v %v", texts, urls)
	}
	if p.Name() != "LinkedIn" || p.MaxPostsPerRun() != 2 {
		t.Error("wrong name or limit")
	}
}

func TestPlatform_PostFailure(t *testing.T) {
	p := NewLinkedInPlatform("token", "urn:li:person:x", 2)
	p.sharePostFunc = func(string, string) (string, error) { return "", errors.New("boom") }
	content, _ := p.Format(testfixtures.SingleVoteAngenommen())
	if _, err := p.Post(content); err == nil {
		t.Fatal("expected error")
	}
}
