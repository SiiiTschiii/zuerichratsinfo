package zurichapi

import (
	"slices"
	"testing"

	"github.com/siiitschiii/parlinfo/pkg/votes"
)

func TestMemberName_GivenNameFirst(t *testing.T) {
	tests := []struct {
		name string
		k    Kontakt
		want string
	}{
		{
			// The combined field joins with a non-breaking space and puts the
			// surname first; neither is what the mapping or a post uses.
			name: "separate fields win over the combined one",
			k:    Kontakt{NameVorname: "Weyermann Karin", Name: "Weyermann", Vorname: "Karin"},
			want: "Karin Weyermann",
		},
		{
			name: "falls back to the combined field, with the space normalised",
			k:    Kontakt{NameVorname: "Weyermann Karin"},
			want: "Weyermann Karin",
		},
		{
			name: "a single name part is used on its own",
			k:    Kontakt{Name: "Weyermann"},
			want: "Weyermann",
		},
		{
			name: "nothing to go on",
			k:    Kontakt{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := memberName(tt.k); got != tt.want {
				t.Errorf("memberName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublishedAccounts(t *testing.T) {
	k := Kontakt{SozialeMedien: SozialeMedienList{Kommunikation: []SozialesMedium{
		{Typ: "Twitter", Adresse: "https://twitter.com/someone"},
		{Typ: "Instagram", Adresse: "www.instagram.com/someone/?hl=de"},
		{Typ: "Bluesky", Adresse: "https://web-cdn.bsky.app/profile/someone.bsky.social"},
		{Typ: "LinkedIn", Adresse: " "},
		{Typ: "Homepage", Adresse: "https://example.ch"},
	}}}

	got := publishedAccounts(k)

	if len(got) != 3 {
		t.Fatalf("got %+v, want the three accounts the mapping has columns for", got)
	}
	if got[0].Platform != "x" || got[0].URL != "https://x.com/someone" {
		t.Errorf("got %+v, want twitter.com normalised to x.com under platform x", got[0])
	}
	// Reported as published, tracking parameters and all: what an account URL
	// should look like in the curated file is cmd/update_contacts' decision,
	// not this adapter's.
	if got[1].URL != "https://www.instagram.com/someone/?hl=de" {
		t.Errorf("got %q, want a scheme added — cmd/validate_contacts rejects bare URLs", got[1].URL)
	}
	// The CDN host and the canonical one are the same profile; storing both
	// records one account twice.
	if got[2].URL != "https://bsky.app/profile/someone.bsky.social" {
		t.Errorf("got %q, want web-cdn.bsky.app canonicalised to bsky.app", got[2].URL)
	}
}

func TestSittingMembers_SeedsEveryoneWhoSitsAndNoOneWhoDoesNot(t *testing.T) {
	withAccount := func(vorname, name string) Kontakt {
		return Kontakt{
			Vorname: vorname, Name: name, Partei: "Grüne",
			SozialeMedien: SozialeMedienList{Kommunikation: []SozialesMedium{
				{Typ: "Instagram", Adresse: "https://www.instagram.com/x/"},
			}},
		}
	}

	kontakte := []Kontakt{
		withAccount("Véronique", "Wavre"),                 // sits, publishes
		withAccount("Anna", "Archiviert"),                 // archived: holds no mandate
		{Vorname: "Stefan", Name: "Stumm", Partei: "SVP"}, // sits, publishes nothing
		withAccount("Moritz", "Bögli"),                    // sits; the register spells it surname-first
	}
	mandates := []Behoerdenmandat{
		{Vorname: "Véronique", Name: "Wavre"},
		{Vorname: "Stefan", Name: "Stumm"},
		{Vorname: "Bögli", Name: "Moritz"},
		{Vorname: "Neu", Name: "Gewählt", Partei: "SP"}, // sits, not in the archive at all
		{Vorname: "Véronique", Name: "Wavre"},           // a second mandate is still one person
	}

	members := sittingMembers(kontakte, mandates)

	var names []string
	got := map[string]votes.Member{}
	for _, m := range members {
		names = append(names, m.Name)
		got[m.Name] = m
	}

	// The slice, not the map: a map would hide a person returned twice.
	want := []string{"Moritz Bögli", "Neu Gewählt", "Stefan Stumm", "Véronique Wavre"}
	if !slices.Equal(names, want) {
		t.Fatalf("members = %v, want %v (one per person, sorted by name)", names, want)
	}
	if _, ok := got["Anna Archiviert"]; ok {
		t.Errorf("an archived contact with no mandate was returned")
	}
	if len(got["Véronique Wavre"].Accounts) != 1 {
		t.Errorf("Wavre's published account was lost: %+v", got["Véronique Wavre"])
	}
	if m := got["Moritz Bögli"]; len(m.Accounts) != 1 || m.Party != "Grüne" {
		t.Errorf("a surname-first mandate did not join its contact: %+v", m)
	}
	if m, ok := got["Stefan Stumm"]; !ok || len(m.Accounts) != 0 || m.Party != "SVP" {
		t.Errorf("a member with no account should arrive bare, with the contact's party: %+v", m)
	}
	if m, ok := got["Neu Gewählt"]; !ok || m.Party != "SP" {
		t.Errorf("a member missing from the archive should still be seeded: %+v", m)
	}
}
