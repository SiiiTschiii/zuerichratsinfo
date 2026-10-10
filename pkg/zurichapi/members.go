package zurichapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// Client enumerates the city council's roster.
var _ votes.MemberSource = (*Client)(nil)

// FetchMembers returns the sitting council members PARIS publishes an account
// for.
//
// Two sources are needed because neither answers the question alone. The contact
// archive carries the accounts but has no field saying who currently sits, and
// keeps members who left years ago; the mandate register says who sits but
// carries no accounts. A contact is therefore kept only when it has a published
// account and holds an active Gemeinderat mandate.
//
// The cost is that a sitting member who publishes nothing never appears here,
// and is added by hand when a handle is found for them. A name the two sources
// spell differently is dropped the same way. Both err towards omitting someone
// who sits rather than adding someone who does not, which is the right way
// round for a file that is append-only: an entry added in error stays forever,
// while a member missed today is picked up on the next run. A body whose source
// lists its sitting members outright — see openparldata.Client.FetchMembers —
// is seeded whole.
func (c *Client) FetchMembers() ([]votes.Member, error) {
	kontakte, err := c.FetchAllKontakte()
	if err != nil {
		return nil, err
	}

	mandates, err := c.FetchActiveGemeinderatMandates()
	if err != nil {
		return nil, err
	}
	if len(mandates) == 0 {
		// Filtering by an empty register would drop everyone, and reporting a
		// chamber with no members is indistinguishable from a source outage.
		return nil, fmt.Errorf("zurichapi: no active Gemeinderat mandates returned")
	}

	return sittingMembers(kontakte, mandates), nil
}

// sittingMembers keeps the contacts that publish an account and hold one of the
// given mandates, matching on the person rather than the spelling of the name.
func sittingMembers(kontakte []Kontakt, mandates []Behoerdenmandat) []votes.Member {
	sitting := make(map[string]bool, len(mandates))
	for _, m := range mandates {
		sitting[contacts.NameKey(m.Vorname+" "+m.Name)] = true
	}

	out := make([]votes.Member, 0, len(sitting))
	for _, k := range kontakte {
		name := memberName(k)
		if name == "" || !sitting[contacts.NameKey(name)] {
			continue
		}

		accounts := publishedAccounts(k)
		if len(accounts) == 0 {
			continue
		}

		out = append(out, votes.Member{
			Name:     name,
			Party:    strings.TrimSpace(k.Partei),
			Fraktion: strings.TrimSpace(k.Fraktion),
			Accounts: accounts,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// memberName renders a contact the way the curated mapping and the council's
// own vote titles both write a person: given name first.
//
// It is assembled from the separate fields rather than taken from NameVorname,
// which is "Weyermann Karin" and, worse, joined with a non-breaking space —
// invisible in a diff, and matching nothing in a post. Reading it as-is added
// every existing contact a second time under their reversed name.
func memberName(k Kontakt) string {
	first := strings.TrimSpace(k.Vorname)
	last := strings.TrimSpace(k.Name)
	if first != "" && last != "" {
		return first + " " + last
	}
	if joined := strings.TrimSpace(first + last); joined != "" {
		return joined
	}
	// Older records carry only the combined field.
	return strings.TrimSpace(strings.ReplaceAll(k.NameVorname, "\u00a0", " "))
}

// publishedAccounts maps PARIS's social media entries onto the platform
// vocabulary the contacts mapping uses, dropping anything it does not name.
func publishedAccounts(k Kontakt) []votes.Account {
	var out []votes.Account

	for _, sm := range k.SozialeMedien.Kommunikation {
		url := strings.TrimSpace(sm.Adresse)
		if url == "" {
			continue
		}

		platform := strings.ToLower(strings.TrimSpace(sm.Typ))
		switch platform {
		case "twitter":
			platform = "x"
		case "x", "facebook", "instagram", "linkedin", "bluesky", "tiktok":
		default:
			// A channel the mapping has no column for — a personal blog, say.
			continue
		}

		if platform == "x" {
			// The mapping stores x.com; twitter.com links still resolve but
			// would sit in the file as a second spelling of the same handle.
			url = strings.ReplaceAll(url, "twitter.com", "x.com")
		}
		if platform == "bluesky" {
			// web-cdn.bsky.app serves the same profile as bsky.app. PARIS
			// publishes both, and stored side by side they are one account
			// recorded twice.
			url = strings.ReplaceAll(url, "web-cdn.bsky.app", "bsky.app")
		}

		// PARIS publishes a fair number of these bare, as
		// "www.instagram.com/…". cmd/validate_contacts rejects a URL with no
		// scheme, so writing one through would break the file's own CI check.
		if !strings.Contains(url, "://") {
			url = "https://" + url
		}

		out = append(out, votes.Account{Platform: platform, URL: url})
	}

	return out
}
