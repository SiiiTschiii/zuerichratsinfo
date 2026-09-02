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

// maxRenderedAuthors bounds what a post will name.
//
// A business is filed by one or two members and co-signed by a few more, and
// only the first signatories are named here, so the cap is a guard rather than
// a policy — but it has to exist. The alternative is an authorship line long
// enough to push the actual subject of the vote out of a 280-character post,
// which trades the thing a reader needs for the thing they merely like.
const maxRenderedAuthors = 3

// affairAuthors returns the members who put an affair before the chamber.
//
// Kanton Zürich's titles are bare subject lines — "Ökologischer Ausgleich im
// Siedlungsraum" — where Stadt Zürich's carry the submitters in the text. This
// call is what closes that gap, and with it the tagging: the mapping matches
// names in a post, so a post with no names in it tags nobody however well
// curated the file is.
//
// Two filters decide who is named, and both matter:
//
//   - Only first signatories. Everyone who signs is a harmonised "author", but
//     the Erstunterzeichnende are the ones the record treats as having filed it
//     and the ones a city title would have named.
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

	leads := make([]contributorDTO, 0, len(resp.Data))
	for _, cont := range resp.Data {
		if cont.Type != "person" || cont.RoleHarmonized != "author" {
			continue
		}
		if !isFirstSignatory(deref(cont.RoleDe)) {
			continue
		}
		if strings.TrimSpace(cont.Fullname) == "" || strings.TrimSpace(deref(cont.PartyDe)) == "" {
			continue
		}
		leads = append(leads, cont)
	}

	// The API returns signatories in descending position, first signatory last.
	sort.SliceStable(leads, func(i, j int) bool {
		return position(leads[i]) < position(leads[j])
	})

	if len(leads) > maxRenderedAuthors {
		leads = leads[:maxRenderedAuthors]
	}

	authors := make([]votes.Author, 0, len(leads))
	for _, cont := range leads {
		authors = append(authors, votes.Author{
			Name:  strings.TrimSpace(cont.Fullname),
			Party: strings.TrimSpace(deref(cont.PartyDe)),
		})
	}
	return authors, nil
}

// isFirstSignatory reads the body's own role label.
//
// The harmonised field cannot answer this — it says "author" for every
// signatory — so the German label is the only thing that separates the member
// who filed a business from the ones who added their name to it. Matching a
// prefix covers both gendered forms and the pair the API serves joined,
// "Erstunterzeichnerin / Erstunterzeichner".
func isFirstSignatory(roleDe string) bool {
	return strings.HasPrefix(strings.TrimSpace(roleDe), "Erstunterzeichn")
}

func position(c contributorDTO) int {
	if c.Position == nil {
		return 0
	}
	return *c.Position
}
