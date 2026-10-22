package facebook

import (
	"errors"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
)

type publishCall struct {
	message string
	link    string
}

func newTestPlatform(calls *[]publishCall, maxPostsPerRun int, err error) *FacebookPlatform {
	return &FacebookPlatform{
		maxPostsPerRun: maxPostsPerRun,
		publishPostFunc: func(message, link string) (string, error) {
			*calls = append(*calls, publishCall{message, link})
			if err != nil {
				return "", err
			}
			return "page_1_post_1", nil
		},
	}
}

func TestPost_PublishesMessageAndLink(t *testing.T) {
	var calls []publishCall
	p := newTestPlatform(&calls, 5, nil)

	content, err := p.Format(testfixtures.SingleVoteAngenommen())
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if _, err := p.Post(content); err != nil {
		t.Fatalf("Post: %v", err)
	}

	fb := content.(*FacebookContent)
	if len(calls) != 1 {
		t.Fatalf("expected 1 publish call, got %d", len(calls))
	}
	if calls[0].message != fb.Message || calls[0].link != fb.Link {
		t.Errorf("published %+v, want message/link from the formatted content", calls[0])
	}
}

func TestPost_StopsAtBudget(t *testing.T) {
	var calls []publishCall
	p := newTestPlatform(&calls, 2, nil)
	content, _ := p.Format(testfixtures.SingleVoteAngenommen())

	if cont, _ := p.Post(content); !cont {
		t.Error("first post should leave budget to continue")
	}
	if cont, _ := p.Post(content); cont {
		t.Error("second post should exhaust a budget of 2")
	}
}

// A failed publish must not count against the budget or report success: the
// pipeline would mark the votes as posted.
func TestPost_PropagatesError(t *testing.T) {
	var calls []publishCall
	p := newTestPlatform(&calls, 5, errors.New("boom"))
	content, _ := p.Format(testfixtures.SingleVoteAngenommen())

	if _, err := p.Post(content); err == nil {
		t.Fatal("expected error")
	}
	if p.postsThisRun != 0 {
		t.Errorf("failed post counted against the budget: %d", p.postsThisRun)
	}
}

func TestPost_StubModeCallsNoAPI(t *testing.T) {
	p := NewFacebookPlatform(1)
	content, _ := p.Format(testfixtures.SingleVoteAngenommen())

	cont, err := p.Post(content)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if cont {
		t.Error("budget of 1 should be exhausted after one post")
	}
}

func TestPost_RejectsForeignContent(t *testing.T) {
	p := NewFacebookPlatform(1)
	if _, err := p.Post(nil); err == nil {
		t.Fatal("expected error for non-Facebook content")
	}
}
