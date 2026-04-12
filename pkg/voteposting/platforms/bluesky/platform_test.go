package bluesky

import (
	"fmt"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
)

// recordCall records a single call to the mock createRecordFunc.
type recordCall struct {
	text    string
	facets  []bskyapi.Facet
	replyTo *bskyapi.ReplyRef
}

// mockCreateRecord returns a CreateRecordFunc that records calls and returns sequential fake PostRefs.
func mockCreateRecord(calls *[]recordCall) CreateRecordFunc {
	counter := 0
	return func(session *bskyapi.Session, text string, facets []bskyapi.Facet, replyTo *bskyapi.ReplyRef) (*bskyapi.PostRef, error) {
		*calls = append(*calls, recordCall{text: text, facets: facets, replyTo: replyTo})
		counter++
		return &bskyapi.PostRef{
			URI: fmt.Sprintf("at://did:fake/app.bsky.feed.post/%d", counter),
			CID: fmt.Sprintf("cid-fake-%d", counter),
		}, nil
	}
}

// mockCreateSession returns a CreateSessionFunc that returns a dummy session and tracks call count.
func mockCreateSession(callCount *int) CreateSessionFunc {
	return func(handle, password string) (*bskyapi.Session, error) {
		*callCount++
		return &bskyapi.Session{
			DID:             "did:plc:fake",
			Handle:          "test.bsky.social",
			AccessJwt:       "fake-jwt",
			ServiceEndpoint: "https://fake.bsky.social",
		}, nil
	}
}

// mockResolveHandle returns a ResolveHandleFunc that returns a synthetic DID.
func mockResolveHandle() ResolveHandleFunc {
	return func(handle string) (string, error) {
		return "did:plc:resolved-" + handle, nil
	}
}

// newTestBlueskyPlatform creates a BlueskyPlatform with all functions mocked.
func newTestBlueskyPlatform(recordCalls *[]recordCall, sessionCallCount *int, maxPostsPerRun int) *BlueskyPlatform {
	return &BlueskyPlatform{
		handle:            "test.bsky.social",
		password:          "test-password",
		maxPostsPerRun:    maxPostsPerRun,
		postsThisRun:      0,
		didCache:          make(map[string]string),
		createRecordFunc:  mockCreateRecord(recordCalls),
		createSessionFunc: mockCreateSession(sessionCallCount),
		resolveHandleFunc: mockResolveHandle(),
	}
}

func TestPost_SinglePost(t *testing.T) {
	var recordCalls []recordCall
	var sessionCalls int
	p := newTestBlueskyPlatform(&recordCalls, &sessionCalls, 10)

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

	if len(recordCalls) < 1 {
		t.Fatalf("expected at least 1 createRecord call, got %d", len(recordCalls))
	}

	// Root should have no replyTo
	if recordCalls[0].replyTo != nil {
		t.Errorf("root post should have nil replyTo, got %+v", recordCalls[0].replyTo)
	}
}

func TestPost_ThreadChaining(t *testing.T) {
	var recordCalls []recordCall
	var sessionCalls int
	p := newTestBlueskyPlatform(&recordCalls, &sessionCalls, 10)

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

	if len(recordCalls) < 2 {
		t.Fatalf("expected at least 2 createRecord calls, got %d", len(recordCalls))
	}

	// Root has no replyTo
	if recordCalls[0].replyTo != nil {
		t.Errorf("root should have nil replyTo, got %+v", recordCalls[0].replyTo)
	}

	rootURI := "at://did:fake/app.bsky.feed.post/1"
	rootCID := "cid-fake-1"

	// All replies should reference the root as Root, and the previous post as Parent
	for i := 1; i < len(recordCalls); i++ {
		call := recordCalls[i]
		if call.replyTo == nil {
			t.Fatalf("reply %d should have replyTo, got nil", i)
		}
		// Root ref should always point to the first post
		if call.replyTo.Root.URI != rootURI {
			t.Errorf("reply %d: Root.URI = %q, want %q", i, call.replyTo.Root.URI, rootURI)
		}
		if call.replyTo.Root.CID != rootCID {
			t.Errorf("reply %d: Root.CID = %q, want %q", i, call.replyTo.Root.CID, rootCID)
		}
		// Parent ref should point to the previous post
		expectedParentURI := fmt.Sprintf("at://did:fake/app.bsky.feed.post/%d", i)
		expectedParentCID := fmt.Sprintf("cid-fake-%d", i)
		if call.replyTo.Parent.URI != expectedParentURI {
			t.Errorf("reply %d: Parent.URI = %q, want %q", i, call.replyTo.Parent.URI, expectedParentURI)
		}
		if call.replyTo.Parent.CID != expectedParentCID {
			t.Errorf("reply %d: Parent.CID = %q, want %q", i, call.replyTo.Parent.CID, expectedParentCID)
		}
	}
}

func TestPost_WithMentionFacets(t *testing.T) {
	var recordCalls []recordCall
	var sessionCalls int
	p := newTestBlueskyPlatform(&recordCalls, &sessionCalls, 10)

	// Use the mention fixture — it has a politician name that triggers mention matching
	// We need a contactMapper for this, but since we mock resolveHandle, we just need
	// to verify that facets from format are passed through to createRecord
	votes := testfixtures.SingleVoteAngenommen()
	content, err := p.Format(votes)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	// Manually inject a facet to verify it's passed through
	bskyContent := content.(*BlueskyContent)
	if len(bskyContent.thread) > 0 {
		bskyContent.thread[0].Facets = append(bskyContent.thread[0].Facets,
			bskyapi.LinkFacet(0, 5, "https://example.com"))
	}

	_, err = p.Post(content)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}

	if len(recordCalls) < 1 {
		t.Fatalf("expected at least 1 call, got %d", len(recordCalls))
	}

	// Root call should have the facet we injected
	if len(recordCalls[0].facets) == 0 {
		t.Error("expected facets to be passed through to createRecord, got none")
	}
}

func TestPost_SessionLazyAuth(t *testing.T) {
	var recordCalls []recordCall
	var sessionCalls int
	p := newTestBlueskyPlatform(&recordCalls, &sessionCalls, 10)

	// Post twice — session should only be created once
	votes := testfixtures.SingleVoteAngenommen()
	content1, _ := p.Format(votes)
	content2, _ := p.Format(votes)

	_, err := p.Post(content1)
	if err != nil {
		t.Fatalf("Post 1 error: %v", err)
	}
	_, err = p.Post(content2)
	if err != nil {
		t.Fatalf("Post 2 error: %v", err)
	}

	if sessionCalls != 1 {
		t.Errorf("expected createSession to be called once, got %d", sessionCalls)
	}
}

func TestPost_EmptyThread(t *testing.T) {
	var recordCalls []recordCall
	var sessionCalls int
	p := newTestBlueskyPlatform(&recordCalls, &sessionCalls, 10)

	content := &BlueskyContent{thread: []*BlueskyPost{}}
	_, err := p.Post(content)
	if err == nil {
		t.Error("expected error for empty thread")
	}
}

func TestPost_PostLimitReached(t *testing.T) {
	var recordCalls []recordCall
	var sessionCalls int
	p := newTestBlueskyPlatform(&recordCalls, &sessionCalls, 1)

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
