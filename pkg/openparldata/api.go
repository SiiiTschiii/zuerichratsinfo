package openparldata

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// Client implements votes.Source.
var _ votes.Source = (*Client)(nil)

// pageSize bounds a single request. The API accepts larger values, but paging
// keeps individual responses small on a body with thousands of votings.
const pageSize = 100

// memberVotePageSize covers the largest chamber in one request; the Nationalrat
// has 200 seats.
const memberVotePageSize = 500

// FetchRecent returns the most recent votings for this body, newest first.
//
// It deliberately does *not* fetch member votes or affair numbers. Those need
// two further calls per voting, and almost every voting fetched here has
// already been posted and is about to be discarded — the enrichment happens in
// GroupByAffair, which only sees what survived dedup.
func (c *Client) FetchRecent(limit int) ([]votes.Vote, error) {
	if limit <= 0 {
		return nil, nil
	}

	var out []votes.Vote
	for offset := 0; offset < limit; offset += pageSize {
		size := min(pageSize, limit-offset)

		params := url.Values{}
		params.Set("body_key", c.bodyKey)
		params.Set("sort_by", "-date")
		params.Set("limit", strconv.Itoa(size))
		params.Set("offset", strconv.Itoa(offset))

		var resp votingsResponse
		if err := c.get("/votings/", params, &resp); err != nil {
			return nil, err
		}

		for _, v := range resp.Data {
			c.rememberVoting(v)
			out = append(out, c.toVote(v))
		}

		if len(resp.Data) < size {
			break
		}
	}

	return out, nil
}

// GroupByAffair enriches the given votes and groups them by business matter and
// sitting day.
//
// Enrichment happens here rather than in FetchRecent because the affair number
// is a grouping input and the member votes are only needed for what actually
// gets posted.
func (c *Client) GroupByAffair(vs []votes.Vote) ([][]votes.Vote, error) {
	if len(vs) == 0 {
		return nil, nil
	}

	complete, err := c.completeGroups(vs)
	if err != nil {
		return nil, err
	}

	for i := range complete {
		if err := c.enrich(&complete[i]); err != nil {
			return nil, err
		}
	}

	// After enrichment, because whether a detail source's verdict may be
	// published depends on the affair type that enrichment fetches.
	c.applyDetails(complete)

	return votes.GroupByAffairAndDate(complete), nil
}

// applyDetails fills in what the configured DetailSource knows and this API
// does not.
//
// The type takes precedence over type_de rather than merely filling its gaps.
// For Kanton Zürich the API's own value is not only incomplete — whole sittings
// arrive null — but wrong often enough to matter: in a 94-vote sample three
// votes typed "Quorum" were ordinary Abstimmungen and one typed "Normal" was a
// Quorumsabstimmung. The detail source is the record those values are derived
// from, so where the two disagree it is the one to believe.
//
// A vote the source says nothing about keeps everything it already had, and a
// source that fails entirely costs nothing but the enrichment: the pipeline
// already refuses to publish a vote whose type it does not recognise.
func (c *Client) applyDetails(vs []votes.Vote) {
	if c.details == nil || len(vs) == 0 {
		return
	}

	voteURLs := make(map[string]string, len(vs))
	for _, v := range vs {
		if u := c.sourceURLs[v.SourceID]; u != "" {
			voteURLs[v.SourceID] = u
		}
	}
	if len(voteURLs) == 0 {
		return
	}

	details, err := c.details.Lookup(voteURLs)
	if err != nil {
		// Partial results come back alongside the error, so this reports the
		// gap and then uses whatever did arrive.
		log.Printf("⚠️  openparldata: vote details incomplete: %v", err)
	}

	for i := range vs {
		d, ok := details[vs[i].SourceID]
		if !ok {
			continue
		}
		if d.Type != "" {
			vs[i].Type = d.Type
		}
		if d.Decision != "" && vs[i].Decision == "" && affairStatesItsOutcome(vs[i].Affair.Type) {
			vs[i].Decision = d.Decision
		}
	}
}

// completeGroups fetches the other votings of each affair already present, so a
// business matter whose earlier votes fell outside the fetch window is posted
// whole rather than truncated.
//
// Completion is confined to sitting days the caller already has. An affair can
// run for years — a Kantonsrat business matter routinely has votes from 2022 and
// 2026 — and those older votes have already been through the age guard, or were
// never eligible for it. Pulling them back in here would smuggle them past the
// only defence against re-posting history, because this runs *after* the guard.
//
// Widening a group to its own sitting day is the same scope the PARIS adapter
// gets for free by completing per session.
func (c *Client) completeGroups(vs []votes.Vote) ([]votes.Vote, error) {
	seenVote := make(map[string]bool, len(vs))
	var affairIDs []string
	// daysWanted[affairID] is the set of sitting days already in play for it.
	daysWanted := make(map[string]map[string]bool)

	for _, v := range vs {
		seenVote[v.SourceID] = true
		if v.Affair.ID == "" {
			continue
		}
		if _, ok := daysWanted[v.Affair.ID]; !ok {
			daysWanted[v.Affair.ID] = make(map[string]bool)
			affairIDs = append(affairIDs, v.Affair.ID)
		}
		daysWanted[v.Affair.ID][v.DateString()] = true
	}

	for _, affairID := range affairIDs {
		found, err := c.votingsForAffairDays(affairID, daysWanted[affairID], seenVote)
		if err != nil {
			// Non-fatal, matching the PARIS adapter: an incomplete group is
			// better than no post, and the next run retries.
			log.Printf("⚠️  openparldata: could not complete affair %s: %v", affairID, err)
			continue
		}
		vs = append(vs, found...)
	}

	return vs, nil
}

// completionMaxPages bounds the paging below. The largest Kanton Zürich affair
// currently has 90 votings; this leaves room for a budget debate several times
// that without ever looping unboundedly against a third-party API.
const completionMaxPages = 10

// votingsForAffairDays returns the affair's votings that fall on one of the
// given sitting days, marking each as seen.
//
// Two things this must not get wrong. The sort is set explicitly rather than
// inherited from the API's default: the days of interest are always the most
// recent ones, so newest-first is what makes the early exit below correct, and
// a default that changed under us would silently truncate groups. And it pages,
// because a single affair can carry more votings than one page holds — the
// largest Kanton Zürich affair already has 90 — so an unpaged call would drop
// the rest of a long debate from its post.
func (c *Client) votingsForAffairDays(affairID string, days map[string]bool, seenVote map[string]bool) ([]votes.Vote, error) {
	oldestWanted := ""
	for day := range days {
		if day != "" && (oldestWanted == "" || day < oldestWanted) {
			oldestWanted = day
		}
	}

	var found []votes.Vote
	for page := 0; page < completionMaxPages; page++ {
		params := url.Values{}
		params.Set("body_key", c.bodyKey)
		params.Set("affair_id", affairID)
		params.Set("sort_by", "-date")
		params.Set("limit", strconv.Itoa(pageSize))
		params.Set("offset", strconv.Itoa(page*pageSize))

		var resp votingsResponse
		if err := c.get("/votings/", params, &resp); err != nil {
			return nil, err
		}

		reachedOlder := false
		for _, dto := range resp.Data {
			candidate := c.toVote(dto)
			day := candidate.DateString()

			// Newest-first, so once a result predates every day we care about,
			// nothing later in the listing can be relevant either.
			if oldestWanted != "" && day != "" && day < oldestWanted {
				reachedOlder = true
				break
			}
			if seenVote[dto.ExternalID] || !days[day] {
				continue
			}
			seenVote[dto.ExternalID] = true
			c.rememberVoting(dto)
			found = append(found, candidate)
		}

		if reachedOlder || len(resp.Data) < pageSize {
			break
		}
	}

	return found, nil
}

// enrich fills in the affair number and URL and the per-member votes.
func (c *Client) enrich(v *votes.Vote) error {
	votingID, err := c.votingID(*v)
	if err != nil {
		return err
	}

	// A vote with no affair has nothing to fetch: procedural motions and
	// attendance roll calls belong to no business matter, and asking anyway
	// spends a request to be told so.
	if v.Affair.ID != "" {
		affairs, err := c.fetchAffairs(votingID)
		if err != nil {
			// The fallback affair number keeps grouping correct, so this
			// degrades to a post without a business number rather than to no
			// post.
			log.Printf("⚠️  openparldata: could not fetch affair for voting %d: %v", votingID, err)
		} else if len(affairs) > 0 {
			applyAffair(v, affairs[0])
		}
	}

	members, err := c.fetchMemberVotes(votingID)
	if err != nil {
		// Totals are reported independently, so a post without the Fraktion
		// breakdown is still correct — just less detailed.
		log.Printf("⚠️  openparldata: could not fetch member votes for voting %d: %v", votingID, err)
		return nil
	}
	v.MemberVotes = members
	return nil
}

// votingID recovers the numeric API id for a vote.
//
// The neutral model keys on the source's stable external id, because that is
// what the vote log stores and what survives across API generations. The
// numeric id is an internal handle needed only to address the sub-resources, so
// the client remembers it from the listing that produced the vote and only
// falls back to a lookup if that memory is missing.
func (c *Client) votingID(v votes.Vote) (int64, error) {
	if id, ok := c.votingIDs[v.SourceID]; ok {
		return id, nil
	}

	params := url.Values{}
	params.Set("body_key", c.bodyKey)
	params.Set("external_id", v.SourceID)
	params.Set("limit", "1")

	var resp votingsResponse
	if err := c.get("/votings/", params, &resp); err != nil {
		return 0, err
	}
	if len(resp.Data) == 0 {
		return 0, fmt.Errorf("openparldata: no voting with external_id %q", v.SourceID)
	}
	c.rememberVoting(resp.Data[0])
	return resp.Data[0].ID, nil
}

func (c *Client) fetchAffairs(votingID int64) ([]affairDTO, error) {
	var resp affairsResponse
	if err := c.get(fmt.Sprintf("/votings/%d/affairs", votingID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) fetchMemberVotes(votingID int64) ([]votes.MemberVote, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(memberVotePageSize))

	var resp votesResponse
	if err := c.get(fmt.Sprintf("/votings/%d/votes", votingID), params, &resp); err != nil {
		return nil, err
	}

	out := make([]votes.MemberVote, 0, len(resp.Data))
	for _, m := range resp.Data {
		out = append(out, toMemberVote(m))
	}
	return out, nil
}

// verdictBearingAffairTypes are the kinds of business for which a carried vote
// really does mean "angenommen".
//
// It is an allowlist because a Ja does not mean the same thing everywhere. A
// Vorlage that carries is adopted. An Einzelinitiative that carries is only
// *vorläufig unterstützt* and goes on to the Regierungsrat — the Kantonsrat's
// own record for the 17.08.2026 Sprungbeschwerde reads "Vorläufig unterstützt
// (79 Stimmen)" and lists the business as still pending, while a post calling
// it "Angenommen" would tell a reader the initiative had passed. The same holds
// for a Parlamentarische Initiative, and a Motion or Postulat is *überwiesen*
// rather than accepted.
//
// Roughly a sixth of recent Kanton Zürich votes belong to one of those, so this
// is not a corner case. Where the wording is not certain the formatters print
// the counts and claim nothing, which is what they did for every cantonal vote
// before a decision was available at all.
//
// Naming the right verb per business type would be better than silence, but the
// archive does not publish which procedural step a given vote served, and
// deriving it would be inventing an outcome — the exact thing this pipeline is
// built not to do.
var verdictBearingAffairTypes = map[string]bool{
	"Vorlage":              true,
	"Geschäftsbericht":     true,
	"Rechenschaftsbericht": true,
	"Tätigkeitsbericht":    true,
}

// affairStatesItsOutcome reports whether "angenommen"/"abgelehnt" is an honest
// label for a vote on this kind of business.
func affairStatesItsOutcome(affairType string) bool {
	return verdictBearingAffairTypes[strings.TrimSpace(affairType)]
}
