package bluesky

import (
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/geschaeftposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func intPtr(i int) *int { return &i }

// allThreadText joins all post texts for simple Contains checks.
func allThreadText(thread []*GeschaeftPost) string {
	var parts []string
	for _, p := range thread {
		parts = append(parts, p.Text)
	}
	return strings.Join(parts, "\n\n")
}

// TestFormatGeschaeftThread_MotionSimple verifies the basic Motion thread structure.
func TestFormatGeschaeftThread_MotionSimple(t *testing.T) {
	g := testfixtures.MotionSimple()
	thread := FormatGeschaeftThread(g, nil)

	if len(thread) != 2 {
		t.Fatalf("expected 2 posts (root + reply), got %d", len(thread))
	}

	root := thread[0].Text
	reply := thread[1].Text
	t.Logf("Root:\n%s\n\nReply:\n%s", root, reply)

	// Root must contain the type header
	for _, want := range []string{
		"📋 Neue Motion",
		"Gemeinderat Zürich",
		"Überdeckung",           // part of the title
		"✍️ Grüne — Martin Busekros",
		"+4 Mitunterzeichnende",
		"📅 24.06.2026",
		"👇 Was ist eine Motion?",
		"geschaefte/detail.php?gid=",
	} {
		if !strings.Contains(root, want) {
			t.Errorf("root missing %q\nRoot:\n%s", want, root)
		}
	}

	// Reply must contain civic education
	for _, want := range []string{
		"ℹ️",
		"Motion",
		"⏭️",
		"gemeinderat-zuerich.ch/gemeinderat/geschaeftsarten/",
	} {
		if !strings.Contains(reply, want) {
			t.Errorf("reply missing %q\nReply:\n%s", want, reply)
		}
	}

	// All posts must be within grapheme limit
	for i, post := range thread {
		if graphemeLen(post.Text) > maxGraphemes {
			t.Errorf("post %d exceeds %d graphemes: %d\n%s", i, maxGraphemes, graphemeLen(post.Text), post.Text)
		}
	}
}

// TestFormatGeschaeftThread_MotionDringlich verifies that dringliche Motions use 🚨.
func TestFormatGeschaeftThread_MotionDringlich(t *testing.T) {
	g := testfixtures.MotionDringlich()
	thread := FormatGeschaeftThread(g, nil)

	if len(thread) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(thread))
	}

	root := thread[0].Text
	t.Logf("Root:\n%s", root)

	if !strings.Contains(root, "🚨") {
		t.Error("dringliche Motion must use 🚨 emoji")
	}
	if !strings.Contains(root, "Dringliche Motion") {
		t.Error("dringliche Motion must say 'Dringliche Motion'")
	}
	if strings.Contains(root, "📋") {
		t.Error("dringliche Motion must NOT use 📋 emoji")
	}

	for i, post := range thread {
		if graphemeLen(post.Text) > maxGraphemes {
			t.Errorf("post %d exceeds %d graphemes: %d", i, maxGraphemes, graphemeLen(post.Text))
		}
	}
}

// TestFormatGeschaeftThread_MotionNoMitunterzeichner verifies that nil
// AnzahlMitunterzeichnende does not produce "+0 Mitunterzeichnende".
func TestFormatGeschaeftThread_MotionNoMitunterzeichner(t *testing.T) {
	g := testfixtures.MotionNoMitunterzeichnende()
	thread := FormatGeschaeftThread(g, nil)

	root := thread[0].Text
	t.Logf("Root:\n%s", root)

	if strings.Contains(root, "Mitunterzeichnende") {
		t.Error("nil AnzahlMitunterzeichnende must NOT produce Mitunterzeichnende line")
	}

	// Submitter line should still be present
	if !strings.Contains(root, "GLP — Lisa Weber") {
		t.Errorf("root missing submitter line\nRoot:\n%s", root)
	}

	for i, post := range thread {
		if graphemeLen(post.Text) > maxGraphemes {
			t.Errorf("post %d exceeds %d graphemes: %d", i, maxGraphemes, graphemeLen(post.Text))
		}
	}
}

// TestFormatGeschaeftThread_PostulatSimple verifies the Postulat-specific content.
func TestFormatGeschaeftThread_PostulatSimple(t *testing.T) {
	g := testfixtures.PostulatSimple()
	thread := FormatGeschaeftThread(g, nil)

	if len(thread) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(thread))
	}

	root := thread[0].Text
	reply := thread[1].Text
	t.Logf("Root:\n%s\n\nReply:\n%s", root, reply)

	for _, want := range []string{
		"📝 Neue Postulat",
		"SP — Anna Graff",
		"👇 Was ist ein Postulat?",
	} {
		if !strings.Contains(root, want) {
			t.Errorf("root missing %q", want)
		}
	}

	// Reply must mention Postulat explanation
	if !strings.Contains(reply, "Postulat") {
		t.Errorf("reply missing Postulat explanation\nReply:\n%s", reply)
	}

	for i, post := range thread {
		if graphemeLen(post.Text) > maxGraphemes {
			t.Errorf("post %d exceeds %d graphemes: %d\n%s", i, maxGraphemes, graphemeLen(post.Text), post.Text)
		}
	}
}

// TestFormatGeschaeftThread_PostulatDringlich verifies dringliches Postulat.
func TestFormatGeschaeftThread_PostulatDringlich(t *testing.T) {
	g := testfixtures.PostulatDringlich()
	thread := FormatGeschaeftThread(g, nil)

	root := thread[0].Text
	if !strings.Contains(root, "🚨") {
		t.Error("dringliches Postulat must use 🚨 emoji")
	}
	if !strings.Contains(root, "Dringliche Postulat") {
		t.Error("dringliches Postulat must say 'Dringliche Postulat'")
	}

	for i, post := range thread {
		if graphemeLen(post.Text) > maxGraphemes {
			t.Errorf("post %d exceeds %d graphemes: %d", i, maxGraphemes, graphemeLen(post.Text))
		}
	}
}

// TestFormatGeschaeftThread_LongTitleTruncation verifies that long titles are
// truncated with "…" to keep the root post within 300 graphemes.
func TestFormatGeschaeftThread_LongTitleTruncation(t *testing.T) {
	g := testfixtures.MotionLongTitle()
	thread := FormatGeschaeftThread(g, nil)

	root := thread[0].Text
	t.Logf("Root:\n%s\nLength: %d graphemes", root, graphemeLen(root))

	if graphemeLen(root) > maxGraphemes {
		t.Errorf("root exceeds %d graphemes: %d", maxGraphemes, graphemeLen(root))
	}
	if !strings.Contains(root, "…") {
		t.Error("truncated root must contain '…'")
	}
	// Header must still be present
	if !strings.Contains(root, "📋 Neue Motion") {
		t.Error("root missing type header after truncation")
	}
}

// TestFormatGeschaeftThread_ReplyWithinLimit verifies the reply stays within 300 graphemes.
func TestFormatGeschaeftThread_ReplyWithinLimit(t *testing.T) {
	fixtures := testfixtures.AllFixtures()
	for name, g := range fixtures {
		t.Run(name, func(t *testing.T) {
			thread := FormatGeschaeftThread(g, nil)
			for i, post := range thread {
				if graphemeLen(post.Text) > maxGraphemes {
					t.Errorf("post %d exceeds %d graphemes: %d\n%s", i, maxGraphemes, graphemeLen(post.Text), post.Text)
				}
			}
		})
	}
}

// TestFormatGeschaeftThread_LinkFacetOnRootPost verifies the Geschaeft link facet.
func TestFormatGeschaeftThread_LinkFacetOnRootPost(t *testing.T) {
	g := testfixtures.MotionSimple()
	thread := FormatGeschaeftThread(g, nil)
	root := thread[0]

	if len(root.Facets) == 0 {
		t.Error("root post should have a link facet for the Geschaeft URL")
	}
}

// TestFormatGeschaeftThread_ExplanationLinkFacetOnReply verifies the reply link facet.
func TestFormatGeschaeftThread_ExplanationLinkFacetOnReply(t *testing.T) {
	g := testfixtures.PostulatSimple()
	thread := FormatGeschaeftThread(g, nil)
	reply := thread[1]

	if len(reply.Facets) == 0 {
		t.Error("reply post should have a link facet for the explanation URL")
	}
}

// TestFormatGeschaeftThread_DateFormatting verifies the date is formatted as DD.MM.YYYY.
func TestFormatGeschaeftThread_DateFormatting(t *testing.T) {
	g := testfixtures.MotionSimple()
	g.Beginn.Start = "2026-06-24 00:00:00"
	thread := FormatGeschaeftThread(g, nil)
	root := thread[0].Text

	if !strings.Contains(root, "24.06.2026") {
		t.Errorf("root should contain formatted date 24.06.2026, root:\n%s", root)
	}
}

// TestFormatGeschaeftThread_CustomVote tests that arbitrary in-scope types work.
func TestFormatGeschaeftThread_CustomVote(t *testing.T) {
	tests := []struct {
		name          string
		geschaeft     zurichapi.Geschaeft
		shouldContain []string
	}{
		{
			name: "Motion with zero co-signers pointer",
			geschaeft: zurichapi.Geschaeft{
				OBJGUID:       "guid-zero-co",
				GRNr:          "2026/300",
				Titel:         "Testtitel Motion",
				Geschaeftsart: "Motion",
				Beginn: struct {
					Start string `xml:"Start"`
					Text  string `xml:"Text"`
				}{Start: "2026-01-01 00:00:00"},
				Erstunterzeichner: struct {
					KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
				}{KontaktGremium: zurichapi.KontaktGremium{Name: "Max Muster", Partei: "FDP"}},
				AnzahlMitunterzeichnende: intPtr(0),
			},
			shouldContain: []string{"📋 Neue Motion", "FDP — Max Muster"},
		},
		{
			name: "Postulat with nil Partei is still rendered",
			geschaeft: zurichapi.Geschaeft{
				OBJGUID:       "guid-no-partei",
				GRNr:          "2026/301",
				Titel:         "Testtitel Postulat",
				Geschaeftsart: "Postulat",
				Beginn: struct {
					Start string `xml:"Start"`
					Text  string `xml:"Text"`
				}{Start: "2026-02-15 00:00:00"},
				Erstunterzeichner: struct {
					KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
				}{KontaktGremium: zurichapi.KontaktGremium{Name: "Eva Schmidt", Partei: ""}},
			},
			shouldContain: []string{"📝 Neue Postulat", "Eva Schmidt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			thread := FormatGeschaeftThread(tt.geschaeft, nil)
			full := allThreadText(thread)
			for _, want := range tt.shouldContain {
				if !strings.Contains(full, want) {
					t.Errorf("thread missing %q\nFull:\n%s", want, full)
				}
			}
			for i, post := range thread {
				if graphemeLen(post.Text) > maxGraphemes {
					t.Errorf("post %d exceeds %d graphemes: %d", i, maxGraphemes, graphemeLen(post.Text))
				}
			}
		})
	}
}
