package facebook

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
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
	Images   [][]byte // JPEG-encoded carousel cards, in order
	Captions []string // each card's own text, same length as Images; "" for none
	Message  string   // post text: the heading and the link block
}

// String returns the text representation for logging/preview
func (c *FacebookContent) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📘 Facebook post: %d image(s)\n\n", len(c.Images)))
	sb.WriteString(c.Message)
	for i, caption := range c.Captions {
		if caption != "" {
			sb.WriteString(fmt.Sprintf("\n\n  🖼️ Caption of image %d:\n%s", i+1, caption))
		}
	}
	return sb.String()
}

// FormatPost renders a group as a single Page post: the Instagram carousel's
// cards as photos, with the text split between the post and the photos.
//
// A multi-vote group's post text is the heading and the links, and each vote's
// counts and Fraktion breakdown ride as the caption of its own card. The
// counts are what makes the post long, and a reader looking at a card wants
// that card's counts, not five votes' worth under the whole grid. A single vote
// has only one card and nothing to split, so its post keeps all of its text.
//
// Facebook has no practical length limit, so unlike Instagram nothing is
// truncated, and the links stay in the text as plain URLs — an attached photo
// gives no preview card to carry them.
//
// Nobody is tagged; see FormatPostWithContacts.
func FormatPost(group []votes.Vote) (*FacebookContent, error) {
	return FormatPostWithContacts(group, nil)
}

// tagsFor builds the tagging a Facebook post applies, from the contacts.
//
// A mention is written with a numeric ID, so only a contact whose Facebook URL
// carries one is tagged; see contacts.ExtractFacebookMentionFromURL. Members
// are mentioned in the title and the label line, and the parties of a Fraktion
// phrase wherever one appears, as on Instagram.
func tagsFor(mapper *contacts.Mapper) voteformat.Tags {
	if mapper == nil {
		return voteformat.Tags{}
	}
	return voteformat.Tags{
		Heading: func(text, jurisdiction string) string {
			return mapper.TagFacebookFraktionen(mapper.TagFacebookHandlesInText(text), jurisdiction)
		},
		SubVote: mapper.TagFacebookFraktionen,
	}
}

// FormatPostWithContacts is FormatPost with the contacts' Facebook accounts
// mentioned where the text names them. Only the post text and the captions are
// tagged; the images carry no mentions.
func FormatPostWithContacts(group []votes.Vote, mapper *contacts.Mapper) (*FacebookContent, error) {
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

	tags := tagsFor(mapper)
	captions := make([]string, len(images))
	var message string
	links := strings.TrimLeft(voteformat.LinkLine(group), "\n")

	switch {
	case votes.IsWahlgeschaeftGroup(group):
		// Same notice as on Instagram: the title, the sentence saying there is
		// no result to report, and the links. See voteformat.WahlgeschaeftBody.
		header := fmt.Sprintf("🗳️ %s\n\n", voteformat.PostHeadline(group))
		message = header + voteformat.WahlgeschaeftBody(group) + voteformat.LinkLine(group)

	case len(group) == 1:
		body := strings.TrimRight(voteformat.LongFormBody(group, tags), "\n")
		message = body + "\n\n" + links

	default:
		// The title card comes first, then one card per vote, in order. Votes
		// the cap left without a card have nowhere to caption, so their text
		// goes in the post rather than being lost.
		var uncovered strings.Builder
		for i := range group {
			block := voteBlock(group, i, tags)
			if card := i + 1; card < len(images) {
				captions[card] = block
				continue
			}
			uncovered.WriteString("\n\n" + block)
		}
		heading := strings.TrimRight(voteformat.LongFormHeading(group, tags), "\n")
		message = heading + uncovered.String() + "\n\n" + links
	}
	message = strings.TrimRight(message, "\n")

	if n := len([]rune(message)); n > maxMessageChars {
		return nil, fmt.Errorf("facebook message is %d characters, over the %d limit", n, maxMessageChars)
	}

	return &FacebookContent{Images: images, Captions: captions, Message: message}, nil
}

// voteBlock is one vote's text for a caption: its counts and Fraktion table,
// then the links that are this vote's own — the recording at its segment — so a
// reader looking at a card can open exactly that vote.
func voteBlock(group []votes.Vote, i int, tags voteformat.Tags) string {
	block := strings.TrimRight(voteformat.LongFormVote(group, i, tags), "\n")
	links := voteformat.VoteLinks(group, i)
	if len(links) == 0 {
		return block
	}
	lines := make([]string, len(links))
	for j, l := range links {
		lines[j] = l.Line(true)
	}
	return block + "\n\n" + strings.Join(lines, "\n")
}
