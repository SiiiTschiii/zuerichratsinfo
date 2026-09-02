package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

func TestMerge_AddsNewMember(t *testing.T) {
	existing := map[string]*Contact{}

	result, added, accounts := merge(existing, []votes.Member{{
		Name:     "Test Person",
		Party:    "SVP",
		Accounts: []votes.Account{{Platform: "x", URL: "https://x.com/testperson"}},
	}})

	if len(result) != 1 || result[0].Name != "Test Person" {
		t.Fatalf("got %+v, want one contact named Test Person", result)
	}
	if added != 1 {
		t.Errorf("added = %d, want 1", added)
	}
	if accounts != 1 {
		t.Errorf("accounts = %d, want 1", accounts)
	}
	if len(result[0].X) != 1 || result[0].X[0] != "https://x.com/testperson" {
		t.Errorf("X = %v, want the published handle", result[0].X)
	}
}

// A roster with no accounts is the canton's case: the names seed the file and
// the handles are curated by hand afterwards.
func TestMerge_AddsNameOnlyMember(t *testing.T) {
	result, added, accounts := merge(map[string]*Contact{}, []votes.Member{
		{Name: "Nameless Only", Party: "SP", Fraktion: "SP"},
	})

	if added != 1 || accounts != 0 {
		t.Fatalf("added = %d, accounts = %d, want 1 and 0", added, accounts)
	}
	if len(result) != 1 || len(result[0].X) != 0 {
		t.Errorf("got %+v, want a name-only contact", result)
	}
}

func TestMerge_AddsNewPlatformToExisting(t *testing.T) {
	existing := map[string]*Contact{
		"Test Person": {Name: "Test Person", X: []string{"https://x.com/testperson"}},
	}

	result, added, accounts := merge(existing, []votes.Member{{
		Name: "Test Person",
		Accounts: []votes.Account{
			{Platform: "x", URL: "https://x.com/testperson"},
			{Platform: "instagram", URL: "https://www.instagram.com/testperson"},
		},
	}})

	if added != 0 {
		t.Errorf("added = %d, want 0 — the contact already existed", added)
	}
	if accounts != 1 {
		t.Errorf("accounts = %d, want 1 — only Instagram is new", accounts)
	}
	if len(result[0].Instagram) != 1 {
		t.Errorf("Instagram = %v, want the new account", result[0].Instagram)
	}
	if len(result[0].X) != 1 {
		t.Errorf("X = %v, want the existing handle kept once", result[0].X)
	}
}

// A curated handle is never replaced: both are kept and a human decides.
func TestMerge_KeepsCuratedHandleBesideDifferentPublishedOne(t *testing.T) {
	existing := map[string]*Contact{
		"Test Person": {Name: "Test Person", X: []string{"https://x.com/curated"}},
	}

	result, _, accounts := merge(existing, []votes.Member{{
		Name:     "Test Person",
		Accounts: []votes.Account{{Platform: "x", URL: "https://x.com/published"}},
	}})

	if accounts != 1 {
		t.Errorf("accounts = %d, want 1", accounts)
	}
	if len(result[0].X) != 2 || result[0].X[0] != "https://x.com/curated" {
		t.Errorf("X = %v, want the curated handle first and both kept", result[0].X)
	}
}

// Nobody is dropped because the roster no longer lists them: removing a former
// member is a deliberate, human edit.
func TestMerge_NeverRemovesCuratedContact(t *testing.T) {
	existing := map[string]*Contact{
		"Departed Member": {Name: "Departed Member", X: []string{"https://x.com/departed"}},
	}

	result, _, _ := merge(existing, []votes.Member{{Name: "Sitting Member"}})

	names := make([]string, len(result))
	for i, c := range result {
		names[i] = c.Name
	}
	if len(result) != 2 {
		t.Fatalf("got %v, want both the departed and the sitting member", names)
	}
}

func TestMerge_SortsCaseInsensitively(t *testing.T) {
	result, _, _ := merge(map[string]*Contact{}, []votes.Member{
		{Name: "Adina Rom"},
		{Name: "AL Stadt Zürich"},
		{Name: "Alana Gerdes"},
	})

	want := []string{"Adina Rom", "AL Stadt Zürich", "Alana Gerdes"}
	for i, name := range want {
		if result[i].Name != name {
			t.Errorf("contact %d = %q, want %q", i, result[i].Name, name)
		}
	}
}

// The written file has to come back through the loader unchanged, in the same
// shape cmd/validate_contacts expects: two-space indent, platforms in
// alphabetical order, header preserved.
func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contacts.yaml")
	header := "# A header explaining this file.\n#\n# Second line.\n"

	err := save(path, header, []Contact{{
		Name:      "Test Person",
		X:         []string{"https://x.com/testperson"},
		Bluesky:   []string{"https://bsky.app/profile/testperson.bsky.social"},
		Instagram: []string{"https://www.instagram.com/testperson/"},
	}})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	got := string(written)

	if !strings.HasPrefix(got, header) {
		t.Errorf("header was not preserved:\n%s", got)
	}
	if !strings.Contains(got, "\n  - name: Test Person\n") {
		t.Errorf("want two-space indented contacts, got:\n%s", got)
	}
	if !strings.Contains(got, "\n    x:\n      - https://x.com/testperson\n") {
		t.Errorf("want block-style platform lists, got:\n%s", got)
	}
	if bsky, insta := strings.Index(got, "bluesky:"), strings.Index(got, "instagram:"); bsky > insta {
		t.Errorf("platforms are not in alphabetical order:\n%s", got)
	}

	reloaded, gotHeader, err := loadExisting(path)
	if err != nil {
		t.Fatalf("loadExisting: %v", err)
	}
	if gotHeader != header {
		t.Errorf("header = %q, want %q", gotHeader, header)
	}
	c, ok := reloaded["Test Person"]
	if !ok {
		t.Fatalf("contact did not survive the round trip: %v", reloaded)
	}
	if len(c.X) != 1 || len(c.Bluesky) != 1 || len(c.Instagram) != 1 {
		t.Errorf("accounts did not survive the round trip: %+v", c)
	}
}

func TestLoadExisting_MissingFileIsNotAnError(t *testing.T) {
	existing, header, err := loadExisting(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatalf("loadExisting: %v", err)
	}
	if len(existing) != 0 || header != "" {
		t.Errorf("got %v / %q, want an empty start", existing, header)
	}
}

func TestAffiliation(t *testing.T) {
	tests := []struct {
		member votes.Member
		want   string
	}{
		{votes.Member{Party: "SVP", Fraktion: "SVP"}, " (SVP)"},
		{votes.Member{Party: "EDU", Fraktion: "SVP"}, " (EDU, Fraktion SVP)"},
		{votes.Member{Party: "SP"}, " (SP)"},
		{votes.Member{Fraktion: "AL"}, " (Fraktion AL)"},
		{votes.Member{}, ""},
	}

	for _, tt := range tests {
		if got := affiliation(tt.member); got != tt.want {
			t.Errorf("affiliation(%+v) = %q, want %q", tt.member, got, tt.want)
		}
	}
}
