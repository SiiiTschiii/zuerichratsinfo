package parlamentch

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/siiitschiii/parlinfo/pkg/votes"
)

// Client implements both roster interfaces: the Nationalrat publishes everyone
// who sits, so the roster and the list of who sits are the same list.
var (
	_ votes.MemberSource  = (*Client)(nil)
	_ votes.SittingLister = (*Client)(nil)
)

// memberDTO is one row of MemberCouncil.
type memberDTO struct {
	ID                    int64  `json:"ID"`
	FirstName             string `json:"FirstName"`
	LastName              string `json:"LastName"`
	PartyName             string `json:"PartyName"`
	ParlGroupAbbreviation string `json:"ParlGroupAbbreviation"`
}

// FetchMembers returns the members sitting in the Nationalrat, named the way
// the vote rows name them so the tagger matches the one against the other.
//
// It is what lets update_contacts -prune tell a Kantonsrat who moved to Bern
// from one who left politics: prune asks every other body who sits there, and
// a body without a roster stops it.
//
// The service publishes no social accounts, so Accounts stays empty.
func (c *Client) FetchMembers() ([]votes.Member, error) {
	var rows []memberDTO
	err := c.get("/MemberCouncil", query{
		"$filter": "Language eq 'DE' and Active eq true and Council eq 1",
		"$select": "ID,FirstName,LastName,PartyName,ParlGroupAbbreviation",
	}, &rows)
	if err != nil {
		return nil, err
	}

	out := make([]votes.Member, 0, len(rows))
	for _, r := range rows {
		name := strings.TrimSpace(r.FirstName + " " + r.LastName)
		if name == "" {
			continue
		}
		out = append(out, votes.Member{
			Name: name,
			// The name, not the abbreviation: the service abbreviates Die
			// Mitte as "M-E", its Fraktion's code.
			Party:      strings.TrimSpace(r.PartyName),
			Fraktion:   fraktionLabel(r.ParlGroupAbbreviation, ""),
			ProfileURL: ProfileURL(r.ID),
		})
	}
	// An empty roster would read to prune as "nobody sits here", which is
	// the one answer that could delete a contact who moved to Bern.
	if len(out) == 0 {
		return nil, fmt.Errorf("parlamentch: no sitting members found")
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SittingNames returns the names FetchMembers does.
func (c *Client) SittingNames() ([]string, error) {
	members, err := c.FetchMembers()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(members))
	for i, m := range members {
		names[i] = m.Name
	}
	return names, nil
}

// ProfileURL is a member's biography on parlament.ch. The ID form redirects to
// the page's current slug, which changes with a name and the ID does not.
func ProfileURL(id int64) string {
	return "https://www.parlament.ch/de/biografie?CouncillorId=" + strconv.FormatInt(id, 10)
}
