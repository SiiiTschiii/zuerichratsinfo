package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
)

func names(cs []Contact) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

func TestBacklog_RotatesRatherThanRepeatingTheSameFew(t *testing.T) {
	cs := []Contact{
		{Name: "A Searched", Searched: "2026-10-11"},
		{Name: "B Never"},
		{Name: "C Older", Searched: "2026-01-05"},
		{Name: "D Never"},
	}

	got := names(backlog(cs, nil, 3))
	want := []string{"B Never", "D Never", "C Older"}
	if !slices.Equal(got, want) {
		t.Errorf("backlog = %v, want %v (never searched first, then least recently)", got, want)
	}
}

func TestBacklog_OnlySittingPeopleWithNoAccount(t *testing.T) {
	cs := []Contact{
		{Name: "Anna Bare"},
		{Name: "Bern Hat", X: contacts.VerifiedAccounts("https://x.com/b")},
		{Name: "Carl Candidate", Instagram: []Account{{URL: "https://www.instagram.com/c/", Confidence: "low"}}},
		{Name: "Dora Weg"},
		{Name: "SP Stadt Zürich", Kind: contacts.KindOrganization},
	}

	got := names(backlog(cs, []string{"Bare Anna", "Hat Bern", "Candidate Carl"}, 10))
	if want := []string{"Anna Bare"}; !slices.Equal(got, want) {
		t.Errorf("backlog = %v, want %v: not someone with an account or a candidate, not someone who has left, not a party", got, want)
	}
}

func TestBacklog_WithoutARosterEveryoneCounts(t *testing.T) {
	got := names(backlog([]Contact{{Name: "Anna Bare"}, {Name: "Dora Weg"}}, nil, 10))
	if len(got) != 2 {
		t.Errorf("backlog = %v, want both when the source cannot say who sits", got)
	}
}

func TestMarkSearched_RefusesWhatItCannotPlace(t *testing.T) {
	existing := map[string]*Contact{
		contacts.NameKey("Anna Aktiv"):      {Name: "Anna Aktiv"},
		contacts.NameKey("SP Stadt Zürich"): {Name: "SP Stadt Zürich", Kind: contacts.KindOrganization},
	}

	for name, args := range map[string]struct {
		names []string
		date  string
	}{
		"a person not on file": {[]string{"Wer Anders"}, "2026-10-11"},
		"a party account":      {[]string{"SP Stadt Zürich"}, "2026-10-11"},
		"a bad date":           {[]string{"Anna Aktiv"}, "11.10.2026"},
		"no names":             {nil, "2026-10-11"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := markSearched(existing, args.names, args.date); err == nil {
				t.Errorf("want an error")
			}
		})
	}
	if existing[contacts.NameKey("Anna Aktiv")].Searched != "" {
		t.Errorf("a refused call still marked someone")
	}
}

func TestMarkSearched_IsKeptThroughARewriteAndMovesThePersonDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contacts.yaml")
	if err := save(path, "", []Contact{{Name: "Anna Aktiv"}, {Name: "Bern Bare"}}); err != nil {
		t.Fatal(err)
	}

	existing, header, err := loadExisting(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := runMarkSearched(path, header, existing, []string{"Bare Bern"}, "2026-10-11", false); err != nil {
		t.Fatal(err)
	}

	written, _ := os.ReadFile(path)
	if !strings.Contains(string(written), "  - name: Bern Bare\n    searched: \"2026-10-11\"\n") && !strings.Contains(string(written), "  - name: Bern Bare\n    searched: 2026-10-11\n") {
		t.Fatalf("searched was not written beside the name:\n%s", written)
	}

	reloaded, _, err := loadExisting(path)
	if err != nil {
		t.Fatal(err)
	}
	cs, _, _ := merge(reloaded, nil)
	if got := names(backlog(cs, nil, 1)); !slices.Equal(got, []string{"Anna Aktiv"}) {
		t.Errorf("next up = %v, want the person not yet searched", got)
	}
}
