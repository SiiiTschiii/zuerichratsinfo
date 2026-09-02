package openparldata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchMembers(t *testing.T) {
	c, _ := newTestClient(t)

	members, err := c.FetchMembers()
	if err != nil {
		t.Fatalf("FetchMembers: %v", err)
	}

	got := make(map[string]bool, len(members))
	for _, m := range members {
		got[m.Name] = true
	}

	// A seat the parliament never closed. The membership still reads active
	// two years on, and only the person's own flag says he has gone.
	if got["Jürg Trachsel"] {
		t.Error("a member whose person record is inactive was counted as sitting")
	}
	// Active as a person — he sits in Bern — but holds no seat in this chamber.
	if got["Gregor Rutz"] {
		t.Error("a person active elsewhere was counted as sitting in this chamber")
	}
	if len(members) != 5 {
		t.Fatalf("got %d members, want the 5 sitting ones: %v", len(members), got)
	}
}

func TestFetchMembers_Affiliation(t *testing.T) {
	c, _ := newTestClient(t)

	members, err := c.FetchMembers()
	if err != nil {
		t.Fatalf("FetchMembers: %v", err)
	}

	byName := make(map[string]int, len(members))
	for i, m := range members {
		byName[m.Name] = i
	}

	// An EDU member sits with the SVP: party and Fraktion are not the same
	// fact, and a curator hunting for the right "Hans Egli" needs the party.
	egli := members[byName["Hans Egli"]]
	if egli.Party != "EDU" || egli.Fraktion != "SVP" {
		t.Errorf("Hans Egli = party %q / Fraktion %q, want EDU / SVP", egli.Party, egli.Fraktion)
	}
	if egli.ProfileURL == "" {
		t.Error("no ProfileURL — it is the first place a curator looks for an account")
	}

	// The source spells the same Fraktion both ways; the trimmed name is what
	// reaches a human either way.
	if walder := members[byName["Patrick Walder"]]; walder.Fraktion != "SVP" {
		t.Errorf(`Patrick Walder Fraktion = %q, want SVP — "SVP-Fraktion" must trim too`, walder.Fraktion)
	}
	if sun := members[byName["Daniela Sun-Güller"]]; sun.Fraktion != "Grünliberale" {
		t.Errorf("Daniela Sun-Güller Fraktion = %q, want Grünliberale", sun.Fraktion)
	}
}

func TestFetchMembers_SortedByName(t *testing.T) {
	c, _ := newTestClient(t)

	members, err := c.FetchMembers()
	if err != nil {
		t.Fatalf("FetchMembers: %v", err)
	}

	for i := 1; i < len(members); i++ {
		if members[i-1].Name > members[i].Name {
			t.Errorf("members are not sorted: %q before %q", members[i-1].Name, members[i].Name)
		}
	}
}

// The roster listings run to hundreds of records, so the paging has to advance
// the offset and stop when the API says there is no more.
func TestPageRoster_PagesUntilExhausted(t *testing.T) {
	var offsets []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		offsets = append(offsets, offset)

		hasMore := offset == "0"
		id := 1
		if !hasMore {
			id = 2
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":[{"person_id":%d,"active":true}],"meta":{"has_more":%t}}`, id, hasMore)
	}))
	t.Cleanup(srv.Close)

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0

	seated, err := c.activeMemberIDs(1)
	if err != nil {
		t.Fatalf("activeMemberIDs: %v", err)
	}

	if len(offsets) != 2 || offsets[0] != "0" || offsets[1] != "500" {
		t.Errorf("offsets = %v, want [0 500]", offsets)
	}
	if !seated[1] || !seated[2] {
		t.Errorf("seated = %v, want both pages collected", seated)
	}
}

// A listing that never reports its end must fail rather than spin against a
// third-party API.
func TestPageRoster_StopsOnEndlessListing(t *testing.T) {
	requests := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"person_id":1,"active":true}],"meta":{"has_more":true}}`))
	}))
	t.Cleanup(srv.Close)

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0

	if _, err := c.activeMemberIDs(1); err == nil {
		t.Fatal("want an error when the listing never ends")
	}
	if requests != rosterMaxPages {
		t.Errorf("made %d requests, want it capped at %d", requests, rosterMaxPages)
	}
}

func TestCouncilGroupID_RequiresALegislativeGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := json.Marshal(map[string]any{
			"data": []map[string]any{
				{"id": 1, "active": true, "type_harmonized": "committee"},
				// The chamber exists but is closed: an old legislative period.
				{"id": 2, "active": false, "type_harmonized": "council_legislative"},
			},
			"meta": map[string]any{"has_more": false},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0

	if _, err := c.councilGroupID(); err == nil {
		t.Fatal("want an error when no active legislative group exists")
	}
}
