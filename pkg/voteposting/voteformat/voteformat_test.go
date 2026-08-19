package voteformat

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
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

func TestIsSchlussabstimmung(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"contains Schlussabstimmung", "2026_0244 Schlussabstimmung", true},
		{"contains Schlussabstimmung with detail", "Schlussabstimmung über die Dispositivziffer 1", true},
		{"case insensitive", "2026_0244 schlussabstimmung", true},
		{"mixed case", "SCHLUSSABSTIMMUNG", true},
		{"Dringlicherklärung", "2026_0244 Dringlicherklärung", false},
		{"empty string", "", false},
		{"unrelated text", "Antrag 1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSchlussabstimmung(tt.input)
			if got != tt.expected {
				t.Errorf("IsSchlussabstimmung(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSingleVoteSubtitlePrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		voteType string
		expected string
	}{
		{"Dringlicherklärung with number", "2026_0244 Dringlicherklärung", "", "Dringlicherklärung"},
		{"Schlussabstimmung returns empty", "2026_0244 Schlussabstimmung", "", ""},
		{"empty returns empty", "", "", ""},
		{"whitespace-only after strip returns empty", "  ", "", ""},
		{"no number prefix", "Dringlicherklärung", "", "Dringlicherklärung"},
		{"Schlussabstimmung case insensitive", "schlussabstimmung über Ziffer 1", "", ""},

		// A lone quorum vote has no sibling to make its tally look unusual, so
		// the label has to stand on its own when the source gives no subtitle —
		// which is exactly the Kanton Zürich case.
		{"quorum without subtitle stands alone", "", "Quorum", "Quorum"},
		{"quorum joins an existing subtitle", "Dringlicherklärung", "Quorum", "Dringlicherklärung · Quorum"},
		{"quorum survives a suppressed Schlussabstimmung", "Schlussabstimmung", "Quorum", "Quorum"},
		{"Normal adds nothing", "Dringlicherklärung", "Normal", "Dringlicherklärung"},
		{"unknown type adds nothing", "Dringlicherklärung", "Offen", "Dringlicherklärung"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SingleVoteSubtitlePrefix(votes.Vote{Subtitle: tt.input, Type: tt.voteType})
			if got != tt.expected {
				t.Errorf("SingleVoteSubtitlePrefix(%q, type=%q) = %q, want %q",
					tt.input, tt.voteType, got, tt.expected)
			}
		})
	}
}

func TestSubVoteLabelNamesTheVoteType(t *testing.T) {
	// The real 15.06.2026 Glattalbahn group: five votes, second-precision
	// timestamps, no per-vote title of any kind.
	at1130 := time.Date(2026, 6, 15, 11, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		subtitle  string
		voteType  string
		date      time.Time
		index     int
		groupSize int
		expected  string
	}{
		// Kanton Zürich publishes no per-vote title, so the ordinal carries the
		// label — and, in a group, the sitting time as well.
		{"quorum on the ordinal fallback", "", "Quorum", time.Time{}, 1, 1, "Abstimmung 2 · Quorum"},
		{"normal keeps the bare ordinal", "", "Normal", time.Time{}, 0, 1, "Abstimmung 1"},
		{"missing type keeps the bare ordinal", "", "", time.Time{}, 0, 1, "Abstimmung 1"},

		// The clock disambiguates the ordinal; the type qualifies the whole
		// label. Both appear, in that order.
		{"clock and type together", "", "Quorum", at1130, 1, 5, "Abstimmung 2 (11:30) · Quorum"},
		{"clock without a type", "", "Normal", at1130, 0, 5, "Abstimmung 1 (11:30)"},

		// Stadt Zürich titles each vote and uses the same type vocabulary. A
		// source-supplied title needs no clock, but still takes the type.
		{"quorum after a subtitle", "Schlussabstimmung", "Quorum", at1130, 0, 5, "Schlussabstimmung · Quorum"},

		{"Auswahl types stay unlabelled", "", "Gleichgerichtete Anträge mit 4 Optionen", time.Time{}, 0, 1, "Abstimmung 1"},
		{"cup is labelled", "", "Cup-Abstimmung", time.Time{}, 2, 1, "Abstimmung 3 · Cup-Abstimmung"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := votes.Vote{Subtitle: tt.subtitle, Type: tt.voteType, Date: tt.date}
			if got := SubVoteLabel(v, tt.index, tt.groupSize); got != tt.expected {
				t.Errorf("SubVoteLabel(subtitle=%q, type=%q, %d, %d) = %q, want %q",
					tt.subtitle, tt.voteType, tt.index, tt.groupSize, got, tt.expected)
			}
		})
	}
}

func intPtr(n int) *int { return &n }

func TestIsDecisionConsistent(t *testing.T) {
	tests := []struct {
		name            string
		schlussresultat string
		ja              *int
		nein            *int
		wantConsistent  bool
	}{
		{
			// Reproduces the 2025/217 bug: API returned "Ja" but counts show Nein > Ja.
			name:            "API says Ja but Nein > Ja — inconsistent",
			schlussresultat: "Ja",
			ja:              intPtr(41),
			nein:            intPtr(75),
			wantConsistent:  false,
		},
		{
			name:            "API says Angenommen but Nein > Ja — inconsistent",
			schlussresultat: "Angenommen",
			ja:              intPtr(41),
			nein:            intPtr(75),
			wantConsistent:  false,
		},
		{
			name:            "API says Ja and Ja > Nein — consistent",
			schlussresultat: "Ja",
			ja:              intPtr(75),
			nein:            intPtr(41),
			wantConsistent:  true,
		},
		{
			name:            "API says Nein with Nein > Ja — consistent",
			schlussresultat: "Nein",
			ja:              intPtr(41),
			nein:            intPtr(75),
			wantConsistent:  true,
		},
		{
			name:            "Tie: Ja == Nein — consistent (no contradiction)",
			schlussresultat: "Ja",
			ja:              intPtr(60),
			nein:            intPtr(60),
			wantConsistent:  true,
		},
		{
			name:            "Nil ja count — consistent (cannot validate)",
			schlussresultat: "Ja",
			ja:              nil,
			nein:            intPtr(75),
			wantConsistent:  true,
		},
		{
			name:            "Nil nein count — consistent (cannot validate)",
			schlussresultat: "Ja",
			ja:              intPtr(41),
			nein:            nil,
			wantConsistent:  true,
		},
		{
			name:            "Auswahl result — always consistent",
			schlussresultat: "Auswahl A",
			ja:              intPtr(0),
			nein:            intPtr(99),
			wantConsistent:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDecisionConsistent(tt.schlussresultat, tt.ja, tt.nein)
			if got != tt.wantConsistent {
				t.Errorf("IsDecisionConsistent(%q, ja=%v, nein=%v) = %v, want %v",
					tt.schlussresultat, tt.ja, tt.nein, got, tt.wantConsistent)
			}
		})
	}
}

func TestPostHeadline(t *testing.T) {
	day := time.Date(2026, 7, 6, 10, 21, 43, 0, time.UTC)

	tests := []struct {
		name  string
		group []votes.Vote
		want  string
	}{
		{
			name:  "named body and known date",
			group: []votes.Vote{{Body: "Kantonsrat ZH", Date: day}},
			want:  "Kantonsrat ZH | Abstimmung vom 06.07.2026",
		},
		{
			// Votes with an unparseable date are deliberately kept rather than
			// discarded, so this is reachable — and "Abstimmung vom " trailing
			// into nothing is not an acceptable way to render it.
			name:  "unknown date drops the date clause",
			group: []votes.Vote{{Body: "Kantonsrat ZH"}},
			want:  "Kantonsrat ZH | Abstimmung",
		},
		{
			name:  "unnamed body omits the chamber",
			group: []votes.Vote{{Date: day}},
			want:  "Abstimmung vom 06.07.2026",
		},
		{
			name:  "empty group",
			group: nil,
			want:  "Abstimmung",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PostHeadline(tc.group); got != tc.want {
				t.Errorf("PostHeadline = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSubVoteLabel(t *testing.T) {
	timed := time.Date(2026, 6, 29, 14, 32, 11, 0, time.UTC)
	midnight := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		vote      votes.Vote
		index     int
		groupSize int
		want      string
	}{
		{
			name:      "a titled vote uses its own title",
			vote:      votes.Vote{Subtitle: "Schlussabstimmung", Date: timed},
			index:     1,
			groupSize: 3,
			want:      "Schlussabstimmung",
		},
		{
			name:      "an untitled vote in a group falls back to ordinal and time",
			vote:      votes.Vote{Date: timed},
			index:     1,
			groupSize: 3,
			want:      "Abstimmung 2 (14:32)",
		},
		{
			// A lone vote is already named by the post's own headline; a clock
			// reading next to it distinguishes it from nothing.
			name:      "a group of one gets no time",
			vote:      votes.Vote{Date: timed},
			index:     0,
			groupSize: 1,
			want:      "Abstimmung 1",
		},
		{
			// PARIS dates are midnight. Printing "(00:00)" would claim a
			// precision the source never supplied.
			name:      "a date without a clock time gets no time",
			vote:      votes.Vote{Date: midnight},
			index:     0,
			groupSize: 4,
			want:      "Abstimmung 1",
		},
		{
			name:      "a zero date gets no time",
			vote:      votes.Vote{},
			index:     2,
			groupSize: 4,
			want:      "Abstimmung 3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SubVoteLabel(tc.vote, tc.index, tc.groupSize); got != tc.want {
				t.Errorf("SubVoteLabel = %q, want %q", got, tc.want)
			}
		})
	}
}

// A vote type nobody has taught the formatters about must not reach a reader.
// The cases below are the real ones: Kanton Zürich serves a null type for
// attendance determinations and for the occasional genuine quorum vote, and
// both look like ordinary lopsided Ja/Nein tallies once rendered.
func TestIsHandledVoteType(t *testing.T) {
	tests := []struct {
		voteType string
		want     bool
	}{
		{"Normal", true},
		{"Quorum", true},
		// The option count varies, so the whole family matches by prefix.
		{"Gleichgerichtete Anträge mit 3 Optionen", true},
		{"Gleichgerichtete Anträge mit 4 Optionen", true},

		// An unset type is the case this guard exists for: it is what Kanton
		// Zürich still serves for Anwesenheitsermittlung, which is a roll call
		// and not a vote at all.
		{"", false},
		{"   ", false},
		// Knockout rounds between more than two proposals. The source reports no
		// aggregate counts for these, so there is nothing correct to render.
		{"Cup-Abstimmung", false},
		{"Something The Source Just Invented", false},
	}

	for _, tt := range tests {
		t.Run(tt.voteType, func(t *testing.T) {
			if got := IsHandledVoteType(tt.voteType); got != tt.want {
				t.Errorf("IsHandledVoteType(%q) = %v, want %v", tt.voteType, got, tt.want)
			}
		})
	}
}

// A quorum vote counts supporters against a threshold and has no Nein to cast.
// Rendering it in the standard four-part line prints "0 Nein | 0 Enth." for
// positions nobody could take, and files every non-supporter under "Abwesend" —
// which the official record shows is wrong for most of them.
func TestQuorumCountsDropThePhantomColumns(t *testing.T) {
	// The real 15.06.2026 Ausgabenbremse: 129 supporters, 51 not.
	quorum := VoteCounts{
		Ja: ptr(129), Nein: ptr(0), Enthaltung: ptr(0), Abwesend: ptr(51),
		Type: "Quorum",
	}
	const want = "📊 129 Zustimmungen | 51 ohne Zustimmung"

	// Both platforms render the same line: it is already shorter than the
	// standard one, so there is nothing to abbreviate for the tight limit.
	if got := FormatVoteCounts(quorum); got != want {
		t.Errorf("FormatVoteCounts() = %q, want %q", got, want)
	}
	if got := FormatVoteCountsLong(quorum); got != want {
		t.Errorf("FormatVoteCountsLong() = %q, want %q", got, want)
	}

	// Upstream #179 may start mapping the Kantonsrat's "Nicht abgestimmt" onto
	// no instead of absent. Reading only Ja and Abwesend would then drop those
	// members from the line — 46 of them on the vote above. Both buckets mean
	// "did not support", so they are summed and the total stays whole.
	afterUpstreamFix := VoteCounts{
		Ja: ptr(128), Nein: ptr(46), Enthaltung: ptr(0), Abwesend: ptr(6),
		Type: "Quorum",
	}
	if got := FormatVoteCounts(afterUpstreamFix); got != "📊 128 Zustimmungen | 52 ohne Zustimmung" {
		t.Errorf("counts would lose the Nein bucket: %q", got)
	}

	// An ordinary vote is untouched, including one that happens to be lopsided.
	normal := VoteCounts{
		Ja: ptr(130), Nein: ptr(44), Enthaltung: ptr(0), Abwesend: ptr(6),
		Type: "Normal",
	}
	if got := FormatVoteCounts(normal); !strings.Contains(got, "Nein") {
		t.Errorf("FormatVoteCounts() dropped Nein from a Normal vote: %q", got)
	}
}

// The faction table has the same problem as the summary line, and is fixed at
// the aggregation step so the image renderer inherits it.
func TestQuorumFraktionColumns(t *testing.T) {
	v := votes.Vote{
		Type: "Quorum",
		MemberVotes: []votes.MemberVote{
			{Fraktion: "SVP", Choice: "Abwesend"},
			{Fraktion: "SVP", Choice: "Abwesend"},
			{Fraktion: "SP", Choice: "Ja"},
		},
	}

	got := FormatFraktionBreakdown(AggregateFraktionCounts(v))

	if !strings.Contains(got, "(Zust./ohne)") {
		t.Errorf("expected the two-column quorum header, got:\n%s", got)
	}
	for _, unwanted := range []string{"Nein", "Enth", "Abw"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("quorum breakdown still names %q:\n%s", unwanted, got)
		}
	}
	if !strings.Contains(got, "SVP 0/2") || !strings.Contains(got, "SP 1/0") {
		t.Errorf("unexpected counts:\n%s", got)
	}

	// Same forward-compatibility concern as the summary line: a "Nein" choice
	// joins the non-supporters instead of opening a third column that would
	// split one group in two.
	withNein := votes.Vote{
		Type: "Quorum",
		MemberVotes: []votes.MemberVote{
			{Fraktion: "SVP", Choice: "Nein"},
			{Fraktion: "SVP", Choice: "Abwesend"},
			{Fraktion: "SP", Choice: "Ja"},
		},
	}
	got = FormatFraktionBreakdown(AggregateFraktionCounts(withNein))
	if !strings.Contains(got, "(Zust./ohne)") || !strings.Contains(got, "SVP 0/2") {
		t.Errorf("a Nein choice should join the non-supporters:\n%s", got)
	}
}

// TestQuorumTallyFoldsNeinIntoTheUnsupportedBucket pins the arithmetic every
// surface of a post shares.
//
// While every quorum vote the source typed happened to report Nein=0, reading
// Abwesend alone looked correct, and the generated image did exactly that. The
// 17.08.2026 Ausgabenbremse reports 141 Ja to 1 Nein with 38 absent, and the
// card then said "38 ohne" under a Fraktion table whose own column summed to
// 39 — the image contradicting itself.
func TestQuorumTallyFoldsNeinIntoTheUnsupportedBucket(t *testing.T) {
	tests := []struct {
		name                     string
		ja, nein, abwesend       *int
		wantSupport, wantWithout int
	}{
		{"Ausgabenbremse with a Nein cast", intPtr(141), intPtr(1), intPtr(38), 141, 39},
		{"no Nein reported", intPtr(128), intPtr(0), intPtr(52), 128, 52},
		{"nothing reported at all", nil, nil, nil, 0, 0},
		{"absent unreported", intPtr(80), intPtr(2), nil, 80, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := VoteCounts{Ja: tt.ja, Nein: tt.nein, Abwesend: tt.abwesend, Type: "Quorum"}
			support, without := QuorumTally(c)
			if support != tt.wantSupport || without != tt.wantWithout {
				t.Errorf("QuorumTally = (%d, %d), want (%d, %d)",
					support, without, tt.wantSupport, tt.wantWithout)
			}
		})
	}
}

// TestQuorumCaptionAgreesWithTheTally guards the two against drifting apart
// again, now that they share one source.
func TestQuorumCaptionAgreesWithTheTally(t *testing.T) {
	c := VoteCounts{Ja: intPtr(141), Nein: intPtr(1), Enthaltung: intPtr(0), Abwesend: intPtr(38), Type: "Quorum"}

	_, without := QuorumTally(c)
	caption := formatQuorumCounts(c)

	if !strings.Contains(caption, fmt.Sprintf("%d ohne Zustimmung", without)) {
		t.Errorf("caption %q does not state the tally's %d", caption, without)
	}
}

// TestGroupPrefixLineNamesTheKindOfBusiness pins the label line that tells a
// reader what they are looking at. A Postulat is a demand addressed to the
// Regierungsrat, not a decision taken, and the tally alone does not say so.
func TestGroupPrefixLineNamesTheKindOfBusiness(t *testing.T) {
	vote := func(affairType, subtitle, voteType string) votes.Vote {
		return votes.Vote{
			Subtitle: subtitle,
			Type:     voteType,
			Affair:   votes.Affair{Type: affairType},
		}
	}

	tests := []struct {
		name  string
		group []votes.Vote
		want  string
	}{
		{
			name:  "the business type alone",
			group: []votes.Vote{vote("Postulat", "", "Normal")},
			want:  "Postulat",
		},
		{
			name:  "a government bill is named too",
			group: []votes.Vote{vote("Vorlage", "", "Normal")},
			want:  "Vorlage",
		},
		{
			name:  "the vote type joins it",
			group: []votes.Vote{vote("Vorlage", "", "Ausgabenbremse")},
			want:  "Vorlage · Ausgabenbremse",
		},
		{
			// Stadt Zürich reports no business type, so its line is unchanged.
			name:  "no business type leaves the existing prefix",
			group: []votes.Vote{vote("", "Änderungsantrag zu Ziffer 1", "Normal")},
			want:  "Änderungsantrag zu Ziffer 1",
		},
		{
			// Which question was put is meaningless when several share the post;
			// the per-vote headings carry that instead.
			name: "a multi-vote group keeps only the business type",
			group: []votes.Vote{
				vote("Vorlage", "Schlussabstimmung", "Normal"),
				vote("Vorlage", "", "Ausgabenbremse"),
			},
			want: "Vorlage",
		},
		{
			name:  "nothing to say",
			group: []votes.Vote{vote("", "", "Normal")},
			want:  "",
		},
		{
			name:  "no votes",
			group: nil,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GroupPrefixLine(tt.group); got != tt.want {
				t.Errorf("GroupPrefixLine() = %q, want %q", got, tt.want)
			}
		})
	}
}
