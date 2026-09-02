package zurichapi

import "testing"

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
		{Typ: "Instagram", Adresse: "www.instagram.com/someone/"},
		{Typ: "LinkedIn", Adresse: " "},
		{Typ: "Homepage", Adresse: "https://example.ch"},
	}}}

	got := publishedAccounts(k)

	if len(got) != 2 {
		t.Fatalf("got %+v, want the two accounts the mapping has columns for", got)
	}
	if got[0].Platform != "x" || got[0].URL != "https://x.com/someone" {
		t.Errorf("got %+v, want twitter.com normalised to x.com under platform x", got[0])
	}
	if got[1].URL != "https://www.instagram.com/someone/" {
		t.Errorf("got %q, want a scheme added — cmd/validate_contacts rejects bare URLs", got[1].URL)
	}
}
