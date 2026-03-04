package voteformat

import (
	"testing"
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
			result := CleanVoteTitle(tt.input)
			if result != tt.expected {
				t.Errorf("CleanVoteTitle() failed\ninput:    %q\nexpected: %q\ngot:      %q", tt.input, tt.expected, result)
			}
		})
	}
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
			result := CleanVoteSubtitle(tt.input)
			if result != tt.expected {
				t.Errorf("CleanVoteSubtitle() failed\ninput:    %q\nexpected: %q\ngot:      %q", tt.input, tt.expected, result)
			}
		})
	}
}

func TestIsGenericAntragTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Simple Antrag",
			input:    "2025/391 Antrag 007.",
			expected: true,
		},
		{
			name:     "Antrag without GR number",
			input:    "Antrag 092.",
			expected: true,
		},
		{
			name:     "Antrag with umlaut (Anträge)",
			input:    "2025/391 Anträge 044.",
			expected: true,
		},
		{
			name:     "Anträge range with bis",
			input:    "2025/391 Anträge 044. bis 046.",
			expected: true,
		},
		{
			name:     "Anträge range with dash",
			input:    "Anträge 001. - 003.",
			expected: true,
		},
		{
			name:     "Anträge range with em-dash",
			input:    "Anträge 001. – 003.",
			expected: true,
		},
		{
			name:     "With newlines",
			input:    "2025/391\nAntrag 005.",
			expected: true,
		},
		{
			name:     "Antrag without dot (API variant)",
			input:    "Antrag 1",
			expected: true,
		},
		{
			name:     "Antrag without dot with GR number",
			input:    "2024/31 Antrag 1",
			expected: true,
		},
		{
			name:     "Anträge range without dots",
			input:    "2025/391 Anträge 44 bis 46",
			expected: true,
		},
		{
			name:     "Descriptive title (not generic)",
			input:    "2025/391 Weisung vom 10.09.2025: Finanzverwaltung, Budgetvorlage 2026",
			expected: false,
		},
		{
			name:     "Postulat (not generic Antrag)",
			input:    "2025/575 Postulat von Ivo Bieri (SP)",
			expected: false,
		},
		{
			name:     "Schlussabstimmung (not generic Antrag)",
			input:    "2025_0391 Schlussabstimmung über die Dispositivziffer 3",
			expected: false,
		},
		{
			name:     "Änderungsanträge (not generic - has description)",
			input:    "2025_0391 Änderungsanträge 1–2 zu Dispositivziffer 3",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGenericAntragTitle(tt.input)
			if result != tt.expected {
				t.Errorf("IsGenericAntragTitle() failed\ninput:    %q\nexpected: %v\ngot:      %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestSelectBestTitle(t *testing.T) {
	tests := []struct {
		name            string
		traktandumTitel string
		geschaeftTitel  string
		expected        string
	}{
		{
			name:            "Generic Antrag - should use Geschäft title",
			traktandumTitel: "2025/391 Antrag 007.",
			geschaeftTitel:  "Finanzverwaltung, Budgetvorlage 2026 (Detailbudgets und Globalbudgets)",
			expected:        "Finanzverwaltung, Budgetvorlage 2026 (Detailbudgets und Globalbudgets)",
		},
		{
			name:            "Generic Antrag without dot - should use Geschäft title",
			traktandumTitel: "2024/31 Antrag 1",
			geschaeftTitel:  "Amt für Städtebau, BZO-Teilrevision «Hochhäuser»",
			expected:        "Amt für Städtebau, BZO-Teilrevision «Hochhäuser»",
		},
		{
			name:            "Generic Anträge range - should use Geschäft title",
			traktandumTitel: "2025/391 Anträge 044. bis 046.",
			geschaeftTitel:  "Finanzverwaltung, Budgetvorlage 2026",
			expected:        "Finanzverwaltung, Budgetvorlage 2026",
		},
		{
			name:            "Descriptive Traktandum - should use Traktandum title",
			traktandumTitel: "2025/575 Postulat von Ivo Bieri (SP) und Liv Mahrer (SP) vom 03.12.2025",
			geschaeftTitel:  "Übergangsweise Ausrichtung von Betriebsbeiträgen",
			expected:        "2025/575 Postulat von Ivo Bieri (SP) und Liv Mahrer (SP) vom 03.12.2025",
		},
		{
			name:            "Weisung - should use Traktandum title",
			traktandumTitel: "2025/391 Weisung vom 10.09.2025: Finanzverwaltung, Budgetvorlage 2026",
			geschaeftTitel:  "Finanzverwaltung, Budgetvorlage 2026 (Detailbudgets und Globalbudgets)",
			expected:        "2025/391 Weisung vom 10.09.2025: Finanzverwaltung, Budgetvorlage 2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectBestTitle(tt.traktandumTitel, tt.geschaeftTitel)
			if result != tt.expected {
				t.Errorf("SelectBestTitle() failed\ntraktandumTitel: %q\ngeschaeftTitel:  %q\nexpected:        %q\ngot:             %q",
					tt.traktandumTitel, tt.geschaeftTitel, tt.expected, result)
			}
		})
	}
}

func TestGenerateGeschaeftLink(t *testing.T) {
	guid := "abfb6cd885df4703a4cdf6cee8440bea"
	expected := "https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=abfb6cd885df4703a4cdf6cee8440bea"
	result := GenerateGeschaeftLink(guid)
	if result != expected {
		t.Errorf("GenerateGeschaeftLink() failed\nexpected: %q\ngot:      %q", expected, result)
	}
}
