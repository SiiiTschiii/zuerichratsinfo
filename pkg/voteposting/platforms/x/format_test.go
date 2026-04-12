package x

import (
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// allThreadText joins all post texts in a thread for simple Contains assertions.
func allThreadText(thread []*XPost) string {
	var parts []string
	for _, p := range thread {
		parts = append(parts, p.Text)
	}
	return strings.Join(parts, "\n\n")
}

func TestFormatVoteThread_PreservesPostulatMotion(t *testing.T) {
	tests := []struct {
		name          string
		votes         []zurichapi.Abstimmung
		expectedParts []string
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
			thread := FormatVoteThread(tt.votes, nil, DefaultMaxChars)
			full := allThreadText(thread)
			t.Logf("Full output:\n%s", full)

			for _, part := range tt.expectedParts {
				if !strings.Contains(full, part) {
					t.Errorf("Expected thread to contain %q, but it didn't.\nFull output:\n%s", part, full)
				}
			}
		})
	}
}

func TestFormatVoteThread_AuswahlVote(t *testing.T) {
	tests := []struct {
		name             string
		votes            []zurichapi.Abstimmung
		shouldContain    []string
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
			thread := FormatVoteThread(tt.votes, nil, DefaultMaxChars)
			full := allThreadText(thread)
			t.Logf("Full output:\n%s", full)
			for _, part := range tt.shouldContain {
				if !strings.Contains(full, part) {
					t.Errorf("Expected thread to contain %q", part)
				}
			}
			for _, part := range tt.shouldNotContain {
				if strings.Contains(full, part) {
					t.Errorf("Expected thread NOT to contain %q", part)
				}
			}
		})
	}
}

func TestFormatVoteThread_SingleVoteStructure(t *testing.T) {
	votes := []zurichapi.Abstimmung{
		{
			OBJGUID:          "struct-guid-1",
			TraktandumTitel:  "Weisung: Testvorlage",
			SitzungDatum:     "2026-01-15",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(80),
			AnzahlNein:       intPtr(30),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		},
	}

	thread := FormatVoteThread(votes, nil, DefaultMaxChars)

	if len(thread) < 2 {
		t.Fatalf("expected root + at least 1 reply, got %d posts", len(thread))
	}

	// Root contains header, title, thread hint
	root := thread[0].Text
	if !strings.Contains(root, "Gemeinderat") {
		t.Error("root should contain header")
	}
	if !strings.Contains(root, "Testvorlage") {
		t.Error("root should contain title")
	}
	if !strings.Contains(root, "👇 Details im Thread") {
		t.Error("root should contain thread hint")
	}

	// Last reply contains link
	lastReply := thread[len(thread)-1].Text
	if !strings.Contains(lastReply, "🔗") {
		t.Error("last reply should contain link")
	}
}

func TestFormatVoteThread_MultiVoteStructure(t *testing.T) {
	votes := []zurichapi.Abstimmung{
		{
			OBJGUID:          "multi-guid-1",
			TraktandumTitel:  "Weisung: Grosses Projekt",
			Abstimmungstitel: "Antrag 1",
			SitzungDatum:     "2026-01-15",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(60),
			AnzahlNein:       intPtr(40),
			AnzahlEnthaltung: intPtr(10),
			AnzahlAbwesend:   intPtr(15),
		},
		{
			OBJGUID:          "multi-guid-2",
			TraktandumTitel:  "Weisung: Grosses Projekt",
			Abstimmungstitel: "Antrag 2",
			SitzungDatum:     "2026-01-15",
			Schlussresultat:  "abgelehnt",
			AnzahlJa:         intPtr(30),
			AnzahlNein:       intPtr(70),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(20),
		},
	}

	thread := FormatVoteThread(votes, nil, DefaultMaxChars)

	if len(thread) < 2 {
		t.Fatalf("expected root + at least 1 reply, got %d posts", len(thread))
	}

	root := thread[0].Text
	if !strings.Contains(root, "Grosses Projekt") {
		t.Error("root should contain title")
	}

	full := allThreadText(thread)
	if !strings.Contains(full, "Antrag 1") {
		t.Error("thread should contain first vote subtitle")
	}
	if !strings.Contains(full, "Antrag 2") {
		t.Error("thread should contain second vote subtitle")
	}

	lastReply := thread[len(thread)-1].Text
	if !strings.Contains(lastReply, "🔗") {
		t.Error("last reply should contain link")
	}
}

func TestFormatVoteThread_LinkPlacement(t *testing.T) {
	votes := []zurichapi.Abstimmung{
		{
			OBJGUID:          "link-guid-1",
			TraktandumTitel:  "Weisung: Linktest",
			SitzungDatum:     "2026-01-15",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(80),
			AnzahlNein:       intPtr(30),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		},
	}

	thread := FormatVoteThread(votes, nil, DefaultMaxChars)

	// Link must NOT be in root
	if strings.Contains(thread[0].Text, "🔗") {
		t.Error("root should not contain link — link belongs in replies")
	}

	// Link must be in last reply
	lastReply := thread[len(thread)-1].Text
	if !strings.Contains(lastReply, "🔗") {
		t.Errorf("last reply should contain link, got: %s", lastReply)
	}
}

func TestFormatVoteThread_RootTruncation(t *testing.T) {
	// Create a vote with a very long title that exceeds DefaultMaxChars
	longTitle := strings.Repeat("A", DefaultMaxChars+500)
	votes := []zurichapi.Abstimmung{
		{
			OBJGUID:          "trunc-guid-1",
			TraktandumTitel:  longTitle,
			SitzungDatum:     "2026-01-15",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(80),
			AnzahlNein:       intPtr(30),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		},
	}

	thread := FormatVoteThread(votes, nil, DefaultMaxChars)

	root := thread[0].Text
	if len(root) > DefaultMaxChars {
		t.Errorf("root post exceeds DefaultMaxChars: %d > %d", len(root), DefaultMaxChars)
	}
	if !strings.Contains(root, "…") {
		t.Error("truncated root should contain '…'")
	}
}

func TestFormatVoteThread_EmptyVotes(t *testing.T) {
	thread := FormatVoteThread(nil, nil, DefaultMaxChars)
	if thread != nil {
		t.Errorf("expected nil for empty votes, got %d posts", len(thread))
	}
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}
