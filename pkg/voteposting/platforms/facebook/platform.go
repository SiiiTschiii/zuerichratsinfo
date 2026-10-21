package facebook

import (
	"fmt"

	"github.com/siiitschiii/zuerichratsinfo/pkg/fbapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// PublishPostFunc is the signature for publishing a Page post. Defaults to
// fbapi.Client.PublishPost.
type PublishPostFunc func(message, link string) (string, error)

// FacebookPlatform implements the platforms.Platform interface for a Facebook Page
type FacebookPlatform struct {
	postsThisRun    int
	maxPostsPerRun  int
	stubMode        bool            // true when no credentials are configured
	publishPostFunc PublishPostFunc // injectable for testing
}

// NewFacebookPlatform creates a Facebook platform in stub mode: Post logs what
// it would publish and calls no API. Used by the preview and dry-run tools.
func NewFacebookPlatform(maxPostsPerRun int) *FacebookPlatform {
	return &FacebookPlatform{
		maxPostsPerRun: maxPostsPerRun,
		stubMode:       true,
	}
}

// NewFacebookPlatformWithCredentials creates a Facebook platform that publishes
// to the given Page with a Page access token.
func NewFacebookPlatformWithCredentials(pageID, accessToken string, maxPostsPerRun int) *FacebookPlatform {
	client := fbapi.NewClient(pageID, accessToken)
	return &FacebookPlatform{
		maxPostsPerRun:  maxPostsPerRun,
		publishPostFunc: client.PublishPost,
	}
}

// Format formats a group of votes into a single Page post
func (p *FacebookPlatform) Format(group []votes.Vote) (platforms.Content, error) {
	return FormatPost(group)
}

// Post publishes content to the Page.
// Returns shouldContinue=false when the post limit is reached.
func (p *FacebookPlatform) Post(content platforms.Content) (bool, error) {
	fbContent, ok := content.(*FacebookContent)
	if !ok {
		return false, fmt.Errorf("unexpected content type for Facebook")
	}

	if p.stubMode {
		fmt.Printf("📘 [Facebook stub] Would post %d chars with preview card %q\n",
			len([]rune(fbContent.Message)), fbContent.Link)
	} else {
		postID, err := p.publishPostFunc(fbContent.Message, fbContent.Link)
		if err != nil {
			return false, fmt.Errorf("publishing Facebook post: %w", err)
		}
		fmt.Printf("✅ Facebook post created (id: %s)\n", postID)
		fmt.Printf("   🔗 %s\n", fbapi.PostURL(postID))
	}

	p.postsThisRun++
	return p.postsThisRun < p.maxPostsPerRun, nil
}

// MaxPostsPerRun returns the configured per-run posting limit.
func (p *FacebookPlatform) MaxPostsPerRun() int {
	return p.maxPostsPerRun
}

// Name returns the platform name
func (p *FacebookPlatform) Name() string {
	return "Facebook"
}
