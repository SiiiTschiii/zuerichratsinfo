package zurichapi

import (
	"testing"
	"time"
)

func intPtr(i int) *int { return &i }

func TestToVote_MapsAllFields(t *testing.T) {
	a := Abstimmung{
		OBJGUID:          "vote-guid",
		SEQ:              "42",
		SitzungGuid:      "sitzung-guid",
		SitzungDatum:     "2025-06-15 10:30:00",
		TraktandumGuid:   "trakt-guid",
		TraktandumTitel:  "Postulat betreffend Veloinfrastruktur",
		GeschaeftGuid:    "geschaeft-guid",
		GeschaeftTitel:   "Veloinfrastruktur Langstrasse",
		GeschaeftGrNr:    "2025/100",
		Abstimmungstitel: "Schlussabstimmung",
		Abstimmungstyp:   "Offen",
		AnzahlJa:         intPtr(90),
		AnzahlNein:       intPtr(30),
		AnzahlEnthaltung: intPtr(0),
		AnzahlAbwesend:   intPtr(5),
		Schlussresultat:  "angenommen",
	}
	a.Stimmabgaben.Stimmabgabe = []Stimmabgabe{
		{Vorname: "Anna", Name: "Graff", Partei: "SP", Fraktion: "SP", Abstimmungsverhalten: "Ja"},
	}

	v := ToVote(a)

	if v.SourceID != "vote-guid" {
		t.Errorf("SourceID = %q, want %q", v.SourceID, "vote-guid")
	}
	if v.Jurisdiction != JurisdictionKey {
		t.Errorf("Jurisdiction = %q, want %q", v.Jurisdiction, JurisdictionKey)
	}
	if v.SessionID != "sitzung-guid" || v.Sequence != "42" {
		t.Errorf("SessionID/Sequence = %q/%q", v.SessionID, v.Sequence)
	}
	if want := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC); !v.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", v.Date, want)
	}
	if v.Title != "Postulat betreffend Veloinfrastruktur" {
		t.Errorf("Title = %q", v.Title)
	}
	if v.Subtitle != "Schlussabstimmung" {
		t.Errorf("Subtitle = %q", v.Subtitle)
	}
	if v.Type != "Offen" {
		t.Errorf("Type = %q", v.Type)
	}
	if v.Decision != "angenommen" {
		t.Errorf("Decision = %q", v.Decision)
	}
	if *v.Yes != 90 || *v.No != 30 || *v.Abstention != 0 || *v.Absent != 5 {
		t.Errorf("totals = %d/%d/%d/%d", *v.Yes, *v.No, *v.Abstention, *v.Absent)
	}
	if v.Affair.Number != "2025/100" || v.Affair.ID != "geschaeft-guid" || v.Affair.Title != "Veloinfrastruktur Langstrasse" {
		t.Errorf("Affair = %+v", v.Affair)
	}
	if len(v.MemberVotes) != 1 {
		t.Fatalf("MemberVotes = %d, want 1", len(v.MemberVotes))
	}
	if got := v.MemberVotes[0]; got.Name != "Anna Graff" || got.Fraktion != "SP" || got.Choice != "Ja" {
		t.Errorf("MemberVotes[0] = %+v", got)
	}
}

// The three link variants used to be selected inside each platform formatter.
// They now come off the mapper, so the selection rules are pinned here.
func TestToVote_LinkSelection(t *testing.T) {
	base := Abstimmung{
		OBJGUID:        "vote-guid",
		SitzungGuid:    "sitzung-guid",
		TraktandumGuid: "trakt-guid",
		GeschaeftGuid:  "geschaeft-guid",
		GeschaeftTitel: "Ein richtiger Titel",
	}

	t.Run("normal traktandum title", func(t *testing.T) {
		a := base
		a.TraktandumTitel = "Postulat betreffend Veloinfrastruktur"
		v := ToVote(a)
		if want := "https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=vote-guid"; v.SourceURL != want {
			t.Errorf("SourceURL = %q, want %q", v.SourceURL, want)
		}
		if want := "https://www.gemeinderat-zuerich.ch/sitzungen/sitzung/?gid=sitzung-guid#trakt-guid"; v.GroupURL != want {
			t.Errorf("GroupURL = %q, want %q", v.GroupURL, want)
		}
	})

	// "Antrag 1." says nothing about what was voted on, so both links point at
	// the Geschäft page instead of an agenda item nobody can interpret.
	t.Run("generic antrag title falls back to the Geschäft page", func(t *testing.T) {
		a := base
		a.TraktandumTitel = "Antrag 1."
		v := ToVote(a)
		want := "https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=geschaeft-guid"
		if v.SourceURL != want || v.GroupURL != want {
			t.Errorf("SourceURL/GroupURL = %q/%q, want both %q", v.SourceURL, v.GroupURL, want)
		}
		if v.Title != "Ein richtiger Titel" {
			t.Errorf("Title = %q, want the Geschäft title", v.Title)
		}
	})
}

func TestParseSitzungDatum(t *testing.T) {
	tests := []struct {
		in   string
		want string // YYYY-MM-DD, or "" for the zero time
	}{
		{"2025-06-15 10:30:00", "2025-06-15"},
		{"2025-06-15", "2025-06-15"},
		{"2025-06-15T10:30:00", "2025-06-15"},
		{"", ""},
		{"not a date", ""},
	}
	for _, tc := range tests {
		got := parseSitzungDatum(tc.in)
		var gotStr string
		if !got.IsZero() {
			gotStr = got.Format("2006-01-02")
		}
		if gotStr != tc.want {
			t.Errorf("parseSitzungDatum(%q) = %q, want %q", tc.in, gotStr, tc.want)
		}
	}
}
