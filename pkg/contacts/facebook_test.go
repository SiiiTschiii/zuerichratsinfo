package contacts

import "testing"

func TestExtractFacebookMentionFromURL(t *testing.T) {
	for raw, want := range map[string]string{
		"https://www.facebook.com/profile.php?id=100070693802425":  "@[100070693802425]",
		"https://facebook.com/profile.php?id=100070693802425&sk=x": "@[100070693802425]",
		"https://www.facebook.com/1063164230212573":                "@[1063164230212573]",
		"https://m.facebook.com/1063164230212573/":                 "@[1063164230212573]",
		"https://www.facebook.com/people/Anna-Graff/1000706/":      "@[1000706]",
		"https://www.facebook.com/pages/FDP-Zuerich/123456789":     "@[123456789]",
		"facebook.com/profile.php?id=42":                           "@[42]",

		// Vanity names carry no ID: they stay untagged until one is found.
		"https://www.facebook.com/fdpstadtzuerich/": "",
		"https://www.facebook.com/alice.kohli":      "",
		"https://www.facebook.com/profile.php":      "",
		"https://www.facebook.com/profile.php?id=x": "",
		"https://www.facebook.com/groups/123/posts": "",
		// Not Facebook, however numeric.
		"https://www.instagram.com/123456789": "",
		"":                                    "",
	} {
		if got := ExtractFacebookMentionFromURL(raw); got != want {
			t.Errorf("ExtractFacebookMentionFromURL(%q) = %q, want %q", raw, got, want)
		}
	}
}

func facebookContact(name, url string, verified bool) Contact {
	return Contact{Name: name, Facebook: []Account{{URL: url, Verified: verified}}}
}

func TestTagFacebookHandlesInText(t *testing.T) {
	m := NewMapperForTest([]Contact{
		facebookContact("Anna Graff", "https://www.facebook.com/profile.php?id=111", true),
		facebookContact("Reto Brüesch", "https://www.facebook.com/reto.brueesch", true),     // vanity: no ID
		facebookContact("Liv Mahrer", "https://www.facebook.com/profile.php?id=333", false), // not verified
		facebookContact("Moritz Bögli", "https://www.facebook.com/profile.php?id=444", true),
	})

	// A vanity URL listed first must not hide the account that has an ID.
	second := NewMapperForTest([]Contact{{Name: "Serap Kahriman", Facebook: []Account{
		{URL: "https://www.facebook.com/serap.kahriman", Verified: true},
		{URL: "https://www.facebook.com/profile.php?id=555", Verified: true},
	}}})
	if got, want := second.TagFacebookHandlesInText("von Serap Kahriman"), "von @[555]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	got := m.TagFacebookHandlesInText("Postulat von Anna Graff (SP), Reto Brüesch (SVP) und Liv Mahrer (GLP)")
	// The mention takes the name's place: it renders the account's own name,
	// so appending it would print the name twice.
	want := "Postulat von @[111] (SP), Reto Brüesch (SVP) und Liv Mahrer (GLP)"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	// The name in either order, as elsewhere.
	if got, want := m.TagFacebookHandlesInText("von Bögli Moritz"), "von @[444]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got := m.TagFacebookHandlesInText("nichts zu markieren"); got != "nichts zu markieren" {
		t.Errorf("untouched text changed: %q", got)
	}
}

func TestTagFacebookFraktionen(t *testing.T) {
	m := NewMapperForTest([]Contact{
		{Name: "SP Stadt Zürich", Kind: "organization", Facebook: []Account{{URL: "https://www.facebook.com/profile.php?id=900", Verified: true}}},
		{Name: "AL Stadt Zürich", Kind: "organization", Facebook: []Account{{URL: "https://www.facebook.com/ALzuerich/", Verified: true}}}, // vanity
	})

	got := m.TagFacebookFraktionen("Motion der SP- und AL-Fraktion", "zurich-city")
	// Only the party whose Page has an ID is mentioned.
	if want := "Motion der SP- und AL-Fraktion @[900]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
