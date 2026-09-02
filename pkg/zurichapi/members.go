package zurichapi

import (
	"sort"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// Client enumerates the city council's roster.
var _ votes.MemberSource = (*Client)(nil)

// FetchMembers returns the council members PARIS publishes an account for.
//
// PARIS serves a contact archive rather than a roster: it has no field saying
// who currently sits, and it keeps members who left years ago. So the accounts
// are used as the relevance signal — someone PARIS publishes a channel for is
// someone worth having in the mapping, and the rest would arrive as several
// hundred bare names of people who may no longer be in the chamber.
//
// The cost of that is a sitting member who publishes nothing never appears
// here, and is added by hand when a handle is found for them. It is the same
// bargain this tool has always struck; a body whose source does list its
// sitting members — see openparldata.Client.FetchMembers — is seeded whole.
func (c *Client) FetchMembers() ([]votes.Member, error) {
	kontakte, err := c.FetchAllKontakte()
	if err != nil {
		return nil, err
	}

	out := make([]votes.Member, 0, len(kontakte))
	for _, k := range kontakte {
		name := memberName(k)
		if name == "" {
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
	return out, nil
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
