package openparldata

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// Client enumerates its body's roster.
var _ votes.MemberSource = (*Client)(nil)

// rosterPageSize bounds one roster request. The chamber has 180 seats but its
// membership history runs to nine hundred records, so these calls page.
const rosterPageSize = 500

// rosterMaxPages bounds the paging, so a listing that never reports its end
// cannot spin against a third-party API.
const rosterMaxPages = 20

// FetchMembers returns the members currently sitting in this body.
//
// Two independent signals have to agree before a person is counted, because
// each is wrong on its own in a way that matters here:
//
//   - An active membership in the chamber's own group is the roster of record,
//     but a seat someone has left is not always closed. Kanton Zürich currently
//     carries one such record, two years stale.
//   - A person's own `active` flag is maintained, but says only that the person
//     is active *somewhere*. Twenty former Kantonsräte who now sit in Bern pass
//     that test and have never left the canton's person list.
//
// Their intersection is exactly the sitting chamber — 180, against 181 and 208
// for the two signals alone, as of writing. Requiring both errs towards omitting
// someone who has left rather than including them, which is the right way round:
// the file this feeds is append-only, so an entry added in error stays forever,
// while a member missed today is picked up on the next run.
func (c *Client) FetchMembers() ([]votes.Member, error) {
	council, err := c.councilGroupID()
	if err != nil {
		return nil, err
	}

	seated, err := c.activeMemberIDs(council)
	if err != nil {
		return nil, err
	}

	people, err := c.people()
	if err != nil {
		return nil, err
	}

	out := make([]votes.Member, 0, len(seated))
	for _, p := range people {
		if !seated[p.ID] || !p.Active {
			continue
		}
		name := strings.TrimSpace(p.Fullname)
		if name == "" {
			continue
		}
		out = append(out, votes.Member{
			Name:       name,
			Party:      strings.TrimSpace(deref(p.PartyDe)),
			Fraktion:   fraktionName(deref(p.ParliamentaryGroupNameDe)),
			ProfileURL: strings.TrimSpace(deref(p.WebsiteParliamentURLDe)),
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("openparldata: no sitting members found for body %q", c.bodyKey)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// councilGroupID finds the group that *is* the chamber, as opposed to the
// committees and parliamentary groups that sit inside it.
//
// It is looked up rather than configured because the numeric ids are the API's
// own and mean nothing outside it: a body is identified by its body_key
// everywhere else in this adapter, and the roster should be no different.
func (c *Client) councilGroupID() (int64, error) {
	var resp groupsResponse
	found := int64(0)

	err := c.pageRoster("/groups/", &resp, func() {
		if found != 0 {
			return
		}
		for _, g := range resp.Data {
			if g.TypeHarmonized == "council_legislative" && g.Active {
				found = g.ID
				return
			}
		}
	})
	if err != nil {
		return 0, err
	}
	if found == 0 {
		return 0, fmt.Errorf("openparldata: body %q has no active legislative council group", c.bodyKey)
	}
	return found, nil
}

// activeMemberIDs returns the person ids holding an open seat in the group.
func (c *Client) activeMemberIDs(groupID int64) (map[int64]bool, error) {
	var resp membershipsResponse
	seated := make(map[int64]bool)

	err := c.pageRoster(fmt.Sprintf("/groups/%d/memberships", groupID), &resp, func() {
		for _, m := range resp.Data {
			if m.Active {
				seated[m.PersonID] = true
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return seated, nil
}

// people returns every person the API knows for this body, sitting or not.
func (c *Client) people() ([]personDTO, error) {
	var resp personsResponse
	var all []personDTO

	err := c.pageRoster("/persons/", &resp, func() {
		all = append(all, resp.Data...)
	})
	if err != nil {
		return nil, err
	}
	return all, nil
}

// pagedRoster is a listing this file pages through. The interface exists only
// so one loop can drive listings that decode into different types.
type pagedRoster interface {
	pageMeta() meta
}

// pageRoster walks a paged listing, calling collect once per page with the
// decoded page still in out. Collect must copy what it needs: the next page
// decodes over the same value.
func (c *Client) pageRoster(path string, out pagedRoster, collect func()) error {
	for page := 0; page < rosterMaxPages; page++ {
		params := url.Values{}
		params.Set("body_key", c.bodyKey)
		params.Set("limit", strconv.Itoa(rosterPageSize))
		params.Set("offset", strconv.Itoa(page*rosterPageSize))

		if err := c.get(path, params, out); err != nil {
			return err
		}
		collect()

		if !out.pageMeta().HasMore {
			return nil
		}
	}
	return fmt.Errorf("openparldata: %s did not end within %d pages", path, rosterMaxPages)
}
