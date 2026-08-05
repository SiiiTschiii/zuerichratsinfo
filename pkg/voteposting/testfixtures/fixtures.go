// Package testfixtures holds hand-built vote groups covering the shapes the
// formatters have to handle: single and multi-vote groups, Auswahl votes,
// titles long enough to truncate, and titles that trigger @mention tagging.
//
// Fixtures are votes.Vote values rather than raw API payloads, so they exercise
// the same neutral model every source produces. They are shaped after Stadt
// Zürich data — including its URLs — because that is the jurisdiction whose
// output the golden snapshot pins.
package testfixtures

import (
	"fmt"
	"strings"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// sitzungDate is the sitting date shared by most fixtures.
var sitzungDate = date(2025, 6, 15)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func intPtr(i int) *int {
	return &i
}

// MustDate parses a YYYY-MM-DD date, panicking on malformed input. Intended for
// fixtures and tests, where a bad literal is a bug in the test itself.
func MustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic("testfixtures: bad date " + s + ": " + err.Error())
	}
	return t
}

// The three URL shapes Stadt Zürich publishes. Spelled out here rather than
// imported from pkg/zurichapi: fixtures describe what an already-mapped vote
// looks like, and must not depend on any source package. pkg/zurichapi's own
// tests pin that these strings match what the mapper produces.
func VoteURL(guid string) string {
	return "https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=" + guid
}

func TraktandumURL(sitzungGuid, traktandumGuid string) string {
	return "https://www.gemeinderat-zuerich.ch/sitzungen/sitzung/?gid=" + sitzungGuid + "#" + traktandumGuid
}

func GeschaeftURL(guid string) string {
	return "https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=" + guid
}

// stimmabgabe creates a single member vote with the given faction and choice.
func stimmabgabe(fraktion, choice string) votes.MemberVote {
	return votes.MemberVote{Fraktion: fraktion, Choice: choice}
}

// makeStimmabgaben creates member votes from faction vote distributions.
// Each entry is (fraktion, ja, nein, enthaltung, abwesend) for Ja/Nein votes.
func makeStimmabgaben(factions []struct {
	Name                string
	Ja, Nein, Enth, Abw int
}) []votes.MemberVote {
	var result []votes.MemberVote
	for _, f := range factions {
		for i := 0; i < f.Ja; i++ {
			result = append(result, stimmabgabe(f.Name, "Ja"))
		}
		for i := 0; i < f.Nein; i++ {
			result = append(result, stimmabgabe(f.Name, "Nein"))
		}
		for i := 0; i < f.Enth; i++ {
			result = append(result, stimmabgabe(f.Name, "Enthaltung"))
		}
		for i := 0; i < f.Abw; i++ {
			result = append(result, stimmabgabe(f.Name, "Abwesend"))
		}
	}
	return result
}

// makeAuswahlStimmabgaben creates member votes for Auswahl votes (A/B/C + Abwesend).
func makeAuswahlStimmabgaben(factions []struct {
	Name         string
	A, B, C, Abw int
}) []votes.MemberVote {
	var result []votes.MemberVote
	for _, f := range factions {
		for i := 0; i < f.A; i++ {
			result = append(result, stimmabgabe(f.Name, "A"))
		}
		for i := 0; i < f.B; i++ {
			result = append(result, stimmabgabe(f.Name, "B"))
		}
		for i := 0; i < f.C; i++ {
			result = append(result, stimmabgabe(f.Name, "C"))
		}
		for i := 0; i < f.Abw; i++ {
			result = append(result, stimmabgabe(f.Name, "Abwesend"))
		}
	}
	return result
}

// shareGroup gives every vote in a group the same session and Geschäft
// identity, as votes from one agenda item really have: one session ID, one
// Affair, and one shared group link.
func shareGroup(group []votes.Vote, stem string) {
	sessionGUID := "sitzung-" + stem
	affairGUID := "geschaeft-" + stem
	groupLink := TraktandumURL(sessionGUID, "trakt-"+stem)
	for i := range group {
		group[i].SessionID = sessionGUID
		group[i].GroupURL = groupLink
		group[i].Affair.ID = affairGUID
		group[i].Affair.URL = GeschaeftURL(affairGUID)
	}
}

// base builds a vote with the ID-derived URLs a Stadt Zürich mapping would
// produce, leaving counts and titles to the caller.
func base(guid, title, grNr string) votes.Vote {
	objGUID := fmt.Sprintf("objguid-%s", guid)
	sitzungGUID := fmt.Sprintf("sitzung-%s", guid)
	traktandumGUID := fmt.Sprintf("trakt-%s", guid)
	geschaeftGUID := fmt.Sprintf("geschaeft-%s", guid)

	return votes.Vote{
		SourceID:     objGUID,
		Jurisdiction: "zurich-city",
		SessionID:    sitzungGUID,
		Date:         sitzungDate,
		Title:        title,
		SourceURL:    VoteURL(objGUID),
		GroupURL:     TraktandumURL(sitzungGUID, traktandumGUID),
		Affair: votes.Affair{
			Number: grNr,
			Title:  title,
			ID:     geschaeftGUID,
			URL:    GeschaeftURL(geschaeftGUID),
		},
	}
}

// vote creates a standard Ja/Nein vote.
func vote(guid, title, grNr, result string, ja, nein, enth, abw int) votes.Vote {
	v := base(guid, title, grNr)
	v.Decision = result
	v.Yes = intPtr(ja)
	v.No = intPtr(nein)
	v.Abstention = intPtr(enth)
	v.Absent = intPtr(abw)
	return v
}

// SingleVoteAngenommen returns a single accepted Postulat (90/30/0/5).
func SingleVoteAngenommen() []votes.Vote {
	v := vote("angenommen-1", "Postulat von Reto Brüesch (SVP) und Martin Götzl (FDP) betreffend Anpassung der Mindest- und Höchstarealfläche bei der städtischen Liegenschaftenverwaltung", "2025/100", "angenommen", 90, 30, 0, 5)
	v.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 0, 30, 0, 1},
		{"SVP", 22, 0, 0, 0},
		{"FDP", 20, 0, 0, 1},
		{"Grüne", 0, 0, 0, 2},
		{"GLP", 18, 0, 0, 0},
		{"Die Mitte", 15, 0, 0, 0},
		{"AL", 15, 0, 0, 1},
	})
	return []votes.Vote{v}
}

// SingleVoteAbgelehnt returns a single rejected Antrag (20/95/5/5).
func SingleVoteAbgelehnt() []votes.Vote {
	v := vote("abgelehnt-1", "Motion von Liv Mahrer (SP) vom 05.02.2025 betreffend Festsetzung der Selnaustrasse als Begegnungszone und Aufhebung des motorisierten Individualverkehrs", "2025/101", "abgelehnt", 20, 95, 5, 5)
	v.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 0, 30, 2, 1},
		{"SVP", 0, 18, 0, 1},
		{"FDP", 0, 20, 0, 1},
		{"Grüne", 0, 15, 3, 0},
		{"GLP", 8, 7, 0, 1},
		{"Die Mitte", 12, 0, 0, 1},
		{"AL", 0, 5, 0, 0},
	})
	return []votes.Vote{v}
}

// SingleVoteDringlicherklaerung returns a single vote whose Abstimmungsgegenstand
// ("Dringlicherklärung") is not a Schlussabstimmung. This exercises the eyebrow/prefix
// that gets prepended above the title on single-vote posts and images.
func SingleVoteDringlicherklaerung() []votes.Vote {
	v := vote("dringlich-1", "Motion von Dr. Jonas Keller (SP), Pascal Lamprecht (SP) und Tanja Maag (AL) vom 27.05.2026: Erhalt kleinerer bis mittlerer Konzertlokale sowie Unterstützung der Kulturanbietenden bei der Suche nach Lokalitäten", "2026/244", "angenommen", 66, 0, 0, 59)
	v.Subtitle = "2026_0244 Dringlicherklärung"
	v.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 37, 0, 0, 4},
		{"FDP", 0, 0, 0, 24},
		{"SVP", 0, 0, 0, 16},
		{"GLP", 0, 0, 0, 15},
		{"Grüne", 14, 0, 0, 0},
		{"AL", 8, 0, 0, 0},
		{"Die Mitte", 7, 0, 0, 0},
	})
	return []votes.Vote{v}
}

// LongTitleTruncation returns a vote with a ~300-char title that triggers truncation.
func LongTitleTruncation() []votes.Vote {
	longTitle := "Schlussabstimmung über die bereinigten Dispositivziffern " +
		"zum Objektkredit von 350 Millionen Franken für das Projekt Erweiterung " +
		"und Neugestaltung des Hauptbahnhofs Zürich mit unterirdischer Durchmesserlinie " +
		"und ergänzenden Massnahmen zur Verbesserung der Verkehrsinfrastruktur im Grossraum Zürich " +
		"inklusive der notwendigen Anpassungen an die bestehende urbane Planung"
	v := vote("longtrunc-1", longTitle, "2025/102", "angenommen", 80, 30, 5, 10)
	v.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 30, 0, 2, 2},
		{"SVP", 0, 18, 0, 1},
		{"FDP", 18, 0, 0, 2},
		{"Grüne", 14, 0, 3, 1},
		{"GLP", 10, 5, 0, 2},
		{"Die Mitte", 8, 5, 0, 1},
		{"AL", 0, 2, 0, 1},
	})
	return []votes.Vote{v}
}

// MultiVoteGroup returns 2 votes from the same Geschäft: Einleitungsartikel + Schlussabstimmung.
func MultiVoteGroup() []votes.Vote {
	const title = "Teilrevision der Gemeindeordnung der Stadt Zürich, Neuordnung der Kompetenzen im Bereich Stadtentwicklung"

	vote1 := vote("multi-1", title, "2025/103", "angenommen", 90, 20, 5, 10)
	vote1.Subtitle = "Einleitungsartikel"
	vote1.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 32, 0, 0, 2},
		{"SVP", 0, 18, 0, 1},
		{"FDP", 20, 0, 0, 2},
		{"Grüne", 12, 0, 5, 1},
		{"GLP", 14, 0, 0, 2},
		{"Die Mitte", 12, 0, 0, 1},
		{"AL", 0, 2, 0, 1},
	})

	vote2 := vote("multi-2", title, "2025/103", "abgelehnt", 40, 70, 5, 10)
	vote2.Subtitle = "Schlussabstimmung"
	vote2.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 0, 30, 2, 2},
		{"SVP", 18, 0, 0, 1},
		{"FDP", 0, 20, 0, 2},
		{"Grüne", 10, 0, 3, 1},
		{"GLP", 12, 0, 0, 2},
		{"Die Mitte", 0, 12, 0, 1},
		{"AL", 0, 8, 0, 1},
	})

	group := []votes.Vote{vote1, vote2}
	shareGroup(group, "multi")
	return group
}

// GenericAntragFallback returns a vote whose Traktandum title was the generic
// "Antrag 1.", so the mapper substituted the Geschäft title and pointed both
// links at the Geschäft page.
func GenericAntragFallback() []votes.Vote {
	v := vote("antrag-1", "Postulat von Max Müller (FDP) und Sarah Weber (Grüne) vom 12.11.2024 betreffend Verbesserung der Veloinfrastruktur entlang der Langstrasse und angrenzender Quartiere", "2025/200", "angenommen", 80, 35, 5, 5)
	// Only the generic-Antrag case sends both links to the Geschäft page.
	v.Affair.ID = "geschaeft-antrag"
	v.Affair.URL = GeschaeftURL(v.Affair.ID)
	v.SourceURL = v.Affair.URL
	v.GroupURL = v.Affair.URL
	v.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 28, 0, 2, 1},
		{"SVP", 0, 20, 0, 0},
		{"FDP", 20, 0, 0, 1},
		{"Grüne", 12, 5, 3, 0},
		{"GLP", 12, 3, 0, 1},
		{"Die Mitte", 8, 5, 0, 1},
		{"AL", 0, 2, 0, 1},
	})
	return []votes.Vote{v}
}

// TenVoteStressTest returns 10 votes forcing multiple reply posts.
func TenVoteStressTest() []votes.Vote {
	const title = "Totalrevision der Bau- und Zonenordnung der Stadt Zürich, Anpassungen an das übergeordnete Recht"

	var group []votes.Vote
	for i := 0; i < 10; i++ {
		v := vote(fmt.Sprintf("stress-%d", i), title, "2025/104", "angenommen", 80+i, 30-i, 5, 10)
		v.Subtitle = fmt.Sprintf("Ziffer %c", 'A'+i)
		v.MemberVotes = makeStimmabgaben([]struct {
			Name                string
			Ja, Nein, Enth, Abw int
		}{
			{"SP", 25 + i, 0, 2, 2},
			{"SVP", 0, 15 - i, 0, 1},
			{"FDP", 18, 2, 0, 2},
			{"Grüne", 15, 0, 3, 1},
			{"GLP", 12 + i, 5 - i, 0, 2},
			{"Die Mitte", 10, 5, 0, 1},
			{"AL", 0, 3, 0, 1},
		})
		group = append(group, v)
	}
	shareGroup(group, "stress")
	return group
}

// InstagramLongMultiVoteTruncation returns a long multi-vote fixture that forces Instagram caption truncation.
func InstagramLongMultiVoteTruncation() []votes.Vote {
	const (
		longVoteTitleRepeatCount = 60
		longMainTitleRepeatCount = 120
	)

	group := TenVoteStressTest()
	for i := range group {
		group[i].Subtitle = strings.Repeat("Sehr langer Abstimmungstitel ", longVoteTitleRepeatCount)
	}
	longTitle := strings.Repeat("Sehr langes Traktandum ", longMainTitleRepeatCount)
	group[0].Title = longTitle
	group[0].Affair.Title = longTitle
	return group
}

// VoteWithMentions returns a vote with a politician name that triggers @mention matching.
func VoteWithMentions() []votes.Vote {
	v := vote("mention-1", "Postulat von Anna Graff (SP) vom 18.09.2024 betreffend Verbesserung der Sicherheit im öffentlichen Raum rund um den Hauptbahnhof", "2025/105", "angenommen", 80, 30, 5, 10)
	v.Affair.Title = "Verbesserung der Sicherheit im öffentlichen Raum rund um den Hauptbahnhof"
	v.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 32, 0, 0, 2},
		{"SVP", 0, 18, 0, 1},
		{"FDP", 15, 5, 0, 2},
		{"Grüne", 16, 0, 2, 1},
		{"GLP", 10, 4, 3, 2},
		{"Die Mitte", 7, 3, 0, 1},
		{"AL", 0, 0, 0, 1},
	})
	return []votes.Vote{v}
}

// AuswahlVote returns a single Auswahl vote with A/B/C counts (no ✅/❌ prefix).
func AuswahlVote() []votes.Vote {
	const title = "Weisung des Stadtrats betreffend Objektkredit für die Erneuerung der Jugendwohnsiedlung Buchegg und Erweiterung des Betreuungsangebots"

	v := base("auswahl-1", title, "2025/106")
	v.Date = date(2026, 3, 4)
	v.Decision = "Auswahl A"
	v.Absent = intPtr(10)
	v.ChoiceA = intPtr(74)
	v.ChoiceB = intPtr(28)
	v.ChoiceC = intPtr(13)
	v.MemberVotes = makeAuswahlStimmabgaben([]struct {
		Name         string
		A, B, C, Abw int
	}{
		{"SP", 30, 0, 0, 2},
		{"FDP", 18, 2, 0, 2},
		{"SVP", 0, 18, 0, 2},
		{"GLP", 14, 0, 0, 1},
		{"Grüne", 0, 0, 13, 1},
		{"Die Mitte", 12, 0, 0, 1},
		{"AL", 0, 8, 0, 1},
	})
	return []votes.Vote{v}
}

// MixedMultiVote returns one Ja/Nein vote + one Auswahl vote in the same group.
func MixedMultiVote() []votes.Vote {
	const title = "Weisung des Stadtrats betreffend Revision der Bau- und Zonenordnung, Anpassung der Bestimmungen für Gewerbe- und Industriezonen"

	v1 := vote("mixed-1", title, "2025/107", "angenommen", 62, 51, 0, 12)
	v1.Date = date(2026, 2, 25)
	v1.Subtitle = "Änderungsantrag 9"
	v1.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 30, 0, 0, 2},
		{"SVP", 0, 18, 0, 2},
		{"FDP", 0, 20, 0, 2},
		{"Grüne", 14, 0, 0, 1},
		{"GLP", 10, 5, 0, 2},
		{"Die Mitte", 8, 5, 0, 2},
		{"AL", 0, 3, 0, 1},
	})

	v2 := base("mixed-2", title, "2025/107")
	v2.Date = date(2026, 2, 25)
	v2.Subtitle = "Änderungsantrag 17, 1. Abstimmung Weisung des Stadtrats betreffend Revision der Bau- und Zonenordnung"
	v2.Decision = "Auswahl A"
	v2.Absent = intPtr(11)
	v2.ChoiceA = intPtr(50)
	v2.ChoiceB = intPtr(24)
	v2.ChoiceC = intPtr(40)
	v2.MemberVotes = makeAuswahlStimmabgaben([]struct {
		Name         string
		A, B, C, Abw int
	}{
		{"SP", 28, 0, 0, 2},
		{"SVP", 0, 0, 20, 2},
		{"FDP", 0, 18, 0, 2},
		{"Grüne", 10, 0, 5, 1},
		{"GLP", 8, 6, 0, 2},
		{"Die Mitte", 4, 0, 10, 1},
		{"AL", 0, 0, 5, 1},
	})

	group := []votes.Vote{v1, v2}
	shareGroup(group, "mixed")
	return group
}

// PostulatWithGrNrPrefix returns a Postulat whose title starts with
// "2025/100 Postulat von ...", exercising Geschäft-number stripping.
func PostulatWithGrNrPrefix() []votes.Vote {
	v := vote("grnr-1", "2025/100 Postulat von Reto Brüesch (SVP) vom 05.03.2025: Anpassung der Mindest- und Höchstarealfläche", "2025/100", "abgelehnt", 21, 38, 56, 10)
	v.Date = date(2025, 11, 26)
	v.Affair.Title = "Anpassung der Mindest- und Höchstarealfläche"
	v.MemberVotes = makeStimmabgaben([]struct {
		Name                string
		Ja, Nein, Enth, Abw int
	}{
		{"SP", 0, 0, 30, 2},
		{"SVP", 18, 0, 0, 1},
		{"FDP", 0, 20, 0, 2},
		{"Grüne", 0, 0, 15, 1},
		{"GLP", 3, 10, 5, 2},
		{"Die Mitte", 0, 8, 6, 1},
		{"AL", 0, 0, 0, 1},
	})
	return []votes.Vote{v}
}

// FixtureNames returns fixture names in definition order.
var FixtureNames = []string{
	"single-vote-angenommen",
	"single-vote-abgelehnt",
	"single-vote-dringlicherklaerung",
	"long-title-truncation",
	"multi-vote-group",
	"generic-antrag-fallback",
	"ten-vote-stress-test",
	"vote-with-mentions",
	"auswahl-vote",
	"mixed-multi-vote",
	"postulat-with-grnr-prefix",
}

// AllFixtures returns all fixtures keyed by kebab-case name.
func AllFixtures() map[string][]votes.Vote {
	return map[string][]votes.Vote{
		"single-vote-angenommen":          SingleVoteAngenommen(),
		"single-vote-abgelehnt":           SingleVoteAbgelehnt(),
		"single-vote-dringlicherklaerung": SingleVoteDringlicherklaerung(),
		"long-title-truncation":           LongTitleTruncation(),
		"multi-vote-group":                MultiVoteGroup(),
		"generic-antrag-fallback":         GenericAntragFallback(),
		"ten-vote-stress-test":            TenVoteStressTest(),
		"vote-with-mentions":              VoteWithMentions(),
		"auswahl-vote":                    AuswahlVote(),
		"mixed-multi-vote":                MixedMultiVote(),
		"postulat-with-grnr-prefix":       PostulatWithGrNrPrefix(),
	}
}
