package x

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/xapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// XContent represents formatted content for X/Twitter
type XContent struct {
	thread []*XPost // [0] = root, [1:] = replies
}

// String returns the text representation for logging/preview
func (c *XContent) String() string {
	var sb strings.Builder
	for i, post := range c.thread {
		if i == 0 {
			sb.WriteString(post.Text)
		} else {
			sb.WriteString(fmt.Sprintf("\n  ↳ Reply %d:\n%s", i, post.Text))
		}
	}
	return sb.String()
}

// XPlatform implements the Platform interface for X/Twitter
type XPlatform struct {
	apiKey         string
	apiSecret      string
	accessToken    string
	accessSecret   string
	contactMapper  *contacts.Mapper
	postsThisRun   int
	maxPostsPerRun int
}

// NewXPlatform creates a new X platform poster
func NewXPlatform(
	apiKey, apiSecret, accessToken, accessSecret string,
	contactMapper *contacts.Mapper,
	maxPostsPerRun int,
) *XPlatform {
	return &XPlatform{
		apiKey:         apiKey,
		apiSecret:      apiSecret,
		accessToken:    accessToken,
		accessSecret:   accessSecret,
		contactMapper:  contactMapper,
		maxPostsPerRun: maxPostsPerRun,
		postsThisRun:   0,
	}
}

// Format formats a group of votes into X-specific content
func (p *XPlatform) Format(votes []zurichapi.Abstimmung) (platforms.Content, error) {
	thread := FormatVoteThread(votes, p.contactMapper)
	return &XContent{thread: thread}, nil
}

// Post posts a thread to X/Twitter (root post + reply chain).
// Returns shouldContinue=false when the post limit is reached.
func (p *XPlatform) Post(content platforms.Content) (bool, error) {
	xContent, ok := content.(*XContent)
	if !ok {
		return false, fmt.Errorf("unexpected content type for X")
	}

	if len(xContent.thread) == 0 {
		return false, fmt.Errorf("empty thread")
	}

	// Post the root
	root := xContent.thread[0]
	rootTweetID, err := xapi.PostTweet(
		p.apiKey, p.apiSecret, p.accessToken, p.accessSecret,
		root.Text, "",
	)
	if err != nil {
		return false, fmt.Errorf("failed to post root: %w", err)
	}

	// Post replies as a chain
	parentTweetID := rootTweetID
	for i, reply := range xContent.thread[1:] {
		tweetID, err := xapi.PostTweet(
			p.apiKey, p.apiSecret, p.accessToken, p.accessSecret,
			reply.Text, parentTweetID,
		)
		if err != nil {
			return false, fmt.Errorf("failed to post reply %d: %w", i+1, err)
		}
		parentTweetID = tweetID
	}

	p.postsThisRun++
	shouldContinue := p.postsThisRun < p.maxPostsPerRun

	return shouldContinue, nil
}

// MaxPostsPerRun returns the configured per-run posting limit.
func (p *XPlatform) MaxPostsPerRun() int {
	return p.maxPostsPerRun
}

// Name returns the platform name
func (p *XPlatform) Name() string {
	return "X"
}
