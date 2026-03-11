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
			name:     "Antrag N zu Dispositivziffer X (generic)",
			input:    "Antrag 1 zu Dispositivziffer 1",
			expected: true,
		},
		{
			name:     "Antrag N zu Dispositivziffer Xa (generic)",
			input:    "2025/391 Antrag 3 zu Dispositivziffer 1a",
			expected: true,
		},
		{
			name:     "Anträge range zu Dispositivziffer (generic)",
			input:    "Anträge 3-4 zu Dispositivziffer 1b",
			expected: true,
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

func ptr(n int) *int { return &n }

func TestIsAuswahlVote(t *testing.T) {
	tests := []struct {
		name     string
		counts   VoteCounts
		expected bool
	}{
		{
			name:     "standard Ja/Nein vote",
			counts:   VoteCounts{Ja: ptr(86), Nein: ptr(13), Enthaltung: ptr(12), Abwesend: ptr(14)},
			expected: false,
		},
		{
			name:     "Auswahl A/B/C vote",
			counts:   VoteCounts{Ja: ptr(0), Nein: ptr(0), Enthaltung: ptr(0), Abwesend: ptr(11), A: ptr(50), B: ptr(24), C: ptr(40)},
			expected: true,
		},
		{
			name:     "Auswahl A/B only",
			counts:   VoteCounts{Abwesend: ptr(10), A: ptr(75), B: ptr(40)},
			expected: true,
		},
		{
			name:     "all nil (unsupported, but not Auswahl)",
			counts:   VoteCounts{},
			expected: false,
		},
		{
			name:     "all zero (unsupported, but not Auswahl)",
			counts:   VoteCounts{A: ptr(0), B: ptr(0)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAuswahlVote(tt.counts)
			if got != tt.expected {
				t.Errorf("IsAuswahlVote() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsUnsupportedVoteType(t *testing.T) {
	tests := []struct {
		name     string
		counts   VoteCounts
		expected bool
	}{
		{
			name:     "standard vote with results",
			counts:   VoteCounts{Ja: ptr(107), Nein: ptr(8), Enthaltung: ptr(0), Abwesend: ptr(10)},
			expected: false,
		},
		{
			name:     "auswahl vote with A/B/C",
			counts:   VoteCounts{Ja: ptr(0), Nein: ptr(0), Enthaltung: ptr(0), Abwesend: ptr(11), A: ptr(50), B: ptr(24), C: ptr(40)},
			expected: false,
		},
		{
			name:     "all nil (no fields parsed)",
			counts:   VoteCounts{},
			expected: true,
		},
		{
			name:     "all zeros (unknown format)",
			counts:   VoteCounts{Ja: ptr(0), Nein: ptr(0), Enthaltung: ptr(0), Abwesend: ptr(11)},
			expected: true,
		},
		{
			name:     "only Abwesend non-zero does not make it supported",
			counts:   VoteCounts{Abwesend: ptr(15)},
			expected: true,
		},
		{
			name:     "single Ja vote is enough",
			counts:   VoteCounts{Ja: ptr(1)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUnsupportedVoteType(tt.counts)
			if got != tt.expected {
				t.Errorf("IsUnsupportedVoteType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatVoteCounts(t *testing.T) {
	tests := []struct {
		name     string
		counts   VoteCounts
		expected string
	}{
		{
			name:     "standard vote short labels",
			counts:   VoteCounts{Ja: ptr(80), Nein: ptr(30), Enthaltung: ptr(5), Abwesend: ptr(10)},
			expected: "📊 80 Ja | 30 Nein | 5 Enth. | 10 Abw.",
		},
		{
			name:     "standard vote nil counts treated as zero",
			counts:   VoteCounts{Ja: ptr(99), Nein: ptr(12), Enthaltung: nil, Abwesend: nil},
			expected: "📊 99 Ja | 12 Nein | 0 Enth. | 0 Abw.",
		},
		{
			name:     "auswahl vote with A/B/C (example 7c90673c)",
			counts:   VoteCounts{Ja: ptr(0), Nein: ptr(0), Enthaltung: ptr(0), Abwesend: ptr(11), A: ptr(50), B: ptr(24), C: ptr(40)},
			expected: "📊 A: 50 | B: 24 | C: 40 | Abw. 11",
		},
		{
			name:     "auswahl vote — only D and E used",
			counts:   VoteCounts{Abwesend: ptr(5), D: ptr(60), E: ptr(55)},
			expected: "📊 D: 60 | E: 55 | Abw. 5",
		},
		{
			name:     "all zero (unsupported) falls back to standard format",
			counts:   VoteCounts{Ja: ptr(0), Nein: ptr(0), Enthaltung: ptr(0), Abwesend: ptr(0)},
			expected: "📊 0 Ja | 0 Nein | 0 Enth. | 0 Abw.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatVoteCounts(tt.counts)
			if got != tt.expected {
				t.Errorf("FormatVoteCounts()\nexpected: %q\ngot:      %q", tt.expected, got)
			}
		})
	}
}

func TestFormatVoteCountsLong(t *testing.T) {
	tests := []struct {
		name     string
		counts   VoteCounts
		expected string
	}{
		{
			name:     "standard vote long labels",
			counts:   VoteCounts{Ja: ptr(107), Nein: ptr(8), Enthaltung: ptr(0), Abwesend: ptr(10)},
			expected: "📊 107 Ja | 8 Nein | 0 Enthaltung | 10 Abwesend",
		},
		{
			name:     "auswahl vote long labels",
			counts:   VoteCounts{Abwesend: ptr(11), A: ptr(50), B: ptr(24), C: ptr(40)},
			expected: "📊 A: 50 | B: 24 | C: 40 | Abwesend 11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatVoteCountsLong(tt.counts)
			if got != tt.expected {
				t.Errorf("FormatVoteCountsLong()\nexpected: %q\ngot:      %q", tt.expected, got)
			}
		})
	}
}
