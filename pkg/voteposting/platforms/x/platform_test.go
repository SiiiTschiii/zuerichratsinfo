package x

import (
	"fmt"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
)

// tweetCall records a single call to the mock postTweetFunc.
type tweetCall struct {
	message          string
	inReplyToTweetID string
}

// mockPostTweet returns a PostTweetFunc that records calls and returns sequential fake tweet IDs.
func mockPostTweet(calls *[]tweetCall) PostTweetFunc {
	counter := 0
	return func(apiKey, apiSecret, accessToken, accessSecret, message, inReplyToTweetID string) (string, error) {
		*calls = append(*calls, tweetCall{message: message, inReplyToTweetID: inReplyToTweetID})
		counter++
		return fmt.Sprintf("fake-tweet-%d", counter), nil
	}
}

// newTestXPlatform creates an XPlatform with a mock postTweetFunc.
func newTestXPlatform(calls *[]tweetCall, maxPostsPerRun int) *XPlatform {
	return &XPlatform{
		apiKey:         "test-key",
		apiSecret:      "test-secret",
		accessToken:    "test-token",
		accessSecret:   "test-access-secret",
		maxPostsPerRun: maxPostsPerRun,
		postsThisRun:   0,
		postTweetFunc:  mockPostTweet(calls),
	}
}

func TestPost_SingleTweet(t *testing.T) {
	var calls []tweetCall
	p := newTestXPlatform(&calls, 10)

	votes := testfixtures.SingleVoteAngenommen()
	content, err := p.Format(votes)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	shouldContinue, err := p.Post(content)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if !shouldContinue {
		t.Error("expected shouldContinue=true")
	}

	if len(calls) < 1 {
		t.Fatalf("expected at least 1 call, got %d", len(calls))
	}

	// Root tweet should have no reply ID
	if calls[0].inReplyToTweetID != "" {
		t.Errorf("root tweet should have empty inReplyToTweetID, got %q", calls[0].inReplyToTweetID)
	}
}

func TestPost_ThreadChaining(t *testing.T) {
	var calls []tweetCall
	p := newTestXPlatform(&calls, 10)

	// Use multi-vote group to generate a thread with replies
	votes := testfixtures.MultiVoteGroup()
	content, err := p.Format(votes)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	shouldContinue, err := p.Post(content)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if !shouldContinue {
		t.Error("expected shouldContinue=true")
	}

	if len(calls) < 2 {
		t.Fatalf("expected at least 2 calls (root + reply), got %d", len(calls))
	}

	// Root has no reply ID
	if calls[0].inReplyToTweetID != "" {
		t.Errorf("root should have empty inReplyToTweetID, got %q", calls[0].inReplyToTweetID)
	}

	// Each reply should reference the previous tweet ID
	for i := 1; i < len(calls); i++ {
		expectedParent := fmt.Sprintf("fake-tweet-%d", i)
		if calls[i].inReplyToTweetID != expectedParent {
			t.Errorf("call %d: expected inReplyToTweetID=%q, got %q", i, expectedParent, calls[i].inReplyToTweetID)
		}
	}
}

func TestPost_EmptyThread(t *testing.T) {
	var calls []tweetCall
	p := newTestXPlatform(&calls, 10)

	// Create empty content
	content := &XContent{thread: []*XPost{}}
	_, err := p.Post(content)
	if err == nil {
		t.Error("expected error for empty thread")
	}
}

func TestPost_PostLimitReached(t *testing.T) {
	var calls []tweetCall
	p := newTestXPlatform(&calls, 1)

	votes := testfixtures.SingleVoteAngenommen()
	content, err := p.Format(votes)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	shouldContinue, err := p.Post(content)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if shouldContinue {
		t.Error("expected shouldContinue=false after reaching maxPostsPerRun=1")
	}
}
