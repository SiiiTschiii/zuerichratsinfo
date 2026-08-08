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
		// OpenParlData has no per-session sequence number, but its timestamps
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
			Number: affairFallbackNumber(v.AffairID),
		},
	}

	if v.AffairID != nil {
		out.Affair.ID = strconv.FormatInt(*v.AffairID, 10)
	}
	// The vote's own page is the link for both the single and the group shape.
	//
	// On Kanton Zürich every vote of one agenda item shares an agendaItemUid,
	// so this page lists the whole group's votes with their tallies and name
	// lists, while the segmentUid lands the reader on this particular one.
	// That is the same thing the Stadt Zürich agenda-item link does.
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

		// The affair page is deliberately *not* the link a post carries. It
		// lists the business matter's documents and a prose summary, but no
		// vote tally and no name list — a reader following it cannot check the
		// numbers in the post, which is the whole point of publishing a link.
		//
		// The vote's own page can. It is only a fallback here, for a vote the
		// source gave no URL of its own.
		if v.SourceURL == "" {
			v.SourceURL = u
		}
		if v.GroupURL == "" {
			v.GroupURL = u
		}
	}
}

// affairFallbackNumber keeps grouping correct when the affair's human-readable
// number cannot be fetched. Votes must not all collapse into one group just
// because a lookup failed.
func affairFallbackNumber(affairID *int64) string {
	if affairID == nil {
		return ""
	}
	return fmt.Sprintf("#%d", *affairID)
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
			return t
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
