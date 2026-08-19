// Package recapp adapts the Kantonsrat Zürich audio archive (zh.recapp.ch) to
// the vote information OpenParlData omits.
//
// It exists because OpenParlData's type_de is incomplete for Kanton Zürich —
// whole sittings arrive with a null type, the 17.08.2026 sitting having served
// all five of its votes that way — and because it cannot express two
// distinctions the parliament makes: the Ausgabenbremse, which it folds into
// "Quorum" or leaves as "Normal" depending on how the ballot was run, and the
// attendance roll call, which it publishes as an ordinary voting.
//
// The archive is what OpenParlData harvests from, so it fills those gaps at the
// source: every vote segment carries the parliament's own label for what kind
// of vote it was, and its outcome.
//
// It does not replace type_de wholesale. Measured over the 300 most recent ZH
// votings on 2026-08-19, type_de agrees with the archive's votingScheme
// wherever it is present (Normal/binary 252, Quorum/quorum 31, Cup 4), while
// the archive's segment titles are editorial free text: 226 of them read only
// "Abstimmung", including every vote on the preliminary support of an
// Einzelinitiative, which is a threshold ballot. A title that names no ballot
// type therefore adds nothing and must not overrule a type the API does carry.
//
// The join is exact rather than heuristic: a segment's extVotingUid is the same
// identifier OpenParlData publishes as external_id.
package recapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the archive's viewer API root.
const DefaultBaseURL = "https://zh.recapp.ch/viewer/api/shareparl"

// Vote types in the neutral vocabulary the formatters already speak. Mapping
// into it here keeps the archive's naming from leaking downstream.
const (
	TypeNormal = "Normal"
	TypeQuorum = "Quorum"
	TypeCup    = "Cup-Abstimmung"

	// TypeAusgabenbremse is the spending brake: a binary Ja/Nein ballot that
	// carries only if 91 of the 180 members vote for it, regardless of how many
	// vote against. It is counted like a quorum vote and named like itself,
	// because "Ausgabenbremse" tells a reader what the threshold was for while
	// "Quorum" only tells them one existed.
	TypeAusgabenbremse = "Ausgabenbremse"

	// TypeAttendance marks a roll call establishing who is in the chamber.
	// It is not a political vote and must never be published as one; it is
	// named rather than left blank so the pipeline can tell "we know what this
	// is and it is not postable" apart from "we have never seen this".
	TypeAttendance = "Anwesenheitsermittlung"
)

// Info is what the archive knows about one vote beyond OpenParlData's listing.
//
// A zero Info means the archive had nothing to say, which callers must treat as
// "unknown" and not as any particular type.
type Info struct {
	// Type is the vote type in the neutral vocabulary, or "" when the archive
	// used a label this package does not recognise.
	Type string
	// TypeUnqualified marks a Type inferred from a label that names no ballot
	// type — a bare "Abstimmung". It says a vote was held and nothing about how
	// it was counted, so a caller holding a more specific type must keep it.
	TypeUnqualified bool
	// Decision is "angenommen" or "abgelehnt", or "" when the archive reports
	// no outcome. OpenParlData leaves this null for every Kanton Zürich vote.
	Decision string
}

// Client reads vote segments from the archive.
//
// Segments are fetched per agenda item and cached, because one agenda item
// covers every vote of a business matter: the five votes of the 17.08.2026
// sitting need three requests, not five.
type Client struct {
	baseURL string
	http    *http.Client

	cache map[string]map[string]segment
}

// New builds a client against the live archive.
func New() *Client {
	return &Client{
		baseURL: DefaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
		cache:   make(map[string]map[string]segment),
	}
}

// SetBaseURL points the client at another host. Used by tests to serve
// recorded fixtures instead of making live calls.
func (c *Client) SetBaseURL(u string) { c.baseURL = u }

// Lookup returns what the archive knows about each vote, keyed by the same
// external voting id the caller passed in.
//
// voteURLs maps an external voting id to that vote's archive URL, which is the
// url_external_de OpenParlData publishes. Votes whose URL does not name an
// agenda item, and votes the archive does not list, are simply absent from the
// result — the caller keeps whatever it already had.
//
// A failed request drops the votes of that agenda item and is reported, but
// never fails the whole lookup: the archive is enrichment, and the callers
// treat a missing answer as "unknown type", which is already safe.
func (c *Client) Lookup(voteURLs map[string]string) (map[string]Info, error) {
	// Group by agenda item so shared items cost one request between them.
	wanted := make(map[string][]string)
	for votingID, rawURL := range voteURLs {
		item := agendaItemUID(rawURL)
		if item == "" {
			continue
		}
		wanted[item] = append(wanted[item], votingID)
	}

	out := make(map[string]Info, len(voteURLs))
	var firstErr error

	for item, votingIDs := range wanted {
		segments, err := c.segmentsFor(item)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, votingID := range votingIDs {
			seg, ok := segments[votingID]
			if !ok {
				continue
			}
			out[votingID] = seg.info()
		}
	}

	return out, firstErr
}

// segmentsFor returns the agenda item's vote segments keyed by external voting
// id, fetching them once per client.
func (c *Client) segmentsFor(agendaItemUID string) (map[string]segment, error) {
	if cached, ok := c.cache[agendaItemUID]; ok {
		return cached, nil
	}

	params := url.Values{}
	params.Set("agendaItemUid", agendaItemUID)
	params.Set("language", "de")
	// The archive serves a different payload shape to its iOS client.
	params.Set("ios", "false")

	endpoint := c.baseURL + "/segments?" + params.Encode()

	resp, err := c.http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("recapp: fetching %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("recapp: reading %s: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recapp: %s: status %d", endpoint, resp.StatusCode)
	}

	var all []segment
	if err := json.Unmarshal(body, &all); err != nil {
		return nil, fmt.Errorf("recapp: decoding %s: %w", endpoint, err)
	}

	// An agenda item is mostly speaker segments; only the votes are of interest.
	byVoting := make(map[string]segment)
	for _, s := range all {
		if s.Type != segmentTypeVote || s.ExtVotingUID == "" {
			continue
		}
		byVoting[s.ExtVotingUID] = s
	}

	c.cache[agendaItemUID] = byVoting
	return byVoting, nil
}

// agendaItemUID pulls the agenda item out of an archive URL, or returns "" when
// the URL is not one — which is the normal case for every other body, whose
// votes link somewhere else entirely.
func agendaItemUID(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("agendaItemUid")
}

const segmentTypeVote = "vote"

// segment is one entry in an agenda item, covering both speeches and votes.
// Only the vote fields are read.
type segment struct {
	SegmentUID string `json:"segmentUid"`
	Type       string `json:"type"`

	// ExtVotingUID is OpenParlData's external_id for the same vote, which is
	// what makes this join exact.
	ExtVotingUID string `json:"extVotingUid"`

	// Title is the parliament's label for the kind of vote, e.g. "Abstimmung",
	// "Abstimmung Ausgabenbremse", "Ermittlung der Anwesenden".
	Title string `json:"title"`

	// VotingResult is "yes" or "no" and refers to whether the question carried,
	// not to how any member voted.
	VotingResult string `json:"votingResult"`
}

func (s segment) info() Info {
	voteType, unqualified := voteTypeFromTitle(s.Title)
	return Info{
		Type:            voteType,
		TypeUnqualified: unqualified,
		Decision:        decisionFrom(s.VotingResult),
	}
}

func decisionFrom(result string) string {
	switch strings.TrimSpace(strings.ToLower(result)) {
	case "yes":
		return "angenommen"
	case "no":
		return "abgelehnt"
	default:
		return ""
	}
}

// voteTypeFromTitle maps the archive's label onto the neutral vocabulary, and
// reports whether the label named a ballot type at all.
//
// Matching is on keywords rather than whole strings because the labels are
// editorial free text and vary: attendance alone appears as
// "Anwesenheitsermittlung", "Präsenzermittlung", "Präsenzabstimmung" and
// "Ermittlung der Anwesenden", and cup rounds are numbered ("Cupabstimmung 3").
//
// Order is load-bearing. "Präsenzabstimmung" contains "abstimmung" and would
// otherwise pass as an ordinary vote, which is the single worst outcome here —
// a roll call published as though parliament had decided something.
//
// A bare "Abstimmung" is the common case — 226 of 302 segments — and names no
// ballot type: the Kantonsrat titles the preliminary support of an
// Einzelinitiative that way, and it is a threshold vote. It therefore returns
// TypeNormal as a fallback and unqualified=true, which tells the caller to
// prefer a type it already holds. Reading it as an ordinary Ja/Nein vote is how
// voting 100969 — an Einzelinitiative supported by 41 of 180 — would have been
// published as "41 Ja | 0 Nein | 139 Abwesend".
//
// An unrecognised label maps to "" and is qualified, which leaves the vote
// unpublishable. That is deliberate: a label we have never seen is exactly when
// a tally is most likely to be read wrongly, and staying silent is recoverable
// where a misleading post is not.
func voteTypeFromTitle(title string) (voteType string, unqualified bool) {
	t := strings.ToLower(strings.TrimSpace(title))
	switch {
	case t == "":
		return "", false
	case containsAny(t, "anwesen", "präsenz", "praesenz"):
		return TypeAttendance, false
	case strings.Contains(t, "cup"):
		return TypeCup, false
	// Kept apart from a plain Quorumsabstimmung, though both are counted the
	// same way, because only this one has a name a reader recognises.
	case strings.Contains(t, "ausgabenbremse"):
		return TypeAusgabenbremse, false
	case strings.Contains(t, "quorum"):
		return TypeQuorum, false
	case strings.Contains(t, "abstimmung"):
		return TypeNormal, true
	default:
		return "", false
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
