package instagram

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/igapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

const (
	// pollInterval is the time between container status checks.
	pollInterval = 10 * time.Second
	// pollTimeout is the maximum time to wait for a container to be published.
	pollTimeout = 5 * time.Minute
	// pagesDeploymentPollInterval is the time between GitHub Pages availability checks.
	pagesDeploymentPollInterval = 10 * time.Second
	// pagesDeploymentTimeout is the maximum time to wait for GitHub Pages deployment.
	pagesDeploymentTimeout = 3 * time.Minute
)

// CreateMediaContainerFunc is the signature for creating an IG media container.
type CreateMediaContainerFunc func(imageURL string) (string, error)

// CreateCarouselContainerFunc is the signature for creating an IG carousel container.
type CreateCarouselContainerFunc func(childIDs []string, caption string) (string, error)

// PublishContainerFunc is the signature for publishing an IG container.
type PublishContainerFunc func(containerID string) (string, error)

// PollContainerStatusFunc is the signature for polling container status.
type PollContainerStatusFunc func(containerID string) (string, error)

// UploadImagesFunc is the signature for uploading images to hosting.
type UploadImagesFunc func(images [][]byte, names []string) ([]string, error)

// CleanupImagesFunc is the signature for cleaning up hosted images.
type CleanupImagesFunc func(names []string) error

// CreateSingleImageContainerFunc is the signature for creating a standalone image container.
type CreateSingleImageContainerFunc func(imageURL, caption string) (string, error)

// InstagramPlatform implements the platforms.Platform interface for Instagram
type InstagramPlatform struct {
	postsThisRun   int
	maxPostsPerRun int
	igClient       *igapi.Client
	imageHoster    *igapi.ImageHoster
	stubMode       bool // true when no credentials are configured

	// Injectable functions for testing
	createMediaContainerFunc       CreateMediaContainerFunc
	createCarouselContainerFunc    CreateCarouselContainerFunc
	createSingleImageContainerFunc CreateSingleImageContainerFunc
	publishContainerFunc           PublishContainerFunc
	pollContainerStatusFunc        PollContainerStatusFunc
	uploadImagesFunc               UploadImagesFunc
	cleanupImagesFunc              CleanupImagesFunc
	waitForImagesFunc              func(urls []string) error
	sleepFunc                      func(time.Duration)
}

// NewInstagramPlatform creates a new Instagram platform poster in stub mode (no real posting).
func NewInstagramPlatform(maxPostsPerRun int) *InstagramPlatform {
	return &InstagramPlatform{
		maxPostsPerRun:    maxPostsPerRun,
		stubMode:          true,
		waitForImagesFunc: waitForImageURLs,
		sleepFunc:         time.Sleep,
	}
}

// NewInstagramPlatformWithCredentials creates a new Instagram platform poster with real API credentials.
func NewInstagramPlatformWithCredentials(igUserID, accessToken, githubToken, repoOwner, repoName string, maxPostsPerRun int) *InstagramPlatform {
	igClient := igapi.NewClient(igUserID, accessToken)
	hoster := igapi.NewImageHoster(repoOwner, repoName, githubToken)

	p := &InstagramPlatform{
		maxPostsPerRun: maxPostsPerRun,
		igClient:       igClient,
		imageHoster:    hoster,
		stubMode:       false,
		sleepFunc:      time.Sleep,
	}

	// Wire real API functions
	p.createMediaContainerFunc = igClient.CreateMediaContainer
	p.createCarouselContainerFunc = igClient.CreateCarouselContainer
	p.createSingleImageContainerFunc = igClient.CreateSingleImageContainer
	p.publishContainerFunc = igClient.PublishContainer
	p.pollContainerStatusFunc = igClient.PollContainerStatus
	p.uploadImagesFunc = hoster.UploadImages
	p.cleanupImagesFunc = hoster.CleanupImages
	p.waitForImagesFunc = waitForImageURLs

	return p
}

// Format formats a group of votes into Instagram-specific content (carousel images + caption).
func (p *InstagramPlatform) Format(votes []zurichapi.Abstimmung) (platforms.Content, error) {
	return FormatCarousel(votes)
}

// Post publishes content to Instagram.
// In stub mode: logs preview, no real API calls.
// In real mode: uploads images to GitHub Pages, creates carousel via IG API, polls for PUBLISHED, cleans up.
// Returns shouldContinue=false when the post limit is reached.
func (p *InstagramPlatform) Post(content platforms.Content) (bool, error) {
	igContent, ok := content.(*InstagramContent)
	if !ok {
		return false, fmt.Errorf("unexpected content type for Instagram")
	}

	if p.stubMode {
		return p.postStub(igContent)
	}

	return p.postReal(igContent)
}

// postStub logs what would be posted without making any API calls.
func (p *InstagramPlatform) postStub(igContent *InstagramContent) (bool, error) {
	fmt.Printf("📷 [Instagram stub] Would post %d image(s) with caption (%d chars):\n",
		len(igContent.Images), len([]rune(igContent.Caption)))

	preview := igContent.Caption
	if len([]rune(preview)) > 200 {
		preview = string([]rune(preview)[:200]) + "…"
	}
	fmt.Printf("   Caption preview: %s\n", preview)

	p.postsThisRun++
	return p.postsThisRun < p.maxPostsPerRun, nil
}

// postReal publishes a carousel to Instagram:
// 1. Upload images to GitHub Pages
// 2. Create carousel item containers (one per image)
// 3. Create carousel container with all children + caption
// 4. Publish the carousel container
// 5. Poll until PUBLISHED or error
// 6. Clean up hosted images
func (p *InstagramPlatform) postReal(igContent *InstagramContent) (bool, error) {
	imageCount := len(igContent.Images)

	// Generate unique filenames for this carousel
	names := make([]string, imageCount)
	ts := time.Now().UnixMilli()
	for i := range names {
		names[i] = fmt.Sprintf("carousel_%d_%d.jpg", ts, i)
	}

	// Step 1: Upload images to GitHub Pages
	fmt.Printf("📤 Uploading %d image(s) to GitHub Pages...\n", imageCount)
	imageURLs, err := p.uploadImagesFunc(igContent.Images, names)
	if err != nil {
		return false, fmt.Errorf("uploading images: %w", err)
	}
	fmt.Printf("   ✅ Images hosted at %d URL(s)\n", len(imageURLs))

	// Step 2: Wait for GitHub Pages deployment by polling image URLs
	fmt.Print("⏳ Waiting for GitHub Pages deployment...\n")
	if err := p.waitForImagesFunc(imageURLs); err != nil {
		logHostedImagesWarning(names)
		return false, fmt.Errorf("waiting for GitHub Pages: %w", err)
	}
	fmt.Print("   ✅ Images accessible\n")

	var publishID string
	if imageCount == 1 {
		// Single image post (carousels require 2+ images)
		fmt.Print("📦 Creating single image container...\n")
		containerID, err := p.createSingleImageContainerFunc(imageURLs[0], igContent.Caption)
		if err != nil {
			logHostedImagesWarning(names)
			return false, fmt.Errorf("creating single image container: %w", err)
		}
		fmt.Printf("   📦 Container: %s\n", containerID)
		publishID = containerID
	} else {
		// Carousel post (2+ images)
		fmt.Printf("📦 Creating %d media container(s)...\n", imageCount)
		childIDs := make([]string, imageCount)
		for i, imageURL := range imageURLs {
			containerID, err := p.createMediaContainerFunc(imageURL)
			if err != nil {
				logHostedImagesWarning(names)
				return false, fmt.Errorf("creating media container %d: %w", i, err)
			}
			childIDs[i] = containerID
			fmt.Printf("   📦 Container %d: %s\n", i, containerID)
		}

		fmt.Print("🎠 Creating carousel container...\n")
		carouselID, err := p.createCarouselContainerFunc(childIDs, igContent.Caption)
		if err != nil {
			logHostedImagesWarning(names)
			return false, fmt.Errorf("creating carousel container: %w", err)
		}
		fmt.Printf("   🎠 Carousel container: %s\n", carouselID)
		publishID = carouselID
	}

	// Wait for container to be ready before publishing
	fmt.Print("⏳ Waiting for container to be ready...\n")
	if err := p.pollUntilReady(publishID); err != nil {
		logHostedImagesWarning(names)
		return false, fmt.Errorf("waiting for container: %w", err)
	}

	// Publish
	fmt.Print("🚀 Publishing...\n")
	mediaID, err := p.publishContainerFunc(publishID)
	if err != nil {
		logHostedImagesWarning(names)
		return false, fmt.Errorf("publishing: %w", err)
	}
	fmt.Printf("   ✅ Published! Media ID: %s\n", mediaID)

	// Poll for PUBLISHED status
	if err := p.pollUntilPublished(publishID); err != nil {
		log.Printf("⚠️  Polling error (media may still be published): %v", err)
		// Don't fail — the post may have succeeded
	}

	// Step 7: Clean up hosted images
	fmt.Print("🧹 Cleaning up hosted images...\n")
	if err := p.cleanupImagesFunc(names); err != nil {
		// Log but don't fail — the post was successful
		log.Printf("⚠️  Cleanup error (images may remain hosted): %v", err)
	} else {
		fmt.Printf("   ✅ Cleaned up %d image(s)\n", imageCount)
	}

	p.postsThisRun++
	return p.postsThisRun < p.maxPostsPerRun, nil
}

// pollUntilPublished polls the container status until it reaches PUBLISHED, ERROR, or EXPIRED,
// or until the timeout is reached.
func (p *InstagramPlatform) pollUntilPublished(containerID string) error {
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		status, err := p.pollContainerStatusFunc(containerID)
		if err != nil {
			return fmt.Errorf("polling status: %w", err)
		}

		switch status {
		case igapi.StatusPublished:
			fmt.Printf("   ✅ Container status: %s\n", status)
			return nil
		case igapi.StatusFinished:
			fmt.Printf("   ✅ Container status: %s\n", status)
			return nil
		case igapi.StatusError:
			return fmt.Errorf("container status: ERROR")
		case igapi.StatusExpired:
			return fmt.Errorf("container status: EXPIRED")
		case igapi.StatusInProgress:
			fmt.Printf("   ⏳ Container status: %s, waiting...\n", status)
		default:
			fmt.Printf("   ⏳ Container status: %s, waiting...\n", status)
		}

		p.sleepFunc(pollInterval)
	}

	return fmt.Errorf("polling timed out after %v", pollTimeout)
}

// pollUntilReady polls the container status until it reaches FINISHED (ready to publish),
// or returns an error on ERROR/EXPIRED/timeout.
func (p *InstagramPlatform) pollUntilReady(containerID string) error {
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		status, err := p.pollContainerStatusFunc(containerID)
		if err != nil {
			return fmt.Errorf("polling status: %w", err)
		}

		switch status {
		case igapi.StatusFinished:
			fmt.Printf("   ✅ Container ready: %s\n", status)
			return nil
		case igapi.StatusPublished:
			fmt.Printf("   ✅ Container ready: %s\n", status)
			return nil
		case igapi.StatusError:
			return fmt.Errorf("container status: ERROR")
		case igapi.StatusExpired:
			return fmt.Errorf("container status: EXPIRED")
		default:
			fmt.Printf("   ⏳ Container status: %s, waiting...\n", status)
		}

		p.sleepFunc(pollInterval)
	}

	return fmt.Errorf("container not ready after %v", pollTimeout)
}

// logHostedImagesWarning logs a warning that images are left hosted for manual debugging.
func logHostedImagesWarning(names []string) {
	log.Printf("⚠️  Images left hosted for debugging: %v", names)
}

// MaxPostsPerRun returns the configured per-run posting limit.
func (p *InstagramPlatform) MaxPostsPerRun() int {
	return p.maxPostsPerRun
}

// Name returns the platform name.
func (p *InstagramPlatform) Name() string {
	return "Instagram"
}

// waitForImageURLs polls the given URLs until they all return HTTP 200, or the timeout is reached.
func waitForImageURLs(urls []string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	deadline := time.Now().Add(pagesDeploymentTimeout)

	for time.Now().Before(deadline) {
		allOK := true
		for _, u := range urls {
			resp, err := client.Head(u)
			if err != nil || resp.StatusCode != http.StatusOK {
				allOK = false
				break
			}
			_ = resp.Body.Close()
		}
		if allOK {
			return nil
		}
		time.Sleep(pagesDeploymentPollInterval)
	}

	return fmt.Errorf("images not accessible after %v", pagesDeploymentTimeout)
}
