package zurichapi

import "github.com/siiitschiii/zuerichratsinfo/pkg/votes"

// Client implements votes.Source for the Stadt Zürich PARIS API.
var _ votes.Source = (*Client)(nil)

// Jurisdiction returns the body this source serves.
func (c *Client) Jurisdiction() votes.Jurisdiction { return Jurisdiction }

// FetchRecent returns the limit most recent votes, newest first.
func (c *Client) FetchRecent(limit int) ([]votes.Vote, error) {
	abstimmungen, err := c.FetchRecentAbstimmungen(limit)
	if err != nil {
		return nil, err
	}
	return ToVotes(abstimmungen), nil
}

// GroupByAffair groups votes by Geschäft and sitting date.
//
// Before grouping it back-fills votes that fell outside the fetch window: the
// API returns the most recent N votes, so a Geschäft whose earlier votes are
// older than that cut-off would otherwise be posted as a partial group.
func (c *Client) GroupByAffair(vs []votes.Vote) ([][]votes.Vote, error) {
	complete := c.completeGroups(vs)
	return votes.GroupByAffairAndDate(complete), nil
}

// completeGroups fetches every vote of every session represented in vs, and
// merges in those belonging to a Geschäft already present. Fetch failures are
// non-fatal: an incomplete group is better than no post at all, and the next
// run will retry.
func (c *Client) completeGroups(vs []votes.Vote) []votes.Vote {
	if len(vs) == 0 {
		return vs
	}

	var sessionIDs []string
	seenSession := make(map[string]bool)
	knownAffair := make(map[string]bool)
	existingIDs := make(map[string]bool)
	for _, v := range vs {
		if v.SessionID != "" && !seenSession[v.SessionID] {
			seenSession[v.SessionID] = true
			sessionIDs = append(sessionIDs, v.SessionID)
		}
		if v.Affair.Number != "" {
			knownAffair[v.Affair.Number] = true
		}
		existingIDs[v.SourceID] = true
	}

	for _, id := range sessionIDs {
		sessionVotes, err := c.FetchAbstimmungenForSitzung(id)
		if err != nil {
			continue
		}
		for _, a := range sessionVotes {
			if existingIDs[a.OBJGUID] || !knownAffair[a.GeschaeftGrNr] {
				continue
			}
			existingIDs[a.OBJGUID] = true
			vs = append(vs, ToVote(a))
		}
	}

	return vs
}
