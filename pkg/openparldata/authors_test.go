package openparldata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// contributorServer serves one contributor list, counting the requests so the
// caching can be asserted on.
func contributorServer(t *testing.T, records []map[string]any) (*Client, *int) {
	t.Helper()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/contributors") {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}
		calls++
		body, _ := json.Marshal(map[string]any{"data": records, "meta": map[string]any{"has_more": false}})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0
	return c, &calls
}

func contributor(kind, role, name, party string, position int) map[string]any {
	c := map[string]any{
		"type":            kind,
		"role_harmonized": "author",
		"role_de":         role,
		"fullname":        name,
		"position":        position,
	}
	if party == "" {
		c["party_de"] = nil
	} else {
		c["party_de"] = party
	}
	return c
}

const (
	first = "Erstunterzeichnerin / Erstunterzeichner"
	co    = "Mitunterzeichnerin / Mitunterzeichner"
)

func TestAffairAuthors_NamesEverySignatoryLeadFirst(t *testing.T) {
	c, _ := contributorServer(t, []map[string]any{
		contributor("person", co, "Carmen Marty Fässler", "SP", 5),
		contributor("person", co, "Gianna Berger", "AL", 2),
		contributor("person", first, "Daniela Sun-Güller", "GLP", 1),
	})

	authors, err := c.affairAuthors(1)
	if err != nil {
		t.Fatalf("affairAuthors: %v", err)
	}

	want := []string{"Daniela Sun-Güller", "Gianna Berger", "Carmen Marty Fässler"}
	if len(authors) != len(want) {
		t.Fatalf("got %+v, want every signatory named", authors)
	}
	for i, name := range want {
		if authors[i].Name != name {
			t.Errorf("author %d = %q, want %q — first signatory first, then by position",
				i, authors[i].Name, name)
		}
	}
	if authors[0].Party != "GLP" {
		t.Errorf("Party = %q, want GLP", authors[0].Party)
	}
}

// A person the record gives no party is not a member acting in the chamber but
// a private individual who filed an Einzelinitiative. Their name is in the
// official record; it does not go on a social media account.
func TestAffairAuthors_SkipsPeopleWithoutAParty(t *testing.T) {
	c, _ := contributorServer(t, []map[string]any{
		contributor("person", first, "Artur Terekhov", "", 1),
	})

	authors, err := c.affairAuthors(1)
	if err != nil {
		t.Fatalf("affairAuthors: %v", err)
	}
	if len(authors) != 0 {
		t.Errorf("got %+v, want nobody named", authors)
	}
}

// Most cantonal business is filed by the government or a committee. Those are
// organisations on the same list, and tagging them is a separate question.
func TestAffairAuthors_SkipsOrganisations(t *testing.T) {
	c, _ := contributorServer(t, []map[string]any{
		contributor("group", first, "Regierungsrat des Kantons Zürich", "", 1),
		contributor("group", "Direktion", "Baudirektion", "", 2),
	})

	authors, err := c.affairAuthors(1)
	if err != nil {
		t.Fatalf("affairAuthors: %v", err)
	}
	if len(authors) != 0 {
		t.Errorf("got %+v, want no authors for government business", authors)
	}
}

func TestAffairAuthors_OrderedByPosition(t *testing.T) {
	c, _ := contributorServer(t, []map[string]any{
		contributor("person", first, "Second Signatory", "SP", 2),
		contributor("person", first, "First Signatory", "SVP", 1),
	})

	authors, err := c.affairAuthors(1)
	if err != nil {
		t.Fatalf("affairAuthors: %v", err)
	}
	if len(authors) != 2 || authors[0].Name != "First Signatory" {
		t.Errorf("got %+v, want the first signatory first", authors)
	}
}

// A long signatory list is named in full. What a card does when it cannot set
// that many is the card's problem — see imagegen.cardTitle — and no reason for
// the adapter to withhold names from the text posts.
func TestAffairAuthors_NamesLongSignatoryLists(t *testing.T) {
	var records []map[string]any
	records = append(records, contributor("person", first, "Lead Member", "SP", 1))
	for i := 2; i <= 6; i++ {
		records = append(records, contributor("person", co, fmt.Sprintf("Member %d", i), "GLP", i))
	}

	c, _ := contributorServer(t, records)

	authors, err := c.affairAuthors(1)
	if err != nil {
		t.Fatalf("affairAuthors: %v", err)
	}
	if len(authors) != 6 {
		t.Errorf("got %d authors, want all six named", len(authors))
	}
	if authors[0].Name != "Lead Member" {
		t.Errorf("got %q first, want the first signatory", authors[0].Name)
	}
}

// The parliament lists the first signatory at position 1, but the role is the
// field that actually says so, and it is what decides the order.
func TestAffairAuthors_LeadFirstEvenWhenPositionDisagrees(t *testing.T) {
	c, _ := contributorServer(t, []map[string]any{
		contributor("person", co, "Co Signer", "SP", 1),
		contributor("person", first, "The Filer", "SVP", 9),
	})

	authors, err := c.affairAuthors(1)
	if err != nil {
		t.Fatalf("affairAuthors: %v", err)
	}
	if len(authors) != 2 || authors[0].Name != "The Filer" {
		t.Errorf("got %+v, want the first signatory named first", authors)
	}
}

// A Geschäft routinely carries several votes from one sitting, and enrichment
// runs per vote.
func TestAuthorsOf_AsksOncePerAffair(t *testing.T) {
	c, calls := contributorServer(t, []map[string]any{
		contributor("person", first, "Daniela Sun-Güller", "GLP", 1),
	})

	for range 3 {
		if authors := c.authorsOf(42); len(authors) != 1 {
			t.Fatalf("got %+v, want the author on every call", authors)
		}
	}

	if *calls != 1 {
		t.Errorf("made %d requests, want 1 — the rest come from the cache", *calls)
	}
}

// Authorship is enrichment: losing it costs the names, not the post.
func TestAuthorsOf_SurvivesAFailingSource(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "nope", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0

	if authors := c.authorsOf(42); authors != nil {
		t.Errorf("got %+v, want no authors", authors)
	}
	// And the failure is not re-tried once per vote in the group.
	c.authorsOf(42)
	if calls != 1 {
		t.Errorf("made %d requests, want the failure cached too", calls)
	}
}

// The real thing, through the recorded fixtures: every affair the suite covers
// is government or committee business, so none of them names a member.
func TestEnrich_AuthorsFromFixtures(t *testing.T) {
	c, rec := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	groups, err := c.GroupByAffair(vs)
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}

	for _, group := range groups {
		for _, v := range group {
			if len(v.Affair.Authors) != 0 {
				t.Errorf("%q: got authors %+v, want none for government business",
					v.Affair.Title, v.Affair.Authors)
			}
		}
	}

	if n := rec.countMatching("/contributors"); n == 0 {
		t.Error("no contributor request was made — enrichment is not asking for authorship")
	}
}
