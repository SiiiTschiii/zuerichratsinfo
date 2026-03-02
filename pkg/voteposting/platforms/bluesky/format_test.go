package bluesky

import (
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// --- Helper Functions ---

func intPtr(i int) *int {
	return &i
}

// sampleVote returns a minimal Abstimmung for testing.
func sampleVote(title, result string, ja, nein, enth, abw int) zurichapi.Abstimmung {
	return zurichapi.Abstimmung{
		OBJGUID:          "vote-guid-1",
		SitzungGuid:      "sitzung-guid-1",
		TraktandumGuid:   "trakt-guid-1",
		GeschaeftGuid:    "geschaeft-guid-1",
		SitzungDatum:     "2025-06-15",
		TraktandumTitel:  title,
		GeschaeftTitel:   title,
		GeschaeftGrNr:    "2025/100",
		Schlussresultat:  result,
		AnzahlJa:         intPtr(ja),
		AnzahlNein:       intPtr(nein),
		AnzahlEnthaltung: intPtr(enth),
		AnzahlAbwesend:   intPtr(abw),
	}
}

// --- Tests ---

func TestFormatVoteThread_EmptyVotes(t *testing.T) {
	result := FormatVoteThread(nil)
	if result != nil {
		t.Errorf("expected nil for empty votes, got %d posts", len(result))
	}

	result = FormatVoteThread([]zurichapi.Abstimmung{})
	if result != nil {
		t.Errorf("expected nil for empty slice, got %d posts", len(result))
	}
}

func TestFormatVoteThread_SingleVote(t *testing.T) {
	votes := []zurichapi.Abstimmung{
		sampleVote("Postulat von Reto Brüesch (SVP): Anpassung der Mindestfläche", "angenommen", 90, 30, 0, 5),
	}
	thread := FormatVoteThread(votes)

	if len(thread) < 2 {
		t.Fatalf("expected at least 2 posts (root + reply), got %d", len(thread))
	}

	root := thread[0]
	// Root must contain header, result, title, and thread hint
	for _, part := range []string{
		"🗳️ Gemeinderat",
		"Abstimmung vom 15.06.2025",
		"✅",
		"Angenommen",
		"Anpassung der Mindestfläche",
		"👇 Details im Thread",
	} {
		if !strings.Contains(root.Text, part) {
			t.Errorf("root post missing %q\nFull root:\n%s", part, root.Text)
		}
	}

	if graphemeLen(root.Text) > maxGraphemes {
		t.Errorf("root post exceeds %d graphemes: %d\n%s", maxGraphemes, graphemeLen(root.Text), root.Text)
	}

	lastReply := thread[len(thread)-1]
	// Last reply must contain vote counts and link
	for _, part := range []string{
		"90 Ja",
		"30 Nein",
		"0 Enth.",
		"5 Abw.",
		"🔗",
	} {
		if !strings.Contains(lastReply.Text, part) {
			t.Errorf("reply missing %q\nFull reply:\n%s", part, lastReply.Text)
		}
	}

	for i, post := range thread {
		if graphemeLen(post.Text) > maxGraphemes {
			t.Errorf("post %d exceeds %d graphemes: %d\n%s", i, maxGraphemes, graphemeLen(post.Text), post.Text)
		}
	}
}

func TestFormatVoteThread_RejectedVote(t *testing.T) {
	votes := []zurichapi.Abstimmung{
		sampleVote("Antrag: Festsetzung der Selnaustrasse", "abgelehnt", 20, 95, 5, 5),
	}
	thread := FormatVoteThread(votes)

	if len(thread) < 2 {
		t.Fatalf("expected at least 2 posts, got %d", len(thread))
	}

	root := thread[0]
	if !strings.Contains(root.Text, "❌") {
		t.Errorf("rejected vote should have ❌ emoji\nFull root:\n%s", root.Text)
	}
	if !strings.Contains(root.Text, "Abgelehnt") {
		t.Errorf("rejected vote should say Abgelehnt\nFull root:\n%s", root.Text)
	}
}

func TestFormatVoteThread_VeryLongTitle(t *testing.T) {
	longTitle := "Schlussabstimmung über die bereinigten Dispositivziffern " +
		"zum Objektkredit von 350 Millionen Franken für das Projekt Erweiterung " +
		"und Neugestaltung des Hauptbahnhofs Zürich mit unterirdischer Durchmesserlinie " +
		"und ergänzenden Massnahmen zur Verbesserung der Verkehrsinfrastruktur im Grossraum Zürich " +
		"inklusive der notwendigen Anpassungen an die bestehende urbane Planung"

	votes := []zurichapi.Abstimmung{
		sampleVote(longTitle, "angenommen", 80, 30, 5, 10),
	}
	thread := FormatVoteThread(votes)

	if len(thread) < 2 {
		t.Fatalf("expected at least 2 posts, got %d", len(thread))
	}

	root := thread[0]
	if graphemeLen(root.Text) > maxGraphemes {
		t.Errorf("root exceeds %d graphemes with long title: %d\n%s", maxGraphemes, graphemeLen(root.Text), root.Text)
	}
	if !strings.Contains(root.Text, "…") {
		t.Errorf("expected truncation ellipsis in root for long title\nFull root:\n%s", root.Text)
	}
	if !strings.Contains(root.Text, "🗳️ Gemeinderat") {
		t.Errorf("root missing header\n%s", root.Text)
	}
	if !strings.Contains(root.Text, "👇 Details im Thread") {
		t.Errorf("root missing thread hint\n%s", root.Text)
	}

	// First reply should contain the FULL untruncated title
	firstReply := thread[1]
	if !strings.Contains(firstReply.Text, "urbane Planung") {
		t.Errorf("first reply should contain the full untruncated title\nFull reply:\n%s", firstReply.Text)
	}
}

func TestFormatVoteThread_MultipleVotes(t *testing.T) {
	votes := []zurichapi.Abstimmung{
		{
			OBJGUID:          "vote-1",
			SitzungGuid:      "sitzung-1",
			TraktandumGuid:   "trakt-1",
			GeschaeftGuid:    "geschaeft-1",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Gesamtrevision der Gemeindeordnung",
			GeschaeftTitel:   "Gesamtrevision der Gemeindeordnung",
			GeschaeftGrNr:    "2025/100",
			Abstimmungstitel: "Einleitungsartikel",
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(90),
			AnzahlNein:       intPtr(20),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		},
		{
			OBJGUID:          "vote-2",
			SitzungGuid:      "sitzung-1",
			TraktandumGuid:   "trakt-1",
			GeschaeftGuid:    "geschaeft-1",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Gesamtrevision der Gemeindeordnung",
			GeschaeftTitel:   "Gesamtrevision der Gemeindeordnung",
			GeschaeftGrNr:    "2025/100",
			Abstimmungstitel: "Schlussabstimmung",
			Schlussresultat:  "abgelehnt",
			AnzahlJa:         intPtr(40),
			AnzahlNein:       intPtr(70),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		},
	}

	thread := FormatVoteThread(votes)
	if len(thread) < 2 {
		t.Fatalf("expected at least 2 posts, got %d", len(thread))
	}

	root := thread[0]
	if !strings.Contains(root.Text, "Gesamtrevision der Gemeindeordnung") {
		t.Errorf("root missing title\n%s", root.Text)
	}
	if !strings.Contains(root.Text, "👇 Details im Thread") {
		t.Errorf("root missing thread hint\n%s", root.Text)
	}

	allReplies := ""
	for _, post := range thread[1:] {
		allReplies += post.Text + "\n"
	}

	for _, part := range []string{
		"Einleitungsartikel",
		"Schlussabstimmung",
		"90 Ja",
		"40 Ja",
		"70 Nein",
		"🔗",
	} {
		if !strings.Contains(allReplies, part) {
			t.Errorf("replies missing %q\nAll replies:\n%s", part, allReplies)
		}
	}

	for i, post := range thread {
		if graphemeLen(post.Text) > maxGraphemes {
			t.Errorf("post %d exceeds %d graphemes: %d\n%s", i, maxGraphemes, graphemeLen(post.Text), post.Text)
		}
	}
}

func TestFormatVoteThread_LinkFacetOnLastReply(t *testing.T) {
	votes := []zurichapi.Abstimmung{
		sampleVote("Budget 2026", "angenommen", 100, 15, 5, 5),
	}
	thread := FormatVoteThread(votes)
	lastReply := thread[len(thread)-1]
	if len(lastReply.Facets) == 0 {
		t.Errorf("last reply should have a link facet\nText: %s", lastReply.Text)
	}
}

func TestFormatVoteThread_GenericAntragUsesGeschaeftTitle(t *testing.T) {
	votes := []zurichapi.Abstimmung{
		{
			OBJGUID:         "vote-1",
			SitzungGuid:     "sitzung-1",
			TraktandumGuid:  "trakt-1",
			GeschaeftGuid:   "geschaeft-1",
			SitzungDatum:    "2025-06-15",
			TraktandumTitel: "Antrag 1.",
			GeschaeftTitel:  "Postulat von Max Müller (FDP): Bessere Veloinfrastruktur",
			GeschaeftGrNr:   "2025/200",
			Schlussresultat: "angenommen",
			AnzahlJa:        intPtr(80),
			AnzahlNein:      intPtr(35),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:  intPtr(5),
		},
	}

	thread := FormatVoteThread(votes)
	root := thread[0]
	if !strings.Contains(root.Text, "Bessere Veloinfrastruktur") {
		t.Errorf("expected GeschaeftTitel to be used for generic Antrag title\nFull root:\n%s", root.Text)
	}
}

func TestFormatVoteThread_AllPostsWithinLimit(t *testing.T) {
	var votes []zurichapi.Abstimmung
	// Stress test: many votes to force multiple reply posts
	for i := 0; i < 10; i++ {
		votes = append(votes, zurichapi.Abstimmung{
			OBJGUID:          "vote-guid-" + string(rune('a'+i)),
			SitzungGuid:      "sitzung-1",
			TraktandumGuid:   "trakt-1",
			GeschaeftGuid:    "geschaeft-1",
			SitzungDatum:     "2025-06-15",
			TraktandumTitel:  "Gesamtrevision der Gemeindeordnung",
			GeschaeftTitel:   "Gesamtrevision der Gemeindeordnung",
			GeschaeftGrNr:    "2025/100",
			Abstimmungstitel: "Ziffer " + string(rune('A'+i)),
			Schlussresultat:  "angenommen",
			AnzahlJa:         intPtr(80 + i),
			AnzahlNein:       intPtr(30 - i),
			AnzahlEnthaltung: intPtr(5),
			AnzahlAbwesend:   intPtr(10),
		})
	}

	thread := FormatVoteThread(votes)
	if len(thread) < 3 {
		t.Errorf("expected at least 3 posts for 10 votes, got %d", len(thread))
	}

	for i, post := range thread {
		gl := graphemeLen(post.Text)
		if gl > maxGraphemes {
			t.Errorf("post %d exceeds %d graphemes: %d\n%s", i, maxGraphemes, gl, post.Text)
		}
	}

	lastReply := thread[len(thread)-1]
	if !strings.Contains(lastReply.Text, "🔗") {
		t.Errorf("last reply missing link\n%s", lastReply.Text)
	}
}