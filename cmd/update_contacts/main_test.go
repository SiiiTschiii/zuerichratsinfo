package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
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
	if len(result[0].X) != 1 || result[0].X[0].URL != "https://x.com/testperson" {
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
		contacts.NameKey("Test Person"): {Name: "Test Person", X: contacts.VerifiedAccounts("https://x.com/testperson")},
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
		contacts.NameKey("Test Person"): {Name: "Test Person", X: contacts.VerifiedAccounts("https://x.com/curated")},
	}

	result, _, accounts := merge(existing, []votes.Member{{
		Name:     "Test Person",
		Accounts: []votes.Account{{Platform: "x", URL: "https://x.com/published"}},
	}})

	if accounts != 1 {
		t.Errorf("accounts = %d, want 1", accounts)
	}
	if len(result[0].X) != 2 || result[0].X[0].URL != "https://x.com/curated" {
		t.Errorf("X = %v, want the curated handle first and both kept", result[0].X)
	}
}

// Nobody is dropped because the roster no longer lists them: removing a former
// member is a deliberate, human edit.
func TestMerge_NeverRemovesCuratedContact(t *testing.T) {
	existing := map[string]*Contact{
		contacts.NameKey("Departed Member"): {Name: "Departed Member", X: contacts.VerifiedAccounts("https://x.com/departed")},
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
		X:         contacts.VerifiedAccounts("https://x.com/testperson"),
		Bluesky:   contacts.VerifiedAccounts("https://bsky.app/profile/testperson.bsky.social"),
		Instagram: contacts.VerifiedAccounts("https://www.instagram.com/testperson/"),
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
	if !strings.Contains(got, "\n    x:\n      - url: https://x.com/testperson\n        verified: true\n") {
		t.Errorf("want every account to state whether it may be published, got:\n%s", got)
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
	c, ok := reloaded[contacts.NameKey("Test Person")]
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

// PARIS writes "Bögli Moritz" where the curated file says "Moritz Bögli". Keyed
// on the string, the roster added a second entry for eight sitting Gemeinderäte
// — and nothing downstream would have complained, because two entries with
// different names are not duplicates to a checker that compares strings.
func TestMerge_MatchesAPersonWhoseNamePartsAreOrderedDifferently(t *testing.T) {
	existing := map[string]*Contact{
		contacts.NameKey("Bögli Moritz"): {
			Name: "Bögli Moritz",
			X:    contacts.VerifiedAccounts("https://x.com/MoritzBoegli"),
		},
	}

	result, added, accounts := merge(existing, []votes.Member{{
		Name:     "Moritz Bögli",
		Party:    "AL",
		Accounts: []votes.Account{{Platform: "instagram", URL: "https://www.instagram.com/moritzboegli/"}},
	}})

	if added != 0 {
		t.Errorf("added = %d, want 0 — this is the same person", added)
	}
	if len(result) != 1 {
		t.Fatalf("got %d contacts, want the roster folded into the existing one: %+v", len(result), result)
	}
	if accounts != 1 || len(result[0].Instagram) != 1 {
		t.Errorf("the published account did not reach the existing contact: %+v", result[0])
	}
	if result[0].Name != "Bögli Moritz" {
		t.Errorf("Name = %q, want the curated spelling left alone", result[0].Name)
	}
}

// PARIS serves the same account under several spellings, and the curated file
// already holds some of them. Compared as strings, each refresh recorded
// another copy of an account that was already there.
func TestAddAccounts_RecognisesAnAccountAlreadyOnFile(t *testing.T) {
	c := &Contact{
		Name:      "Përparim Avdili",
		Instagram: contacts.VerifiedAccounts("https://www.instagram.com/perparim.avdili/?hl=de"),
	}

	added := addAccounts(c, []votes.Account{
		{Platform: "instagram", URL: "https://www.instagram.com/perparim.avdili/"},
		{Platform: "instagram", URL: "https://www.instagram.com/perparim.avdili/?igsh=abc123"},
	})

	if added != 0 {
		t.Errorf("added = %d, want 0 — both are the account already on file", added)
	}
	if len(c.Instagram) != 1 {
		t.Errorf("Instagram = %v, want the one entry left alone", c.Instagram)
	}
}

func TestAddAccounts_StoresNewAccountsWithoutTracking(t *testing.T) {
	c := &Contact{Name: "Alex Guggenheim"}

	added := addAccounts(c, []votes.Account{
		{Platform: "instagram", URL: "https://www.instagram.com/alex.guggenheim?igsh=d2pvbmpjNzV5Z2lp&utm_source=qr"},
	})

	if added != 1 {
		t.Fatalf("added = %d, want 1", added)
	}
	if c.Instagram[0].URL != "https://www.instagram.com/alex.guggenheim" {
		t.Errorf("stored %q, want the share token dropped", c.Instagram[0].URL)
	}
}

// A Facebook profile is its query string, so the tidying must not touch it.
func TestStripTracking_KeepsAQueryThatIdentifiesTheAccount(t *testing.T) {
	const url = "https://www.facebook.com/profile.php?id=100070693802425"
	if got := stripTracking(url); got != url {
		t.Errorf("stripTracking(%q) = %q, want it untouched", url, got)
	}
}

// Every way a URL varies without pointing somewhere else. Each of these,
// compared literally, added a second copy of an account already on file.
func TestAccountKey_SameAccountDifferentSpellings(t *testing.T) {
	same := [][2]string{
		{"https://www.facebook.com/attila.kipfer", "https://facebook.com/attila.kipfer"},
		{"https://www.linkedin.com/in/alana-gerdes", "https://www.linkedin.com/in/alana-gerdes/"},
		{"https://www.linkedin.com/in/attila-kipfer", "https://ch.linkedin.com/in/attila-kipfer"},
		{"https://www.instagram.com/perparim.avdili/?hl=de", "https://www.instagram.com/perparim.avdili/"},
		{"https://www.instagram.com/alex.guggenheim?igsh=abc&utm_source=qr", "https://instagram.com/alex.guggenheim"},
		{"https://x.com/MoritzBoegli", "https://x.com/moritzboegli"},
		// Bluesky's CDN host: the same profile, and PARIS publishes both.
		{"https://bsky.app/profile/michaamstad.bsky.social", "https://web-cdn.bsky.app/profile/michaamstad.bsky.social"},
	}
	for _, pair := range same {
		if accountKey(pair[0]) != accountKey(pair[1]) {
			t.Errorf("accountKey(%q) = %q\naccountKey(%q) = %q\nwant them equal",
				pair[0], accountKey(pair[0]), pair[1], accountKey(pair[1]))
		}
	}

	different := [][2]string{
		{"https://x.com/someone", "https://x.com/someone_else"},
		// Two Facebook profiles differ only in the id their query carries.
		{"https://www.facebook.com/profile.php?id=100070693802425", "https://www.facebook.com/profile.php?id=100063702649427"},
		{"https://www.instagram.com/annagraff_", "https://x.com/annagraff_"},
	}
	for _, pair := range different {
		if accountKey(pair[0]) == accountKey(pair[1]) {
			t.Errorf("accountKey collapsed two different accounts: %q and %q", pair[0], pair[1])
		}
	}
}

// Accounts the parliament publishes for its own members arrive confirmed: that
// is a stronger claim than any search result, and it is what the file has
// always recorded for them.
func TestAddAccounts_PublishedAccountsArriveVerified(t *testing.T) {
	c := &Contact{Name: "Test Person"}

	addAccounts(c, []votes.Account{{Platform: "x", URL: "https://x.com/testperson"}})

	if len(c.X) != 1 || !c.X[0].Verified {
		t.Errorf("got %+v, want the published account marked verified", c.X)
	}
	if c.X[0].Confidence != "" {
		t.Errorf("confidence = %q, want none on a published account", c.X[0].Confidence)
	}
}

// A candidate already on file is exactly what the parliament publishing the
// same account confirms. Skipping it would let a harvested guess permanently
// shadow the authoritative record.
func TestAddAccounts_PromotesACandidateTheSourcePublishes(t *testing.T) {
	c := &Contact{
		Name: "Test Person",
		X: []Account{{
			URL:        "https://x.com/testperson",
			Verified:   false,
			Confidence: "high",
		}},
	}

	added := addAccounts(c, []votes.Account{
		{Platform: "x", URL: "https://x.com/testperson"},
	})

	if added != 1 {
		t.Errorf("added = %d, want the promotion counted as a change", added)
	}
	if len(c.X) != 1 {
		t.Fatalf("X = %+v, want the existing entry promoted, not duplicated", c.X)
	}
	if !c.X[0].Verified {
		t.Error("the candidate was not promoted by the published record")
	}
	if c.X[0].Confidence != "" {
		t.Errorf("confidence = %q, want it dropped once the account is confirmed", c.X[0].Confidence)
	}
}

// Promotion also has to see through the spellings accountKey already handles.
func TestAddAccounts_PromotesAcrossURLSpellings(t *testing.T) {
	c := &Contact{
		Name:    "Test Person",
		Bluesky: []Account{{URL: "https://web-cdn.bsky.app/profile/tp.bsky.social", Confidence: "low"}},
	}

	addAccounts(c, []votes.Account{
		{Platform: "bluesky", URL: "https://bsky.app/profile/tp.bsky.social"},
	})

	if len(c.Bluesky) != 1 || !c.Bluesky[0].Verified {
		t.Errorf("Bluesky = %+v, want the CDN-host candidate promoted in place", c.Bluesky)
	}
}

// An account already confirmed needs no change and must not be recounted.
func TestAddAccounts_LeavesAVerifiedAccountAlone(t *testing.T) {
	c := &Contact{
		Name: "Test Person",
		X:    contacts.VerifiedAccounts("https://x.com/testperson"),
	}

	added := addAccounts(c, []votes.Account{{Platform: "x", URL: "https://x.com/testperson"}})

	if added != 0 {
		t.Errorf("added = %d, want nothing counted for an account already confirmed", added)
	}
	if len(c.X) != 1 {
		t.Errorf("X = %+v, want no duplicate", c.X)
	}
}
