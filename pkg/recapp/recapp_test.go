package recapp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Agenda items present in testdata, with the archive URL shape the real
// OpenParlData listing publishes.
const (
	sihlItem         = "6e20a24f-3a9e-49ab-a855-269abd8457cd"
	mitteilungenItem = "86a9e704-63f8-41c5-8223-c30eb56ac4bc"
	cupItem          = "ffcd1ff7-fb00-475a-82c8-141b9d5bb054"
)

var fixtures = map[string]string{
	sihlItem:         "segments_sihl.json",
	mitteilungenItem: "segments_mitteilungen.json",
	cupItem:          "segments_cup.json",
}

func archiveURL(agendaItem, segment string) string {
	return "https://zh.recapp.ch/shareparl?agendaItemUid=" + agendaItem + "&segmentUid=" + segment
}

// newTestClient serves recorded responses so the suite never touches the live
// archive: CI must not depend on a third-party service being up, and a test
// must not change its answer because the Kantonsrat sat yesterday.
func newTestClient(t *testing.T) (*Client, *[]string) {
	t.Helper()

	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())

		name, ok := fixtures[r.URL.Query().Get("agendaItemUid")]
		if !ok {
			http.Error(w, "no fixture for "+r.URL.String(), http.StatusNotFound)
			return
		}
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	c := New()
	c.SetBaseURL(srv.URL)
	return c, &requests
}

// TestLookupSeparatesOrdinaryVotesFromThresholdVotes covers the case this
// package exists for. Both votes belong to the same business matter and both
// are plain Ja/Nein tallies to look at, but one is an Ausgabenbremse decided
// against a threshold and the other is not — a distinction OpenParlData served
// as null for the whole 17.08.2026 sitting.
func TestLookupSeparatesOrdinaryVotesFromThresholdVotes(t *testing.T) {
	c, _ := newTestClient(t)

	const (
		ordinary = "8FDBDDFC-C068-420D-476A-F704C08D005B"
		brake    = "088BAC94-081E-08E6-BC6F-4AC3358FC790"
	)

	got, err := c.Lookup(map[string]string{
		ordinary: archiveURL(sihlItem, "d694e78d-5c6c-4d81-9f16-1ca42bc76c54"),
		brake:    archiveURL(sihlItem, "b76547a1-5c7e-4e2a-8669-ab5a6317ac92"),
	})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}

	if got[ordinary].Type != TypeNormal {
		t.Errorf("ordinary vote: got type %q, want %q", got[ordinary].Type, TypeNormal)
	}
	if got[brake].Type != TypeAusgabenbremse {
		t.Errorf("Ausgabenbremse: got type %q, want %q", got[brake].Type, TypeAusgabenbremse)
	}
}

// TestLookupMarksAttendanceRollCalls is the one that protects the account. An
// attendance roll call is not a political vote, but it reports as a lopsided
// Ja tally and would publish as a near-unanimous decision on nothing.
func TestLookupMarksAttendanceRollCalls(t *testing.T) {
	c, _ := newTestClient(t)

	const rollCall = "8AFCE43D-5949-B59B-873C-B2B9A8B75443"

	got, err := c.Lookup(map[string]string{
		rollCall: archiveURL(mitteilungenItem, "22c6950f-6167-469f-b802-35fa630e1f6f"),
	})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got[rollCall].Type != TypeAttendance {
		t.Errorf("roll call: got type %q, want %q", got[rollCall].Type, TypeAttendance)
	}
}

// TestLookupReportsDecision covers what OpenParlData leaves null for every
// Kanton Zürich vote, which is why canton posts never used to state an outcome.
func TestLookupReportsDecision(t *testing.T) {
	c, _ := newTestClient(t)

	const brake = "088BAC94-081E-08E6-BC6F-4AC3358FC790"

	got, err := c.Lookup(map[string]string{
		brake: archiveURL(sihlItem, "b76547a1-5c7e-4e2a-8669-ab5a6317ac92"),
	})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got[brake].Decision != "angenommen" {
		t.Errorf("got decision %q, want %q", got[brake].Decision, "angenommen")
	}
}

// TestLookupFetchesEachAgendaItemOnce guards the request budget. Votes are
// grouped by business matter, so a group's votes almost always share an agenda
// item; fetching per vote would multiply calls to a third-party archive for
// nothing.
func TestLookupFetchesEachAgendaItemOnce(t *testing.T) {
	c, requests := newTestClient(t)

	_, err := c.Lookup(map[string]string{
		"8FDBDDFC-C068-420D-476A-F704C08D005B": archiveURL(sihlItem, "d694e78d-5c6c-4d81-9f16-1ca42bc76c54"),
		"088BAC94-081E-08E6-BC6F-4AC3358FC790": archiveURL(sihlItem, "b76547a1-5c7e-4e2a-8669-ab5a6317ac92"),
	})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(*requests) != 1 {
		t.Errorf("got %d requests for one agenda item, want 1: %v", len(*requests), *requests)
	}
}

// TestLookupIgnoresUnknownVotesAndURLs checks the degradation path. Every other
// body's votes link somewhere that is not the archive, and the archive does not
// list every vote OpenParlData knows about.
func TestLookupIgnoresUnknownVotesAndURLs(t *testing.T) {
	c, _ := newTestClient(t)

	got, err := c.Lookup(map[string]string{
		"elsewhere": "https://www.gemeinderat-zuerich.ch/geschaefte/1234",
		"empty":     "",
		"unlisted":  archiveURL(sihlItem, "00000000-0000-0000-0000-000000000000"),
	})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want no entries for unknown votes, got %v", got)
	}
}

// TestLookupSurvivesAFailedAgendaItem checks that one unreachable agenda item
// does not cost the others their answer. The archive is enrichment; losing it
// must degrade to "unknown type", which is already safe.
func TestLookupSurvivesAFailedAgendaItem(t *testing.T) {
	c, _ := newTestClient(t)

	const known = "8FDBDDFC-C068-420D-476A-F704C08D005B"

	got, err := c.Lookup(map[string]string{
		known:     archiveURL(sihlItem, "d694e78d-5c6c-4d81-9f16-1ca42bc76c54"),
		"missing": archiveURL("no-such-agenda-item", "irrelevant"),
	})
	if err == nil {
		t.Error("want the failed agenda item reported, got nil error")
	}
	if got[known].Type != TypeNormal {
		t.Errorf("healthy vote lost its answer: got %q, want %q", got[known].Type, TypeNormal)
	}
}

// TestVoteTypeFromTitle pins the label vocabulary, including the spelling
// variants the archive actually uses. The attendance cases are the ones that
// matter: "Präsenzabstimmung" contains "abstimmung" and must not fall through
// to an ordinary vote.
func TestVoteTypeFromTitle(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Abstimmung", TypeNormal},
		{"Schlussabstimmung", TypeNormal},
		{"Abstimmung Ausgabenbremse", TypeAusgabenbremse},
		{"Quorumsabstimmung", TypeQuorum},
		{"Anwesenheitsermittlung", TypeAttendance},
		{"Präsenzermittlung", TypeAttendance},
		{"Präsenzabstimmung", TypeAttendance},
		{"Ermittlung der Anwesenden", TypeAttendance},
		{"Cupabstimmung", TypeCup},
		{"Cupabstimmung 3", TypeCup},
		{"Abstimmung Cup-System", TypeCup},
		// Case and padding are editorial noise, not meaning.
		{"  ABSTIMMUNG AUSGABENBREMSE  ", TypeAusgabenbremse},
		// A label we have never seen must stay unpublishable rather than
		// guessing its way into a post.
		{"Wahlgang", ""},
		{"", ""},
	}

	for _, tt := range tests {
		if got := voteTypeFromTitle(tt.title); got != tt.want {
			t.Errorf("voteTypeFromTitle(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}
