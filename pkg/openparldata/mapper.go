package openparldata

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// dateLayouts are the timestamp shapes OpenParlData has been observed to return.
var dateLayouts = []string{"2006-01-02T15:04:05", time.RFC3339, "2006-01-02"}

// bodyLocation is the wall clock every body this adapter serves actually sits by.
//
// OpenParlData publishes timestamps in UTC with no zone marker, while the
// official archives and any reader use local time: a vote the Kantonsrat's own
// archive lists at 11:29 arrives here as 09:29. Posts have to agree with the
// archive they link to, or the time in a post sends a reader to the wrong entry
// — and a vote taken late in a sitting would otherwise be grouped under the
// previous day.
var bodyLocation = loadBodyLocation()

func loadBodyLocation() *time.Location {
	loc, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		// Falls back to a fixed winter offset rather than silently reverting to
		// UTC, which would be an hour further out for half the year.
		return time.FixedZone("CET", 60*60)
	}
	return loc
}

// toVote maps a voting to the neutral model, without the member votes and
// affair number that need further calls (see Client.enrich).
func (c *Client) toVote(v votingDTO) votes.Vote {
	date := parseDate(v.Date)

	title := deref(v.AffairTitleDe)
	subtitle := deref(v.TitleDe)
	if title == "" {
		// Some bodies title the voting but not the affair.
		title, subtitle = subtitle, ""
	}
	if subtitle == title {
		// Kanton Zürich has no agenda-item concept and repeats the affair title
		// as the voting title. A subtitle that restates the headline adds
		// nothing and would render as "X: X".
		subtitle = ""
	}

	out := votes.Vote{
		SourceID:     v.ExternalID,
		Jurisdiction: c.jurisdiction.Key,
		Body:         c.jurisdiction.ShortName,
		Date:         date,
		// OpenParlData has no per-sitting sequence number, but its timestamps
		// are second-precision, so they order votes within a sitting exactly.
		Sequence: sequenceFromDate(date),

		Title:    title,
		Subtitle: subtitle,
		Type:     deref(v.TypeDe),

		SourceURL: deref(v.URLExternalDe),

		Yes:        v.ResultsYes,
		No:         v.ResultsNo,
		Abstention: v.ResultsAbstention,
		Absent:     v.ResultsAbsent,

		Decision: decision(v),

		// CC BY 4.0 obliges us to credit the source wherever the data appears.
		Attribution: Attribution,

		Affair: votes.Affair{
			Title:  deref(v.AffairTitleDe),
			Number: groupingNumber(v.AffairID, v.ExternalID),
		},
	}

	if v.AffairID != nil {
		out.Affair.ID = strconv.FormatInt(*v.AffairID, 10)
	}
	// Provisional until the affair is fetched, which replaces it.
	out.GroupURL = out.SourceURL

	return out
}

// applyAffair fills in the fields that only the /affairs call provides.
func applyAffair(v *votes.Vote, a affairDTO) {
	if n := deref(a.Number); n != "" {
		v.Affair.Number = n
	}
	if t := deref(a.TitleDe); t != "" {
		v.Affair.Title = t
		if v.Title == "" {
			v.Title = t
		}
	}
	if u := deref(a.URLExternalDe); u != "" {
		v.Affair.URL = u

		// Every post about this business links to the parliament's own Geschäft
		// page, whether it covers one vote or five.
		//
		// It is not the page where a tally is easiest to read — that is the
		// vote's own archive page, and this one gives the totals in prose with
		// a name list for only some votes. It is chosen for what a published
		// link has to survive: it is the canton's own permalink, keyed by the
		// Geschäft id, on the parliament's own domain, and it carries the
		// documents, committee reports and every step of the business. A link
		// that rots or points at a third party in a year cannot be repaired
		// after the fact; a number that takes one more click can.
		//
		// Both URLs get it, because the reason applies to both. What the source
		// gives a single vote is a zh.recapp.ch deep link keyed by two opaque
		// uuids — precisely the third-party link the paragraph above rules out,
		// and letting a group of one keep it would make durability depend on how
		// many votes a sitting happened to hold. The recapp link stays available
		// through Affair and the source; it just does not go in a post.
		v.GroupURL = u
		v.SourceURL = u
	}
}

// groupingNumber is a placeholder business number, used until the /affairs call
// supplies the real one.
//
// It must never be empty. Votes are grouped on business number plus sitting
// day, so an empty value makes every such vote of one day look like the same
// business matter: 47 Kanton Zürich votings carry no affair_id at all, eight of
// them on 2025-11-24 alone, and those would be published as one post claiming
// eight unrelated votes belong together.
//
// Without an affair the vote is its own group, so it falls back to its own id.
func groupingNumber(affairID *int64, externalID string) string {
	if affairID != nil {
		return fmt.Sprintf("#%d", *affairID)
	}
	return "#vote-" + externalID
}

// decision returns the source's outcome label, deriving one when the source
// leaves it null — which Kanton Zürich always does.
//
// It is deliberately expressed in the same vocabulary the formatters already
// understand, so a derived decision renders identically to a reported one.
func decision(v votingDTO) string {
	if d := deref(v.Decision); d != "" {
		return d
	}
	if v.ResultsYes == nil || v.ResultsNo == nil {
		return ""
	}
	if *v.ResultsYes > *v.ResultsNo {
		return "Ja"
	}
	return "Nein"
}

// choiceLabels maps OpenParlData's harmonised vote values onto the German
// vocabulary the formatters render and the Fraktion table groups by.
//
// Mapping from the harmonised value rather than the display string is what
// makes two bodies' breakdowns line up: Kanton Zürich labels an abstention
// "Enthalten" and Stadt Zürich "Enthaltung", and a table with both columns
// would be wrong rather than merely ugly.
var choiceLabels = map[string]string{
	"yes":        "Ja",
	"no":         "Nein",
	"abstention": "Enthaltung",
	"absent":     "Abwesend",
}

func toMemberVote(v voteDTO) votes.MemberVote {
	choice, ok := choiceLabels[strings.ToLower(v.Vote)]
	if !ok {
		// An unrecognised value still belongs in the table under its own
		// label — dropping the member would understate their faction.
		choice = deref(v.VoteDisplayDe)
		if choice == "" {
			choice = v.Vote
		}
	}

	return votes.MemberVote{
		Name:     deref(v.PersonFullname),
		Party:    deref(v.PersonPartyDe),
		Fraktion: fraktionName(deref(v.PersonParliamentaryGroupNameDe)),
		Choice:   choice,
	}
}

// fraktionName trims the redundant "Fraktion " prefix some bodies include
// ("Fraktion SVP"), so faction labels read the same across sources.
func fraktionName(name string) string {
	name = strings.TrimSpace(name)
	if rest, ok := strings.CutPrefix(name, "Fraktion "); ok {
		return strings.TrimSpace(rest)
	}
	return name
}

// parseDate returns the zero time for anything unparseable; callers treat that
// as "unknown", not "old".
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.In(bodyLocation)
		}
	}
	return time.Time{}
}

func sequenceFromDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return strconv.FormatInt(t.Unix(), 10)
}
