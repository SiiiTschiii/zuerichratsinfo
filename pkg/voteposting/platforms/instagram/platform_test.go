package instagram

import (
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
)

// Verify InstagramPlatform satisfies platforms.Platform at compile time.
var _ platforms.Platform = (*InstagramPlatform)(nil)

func TestNewInstagramPlatform(t *testing.T) {
	p := NewInstagramPlatform(5)
	if p.MaxPostsPerRun() != 5 {
		t.Errorf("expected MaxPostsPerRun=5, got %d", p.MaxPostsPerRun())
	}
	if p.Name() != "Instagram" {
		t.Errorf("expected Name()=Instagram, got %q", p.Name())
	}
}

func TestPlatform_FormatAndPost(t *testing.T) {
	p := NewInstagramPlatform(5)

	votes := testfixtures.SingleVoteAngenommen()
	content, err := p.Format(votes)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	igContent, ok := content.(*InstagramContent)
	if !ok {
		t.Fatal("expected *InstagramContent from Format")
	}

	if len(igContent.Images) == 0 {
		t.Error("expected at least 1 image")
	}
	if igContent.Caption == "" {
		t.Error("expected non-empty caption")
	}

	shouldContinue, err := p.Post(content)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if !shouldContinue {
		t.Error("expected shouldContinue=true after first post with limit=5")
	}
}

func TestPlatform_PostLimitReached(t *testing.T) {
	p := NewInstagramPlatform(1)

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

func TestPlatform_PostWrongContentType(t *testing.T) {
	p := NewInstagramPlatform(5)

	// Create a mock content that doesn't implement InstagramContent
	_, err := p.Post(mockContent{})
	if err == nil {
		t.Error("expected error for wrong content type")
	}
}

type mockContent struct{}

func (m mockContent) String() string { return "mock" }
