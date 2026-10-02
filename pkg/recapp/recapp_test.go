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
	initiativeItem   = "d5414a84-7da0-403e-81ad-8081ebc3f219"
	praesenzItem     = "96e0b324-74e7-484f-bb4a-2e4d906cb545"
)

var fixtures = map[string]string{
	sihlItem:         "segments_sihl.json",
	mitteilungenItem: "segments_mitteilungen.json",
	cupItem:          "segments_cup.json",
	initiativeItem:   "segments_einzelinitiative.json",
	praesenzItem:     "segments_praesenz.json",
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

// TestLookupReadsThresholdVotesFromTheScheme is the bug that made the scheme
// worth reading. Voting 100969, the preliminary support of an Einzelinitiative,
// is titled only "Abstimmung" and ran as a "quorum" ballot; it failed with 41
// of the 60 members it needed. Typed from its title it read as
// "41 Ja | 0 Nein | 139 Abwesend" — unanimous approval of something that fell.
func TestLookupReadsThresholdVotesFromTheScheme(t *testing.T) {
	c, _ := newTestClient(t)

	const initiative = "BF99B79E-4CE7-1E27-2B42-926C87E3E9B9"

	got, err := c.Lookup(map[string]string{
		initiative: archiveURL(initiativeItem, "a30add5b-c490-48c0-93d7-7777510c8317"),
	})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got[initiative].Type != TypeQuorum {
		t.Errorf("Einzelinitiative: got type %q, want %q", got[initiative].Type, TypeQuorum)
	}
}

// TestLookupKeepsRollCallsOverTheScheme is the reverse, and the reason the
// title is still read first. Präsenzabstimmung 101308 ran as a "quorum" ballot,
// and trusting the scheme would publish the roll call as a threshold vote.
func TestLookupKeepsRollCallsOverTheScheme(t *testing.T) {
	c, _ := newTestClient(t)

	const (
		rollCall = "36188051-92F9-87F4-5D7B-B7DCD27BE7DF"
		quorum   = "3C90F992-ED88-2470-AFAA-F9B441B69B0D"
	)

	got, err := c.Lookup(map[string]string{
		rollCall: archiveURL(praesenzItem, "b86a166b-f485-41a3-a23d-8531969067f3"),
		quorum:   archiveURL(praesenzItem, "8a3813fb-51e1-42b1-ac5e-2e7c123e058b"),
	})
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got[rollCall].Type != TypeAttendance {
		t.Errorf("roll call: got type %q, want %q", got[rollCall].Type, TypeAttendance)
	}
	if got[quorum].Type != TypeQuorum {
		t.Errorf("quorum vote beside it: got type %q, want %q", got[quorum].Type, TypeQuorum)
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

// TestVoteType pins how scheme and title combine: the title can make a type
// stricter than the scheme, never looser. Every row with a scheme is a pairing
// the 300 most recent ZH votings actually contain.
func TestVoteType(t *testing.T) {
	tests := []struct {
		title, scheme string
		want          string
	}{
		// A title naming no ballot type yields to the scheme.
		{"Abstimmung", "binary", TypeNormal},
		{"Abstimmung", "quorum", TypeQuorum},
		{"Abstimmung über Rückkommen", "quorum", TypeQuorum},
		{"Schlussabstimmung", "binary", TypeNormal},
		// A title naming one wins, whatever the scheme says.
		{"Präsenzabstimmung", "quorum", TypeAttendance},
		{"Präsenzermittlung", "quorum", TypeAttendance},
		{"Cupabstimmung 2", "binary", TypeCup},
		{"Abstimmung Ausgabenbremse", "binary", TypeAusgabenbremse},
		{"Abstimmung Ausgabenbremse", "quorum", TypeAusgabenbremse},
		{"Quorumsabstimmung", "binary", TypeQuorum},
		{"Cupabstimmung", "cup", TypeCup},
		// No scheme: the title alone, as before. Roll calls since 2023 arrive
		// like this.
		{"Anwesenheitsermittlung", "", TypeAttendance},
		{"Quorumsabstimmung", "", TypeQuorum},
		{"Abstimmung", "", TypeNormal},
		{"Wahlgang", "", ""},
		// An unrecognised title is not rescued by the scheme. Voting 99904
		// is titled with its business matter and records 8 Ja to 0 with 172
		// absent; the scheme calling it binary does not make that a result.
		{"Bargeldannahmepflicht im Kanton Zürich", "binary", ""},
		// A scheme nobody has mapped is a structured signal disagreeing with a
		// plain title, so the vote stays unpublishable.
		{"Abstimmung", "weighted", ""},
	}

	for _, tt := range tests {
		if got := voteType(tt.title, tt.scheme); got != tt.want {
			t.Errorf("voteType(%q, %q) = %q, want %q", tt.title, tt.scheme, got, tt.want)
		}
	}
}

func TestItemURL(t *testing.T) {
	const item = "https://zh.recapp.ch/shareparl?agendaItemUid=c1c8edec-fde9-47ac-884a-987165474821"

	got := ItemURL("https://zh.recapp.ch/shareparl?agendaItemUid=c1c8edec-fde9-47ac-884a-987165474821" +
		"&segmentUid=c838b2b9-f98b-49e2-8843-81adb1b5b3f3")
	if got != item {
		t.Errorf("ItemURL() = %q, want the segment dropped: %q", got, item)
	}

	for _, in := range []string{
		"",
		"https://zh.recapp.ch/shareparl?segmentUid=c838b2b9-f98b-49e2-8843-81adb1b5b3f3",
		"https://www.kantonsrat.zh.ch/geschaefte/geschaeft/?id=374e1a29c38343e8b4ee2ef8acb6ed0c",
	} {
		if got := ItemURL(in); got != "" {
			// Returning the input, or a bare shareparl URL, would publish a
			// link to the archive's front door as if it named this vote.
			t.Errorf("ItemURL(%q) = %q, want no link at all", in, got)
		}
	}
}
