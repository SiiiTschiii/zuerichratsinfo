package instagram

import (
	"fmt"
	"testing"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/igapi"
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

func TestPlatform_StubMode(t *testing.T) {
	p := NewInstagramPlatform(5)
	if !p.stubMode {
		t.Error("expected stub mode when created without credentials")
	}
}

func TestPlatform_RealPostFlow(t *testing.T) {
	// Create a platform with injected mock functions to test the full posting flow
	p := NewInstagramPlatform(5)
	p.stubMode = false
	p.sleepFunc = func(_ time.Duration) {} // no-op sleep for testing
	p.waitForImagesFunc = func(_ []string) error { return nil }

	var uploadedNames []string
	var cleanedUpNames []string
	singleImageCreated := false

	p.uploadImagesFunc = func(images [][]byte, names []string) ([]string, error) {
		uploadedNames = names
		urls := make([]string, len(images))
		for i := range images {
			urls[i] = "https://example.github.io/repo/ig-images/" + names[i]
		}
		return urls, nil
	}

	p.createSingleImageContainerFunc = func(_ string, _ string) (string, error) {
		singleImageCreated = true
		return "single_container", nil
	}

	p.createMediaContainerFunc = func(_ string) (string, error) {
		t.Error("createMediaContainerFunc should not be called for single image")
		return "", nil
	}

	p.createCarouselContainerFunc = func(_ []string, _ string) (string, error) {
		t.Error("createCarouselContainerFunc should not be called for single image")
		return "", nil
	}

	p.publishContainerFunc = func(_ string) (string, error) {
		return "media_123", nil
	}

	p.pollContainerStatusFunc = func(_ string) (string, error) {
		return igapi.StatusPublished, nil
	}

	p.cleanupImagesFunc = func(names []string) error {
		cleanedUpNames = names
		return nil
	}

	// Format and post
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
		t.Error("expected shouldContinue=true with limit=5")
	}

	// Verify the full flow was executed
	if len(uploadedNames) == 0 {
		t.Error("expected images to be uploaded")
	}
	if !singleImageCreated {
		t.Error("expected single image container to be created")
	}
	if len(cleanedUpNames) == 0 {
		t.Error("expected images to be cleaned up")
	}
	// Upload and cleanup should use same names
	if len(uploadedNames) != len(cleanedUpNames) {
		t.Errorf("upload names (%d) != cleanup names (%d)", len(uploadedNames), len(cleanedUpNames))
	}
}

func TestPlatform_RealPostFlow_UploadError(t *testing.T) {
	p := NewInstagramPlatform(5)
	p.stubMode = false
	p.sleepFunc = func(_ time.Duration) {}
	p.waitForImagesFunc = func(_ []string) error { return nil }

	p.uploadImagesFunc = func(_ [][]byte, _ []string) ([]string, error) {
		return nil, errTest
	}

	votes := testfixtures.SingleVoteAngenommen()
	content, err := p.Format(votes)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	_, err = p.Post(content)
	if err == nil {
		t.Fatal("expected error when upload fails")
	}
}

func TestPlatform_RealPostFlow_ContainerError(t *testing.T) {
	p := NewInstagramPlatform(5)
	p.stubMode = false
	p.sleepFunc = func(_ time.Duration) {}
	p.waitForImagesFunc = func(_ []string) error { return nil }

	p.uploadImagesFunc = func(images [][]byte, names []string) ([]string, error) {
		urls := make([]string, len(images))
		for i := range images {
			urls[i] = "https://example.com/" + names[i]
		}
		return urls, nil
	}

	p.createSingleImageContainerFunc = func(_ string, _ string) (string, error) {
		return "", errTest
	}

	p.createMediaContainerFunc = func(_ string) (string, error) {
		return "", errTest
	}

	votes := testfixtures.SingleVoteAngenommen()
	content, err := p.Format(votes)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	_, err = p.Post(content)
	if err == nil {
		t.Fatal("expected error when container creation fails")
	}
}

func TestPlatform_PollUntilPublished_ImmediateSuccess(t *testing.T) {
	p := NewInstagramPlatform(5)
	p.sleepFunc = func(_ time.Duration) {}
	p.pollContainerStatusFunc = func(_ string) (string, error) {
		return igapi.StatusPublished, nil
	}

	err := p.pollUntilPublished("container_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlatform_PollUntilPublished_ErrorStatus(t *testing.T) {
	p := NewInstagramPlatform(5)
	p.sleepFunc = func(_ time.Duration) {}
	p.pollContainerStatusFunc = func(_ string) (string, error) {
		return igapi.StatusError, nil
	}

	err := p.pollUntilPublished("container_1")
	if err == nil {
		t.Fatal("expected error for ERROR status")
	}
}

func TestPlatform_PollUntilPublished_ExpiredStatus(t *testing.T) {
	p := NewInstagramPlatform(5)
	p.sleepFunc = func(_ time.Duration) {}
	p.pollContainerStatusFunc = func(_ string) (string, error) {
		return igapi.StatusExpired, nil
	}

	err := p.pollUntilPublished("container_1")
	if err == nil {
		t.Fatal("expected error for EXPIRED status")
	}
}

func TestPlatform_PollUntilPublished_FinishedStatus(t *testing.T) {
	p := NewInstagramPlatform(5)
	p.sleepFunc = func(_ time.Duration) {}
	p.pollContainerStatusFunc = func(_ string) (string, error) {
		return igapi.StatusFinished, nil
	}

	err := p.pollUntilPublished("container_1")
	if err != nil {
		t.Fatalf("unexpected error for FINISHED status: %v", err)
	}
}

func TestPlatform_NewWithCredentials(t *testing.T) {
	p := NewInstagramPlatformWithCredentials("ig_user", "token", "gh_token", "owner", "repo", 10)
	if p.stubMode {
		t.Error("expected real mode when created with credentials")
	}
	if p.MaxPostsPerRun() != 10 {
		t.Errorf("expected MaxPostsPerRun=10, got %d", p.MaxPostsPerRun())
	}
	if p.igClient == nil {
		t.Error("expected igClient to be set")
	}
	if p.imageHoster == nil {
		t.Error("expected imageHoster to be set")
	}
}

var errTest = fmt.Errorf("test error")
