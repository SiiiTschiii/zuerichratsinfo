package bluesky

import (
	"fmt"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// BlueskyContent implements platforms.Content for Bluesky
type BlueskyContent struct {
	post *BlueskyPost
}

// String returns the text representation for logging/preview
func (c *BlueskyContent) String() string {
	return c.post.Text
}

// BlueskyPlatform implements the platforms.Platform interface for Bluesky
type BlueskyPlatform struct {
	handle         string
	password       string
	session        *bskyapi.Session
	postsThisRun   int
	maxPostsPerRun int
}

// NewBlueskyPlatform creates a new Bluesky platform poster
func NewBlueskyPlatform(handle, password string, maxPostsPerRun int) *BlueskyPlatform {
	return &BlueskyPlatform{
		handle:         handle,
		password:       password,
		maxPostsPerRun: maxPostsPerRun,
		postsThisRun:   0,
	}
}

// ensureSession creates or reuses a Bluesky session
func (p *BlueskyPlatform) ensureSession() error {
	if p.session != nil {
		return nil
	}

	session, err := bskyapi.CreateSession(p.handle, p.password)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Bluesky: %w", err)
	}

	p.session = session
	fmt.Printf("🔑 Authenticated as %s (DID: %s)\n", session.Handle, session.DID)
	return nil
}

// Format formats a group of votes into Bluesky-specific content
func (p *BlueskyPlatform) Format(votes []zurichapi.Abstimmung) (platforms.Content, error) {
	post := FormatVoteGroupPost(votes)
	return &BlueskyContent{post: post}, nil
}

// Post posts content to Bluesky
// Returns shouldContinue=false when the post limit is reached
func (p *BlueskyPlatform) Post(content platforms.Content) (bool, error) {
	bskyContent, ok := content.(*BlueskyContent)
	if !ok {
		return false, fmt.Errorf("unexpected content type for Bluesky")
	}

	// Authenticate lazily on first post
	if err := p.ensureSession(); err != nil {
		return false, err
	}

	err := bskyapi.CreateRecord(p.session, bskyContent.post.Text, bskyContent.post.Facets)
	if err != nil {
		return false, err
	}

	p.postsThisRun++
	shouldContinue := p.postsThisRun < p.maxPostsPerRun

	return shouldContinue, nil
}

// Name returns the platform name
func (p *BlueskyPlatform) Name() string {
	return "Bluesky"
}
