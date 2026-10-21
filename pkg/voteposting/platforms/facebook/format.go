package facebook

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// maxMessageChars is the Pages API limit on a post's message. Nothing this bot
// renders comes near it; it is checked so a runaway group fails in Format
// rather than as an opaque Graph API error after the other platforms posted.
const maxMessageChars = 63206

// FacebookContent implements platforms.Content for Facebook: one Page post.
type FacebookContent struct {
	Message string // post text, including the link block
	Link    string // URL Facebook renders as the preview card; may be empty
}

// String returns the text representation for logging/preview
func (c *FacebookContent) String() string {
	var sb strings.Builder
	sb.WriteString(c.Message)
	if c.Link != "" {
		sb.WriteString(fmt.Sprintf("\n  🃏 Preview card: %s", c.Link))
	}
	return sb.String()
}

// FormatPost renders a group as a single Page post.
//
// The message is the same long-form body as the Instagram caption, followed by
// the link block. Facebook has no practical length limit, so unlike Instagram
// nothing is truncated, and unlike X and Bluesky nothing is threaded.
//
// Names are left untagged. A Page can only mention another Page or profile by
// its numeric ID, which the contacts file does not hold — it records profile
// URLs — so a tag here would publish as literal "@name" text.
func FormatPost(group []votes.Vote) (*FacebookContent, error) {
	if len(group) == 0 {
		return nil, fmt.Errorf("no votes provided")
	}

	var message string
	if votes.IsWahlgeschaeftGroup(group) {
		// Same notice as on Instagram: the title, the sentence saying there is
		// no result to report, and the links. See voteformat.WahlgeschaeftBody.
		header := fmt.Sprintf("🗳️ %s\n\n", voteformat.PostHeadline(group))
		message = header + voteformat.WahlgeschaeftBody(group) + voteformat.LinkLine(group)
	} else {
		body := strings.TrimRight(voteformat.LongFormBody(group, nil), "\n")
		message = body + "\n\n" + strings.TrimLeft(voteformat.LinkLine(group), "\n")
	}
	message = strings.TrimRight(message, "\n")

	if n := len([]rune(message)); n > maxMessageChars {
		return nil, fmt.Errorf("facebook message is %d characters, over the %d limit", n, maxMessageChars)
	}

	return &FacebookContent{Message: message, Link: previewLink(group)}, nil
}

// previewLink picks the URL Facebook turns into the preview card: the first
// link the post lists, which is the business page wherever there is one. The
// business page is the one with a title and a description worth a card; the
// video archive and the sitting page are generic for every vote of a day.
func previewLink(group []votes.Vote) string {
	links := voteformat.GroupLinks(group)
	if len(links) == 0 {
		return ""
	}
	return links[0].URL
}
