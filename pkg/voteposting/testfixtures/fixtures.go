package testfixtures

import (
	"fmt"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func intPtr(i int) *int {
	return &i
}

// stimmabgabe creates a single Stimmabgabe with the given faction and vote behavior.
func stimmabgabe(fraktion, verhalten string) zurichapi.Stimmabgabe {
	return zurichapi.Stimmabgabe{
		Fraktion:             fraktion,
		Abstimmungsverhalten: verhalten,
	}
}

// makeStimmabgaben creates a Stimmabgaben slice from faction vote distributions.
// Each entry is (fraktion, ja, nein, enthaltung, abwesend) for Ja/Nein votes.
func makeStimmabgaben(factions []struct {
	Name                string
	Ja, Nein, Enth, Abw int
}) []zurichapi.Stimmabgabe {
	var result []zurichapi.Stimmabgabe
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

// makeAuswahlStimmabgaben creates Stimmabgaben for Auswahl votes (A/B/C + Abwesend).
func makeAuswahlStimmabgaben(factions []struct {
	Name         string
	A, B, C, Abw int
}) []zurichapi.Stimmabgabe {
	var result []zurichapi.Stimmabgabe
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

// vote creates an Abstimmung with standard Ja/Nein fields and unique GUID values.
func vote(guid, title, grNr, result string, ja, nein, enth, abw int) zurichapi.Abstimmung {
	return zurichapi.Abstimmung{
		OBJGUID:          fmt.Sprintf("objguid-%s", guid),
		SitzungGuid:      fmt.Sprintf("sitzung-%s", guid),
		TraktandumGuid:   fmt.Sprintf("trakt-%s", guid),
		GeschaeftGuid:    fmt.Sprintf("geschaeft-%s", guid),
		SitzungDatum:     "2025-06-15",
		TraktandumTitel:  title,
		GeschaeftTitel:   title,
		GeschaeftGrNr:    grNr,
		Schlussresultat:  result,
		AnzahlJa:         intPtr(ja),
		AnzahlNein:       intPtr(nein),
		AnzahlEnthaltung: intPtr(enth),
		AnzahlAbwesend:   intPtr(abw),
	}
}

// SingleVoteAngenommen returns a single accepted Postulat (90/30/0/5).
func SingleVoteAngenommen() []zurichapi.Abstimmung {
	v := vote("angenommen-1", "Postulat von Reto Brüesch (SVP) und Martin Götzl (FDP) betreffend Anpassung der Mindest- und Höchstarealfläche bei der städtischen Liegenschaftenverwaltung", "2025/100", "angenommen", 90, 30, 0, 5)
	v.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{v}
}

// SingleVoteAbgelehnt returns a single rejected Antrag (20/95/5/5).
func SingleVoteAbgelehnt() []zurichapi.Abstimmung {
	v := vote("abgelehnt-1", "Motion von Liv Mahrer (SP) vom 05.02.2025 betreffend Festsetzung der Selnaustrasse als Begegnungszone und Aufhebung des motorisierten Individualverkehrs", "2025/101", "abgelehnt", 20, 95, 5, 5)
	v.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{v}
}

// LongTitleTruncation returns a vote with a ~300-char title that triggers truncation.
func LongTitleTruncation() []zurichapi.Abstimmung {
	longTitle := "Schlussabstimmung über die bereinigten Dispositivziffern " +
		"zum Objektkredit von 350 Millionen Franken für das Projekt Erweiterung " +
		"und Neugestaltung des Hauptbahnhofs Zürich mit unterirdischer Durchmesserlinie " +
		"und ergänzenden Massnahmen zur Verbesserung der Verkehrsinfrastruktur im Grossraum Zürich " +
		"inklusive der notwendigen Anpassungen an die bestehende urbane Planung"
	v := vote("longtrunc-1", longTitle, "2025/102", "angenommen", 80, 30, 5, 10)
	v.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{v}
}

// MultiVoteGroup returns 2 votes from the same Geschäft: Einleitungsartikel + Schlussabstimmung.
func MultiVoteGroup() []zurichapi.Abstimmung {
	vote1 := zurichapi.Abstimmung{
		OBJGUID:          "objguid-multi-1",
		SitzungGuid:      "sitzung-multi",
		TraktandumGuid:   "trakt-multi",
		GeschaeftGuid:    "geschaeft-multi",
		SitzungDatum:     "2025-06-15",
		TraktandumTitel:  "Teilrevision der Gemeindeordnung der Stadt Zürich, Neuordnung der Kompetenzen im Bereich Stadtentwicklung",
		GeschaeftTitel:   "Teilrevision der Gemeindeordnung der Stadt Zürich, Neuordnung der Kompetenzen im Bereich Stadtentwicklung",
		GeschaeftGrNr:    "2025/103",
		Abstimmungstitel: "Einleitungsartikel",
		Schlussresultat:  "angenommen",
		AnzahlJa:         intPtr(90),
		AnzahlNein:       intPtr(20),
		AnzahlEnthaltung: intPtr(5),
		AnzahlAbwesend:   intPtr(10),
	}
	vote1.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	vote2 := zurichapi.Abstimmung{
		OBJGUID:          "objguid-multi-2",
		SitzungGuid:      "sitzung-multi",
		TraktandumGuid:   "trakt-multi",
		GeschaeftGuid:    "geschaeft-multi",
		SitzungDatum:     "2025-06-15",
		TraktandumTitel:  "Teilrevision der Gemeindeordnung der Stadt Zürich, Neuordnung der Kompetenzen im Bereich Stadtentwicklung",
		GeschaeftTitel:   "Teilrevision der Gemeindeordnung der Stadt Zürich, Neuordnung der Kompetenzen im Bereich Stadtentwicklung",
		GeschaeftGrNr:    "2025/103",
		Abstimmungstitel: "Schlussabstimmung",
		Schlussresultat:  "abgelehnt",
		AnzahlJa:         intPtr(40),
		AnzahlNein:       intPtr(70),
		AnzahlEnthaltung: intPtr(5),
		AnzahlAbwesend:   intPtr(10),
	}
	vote2.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{vote1, vote2}
}

// GenericAntragFallback returns a vote where TraktandumTitel is "Antrag 1."
// which should trigger fallback to GeschaeftTitel.
func GenericAntragFallback() []zurichapi.Abstimmung {
	v := zurichapi.Abstimmung{
		OBJGUID:          "objguid-antrag-1",
		SitzungGuid:      "sitzung-antrag",
		TraktandumGuid:   "trakt-antrag",
		GeschaeftGuid:    "geschaeft-antrag",
		SitzungDatum:     "2025-06-15",
		TraktandumTitel:  "Antrag 1.",
		GeschaeftTitel:   "Postulat von Max Müller (FDP) und Sarah Weber (Grüne) vom 12.11.2024 betreffend Verbesserung der Veloinfrastruktur entlang der Langstrasse und angrenzender Quartiere",
		GeschaeftGrNr:    "2025/200",
		Schlussresultat:  "angenommen",
		AnzahlJa:         intPtr(80),
		AnzahlNein:       intPtr(35),
		AnzahlEnthaltung: intPtr(5),
		AnzahlAbwesend:   intPtr(5),
	}
	v.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{v}
}

// TenVoteStressTest returns 10 votes forcing multiple reply posts.
func TenVoteStressTest() []zurichapi.Abstimmung {
	var votes []zurichapi.Abstimmung
	for i := 0; i < 10; i++ {
		v := zurichapi.Abstimmung{
			OBJGUID:          fmt.Sprintf("objguid-stress-%d", i),
			SitzungGuid:      "sitzung-stress",
			TraktandumGuid:   "trakt-stress",
			GeschaeftGuid:    "geschaeft-stress",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Totalrevision der Bau- und Zonenordnung der Stadt Zürich, Anpassungen an das übergeordnete Recht",
			GeschaeftTitel:   "Totalrevision der Bau- und Zonenordnung der Stadt Zürich, Anpassungen an das übergeordnete Recht",
			GeschaeftGrNr:    "2025/104",
			Abstimmungstitel: fmt.Sprintf("Ziffer %c", 'A'+i),
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(80 + i),
			AnzahlNein:       intPtr(30 - i),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		}
		v.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
		votes = append(votes, v)
	}
	return votes
}

// InstagramLongMultiVoteTruncation returns a long multi-vote fixture that forces Instagram caption truncation.
func InstagramLongMultiVoteTruncation() []zurichapi.Abstimmung {
	const (
		longVoteTitleRepeatCount = 60
		longMainTitleRepeatCount = 120
	)

	votes := TenVoteStressTest()
	for i := range votes {
		votes[i].Abstimmungstitel = strings.Repeat("Sehr langer Abstimmungstitel ", longVoteTitleRepeatCount)
	}
	votes[0].TraktandumTitel = strings.Repeat("Sehr langes Traktandum ", longMainTitleRepeatCount)
	votes[0].GeschaeftTitel = votes[0].TraktandumTitel
	return votes
}

// VoteWithMentions returns a vote with a politician name that triggers @mention matching.
func VoteWithMentions() []zurichapi.Abstimmung {
	v := zurichapi.Abstimmung{
		OBJGUID:          "objguid-mention-1",
		SitzungGuid:      "sitzung-mention",
		TraktandumGuid:   "trakt-mention",
		GeschaeftGuid:    "geschaeft-mention",
		SitzungDatum:     "2025-06-15",
		TraktandumTitel:  "Postulat von Anna Graff (SP) vom 18.09.2024 betreffend Verbesserung der Sicherheit im öffentlichen Raum rund um den Hauptbahnhof",
		GeschaeftTitel:   "Verbesserung der Sicherheit im öffentlichen Raum rund um den Hauptbahnhof",
		GeschaeftGrNr:    "2025/105",
		Schlussresultat:  "angenommen",
		AnzahlJa:         intPtr(80),
		AnzahlNein:       intPtr(30),
		AnzahlEnthaltung: intPtr(5),
		AnzahlAbwesend:   intPtr(10),
	}
	v.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{v}
}

// AuswahlVote returns a single Auswahl vote with A/B/C counts (no ✅/❌ prefix).
func AuswahlVote() []zurichapi.Abstimmung {
	v := zurichapi.Abstimmung{
		OBJGUID:         "objguid-auswahl-1",
		SitzungGuid:     "sitzung-auswahl",
		TraktandumGuid:  "trakt-auswahl",
		GeschaeftGuid:   "geschaeft-auswahl",
		SitzungDatum:    "2026-03-04",
		TraktandumTitel: "Weisung des Stadtrats betreffend Objektkredit für die Erneuerung der Jugendwohnsiedlung Buchegg und Erweiterung des Betreuungsangebots",
		GeschaeftTitel:  "Weisung des Stadtrats betreffend Objektkredit für die Erneuerung der Jugendwohnsiedlung Buchegg und Erweiterung des Betreuungsangebots",
		GeschaeftGrNr:   "2025/106",
		Schlussresultat: "Auswahl A",
		AnzahlAbwesend:  intPtr(10),
		AnzahlA:         intPtr(74),
		AnzahlB:         intPtr(28),
		AnzahlC:         intPtr(13),
	}
	v.Stimmabgaben.Stimmabgabe = makeAuswahlStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{v}
}

// MixedMultiVote returns one Ja/Nein vote + one Auswahl vote in the same group.
func MixedMultiVote() []zurichapi.Abstimmung {
	v1 := zurichapi.Abstimmung{
		OBJGUID:          "objguid-mixed-1",
		SitzungGuid:      "sitzung-mixed",
		TraktandumGuid:   "trakt-mixed",
		GeschaeftGuid:    "geschaeft-mixed",
		SitzungDatum:     "2026-02-25",
		TraktandumTitel:  "Weisung des Stadtrats betreffend Revision der Bau- und Zonenordnung, Anpassung der Bestimmungen für Gewerbe- und Industriezonen",
		GeschaeftTitel:   "Weisung des Stadtrats betreffend Revision der Bau- und Zonenordnung, Anpassung der Bestimmungen für Gewerbe- und Industriezonen",
		GeschaeftGrNr:    "2025/107",
		Abstimmungstitel: "Änderungsantrag 9",
		Schlussresultat:  "angenommen",
		AnzahlJa:         intPtr(62),
		AnzahlNein:       intPtr(51),
		AnzahlEnthaltung: intPtr(0),
		AnzahlAbwesend:   intPtr(12),
	}
	v1.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	v2 := zurichapi.Abstimmung{
		OBJGUID:          "objguid-mixed-2",
		SitzungGuid:      "sitzung-mixed",
		TraktandumGuid:   "trakt-mixed",
		GeschaeftGuid:    "geschaeft-mixed",
		SitzungDatum:     "2026-02-25",
		TraktandumTitel:  "Weisung des Stadtrats betreffend Revision der Bau- und Zonenordnung, Anpassung der Bestimmungen für Gewerbe- und Industriezonen",
		GeschaeftTitel:   "Weisung des Stadtrats betreffend Revision der Bau- und Zonenordnung, Anpassung der Bestimmungen für Gewerbe- und Industriezonen",
		GeschaeftGrNr:    "2025/107",
		Abstimmungstitel: "Änderungsantrag 17, 1. Abstimmung",
		Schlussresultat:  "Auswahl A",
		AnzahlAbwesend:   intPtr(11),
		AnzahlA:          intPtr(50),
		AnzahlB:          intPtr(24),
		AnzahlC:          intPtr(40),
	}
	v2.Stimmabgaben.Stimmabgabe = makeAuswahlStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{v1, v2}
}

// PostulatWithGrNrPrefix returns a Postulat where the title starts with "2025/100 Postulat von ..."
// which tests GrNr stripping logic.
func PostulatWithGrNrPrefix() []zurichapi.Abstimmung {
	v := zurichapi.Abstimmung{
		OBJGUID:          "objguid-grnr-1",
		SitzungGuid:      "sitzung-grnr",
		TraktandumGuid:   "trakt-grnr",
		GeschaeftGuid:    "geschaeft-grnr",
		SitzungDatum:     "2025-11-26",
		TraktandumTitel:  "2025/100 Postulat von Reto Brüesch (SVP) vom 05.03.2025: Anpassung der Mindest- und Höchstarealfläche",
		GeschaeftTitel:   "Anpassung der Mindest- und Höchstarealfläche",
		GeschaeftGrNr:    "2025/100",
		Schlussresultat:  "abgelehnt",
		AnzahlJa:         intPtr(21),
		AnzahlNein:       intPtr(38),
		AnzahlEnthaltung: intPtr(56),
		AnzahlAbwesend:   intPtr(10),
	}
	v.Stimmabgaben.Stimmabgabe = makeStimmabgaben([]struct {
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
	return []zurichapi.Abstimmung{v}
}

// FixtureNames returns fixture names in definition order.
var FixtureNames = []string{
	"single-vote-angenommen",
	"single-vote-abgelehnt",
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
func AllFixtures() map[string][]zurichapi.Abstimmung {
	return map[string][]zurichapi.Abstimmung{
		"single-vote-angenommen":    SingleVoteAngenommen(),
		"single-vote-abgelehnt":     SingleVoteAbgelehnt(),
		"long-title-truncation":     LongTitleTruncation(),
		"multi-vote-group":          MultiVoteGroup(),
		"generic-antrag-fallback":   GenericAntragFallback(),
		"ten-vote-stress-test":      TenVoteStressTest(),
		"vote-with-mentions":        VoteWithMentions(),
		"auswahl-vote":              AuswahlVote(),
		"mixed-multi-vote":          MixedMultiVote(),
		"postulat-with-grnr-prefix": PostulatWithGrNrPrefix(),
	}
}
