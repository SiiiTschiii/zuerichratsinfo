package openparldata

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// contributorPageSize covers any realistic signatory list in one request. The
// largest on record for Kanton Zürich is well under a hundred.
const contributorPageSize = 100

// affairAuthors returns the members who put an affair before the chamber.
//
// Kanton Zürich's titles are bare subject lines — "Ökologischer Ausgleich im
// Siedlungsraum" — where Stadt Zürich's carry the submitters in the text. This
// call is what closes that gap, and with it the tagging: the mapping matches
// names in a post, so a post with no names in it tags nobody however well
// curated the file is.
//
// Everyone who signed is named, first signatory first. Stadt Zürich's titles
// name every submitter — city Postulate simply happen to have two — and the
// full list carries something the lead name alone hides: a Postulat signed
// across SP, Die Mitte, GLP, Grüne and AL is a broad coalition, and that is
// visible at a glance only if the names are there.
//
// One filter decides who is named:
//
//   - Only people the record gives a party. A person with none is not a member
//     acting in the chamber but a private individual exercising the right to
//     file an Einzelinitiative. The parliament publishes their name; putting it
//     on a social media account is a different act, and not one this project
//     has any business performing.
//
// Roughly a third of recent cantonal business has authors at all. The rest —
// Vorlagen from the Regierungsrat, committee reports — is filed by
// organisations, which come back on the same list and are skipped here: those
// have their own accounts, and tagging them is a separate question.
func (c *Client) affairAuthors(affairID int64) ([]votes.Author, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(contributorPageSize))

	var resp contributorsResponse
	if err := c.get(fmt.Sprintf("/affairs/%d/contributors", affairID), params, &resp); err != nil {
		return nil, err
	}

	signatories := make([]contributorDTO, 0, len(resp.Data))
	for _, cont := range resp.Data {
		if cont.Type != "person" || cont.RoleHarmonized != "author" {
			continue
		}
		if strings.TrimSpace(cont.Fullname) == "" || strings.TrimSpace(deref(cont.PartyDe)) == "" {
			continue
		}
		signatories = append(signatories, cont)
	}

	// First signatory first, then the rest in the order the parliament lists
	// them. Position alone almost always gives that — the Erstunterzeichnende
	// carry position 1 — but the role is the field that actually says so, and
	// it costs nothing to sort on both.
	sort.SliceStable(signatories, func(i, j int) bool {
		li := isFirstSignatory(deref(signatories[i].RoleDe))
		lj := isFirstSignatory(deref(signatories[j].RoleDe))
		if li != lj {
			return li
		}
		return position(signatories[i]) < position(signatories[j])
	})

	authors := make([]votes.Author, 0, len(signatories))
	for _, cont := range signatories {
		authors = append(authors, votes.Author{
			Name:  strings.TrimSpace(cont.Fullname),
			Party: strings.TrimSpace(deref(cont.PartyDe)),
		})
	}
	return authors, nil
}

// isFirstSignatory reads the body's own role label, which is the only field
// that separates the member who filed a business from the ones who added their
// name to it — the harmonised field says "author" for every signatory. Matching
// a prefix covers both gendered forms and the pair the API serves joined,
// "Erstunterzeichnerin / Erstunterzeichner". It orders the list; it no longer
// decides who is on it.
func isFirstSignatory(roleDe string) bool {
	return strings.HasPrefix(strings.TrimSpace(roleDe), "Erstunterzeichn")
}

func position(c contributorDTO) int {
	if c.Position == nil {
		return 0
	}
	return *c.Position
}
