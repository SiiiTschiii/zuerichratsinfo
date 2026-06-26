package geschaeftformat

import (
	"fmt"
	"strings"
)

const (
	// ExplanationLink is the URL for the Geschaeftsarten explanation page
	ExplanationLink = "https://www.gemeinderat-zuerich.ch/gemeinderat/geschaeftsarten/"
)

// typeInfo holds display metadata for a Geschaeftsart
type typeInfo struct {
	emoji       string
	article     string // German article for "Was ist eine/ein …?"
	explanation string // ℹ️ civic education text
	whatsnext   string // ⏭️ "what happens next" text
}

// geschaeftsartInfo maps Geschaeftsart values to their display metadata.
var geschaeftsartInfo = map[string]typeInfo{
	"Motion": {
		emoji:   "📋",
		article: "eine",
		explanation: "Eine Motion verpflichtet den Stadtrat, dem Gemeinderat " +
			"einen Gesetzesentwurf auszuarbeiten — wenn der Rat sie überweist.",
		whatsnext: "Der Rat stimmt demnächst über die Überweisung ab. " +
			"Bei Annahme hat der Stadtrat 2 Jahre Zeit, einen Entwurf vorzulegen.",
	},
	"Postulat": {
		emoji:   "📝",
		article: "ein",
		explanation: "Ein Postulat beauftragt den Stadtrat, die Machbarkeit einer " +
			"Massnahme zu prüfen oder einen Bericht dazu zu verfassen.",
		whatsnext: "Der Rat stimmt demnächst über die Überweisung ab. " +
			"Bei Annahme hat der Stadtrat 2 Jahre Zeit, den Bericht vorzulegen.",
	},
	"Interpellation": {
		emoji:   "💬",
		article: "eine",
		explanation: "Eine Interpellation richtet Fragen an den Stadtrat, " +
			"der sie schriftlich beantworten muss.",
		whatsnext: "Der Stadtrat beantwortet die Interpellation schriftlich. " +
			"Der Rat kann die Antwort in einer Sitzung diskutieren.",
	},
	"Schriftliche Anfrage": {
		emoji:   "❓",
		article: "eine",
		explanation: "Eine Schriftliche Anfrage richtet eine oder mehrere Fragen " +
			"an den Stadtrat, der sie schriftlich beantwortet.",
		whatsnext: "Der Stadtrat beantwortet die Anfrage schriftlich. " +
			"Keine Diskussion im Rat.",
	},
	"Weisung": {
		emoji:   "📄",
		article: "eine",
		explanation: "Eine Weisung ist ein Antrag des Stadtrats an den Gemeinderat, " +
			"beispielsweise für einen Kredit oder eine Rechtsänderung.",
		whatsnext: "Der Rat behandelt die Weisung in einer der nächsten Sitzungen.",
	},
	"Parlamentarische Initiative": {
		emoji:   "⚖️",
		article: "eine",
		explanation: "Eine Parlamentarische Initiative ermöglicht Ratsmitgliedern, " +
			"direkt einen Gesetzesentwurf einzubringen.",
		whatsnext: "Der Rat entscheidet, ob er auf die Initiative eintreten will.",
	},
}

// IsPostable returns true for Geschaeftsarten that should be posted in v1 scope.
// Motion and Postulat are supported; dringliche variants use the same check.
func IsPostable(geschaeftsart string) bool {
	switch geschaeftsart {
	case "Motion", "Postulat":
		return true
	default:
		return false
	}
}

// GetEmoji returns the emoji for a Geschaeftsart.
// If dringlich is true, the emergency emoji 🚨 is returned instead.
func GetEmoji(geschaeftsart string, dringlich bool) string {
	if dringlich {
		return "🚨"
	}
	if info, ok := geschaeftsartInfo[geschaeftsart]; ok {
		return info.emoji
	}
	return "📋"
}

// GetArticle returns the German article ("ein" or "eine") for the type label in
// "Was ist eine Motion?" — used in the thread hint.
func GetArticle(geschaeftsart string) string {
	if info, ok := geschaeftsartInfo[geschaeftsart]; ok {
		return info.article
	}
	return "ein"
}

// GetExplanation returns the ℹ️ civic education text for a Geschaeftsart.
func GetExplanation(geschaeftsart string) string {
	if info, ok := geschaeftsartInfo[geschaeftsart]; ok {
		return info.explanation
	}
	return ""
}

// GetWhatsnext returns the ⏭️ "what happens next" text for a Geschaeftsart.
// For Schriftliche Anfrage the fristBis date (YYYY-MM-DD or datetime string) is
// substituted into the text when provided.
func GetWhatsnext(geschaeftsart, fristBis string) string {
	if geschaeftsart == "Schriftliche Anfrage" && fristBis != "" {
		date := FormatDate(fristBis)
		return fmt.Sprintf("Der Stadtrat beantwortet die Anfrage schriftlich bis %s. "+
			"Keine Diskussion im Rat.", date)
	}
	if info, ok := geschaeftsartInfo[geschaeftsart]; ok {
		return info.whatsnext
	}
	return ""
}

// GetTypeLabel returns the display label for a Geschaeftsart, taking dringlich into account.
func GetTypeLabel(geschaeftsart string, dringlich bool) string {
	if dringlich {
		return "Dringliche " + geschaeftsart
	}
	return geschaeftsart
}

// FormatDate converts an ISO datetime string (YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)
// to the German format DD.MM.YYYY. Returns an empty string for invalid/empty input.
func FormatDate(isoDate string) string {
	if len(isoDate) < 10 {
		return ""
	}
	parts := strings.Split(isoDate[:10], "-")
	if len(parts) == 3 {
		return fmt.Sprintf("%s.%s.%s", parts[2], parts[1], parts[0])
	}
	return ""
}

// FormatSubmitterLine formats the ✍️ submitter line for a post.
// name is the Erstunterzeichner's name, partei is the party abbreviation,
// anzahlMitunterzeichnende is the count of co-signers (may be nil).
func FormatSubmitterLine(name, partei string, anzahlMitunterzeichnende *int) string {
	base := fmt.Sprintf("✍️ %s — %s", partei, name)
	if anzahlMitunterzeichnende != nil && *anzahlMitunterzeichnende > 0 {
		return fmt.Sprintf("%s (+%d Mitunterzeichnende)", base, *anzahlMitunterzeichnende)
	}
	return base
}

// CleanTitle normalises whitespace and strips leading GRNr prefixes (e.g. "2026/123 ").
func CleanTitle(title string) string {
	title = strings.ReplaceAll(title, "\r\n", " ")
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")
	title = strings.Join(strings.Fields(title), " ")
	return title
}
