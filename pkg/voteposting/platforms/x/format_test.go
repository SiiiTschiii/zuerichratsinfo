package x

import (
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func TestCleanVoteTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple title with Geschäft number and Postulat",
			input:    "2025/369 Postulat von Reto Brüesch (SVP) und Jean-Marc Jung (SVP) vom 05.03.2025: Anpassung der Mindest- und Höchstarealfläche",
			expected: "Postulat von Reto Brüesch (SVP) und Jean-Marc Jung (SVP) vom 05.03.2025: Anpassung der Mindest- und Höchstarealfläche",
		},
		{
			name:     "Title with Motion",
			input:    "2025/370 Motion von Liv Mahrer (SP) vom 05.02.2025: Festsetzung der Selnaustrasse",
			expected: "Motion von Liv Mahrer (SP) vom 05.02.2025: Festsetzung der Selnaustrasse",
		},
		{
			name:     "Title without type word",
			input:    "2024/431 Anpassung der Bau- und Zonenordnung",
			expected: "Anpassung der Bau- und Zonenordnung",
		},
		{
			name:     "Title with newlines",
			input:    "2025/369 Postulat von\nReto Brüesch\r\n(SVP) vom 05.03.2025",
			expected: "Postulat von Reto Brüesch (SVP) vom 05.03.2025",
		},
		{
			name:     "Title with extra spaces",
			input:    "2025/369   Postulat  von   Reto   Brüesch",
			expected: "Postulat von Reto Brüesch",
		},
		{
			name:     "Title without Geschäft number",
			input:    "Postulat von Reto Brüesch (SVP) vom 05.03.2025",
			expected: "Postulat von Reto Brüesch (SVP) vom 05.03.2025",
		},
		{
			name:     "Empty title",
			input:    "",
			expected: "",
		},
		{
			name:     "Only Geschäft number",
			input:    "2025/369",
			expected: "2025/369",
		},
		{
			name:     "Title with der/die/das prefix",
			input:    "2025/369 der SP-, AL- und Die Mitte/EVP-Fraktion vom 05.02.2025: Abgeltung der Kosten",
			expected: "der SP-, AL- und Die Mitte/EVP-Fraktion vom 05.02.2025: Abgeltung der Kosten",
		},
		{
			name:     "Real example - should preserve Postulat",
			input:    "2025/100 Postulat von Reto Brüesch (SVP) und Jean-Marc Jung (SVP) vom 05.03.2025: Anpassung der Mindest- und Höchstarealfläche im Rahmen der geplanten BZO-Revision",
			expected: "Postulat von Reto Brüesch (SVP) und Jean-Marc Jung (SVP) vom 05.03.2025: Anpassung der Mindest- und Höchstarealfläche im Rahmen der geplanten BZO-Revision",
		},
		{
			name:     "Real API data - Postulat with carriage returns (was correct before)",
			input:    "2024/588\r\nPostulat von Urs Riklin (Grüne) und Dr. Tamara Bosshardt (SP) vom 18.12.2024:\r\nBarrierefreie und familiengerechte öffentliche Toiletten, Anpassung der Raumstandards von Schul- und Sportanlagen",
			expected: "Postulat von Urs Riklin (Grüne) und Dr. Tamara Bosshardt (SP) vom 18.12.2024: Barrierefreie und familiengerechte öffentliche Toiletten, Anpassung der Raumstandards von Schul- und Sportanlagen",
		},
		{
			name:     "Real API data - Motion with carriage returns (was incorrect - cutoff bug)",
			input:    "2025/51\r\nMotion von Liv Mahrer (SP), Marco Denoth (SP), Beat Oberholzer (GLP) und 3 Mitunterzeichnenden vom 05.02.2025:\r\nFestsetzung der Selnaustrasse zwischen Sihlstrasse und Stauffacherbrücke als Strassenraum mit einer dem Platz- oder Strassenraum zugewandten Erdgeschossnutzung, Änderung der Bau- und Zonenordnung (BZO)",
			expected: "Motion von Liv Mahrer (SP), Marco Denoth (SP), Beat Oberholzer (GLP) und 3 Mitunterzeichnenden vom 05.02.2025: Festsetzung der Selnaustrasse zwischen Sihlstrasse und Stauffacherbrücke als Strassenraum mit einer dem Platz- oder Strassenraum zugewandten Erdgeschossnutzung, Änderung der Bau- und Zonenordnung (BZO)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanVoteTitle(tt.input)
			if result != tt.expected {
				t.Errorf("cleanVoteTitle() failed\ninput:    %q\nexpected: %q\ngot:      %q", tt.input, tt.expected, result)
			}
		})
	}
}

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

func TestCleanVoteSubtitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Subtitle with slash pattern",
			input:    "2025/369 Abstimmung über Postulat",
			expected: "Abstimmung über Postulat",
		},
		{
			name:     "Subtitle with underscore pattern",
			input:    "2025_0369 Abstimmung über Motion",
			expected: "Abstimmung über Motion",
		},
		{
			name:     "Subtitle with newlines",
			input:    "2025/369 Abstimmung\nüber\r\nPostulat",
			expected: "Abstimmung über Postulat",
		},
		{
			name:     "Subtitle without number",
			input:    "Abstimmung über Postulat",
			expected: "Abstimmung über Postulat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanVoteSubtitle(tt.input)
			if result != tt.expected {
				t.Errorf("cleanVoteSubtitle() failed\ninput:    %q\nexpected: %q\ngot:      %q", tt.input, tt.expected, result)
			}
		})
	}
}
