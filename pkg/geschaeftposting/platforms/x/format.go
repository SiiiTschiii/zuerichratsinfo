package x

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/geschaeftposting/geschaeftformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// DefaultMaxChars is the X post character limit for free accounts.
// X shortens all URLs to 23 chars (t.co). Premium accounts support up to 2000 chars.
const DefaultMaxChars = 280

// URLLength is the number of characters X counts for any URL (t.co shortening).
const URLLength = 23

// XGeschaeftPost holds the formatted text for a single post in an X thread
type XGeschaeftPost struct {
	Text string
}

// FormatGeschaeftThread creates an X thread for a newly submitted Geschaeft.
// Returns [0] = root post (news hook), [1] = reply (civic education + what's next).
// charLimit sets the per-post character limit (e.g. 280 for free, 2000 for Premium).
func FormatGeschaeftThread(g zurichapi.Geschaeft, contactMapper *contacts.Mapper, charLimit int) []*XGeschaeftPost {
	typeLabel := geschaeftformat.GetTypeLabel(g.Geschaeftsart, g.Dringlich)
	emoji := geschaeftformat.GetEmoji(g.Geschaeftsart, g.Dringlich)
	date := geschaeftformat.FormatDate(g.Beginn.Start)
	title := geschaeftformat.CleanTitle(g.Titel)
	submitterName := g.Erstunterzeichner.KontaktGremium.Name
	submitterPartei := g.Erstunterzeichner.KontaktGremium.Partei
	submitterLine := geschaeftformat.FormatSubmitterLine(submitterName, submitterPartei, g.AnzahlMitunterzeichnende)
	link := voteformat.GenerateGeschaeftLink(g.OBJGUID)

	// Tag X handles in the submitter name/party if contact mapper is available
	if contactMapper != nil {
		submitterLine = contactMapper.TagXHandlesInText(submitterLine)
	}

	root := buildRootPost(emoji, typeLabel, title, submitterLine, date, link, g.Geschaeftsart, charLimit)
	reply := buildReplyPost(g.Geschaeftsart, g.Ablaufschritte, charLimit)

	return []*XGeschaeftPost{root, reply}
}

// buildRootPost creates the root (news hook) post.
// URLs count as URLLength characters on X regardless of actual length.
func buildRootPost(emoji, typeLabel, title, submitterLine, date, link, geschaeftsart string, charLimit int) *XGeschaeftPost {
	article := geschaeftformat.GetArticle(geschaeftsart)
	header := fmt.Sprintf("%s Neue %s | Gemeinderat Zürich\n\n", emoji, typeLabel)
	footer := fmt.Sprintf("\n\n%s\n📅 %s\n\n👇 Was ist %s %s?\n\n%s",
		submitterLine, date, article, geschaeftsart, link)

	// On X, the URL in footer counts as URLLength (not its real length).
	// Use rune counts because X counts Unicode code points, not bytes.
	footerCharCount := len([]rune(footer)) - len([]rune(link)) + URLLength
	overhead := len([]rune(header)) + footerCharCount
	available := charLimit - overhead
	if available < 1 {
		available = 1
	}

	if len([]rune(title)) > available {
		title = truncateText(title, available)
	}

	return &XGeschaeftPost{Text: header + title + footer}
}

// buildReplyPost creates the civic education reply post.
func buildReplyPost(geschaeftsart string, ablaufschritte []zurichapi.Aufgabe, charLimit int) *XGeschaeftPost {
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

	// X counts URL as URLLength chars; check effective length using rune counts.
	effectiveLen := len([]rune(text)) - len([]rune(link)) + URLLength
	if effectiveLen > charLimit {
		text = trimReplyToLimit(explanation, whatsnext, link, charLimit)
	}

	return &XGeschaeftPost{Text: text}
}

// trimReplyToLimit builds the reply text trimming whatsnext if needed.
func trimReplyToLimit(explanation, whatsnext, link string, charLimit int) string {
	linkPart := fmt.Sprintf("\n\n%s", link)
	whatsnextHeader := "⏭️ Was passiert als nächstes?\n"
	explanationPart := fmt.Sprintf("ℹ️ %s", explanation)

	// Use rune counts for accurate X character counting (X counts Unicode code points).
	// The link contributes URLLength chars (t.co shortening), not its real rune count.
	overhead := len([]rune(explanationPart)) + len([]rune("\n\n")) +
		len([]rune(whatsnextHeader)) + URLLength
	available := charLimit - overhead
	if available > 0 && len([]rune(whatsnext)) > available {
		// truncateText ensures the result (including "…") fits in available runes.
		whatsnext = truncateText(whatsnext, available)
	}

	return explanationPart + "\n\n" + whatsnextHeader + whatsnext + linkPart
}

// truncateText truncates a string so the result (including the appended "…") fits
// within maxLen runes. If the string already fits, it is returned unchanged.
func truncateText(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	// Reserve 1 slot for "…"
	end := maxLen - 1
	if end < 0 {
		end = 0
	}
	truncated := strings.TrimRight(string(runes[:end]), " \n")
	return truncated + "…"
}
