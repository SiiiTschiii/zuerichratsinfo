package linkedin

import (
	"fmt"

	"github.com/siiitschiii/zuerichratsinfo/pkg/linkedinapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// LinkedInContent implements platforms.Content for LinkedIn.
type LinkedInContent struct {
	post *LinkedInPost
}

// String returns the text representation for logging/preview.
func (c *LinkedInContent) String() string {
	if c.post.LinkURL == "" {
		return c.post.Text
	}
	return c.post.Text + "\n  🖼️ link preview: " + c.post.LinkURL
}

// SharePostFunc is the signature for publishing a share. Defaults to
// (*linkedinapi.Client).SharePost.
type SharePostFunc func(text, articleURL string) (string, error)

// LinkedInPlatform implements platforms.Platform for LinkedIn.
type LinkedInPlatform struct {
	postsThisRun   int
	maxPostsPerRun int
	sharePostFunc  SharePostFunc // injectable for testing
}

// NewLinkedInPlatform creates a poster for the member who owns accessToken.
// authorURN is "urn:li:person:<id>".
func NewLinkedInPlatform(accessToken, authorURN string, maxPostsPerRun int) *LinkedInPlatform {
	return &LinkedInPlatform{
		maxPostsPerRun: maxPostsPerRun,
		sharePostFunc:  linkedinapi.New(accessToken, authorURN).SharePost,
	}
}

// Format formats a group of votes into one LinkedIn post.
func (p *LinkedInPlatform) Format(group []votes.Vote) (platforms.Content, error) {
	post := FormatVotePost(group)
	if post == nil {
		return nil, fmt.Errorf("empty vote group")
	}
	return &LinkedInContent{post: post}, nil
}

// Post publishes the post. Returns shouldContinue=false when the limit is reached.
func (p *LinkedInPlatform) Post(content platforms.Content) (bool, error) {
	c, ok := content.(*LinkedInContent)
	if !ok {
		return false, fmt.Errorf("unexpected content type for LinkedIn")
	}

	urn, err := p.sharePostFunc(c.post.Text, c.post.LinkURL)
	if err != nil {
		return false, fmt.Errorf("failed to post: %w", err)
	}
	fmt.Printf("   🔗 %s\n", linkedinapi.PostURL(urn))

	p.postsThisRun++
	return p.postsThisRun < p.maxPostsPerRun, nil
}

// MaxPostsPerRun returns the configured per-run posting limit.
func (p *LinkedInPlatform) MaxPostsPerRun() int {
	return p.maxPostsPerRun
}

// Name returns the platform name.
func (p *LinkedInPlatform) Name() string {
	return "LinkedIn"
}
