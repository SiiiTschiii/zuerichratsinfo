package bluesky

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/geschaeftposting/geschaeftformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// maxGraphemes is the Bluesky post character limit (graphemes)
const maxGraphemes = 300

// GeschaeftPost holds the formatted text and rich text facets for a Bluesky post
type GeschaeftPost struct {
	Text     string
	Facets   []bskyapi.Facet
	Mentions []contacts.BlueskyMention
}

// FormatGeschaeftThread creates a Bluesky thread for a newly submitted Geschaeft.
// Returns a slice of posts: [0] is the root post (news hook), [1] is the reply
// (civic education + what's next).
func FormatGeschaeftThread(g zurichapi.Geschaeft, contactMapper *contacts.Mapper) []*GeschaeftPost {
	typeLabel := geschaeftformat.GetTypeLabel(g.Geschaeftsart, g.Dringlich)
	emoji := geschaeftformat.GetEmoji(g.Geschaeftsart, g.Dringlich)
	date := geschaeftformat.FormatDate(g.Beginn.Start)
	title := geschaeftformat.CleanTitle(g.Titel)
	submitterName := g.Erstunterzeichner.KontaktGremium.Name
	submitterPartei := g.Erstunterzeichner.KontaktGremium.Partei
	submitterLine := geschaeftformat.FormatSubmitterLine(submitterName, submitterPartei, g.AnzahlMitunterzeichnende)
	link := voteformat.GenerateGeschaeftLink(g.OBJGUID)

	root := buildRootPost(emoji, typeLabel, title, submitterLine, date, link, g.Geschaeftsart)
	reply := buildReplyPost(g.Geschaeftsart, g.Dringlich, g.Ablaufschritte)

	thread := []*GeschaeftPost{root, reply}

	// Scan all posts for Bluesky mentions
	if contactMapper != nil {
		for _, post := range thread {
			post.Mentions = contactMapper.FindBlueskyMentions(post.Text)
		}
	}

	return thread
}

// buildRootPost creates the root (news hook) post.
// Format:
//
//	{emoji} Neue {typeLabel} | Gemeinderat Zürich
//
//	{title}
//
//	{submitterLine}
//	📅 {date}
//
//	👇 Was ist {article} {geschaeftsart}?
//
//	{link}
func buildRootPost(emoji, typeLabel, title, submitterLine, date, link, geschaeftsart string) *GeschaeftPost {
	article := geschaeftformat.GetArticle(geschaeftsart)
	header := fmt.Sprintf("%s Neue %s | Gemeinderat Zürich\n\n", emoji, typeLabel)
	footer := fmt.Sprintf("\n\n%s\n📅 %s\n\n👇 Was ist %s %s?\n\n%s",
		submitterLine, date, article, geschaeftsart, link)

	overhead := graphemeLen(header) + graphemeLen(footer)
	available := maxGraphemes - overhead
	if available < 1 {
		available = 1
	}

	if graphemeLen(title) > available {
		title = truncateText(title, available)
	}

	fullText := header + title + footer

	post := &GeschaeftPost{Text: fullText}
	// Add link facet for the Geschaeft URL
	post.Facets = buildLinkFacets(fullText, link)
	return post
}

// buildReplyPost creates the civic education reply post.
// Format:
//
//	ℹ️ {explanation}
//
//	⏭️ Was passiert als nächstes?
//	{whatsnext}
//
//	{explanationLink}
func buildReplyPost(geschaeftsart string, dringlich bool, ablaufschritte []zurichapi.Aufgabe) *GeschaeftPost {
	// For Schriftliche Anfrage, extract deadline from first Ablaufschritt
	var fristBis string
	if geschaeftsart == "Schriftliche Anfrage" && len(ablaufschritte) > 0 {
		fristBis = ablaufschritte[0].FristBis.Start
	}

	explanation := geschaeftformat.GetExplanation(geschaeftsart)
	whatsnext := geschaeftformat.GetWhatsnext(geschaeftsart, fristBis)
	link := geschaeftformat.ExplanationLink

	var sb strings.Builder
	if explanation != "" {
		sb.WriteString(fmt.Sprintf("ℹ️ %s", explanation))
	}
	if whatsnext != "" {
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("⏭️ Was passiert als nächstes?\n%s", whatsnext))
	}
	sb.WriteString(fmt.Sprintf("\n\n%s", link))

	text := sb.String()

	// If the reply exceeds the limit, try to fit by trimming whatsnext
	if graphemeLen(text) > maxGraphemes {
		text = trimReplyToLimit(explanation, whatsnext, link)
	}

	post := &GeschaeftPost{Text: text}
	post.Facets = buildLinkFacets(text, link)
	return post
}

// trimReplyToLimit builds the reply text and trims whatsnext if needed to fit within maxGraphemes.
func trimReplyToLimit(explanation, whatsnext, link string) string {
	linkPart := fmt.Sprintf("\n\n%s", link)
	whatsnextHeader := "⏭️ Was passiert als nächstes?\n"

	explanationPart := fmt.Sprintf("ℹ️ %s", explanation)
	overhead := graphemeLen(explanationPart) + graphemeLen("\n\n") +
		graphemeLen(whatsnextHeader) + graphemeLen(linkPart) + 1 // +1 for "…"
	available := maxGraphemes - overhead
	if available > 0 && graphemeLen(whatsnext) > available {
		whatsnext = truncateText(whatsnext, available)
	}

	return explanationPart + "\n\n" + whatsnextHeader + whatsnext + linkPart
}

// buildLinkFacets finds the URL in the text and creates a link facet for it.
func buildLinkFacets(text, url string) []bskyapi.Facet {
	idx := strings.Index(text, url)
	if idx < 0 {
		return nil
	}
	byteStart := len(text[:idx])
	byteEnd := byteStart + len(url)
	return []bskyapi.Facet{
		bskyapi.LinkFacet(byteStart, byteEnd, url),
	}
}

// truncateText truncates a string so the result (including the appended "…") fits
// within maxRunes graphemes. If the string already fits, it is returned unchanged.
func truncateText(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	// Reserve 1 slot for "…"
	end := maxRunes - 1
	if end < 0 {
		end = 0
	}
	truncated := strings.TrimRight(string(runes[:end]), " \n")
	return truncated + "…"
}

// graphemeLen returns the number of graphemes (runes) in a string.
func graphemeLen(s string) int {
	return len([]rune(s))
}
