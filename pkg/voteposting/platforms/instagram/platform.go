package instagram

import (
	"fmt"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// InstagramPlatform implements the platforms.Platform interface for Instagram
type InstagramPlatform struct {
	postsThisRun   int
	maxPostsPerRun int
}

// NewInstagramPlatform creates a new Instagram platform poster.
func NewInstagramPlatform(maxPostsPerRun int) *InstagramPlatform {
	return &InstagramPlatform{
		maxPostsPerRun: maxPostsPerRun,
	}
}

// Format formats a group of votes into Instagram-specific content (carousel images + caption).
func (p *InstagramPlatform) Format(votes []zurichapi.Abstimmung) (platforms.Content, error) {
	return FormatCarousel(votes)
}

// Post is a stub that logs what would be posted. Real posting comes in Phase 3.
// Returns shouldContinue=false when the post limit is reached.
func (p *InstagramPlatform) Post(content platforms.Content) (bool, error) {
	igContent, ok := content.(*InstagramContent)
	if !ok {
		return false, fmt.Errorf("unexpected content type for Instagram")
	}

	// Stub: preview what would be posted
	fmt.Printf("📷 [Instagram stub] Would post %d image(s) with caption (%d chars):\n",
		len(igContent.Images), len([]rune(igContent.Caption)))

	// Show a truncated caption preview
	preview := igContent.Caption
	if len([]rune(preview)) > 200 {
		preview = string([]rune(preview)[:200]) + "…"
	}
	fmt.Printf("   Caption preview: %s\n", preview)

	p.postsThisRun++
	shouldContinue := p.postsThisRun < p.maxPostsPerRun

	return shouldContinue, nil
}

// MaxPostsPerRun returns the configured per-run posting limit.
func (p *InstagramPlatform) MaxPostsPerRun() int {
	return p.maxPostsPerRun
}

// Name returns the platform name.
func (p *InstagramPlatform) Name() string {
	return "Instagram"
}
