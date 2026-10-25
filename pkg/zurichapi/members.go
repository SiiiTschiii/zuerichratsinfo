package zurichapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/siiitschiii/parlinfo/pkg/contacts"
	"github.com/siiitschiii/parlinfo/pkg/votes"
)

// Client enumerates the city council's roster.
var (
	_ votes.MemberSource  = (*Client)(nil)
	_ votes.SittingLister = (*Client)(nil)
)

// FetchMembers returns everyone who currently sits on the council, with the
// accounts PARIS publishes for them.
//
// The roster of record is the mandate register: an active Gemeinderat mandate is
// what makes someone a member. PARIS's contact archive cannot supply that, since
// it has no current-membership field and keeps members who left years ago; what
// it does carry is the accounts, and the party and Fraktion in the form the
// mapping uses. So the two are joined on the person: a member with no account
// published arrives as a bare name, as in the Kantonsrat, and an archived contact
// who holds no mandate is left out however many accounts they have.
//
// A member the archive spells differently from the register is still returned,
// under the register's name and without accounts. The duplicate that risks is
// caught by NameKey when the roster is merged.
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
		// An empty register is indistinguishable from a source outage, and
		// reporting a chamber with no members would hide it.
		return nil, fmt.Errorf("zurichapi: no active Gemeinderat mandates returned")
	}

	return sittingMembers(kontakte, mandates), nil
}

// SittingNames returns everyone holding an active Gemeinderat mandate, whether
// or not PARIS publishes an account for them.
func (c *Client) SittingNames() ([]string, error) {
	mandates, err := c.FetchActiveGemeinderatMandates()
	if err != nil {
		return nil, err
	}
	if len(mandates) == 0 {
		return nil, fmt.Errorf("zurichapi: no active Gemeinderat mandates returned")
	}

	names := make([]string, 0, len(mandates))
	for _, m := range mandates {
		names = append(names, strings.TrimSpace(m.Vorname+" "+m.Name))
	}
	return names, nil
}

// sittingMembers returns one member per person holding a mandate, taking the
// name, party, Fraktion and accounts from the matching contact when there is one.
func sittingMembers(kontakte []Kontakt, mandates []Behoerdenmandat) []votes.Member {
	byPerson := make(map[string]Kontakt, len(kontakte))
	for _, k := range kontakte {
		if name := memberName(k); name != "" {
			byPerson[contacts.NameKey(name)] = k
		}
	}

	seen := make(map[string]bool, len(mandates))
	out := make([]votes.Member, 0, len(mandates))
	for _, m := range mandates {
		name := strings.TrimSpace(m.Vorname + " " + m.Name)
		key := contacts.NameKey(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true

		member := votes.Member{
			Name:     name,
			Party:    strings.TrimSpace(m.Partei),
			Fraktion: strings.TrimSpace(m.Fraktion),
		}
		if k, ok := byPerson[key]; ok {
			member.Name = memberName(k)
			member.Party = strings.TrimSpace(k.Partei)
			member.Fraktion = strings.TrimSpace(k.Fraktion)
			member.Accounts = publishedAccounts(k)
		}
		out = append(out, member)
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
