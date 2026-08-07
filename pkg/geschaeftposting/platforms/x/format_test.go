package x

import (
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/geschaeftposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func intPtr(i int) *int { return &i }

// allThreadText joins all post texts for simple Contains checks.
func allThreadText(thread []*XGeschaeftPost) string {
	var parts []string
	for _, p := range thread {
		parts = append(parts, p.Text)
	}
	return strings.Join(parts, "\n\n")
}

// charCount returns the effective X character count for a post
// (URLs counted as URLLength). X counts Unicode code points (runes).
func charCount(text string, link string) int {
	return len([]rune(text)) - len([]rune(link)) + URLLength
}

// TestFormatGeschaeftThread_MotionSimple verifies basic Motion thread structure.
func TestFormatGeschaeftThread_MotionSimple(t *testing.T) {
	g := testfixtures.MotionSimple()
	thread := FormatGeschaeftThread(g, nil, DefaultMaxChars)

	if len(thread) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(thread))
	}

	root := thread[0].Text
	reply := thread[1].Text
	t.Logf("Root:\n%s\n\nReply:\n%s", root, reply)

	for _, want := range []string{
		"📋 Neue Motion",
		"Gemeinderat Zürich",
		"Überdeckung",
		"Grüne — Martin Busekros",
		"+4 Mitunterzeichnende",
		"📅 24.06.2026",
		"👇 Was ist eine Motion?",
		"geschaefte/detail.php",
	} {
		if !strings.Contains(root, want) {
			t.Errorf("root missing %q\nRoot:\n%s", want, root)
		}
	}

	for _, want := range []string{
		"ℹ️",
		"⏭️",
		"gemeinderat-zuerich.ch/gemeinderat/geschaeftsarten/",
	} {
		if !strings.Contains(reply, want) {
			t.Errorf("reply missing %q\nReply:\n%s", want, reply)
		}
	}
}

// TestFormatGeschaeftThread_MotionDringlich verifies 🚨 for dringliche types.
func TestFormatGeschaeftThread_MotionDringlich(t *testing.T) {
	g := testfixtures.MotionDringlich()
	thread := FormatGeschaeftThread(g, nil, DefaultMaxChars)
	root := thread[0].Text
	t.Logf("Root:\n%s", root)

	if !strings.Contains(root, "🚨") {
		t.Error("dringliche Motion must use 🚨 emoji")
	}
	if !strings.Contains(root, "Dringliche Motion") {
		t.Error("dringliche Motion must say 'Dringliche Motion'")
	}
}

// TestFormatGeschaeftThread_PostulatSimple verifies Postulat-specific content.
func TestFormatGeschaeftThread_PostulatSimple(t *testing.T) {
	g := testfixtures.PostulatSimple()
	thread := FormatGeschaeftThread(g, nil, DefaultMaxChars)
	root := thread[0].Text
	t.Logf("Root:\n%s", root)

	for _, want := range []string{
		"📝 Neue Postulat",
		"👇 Was ist ein Postulat?",
		"SP — Anna Graff",
	} {
		if !strings.Contains(root, want) {
			t.Errorf("root missing %q", want)
		}
	}

	reply := thread[1].Text
	if !strings.Contains(reply, "Postulat") {
		t.Errorf("reply missing Postulat text\nReply:\n%s", reply)
	}
}

// TestFormatGeschaeftThread_LongTitleTruncation verifies title truncation.
func TestFormatGeschaeftThread_LongTitleTruncation(t *testing.T) {
	g := testfixtures.MotionLongTitle()
	thread := FormatGeschaeftThread(g, nil, DefaultMaxChars)
	root := thread[0].Text
	t.Logf("Root (length %d):\n%s", len(root), root)

	link := "https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=" + g.OBJGUID
	effective := charCount(root, link)
	if effective > DefaultMaxChars {
		t.Errorf("root effective length exceeds %d: %d", DefaultMaxChars, effective)
	}
	if !strings.Contains(root, "…") {
		t.Error("truncated root must contain '…'")
	}
}

// TestFormatGeschaeftThread_PremiumCharLimit verifies that Premium (2000-char) posts
// do not truncate the same title that is truncated in free mode.
func TestFormatGeschaeftThread_PremiumCharLimit(t *testing.T) {
	g := testfixtures.MotionLongTitle()
	threadFree := FormatGeschaeftThread(g, nil, DefaultMaxChars)
	threadPremium := FormatGeschaeftThread(g, nil, 2000)

	rootFree := threadFree[0].Text
	rootPremium := threadPremium[0].Text

	if !strings.Contains(rootFree, "…") {
		t.Error("free root should be truncated")
	}
	if strings.Contains(rootPremium, "…") {
		t.Errorf("premium root should NOT be truncated\nPremium root:\n%s", rootPremium)
	}
	// Premium root must contain the full original title
	if !strings.Contains(rootPremium, "inklusive Ladeinfrastruktur") {
		t.Error("premium root should contain full title")
	}
}

// TestFormatGeschaeftThread_NoMitunterzeichner verifies nil AnzahlMitunterzeichnende.
func TestFormatGeschaeftThread_NoMitunterzeichner(t *testing.T) {
	g := testfixtures.MotionNoMitunterzeichnende()
	thread := FormatGeschaeftThread(g, nil, DefaultMaxChars)
	root := thread[0].Text

	if strings.Contains(root, "Mitunterzeichnende") {
		t.Error("nil AnzahlMitunterzeichnende must NOT produce Mitunterzeichnende text")
	}
	if !strings.Contains(root, "GLP — Lisa Weber") {
		t.Errorf("root missing submitter line\nRoot:\n%s", root)
	}
}

// TestFormatGeschaeftThread_AllFixturesWithinLimit checks all fixtures at free tier.
func TestFormatGeschaeftThread_AllFixturesWithinLimit(t *testing.T) {
	for _, name := range testfixtures.FixtureNames {
		g := testfixtures.AllFixtures()[name]
		t.Run(name, func(t *testing.T) {
			thread := FormatGeschaeftThread(g, nil, DefaultMaxChars)
			for i, post := range thread {
				link := "https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=" + g.OBJGUID
				effective := charCount(post.Text, link)
				// Allow a bit of leeway for the reply's explanation link (different URL)
				// The main concern is the root not exceeding charLimit
				if i == 0 && effective > DefaultMaxChars {
					t.Errorf("root post effective length exceeds %d: %d\n%s",
						DefaultMaxChars, effective, post.Text)
				}
			}
		})
	}
}

// TestFormatGeschaeftThread_ZeroMitunterzeichnende verifies pointer-to-zero is suppressed.
func TestFormatGeschaeftThread_ZeroMitunterzeichnende(t *testing.T) {
	g := zurichapi.Geschaeft{
		OBJGUID:       "guid-zero",
		GRNr:          "2026/400",
		Titel:         "Testtitel",
		Geschaeftsart: "Motion",
		Beginn: struct {
			Start string `xml:"Start"`
			Text  string `xml:"Text"`
		}{Start: "2026-01-01 00:00:00"},
		Erstunterzeichner: struct {
			KontaktGremium zurichapi.KontaktGremium `xml:"KontaktGremium"`
		}{KontaktGremium: zurichapi.KontaktGremium{Name: "Test Person", Partei: "FDP"}},
		AnzahlMitunterzeichnende: intPtr(0),
	}
	thread := FormatGeschaeftThread(g, nil, DefaultMaxChars)
	root := thread[0].Text

	if strings.Contains(root, "Mitunterzeichnende") {
		t.Error("zero AnzahlMitunterzeichnende must NOT produce Mitunterzeichnende text")
	}
}
