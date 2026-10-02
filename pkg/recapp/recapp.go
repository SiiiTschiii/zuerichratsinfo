// Package recapp adapts the Kantonsrat Zürich audio archive (zh.recapp.ch) to
// the vote information OpenParlData omits.
//
// It exists because OpenParlData's type_de is incomplete for Kanton Zürich —
// whole sittings arrive with a null type, the 17.08.2026 sitting having served
// all five of its votes that way — and because it cannot express distinctions
// the parliament makes: the attendance roll call, which it publishes as an
// ordinary or a quorum voting, and the Ausgabenbremse, which it folds into
// "Quorum" or "Normal" depending on how the ballot was run.
//
// The archive is what OpenParlData harvests from, and each vote segment
// carries two signals. votingScheme is structured and is what type_de is
// derived from: over the 300 most recent ZH votings on 2026-09-30 the two
// agreed in every case where both were set. The segment title is editorial free
// text, and loose — 219 of those 300 read only "Abstimmung", the preliminary
// support of an Einzelinitiative included, which is a threshold vote. See
// voteType for how the two are combined.
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

// shareBaseURL is the archive's reader-facing viewer, which is a different host
// path from the API above and is never redirected to a test server: ItemURL
// produces a link for publication, not a request this package makes.
const shareBaseURL = "https://zh.recapp.ch/shareparl"

// ItemURL widens a vote's archive URL to the agenda item that holds it.
//
// The URL the source publishes for a vote names two things — the agenda item
// and the one segment within it where that vote was taken. Dropping the segment
// leaves a page covering the whole item: the same debate, opened at the top
// instead of at one tally. That is what a post covering several votes of one
// business matter should link to, because linking to the first vote's segment
// would be arbitrary and linking to none of them loses the archive entirely.
//
// Returns "" for a URL that names no agenda item, which callers must treat as
// "no such page" rather than substituting the vote's own link.
func ItemURL(voteURL string) string {
	uid := agendaItemUID(voteURL)
	if uid == "" {
		return ""
	}
	q := url.Values{}
	q.Set("agendaItemUid", uid)
	return shareBaseURL + "?" + q.Encode()
}

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
	// used a label or a votingScheme this package does not recognise.
	Type string
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

	// VotingScheme is how the ballot was run: "binary", "quorum" or "cup".
	// Absent on segments before 11.09.2023 and on some roll calls since.
	VotingScheme string `json:"votingScheme"`

	// VotingResult is "yes" or "no" and refers to whether the question carried,
	// not to how any member voted.
	VotingResult string `json:"votingResult"`
}

func (s segment) info() Info {
	return Info{
		Type:     voteType(s.Title, s.VotingScheme),
		Decision: decisionFrom(s.VotingResult),
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

// voteType combines a segment's scheme and title into one type.
//
// The scheme decides, and the title can only make the type stricter than the
// scheme says, never looser. A title that names a kind of ballot — attendance,
// cup, Ausgabenbremse, quorum — wins, because each of those occurs under a
// scheme that would misrepresent it: Präsenzabstimmung 101308 ran as "quorum",
// Cupabstimmung 2 on 15.12.2025 as "binary", the two Ausgabenbremse votes of
// 17.08.2026 as "binary". A generic title — "Abstimmung", "Schlussabstimmung" —
// yields to the scheme. Reading "Abstimmung" as an ordinary vote over a
// "quorum" scheme is how voting 100969, an Einzelinitiative supported by 41 of
// 180, would have published as "41 Ja | 0 Nein | 139 Abwesend".
//
// A title this package does not recognise stays unrecognised whatever the
// scheme: of the three recent ones, voting 99904 is titled with its business
// matter, ran as "binary", and records 8 Ja to 0 with 172 absent — not a tally
// to publish as a decision on the strength of the scheme alone. Nor does a
// scheme this package does not know fall back on a generic title. Without any
// scheme the title is all there is and is read as before.
func voteType(title, scheme string) string {
	fromTitle := voteTypeFromTitle(title)
	if fromTitle != TypeNormal {
		return fromTitle
	}
	switch strings.TrimSpace(scheme) {
	case "", schemeBinary:
		return TypeNormal
	case schemeQuorum:
		return TypeQuorum
	case schemeCup:
		return TypeCup
	default:
		return ""
	}
}

// The archive's votingScheme values. OpenParlData checked all 1568 Kantonsrat
// agenda items and found exactly these three.
const (
	schemeBinary = "binary"
	schemeQuorum = "quorum"
	schemeCup    = "cup"
)

// voteTypeFromTitle maps the archive's label onto the neutral vocabulary.
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
// An unrecognised label maps to "", which leaves the vote unpublishable. That
// is deliberate: a label we have never seen is exactly when a tally is most
// likely to be read wrongly, and staying silent is recoverable where a
// misleading post is not.
func voteTypeFromTitle(title string) string {
	t := strings.ToLower(strings.TrimSpace(title))
	switch {
	case t == "":
		return ""
	case containsAny(t, "anwesen", "präsenz", "praesenz"):
		return TypeAttendance
	case strings.Contains(t, "cup"):
		return TypeCup
	// Kept apart from a plain Quorumsabstimmung, though both are counted the
	// same way, because only this one has a name a reader recognises.
	case strings.Contains(t, "ausgabenbremse"):
		return TypeAusgabenbremse
	case strings.Contains(t, "quorum"):
		return TypeQuorum
	case strings.Contains(t, "abstimmung"):
		return TypeNormal
	default:
		return ""
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
