package zurichapi

// These pin the PARIS-specific rules that decide which of the API's two title
// fields a post uses, and which page it links to. They moved here from
// pkg/voteposting/voteformat when the rules did: both are facts about this
// API's field layout, not about how a post is rendered.

import "testing"

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

func TestGeschaeftLink(t *testing.T) {
	guid := "abfb6cd885df4703a4cdf6cee8440bea"
	expected := "https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=abfb6cd885df4703a4cdf6cee8440bea"
	result := GeschaeftLink(guid)
	if result != expected {
		t.Errorf("GeschaeftLink() failed\nexpected: %q\ngot:      %q", expected, result)
	}
}
