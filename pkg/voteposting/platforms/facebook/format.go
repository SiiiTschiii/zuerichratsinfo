package facebook

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/imagegen"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// maxMessageChars is the Pages API limit on a post's message. Nothing this bot
// renders comes near it; it is checked so a runaway group fails in Format
// rather than as an opaque Graph API error after the other platforms posted.
const maxMessageChars = 63206

// maxPhotos caps the photos in one post at what an Instagram carousel holds, so
// both platforms show the same cards.
const maxPhotos = 10

// FacebookContent implements platforms.Content for Facebook: one Page post with
// the carousel's photos attached.
type FacebookContent struct {
	Images  [][]byte // JPEG-encoded carousel cards, in order
	Message string   // post text, including the link block
}

// String returns the text representation for logging/preview
func (c *FacebookContent) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📘 Facebook post: %d image(s)\n\n", len(c.Images)))
	sb.WriteString(c.Message)
	return sb.String()
}

// FormatPost renders a group as a single Page post: the Instagram carousel's
// cards as photos, and the Instagram caption's text as the message.
//
// Facebook has no practical length limit, so unlike Instagram the text is never
// truncated, and the links stay in it as plain URLs — an attached photo gives no
// preview card to carry them.
//
// Names are left untagged. A Page can only mention another Page or profile by
// its numeric ID, which the contacts file does not hold — it records profile
// URLs — so a tag here would publish as literal "@name" text.
func FormatPost(group []votes.Vote) (*FacebookContent, error) {
	if len(group) == 0 {
		return nil, fmt.Errorf("no votes provided")
	}

	images, err := imagegen.GenerateCarousel(group)
	if err != nil {
		return nil, fmt.Errorf("generating carousel images: %w", err)
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("no carousel images generated")
	}
	if len(images) > maxPhotos {
		images = images[:maxPhotos]
	}

	var message string
	if votes.IsWahlgeschaeftGroup(group) {
		// Same notice as on Instagram: the title, the sentence saying there is
		// no result to report, and the links. See voteformat.WahlgeschaeftBody.
		header := fmt.Sprintf("🗳️ %s\n\n", voteformat.PostHeadline(group))
		message = header + voteformat.WahlgeschaeftBody(group) + voteformat.LinkLine(group)
	} else {
		body := strings.TrimRight(voteformat.LongFormBody(group, voteformat.Tags{}), "\n")
		message = body + "\n\n" + strings.TrimLeft(voteformat.LinkLine(group), "\n")
	}
	message = strings.TrimRight(message, "\n")

	if n := len([]rune(message)); n > maxMessageChars {
		return nil, fmt.Errorf("facebook message is %d characters, over the %d limit", n, maxMessageChars)
	}

	return &FacebookContent{Images: images, Message: message}, nil
}
