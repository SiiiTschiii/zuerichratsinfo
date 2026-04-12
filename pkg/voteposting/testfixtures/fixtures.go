package testfixtures

import (
	"fmt"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func intPtr(i int) *int {
	return &i
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
	return []zurichapi.Abstimmung{
		vote("angenommen-1", "Postulat von Reto Brüesch (SVP): Anpassung der Mindestfläche", "2025/100", "angenommen", 90, 30, 0, 5),
	}
}

// SingleVoteAbgelehnt returns a single rejected Antrag (20/95/5/5).
func SingleVoteAbgelehnt() []zurichapi.Abstimmung {
	return []zurichapi.Abstimmung{
		vote("abgelehnt-1", "Antrag: Festsetzung der Selnaustrasse", "2025/101", "abgelehnt", 20, 95, 5, 5),
	}
}

// LongTitleTruncation returns a vote with a ~300-char title that triggers truncation.
func LongTitleTruncation() []zurichapi.Abstimmung {
	longTitle := "Schlussabstimmung über die bereinigten Dispositivziffern " +
		"zum Objektkredit von 350 Millionen Franken für das Projekt Erweiterung " +
		"und Neugestaltung des Hauptbahnhofs Zürich mit unterirdischer Durchmesserlinie " +
		"und ergänzenden Massnahmen zur Verbesserung der Verkehrsinfrastruktur im Grossraum Zürich " +
		"inklusive der notwendigen Anpassungen an die bestehende urbane Planung"
	return []zurichapi.Abstimmung{
		vote("longtrunc-1", longTitle, "2025/102", "angenommen", 80, 30, 5, 10),
	}
}

// MultiVoteGroup returns 2 votes from the same Geschäft: Einleitungsartikel + Schlussabstimmung.
func MultiVoteGroup() []zurichapi.Abstimmung {
	return []zurichapi.Abstimmung{
		{
			OBJGUID:          "objguid-multi-1",
			SitzungGuid:      "sitzung-multi",
			TraktandumGuid:   "trakt-multi",
			GeschaeftGuid:    "geschaeft-multi",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Gesamtrevision der Gemeindeordnung",
			GeschaeftTitel:   "Gesamtrevision der Gemeindeordnung",
			GeschaeftGrNr:    "2025/103",
			Abstimmungstitel: "Einleitungsartikel",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(90),
			AnzahlNein:       intPtr(20),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		},
		{
			OBJGUID:          "objguid-multi-2",
			SitzungGuid:      "sitzung-multi",
			TraktandumGuid:   "trakt-multi",
			GeschaeftGuid:    "geschaeft-multi",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Gesamtrevision der Gemeindeordnung",
			GeschaeftTitel:   "Gesamtrevision der Gemeindeordnung",
			GeschaeftGrNr:    "2025/103",
			Abstimmungstitel: "Schlussabstimmung",
			Schlussresultat:  "abgelehnt",
			AnzahlJa:         intPtr(40),
			AnzahlNein:       intPtr(70),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		},
	}
}

// GenericAntragFallback returns a vote where TraktandumTitel is "Antrag 1."
// which should trigger fallback to GeschaeftTitel.
func GenericAntragFallback() []zurichapi.Abstimmung {
	return []zurichapi.Abstimmung{
		{
			OBJGUID:          "objguid-antrag-1",
			SitzungGuid:      "sitzung-antrag",
			TraktandumGuid:   "trakt-antrag",
			GeschaeftGuid:    "geschaeft-antrag",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Antrag 1.",
			GeschaeftTitel:   "Postulat von Max Müller (FDP): Bessere Veloinfrastruktur",
			GeschaeftGrNr:    "2025/200",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(80),
			AnzahlNein:       intPtr(35),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(5),
		},
	}
}

// TenVoteStressTest returns 10 votes forcing multiple reply posts.
func TenVoteStressTest() []zurichapi.Abstimmung {
	var votes []zurichapi.Abstimmung
	for i := 0; i < 10; i++ {
		votes = append(votes, zurichapi.Abstimmung{
			OBJGUID:          fmt.Sprintf("objguid-stress-%d", i),
			SitzungGuid:      "sitzung-stress",
			TraktandumGuid:   "trakt-stress",
			GeschaeftGuid:    "geschaeft-stress",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Totalrevision der Bau- und Zonenordnung",
			GeschaeftTitel:   "Totalrevision der Bau- und Zonenordnung",
			GeschaeftGrNr:    "2025/104",
			Abstimmungstitel: fmt.Sprintf("Ziffer %c", 'A'+i),
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(80 + i),
			AnzahlNein:       intPtr(30 - i),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		})
	}
	return votes
}

// VoteWithMentions returns a vote with a politician name that triggers @mention matching.
func VoteWithMentions() []zurichapi.Abstimmung {
	return []zurichapi.Abstimmung{
		{
			OBJGUID:          "objguid-mention-1",
			SitzungGuid:      "sitzung-mention",
			TraktandumGuid:   "trakt-mention",
			GeschaeftGuid:    "geschaeft-mention",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Postulat von Anna Graff (SP): Bessere Sicherheit",
			GeschaeftTitel:   "Bessere Sicherheit",
			GeschaeftGrNr:    "2025/105",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(80),
			AnzahlNein:       intPtr(30),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		},
	}
}

// AuswahlVote returns a single Auswahl vote with A/B/C counts (no ✅/❌ prefix).
func AuswahlVote() []zurichapi.Abstimmung {
	return []zurichapi.Abstimmung{
		{
			OBJGUID:         "objguid-auswahl-1",
			SitzungGuid:     "sitzung-auswahl",
			TraktandumGuid:  "trakt-auswahl",
			GeschaeftGuid:   "geschaeft-auswahl",
			SitzungDatum:    "2026-03-04",
			TraktandumTitel: "Weisung: Jugendwohnkredit 2025",
			GeschaeftTitel:  "Weisung: Jugendwohnkredit 2025",
			GeschaeftGrNr:   "2025/106",
			Schlussresultat: "Auswahl A",
			AnzahlAbwesend:  intPtr(10),
			AnzahlA:         intPtr(74),
			AnzahlB:         intPtr(28),
			AnzahlC:         intPtr(13),
		},
	}
}

// MixedMultiVote returns one Ja/Nein vote + one Auswahl vote in the same group.
func MixedMultiVote() []zurichapi.Abstimmung {
	return []zurichapi.Abstimmung{
		{
			OBJGUID:          "objguid-mixed-1",
			SitzungGuid:      "sitzung-mixed",
			TraktandumGuid:   "trakt-mixed",
			GeschaeftGuid:    "geschaeft-mixed",
			SitzungDatum:     "2026-02-25",
			TraktandumTitel:  "Weisung: BZO",
			GeschaeftTitel:   "Weisung: BZO",
			GeschaeftGrNr:    "2025/107",
			Abstimmungstitel: "Änderungsantrag 9",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(62),
			AnzahlNein:       intPtr(51),
			AnzahlEnthaltung: intPtr(0),
			AnzahlAbwesend:   intPtr(12),
		},
		{
			OBJGUID:          "objguid-mixed-2",
			SitzungGuid:      "sitzung-mixed",
			TraktandumGuid:   "trakt-mixed",
			GeschaeftGuid:    "geschaeft-mixed",
			SitzungDatum:     "2026-02-25",
			TraktandumTitel:  "Weisung: BZO",
			GeschaeftTitel:   "Weisung: BZO",
			GeschaeftGrNr:    "2025/107",
			Abstimmungstitel: "Änderungsantrag 17, 1. Abstimmung",
			Schlussresultat:  "Auswahl A",
			AnzahlAbwesend:   intPtr(11),
			AnzahlA:          intPtr(50),
			AnzahlB:          intPtr(24),
			AnzahlC:          intPtr(40),
		},
	}
}

// PostulatWithGrNrPrefix returns a Postulat where the title starts with "2025/100 Postulat von ..."
// which tests GrNr stripping logic.
func PostulatWithGrNrPrefix() []zurichapi.Abstimmung {
	return []zurichapi.Abstimmung{
		{
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
		},
	}
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
