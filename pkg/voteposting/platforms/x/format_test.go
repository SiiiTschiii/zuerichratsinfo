package x

import (
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func TestFormatVoteGroupPost_PreservesPostulatMotion(t *testing.T) {
	tests := []struct {
		name          string
		votes         []zurichapi.Abstimmung
		expectedParts []string // Parts that should appear in the output
	}{
		{
			name: "Single vote with Postulat in title",
			votes: []zurichapi.Abstimmung{
				{
					OBJGUID:          "test-guid-1",
					GeschaeftGrNr:    "2025/100",
					TraktandumTitel:  "2025/100 Postulat von Reto Brüesch (SVP) vom 05.03.2025: Anpassung der Mindest- und Höchstarealfläche",
					SitzungDatum:     "2025-11-26",
					Schlussresultat:  "abgelehnt",
					AnzahlJa:         intPtr(21),
					AnzahlNein:       intPtr(38),
					AnzahlEnthaltung: intPtr(56),
					AnzahlAbwesend:   intPtr(10),
				},
			},
			expectedParts: []string{
				"Postulat",
				"von Reto Brüesch (SVP)",
				"Anpassung der Mindest- und Höchstarealfläche",
			},
		},
		{
			name: "Single vote with Motion in title",
			votes: []zurichapi.Abstimmung{
				{
					OBJGUID:          "test-guid-2",
					GeschaeftGrNr:    "2025/200",
					TraktandumTitel:  "2025/200 Motion von Liv Mahrer (SP) vom 05.02.2025: Festsetzung der Selnaustrasse",
					SitzungDatum:     "2025-11-26",
					Schlussresultat:  "angenommen",
					AnzahlJa:         intPtr(90),
					AnzahlNein:       intPtr(30),
					AnzahlEnthaltung: intPtr(0),
					AnzahlAbwesend:   intPtr(5),
				},
			},
			expectedParts: []string{
				"Motion",
				"von Liv Mahrer (SP)",
				"Festsetzung der Selnaustrasse",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatVoteGroupPost(tt.votes, nil)

			t.Logf("Full output:\n%s", result)

			for _, part := range tt.expectedParts {
				if !strings.Contains(result, part) {
					t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", part, result)
				}
			}
		})
	}
}

func TestFormatVoteGroupPost_AuswahlVote(t *testing.T) {
	tests := []struct {
		name           string
		votes          []zurichapi.Abstimmung
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name: "Single Auswahl vote — no result prefix",
			votes: []zurichapi.Abstimmung{
				{
					OBJGUID:         "auswahl-guid-1",
					TraktandumTitel: "Weisung: Jugendwohnkredit 2025",
					SitzungDatum:    "2026-03-04",
					Schlussresultat: "Auswahl A",
					AnzahlAbwesend:  intPtr(10),
					AnzahlA:         intPtr(74),
					AnzahlB:         intPtr(28),
					AnzahlC:         intPtr(13),
				},
			},
			shouldContain:    []string{"📊 A: 74 | B: 28 | C: 13 | Abwesend 10", "Jugendwohnkredit"},
			shouldNotContain: []string{"✅", "❌", "Angenommen", "Abgelehnt"},
		},
		{
			name: "Multi vote with Auswahl entry — no emoji before subtitle",
			votes: []zurichapi.Abstimmung{
				{
					OBJGUID:          "guid-ja-nein",
					TraktandumTitel:  "Weisung: BZO",
					Abstimmungstitel: "Änderungsantrag 9",
					SitzungDatum:     "2026-02-25",
					Schlussresultat:  "angenommen",
					AnzahlJa:         intPtr(62),
					AnzahlNein:       intPtr(51),
					AnzahlEnthaltung: intPtr(0),
					AnzahlAbwesend:   intPtr(12),
				},
				{
					OBJGUID:          "guid-auswahl",
					TraktandumTitel:  "Weisung: BZO",
					Abstimmungstitel: "Änderungsantrag 17, 1. Abstimmung",
					SitzungDatum:     "2026-02-25",
					Schlussresultat:  "Auswahl A",
					AnzahlAbwesend:   intPtr(11),
					AnzahlA:          intPtr(50),
					AnzahlB:          intPtr(24),
					AnzahlC:          intPtr(40),
				},
			},
			shouldContain: []string{
				"✅ Änderungsantrag 9",
				"Änderungsantrag 17, 1. Abstimmung",
				"📊 A: 50 | B: 24 | C: 40 | Abwesend 11",
			},
			shouldNotContain: []string{"❌ Änderungsantrag 17", "✅ Änderungsantrag 17"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatVoteGroupPost(tt.votes, nil)
			t.Logf("Full output:\n%s", result)
			for _, part := range tt.shouldContain {
				if !strings.Contains(result, part) {
					t.Errorf("Expected output to contain %q", part)
				}
			}
			for _, part := range tt.shouldNotContain {
				if strings.Contains(result, part) {
					t.Errorf("Expected output NOT to contain %q", part)
				}
			}
		})
	}
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}
