package x

import (
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/xapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// XContent represents formatted content for X/Twitter
type XContent struct {
	message string
}

// String returns the text content
func (c *XContent) String() string {
	return c.message
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
	message := FormatVoteGroupPost(votes, p.contactMapper)
	return &XContent{message: message}, nil
}

// Post posts content to X/Twitter
// Returns shouldContinue=false when the post limit is reached
func (p *XPlatform) Post(content platforms.Content) (bool, error) {
	xContent, ok := content.(*XContent)
	if !ok {
		return false, nil
	}

	err := xapi.PostTweet(
p.apiKey,
p.apiSecret,
p.accessToken,
p.accessSecret,
xContent.message,
)

	if err != nil {
		return false, err
	}

	p.postsThisRun++
	shouldContinue := p.postsThisRun < p.maxPostsPerRun

	return shouldContinue, nil
}

// Name returns the platform name
func (p *XPlatform) Name() string {
	return "X"
}
