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

// Helper function for tests
func intPtr(i int) *int {
	return &i
}
