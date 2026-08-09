package x

import "testing"

func TestWeightedLen(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"ascii", "Ja", 2},
		// Umlauts are inside the weight-1 range, so German text costs its rune
		// count — not its byte count, which is what the packer used to charge.
		{"umlaut", "Grünliberale", 12},
		{"emoji", "📊", 2},
		// A variation selector is a code point of its own and X charges for it.
		{"emoji with variation selector", "🏛️", 4},
		{"url replaces its own length", "https://zh.recapp.ch/shareparl?agendaItemUid=82166c96-87f8-4fdb-8fd7-20af55278ec4&segmentUid=e815525c-45ef-475e-a2db-3f644e0f0c0b", urlWeightedLen},
		{"short url costs the same", "https://a.ch", urlWeightedLen},
		{"url after text", "🔗 https://a.ch", 2 + 1 + urlWeightedLen},
		{"two urls", "https://a.ch https://b.ch", urlWeightedLen + 1 + urlWeightedLen},
		// Only URLs at a boundary count as URLs; anything else is charged per
		// rune, which overcounts rather than under.
		{"not a url", "xhttps://a.ch", 13},
		{"ellipsis", "…", 2},
		// X linkifies bare domains. This one is on every Kanton Zürich post.
		{"attribution line", "Source: OpenParlData.ch", 8 + urlWeightedLen},
		{"bare domain with path", "kantonsrat.zh.ch/geschaefte", urlWeightedLen},
		// A date is not a link, and charging it 23 would waste header room.
		{"date is not a domain", "15.06.2026", 10},
		{"abbreviation is not a domain", "z.B.", 4},
		{"decimal is not a domain", "3.5", 3},
		{"word is not a domain", "Verkehr", 7},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := weightedLen(tc.in); got != tc.want {
				t.Errorf("weightedLen(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestTruncateToWeighted(t *testing.T) {
	// Truncation has to respect the same weights, or a title trimmed to "fit"
	// still overshoots and the API rejects the root post.
	if got := truncateToWeighted("Grünliberale", 5); got != "Grünl" {
		t.Errorf("truncateToWeighted() = %q, want %q", got, "Grünl")
	}
	if got := truncateToWeighted("📊📊📊", 4); got != "📊📊" {
		t.Errorf("truncateToWeighted() = %q, want %q", got, "📊📊")
	}
	if got := truncateToWeighted("kurz", 99); got != "kurz" {
		t.Errorf("truncateToWeighted() = %q, want %q", got, "kurz")
	}
}

// TestWeightedLenNeverUndercounts is the property that matters operationally:
// a post the formatter believes fits must fit, because the alternative is a
// published root post whose replies were rejected mid-thread.
func TestWeightedLenNeverUndercounts(t *testing.T) {
	for _, s := range []string{
		"🗳️  Kantonsrat ZH | Abstimmung vom 15.06.2026",
		"🏛️ Fraktionen (Ja/Nein/Enth/Abw):\nGrünliberale 23/0/0/0",
		"📊 130 Ja | 44 Nein | 0 Enthaltung | 6 Abwesend",
		"✅ Angenommen: Änderungsantrag zu Dispositivziffer 1",
	} {
		if got, runes := weightedLen(s), len([]rune(s)); got < runes {
			t.Errorf("weightedLen(%q) = %d, below rune count %d", s, got, runes)
		}
	}
}
