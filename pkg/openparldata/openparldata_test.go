package openparldata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

var testJurisdiction = votes.Jurisdiction{
	Key:       "zurich-canton",
	Name:      "Kantonsrat Zürich",
	ShortName: "Kantonsrat",
}

// newTestClient serves recorded responses so the suite never touches the live
// API: CI must not depend on a third-party service being up, and re-running a
// test must not depend on what the Kantonsrat did yesterday.
func newTestClient(t *testing.T) (*Client, *recorder) {
	t.Helper()

	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.requests = append(rec.requests, r.URL.String())

		if r.URL.Query().Get("lang_format") != "flat" {
			// Without lang_format=flat the real API returns null for every
			// localised field, with a 200 and no other signal. Failing loudly
			// here is the only way that stays caught.
			t.Errorf("request without lang_format=flat: %s", r.URL)
		}

		name, ok := fixtureFor(r)
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

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0 // no test should ever wait on backoff
	return c, rec
}

type recorder struct{ requests []string }

func (r *recorder) countMatching(substr string) int {
	n := 0
	for _, req := range r.requests {
		if strings.Contains(req, substr) {
			n++
		}
	}
	return n
}

// fixtureFor dispatches on the request the same way the real API would, so a
// test asserting on one voting cannot accidentally be served another's data.
func fixtureFor(r *http.Request) (string, bool) {
	q := r.URL.Query()
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	switch {
	case len(parts) == 3 && parts[0] == "votings" && parts[2] == "votes":
		return "zh_votes_" + parts[1] + ".json", true
	case len(parts) == 3 && parts[0] == "votings" && parts[2] == "affairs":
		return "zh_affairs_" + parts[1] + ".json", true
	case q.Get("affair_id") != "":
		return "zh_votings_affair_" + q.Get("affair_id") + ".json", true
	case parts[0] == "votings":
		return "zh_votings.json", true
	}
	return "", false
}

func TestFetchRecent(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	if len(vs) != 12 {
		t.Fatalf("got %d votes, want 12", len(vs))
	}

	v := vs[0]
	if v.SourceID != "384F0C84-545C-C0BE-496E-4172BC25E31E" {
		t.Errorf("SourceID = %q", v.SourceID)
	}
	if v.Jurisdiction != "zurich-canton" {
		t.Errorf("Jurisdiction = %q", v.Jurisdiction)
	}
	if got := v.DateString(); got != "2026-07-06" {
		t.Errorf("DateString = %q, want 2026-07-06", got)
	}
	if v.Title != "Tätigkeitsbericht der Finanzkontrolle des Kantons Zürich über das Jahr 2025" {
		t.Errorf("Title = %q", v.Title)
	}
	// Kanton Zürich has no agenda item and repeats the affair title as the
	// voting title; restating it as a subtitle would render "X: X".
	if v.Subtitle != "" {
		t.Errorf("Subtitle = %q, want empty when it merely repeats the title", v.Subtitle)
	}
	if *v.Yes != 167 || *v.No != 0 || *v.Abstention != 0 || *v.Absent != 13 {
		t.Errorf("totals = %d/%d/%d/%d", *v.Yes, *v.No, *v.Abstention, *v.Absent)
	}
	// What the listing carries. The affair call replaces it with the canton's
	// own permalink before anything is posted — see TestAffairURLReplacesBothURLs.
	if !strings.HasPrefix(v.SourceURL, "https://zh.recapp.ch/") {
		t.Errorf("SourceURL = %q", v.SourceURL)
	}

	// FetchRecent must stay a single listing call: enriching here would mean
	// two further calls for every vote that dedup is about to discard.
	if v.MemberVotes != nil {
		t.Error("FetchRecent should not fetch member votes")
	}
}

// Kanton Zürich populates `decision` for no vote at all, and we do not invent
// one. Deriving it from Ja vs Nein looks safe and is not: in a quorum vote Nein
// is structurally always 0, so every one of them derived as "Ja", including
// those that fell short of their threshold. The posts now state the counts and
// no outcome.
func TestDecisionIsNeverDerived(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	if len(vs) == 0 {
		t.Fatal("fixture returned no votes")
	}

	for _, v := range vs {
		if v.Decision != "" {
			t.Errorf("%s: decision %q was invented; the source reports none",
				v.SourceID, v.Decision)
		}
		// An absent decision must not read as a rejection downstream.
		if voteformat.HasVerdict(v) {
			t.Errorf("%s: renders a verdict despite the source reporting none", v.SourceID)
		}
		// An empty decision must never trip the consistency gate, which would
		// abort the whole run rather than merely omit an outcome line.
		if !voteformat.IsDecisionConsistent(v.Decision, v.Yes, v.No) {
			t.Errorf("%s: empty decision treated as inconsistent with its counts", v.SourceID)
		}
	}
}

// A source that does report an outcome keeps it — this is the Stadt Zürich path
// through the same adapter, and the reason decision() is not simply dropped.
func TestReportedDecisionIsPreserved(t *testing.T) {
	for _, reported := range []string{"Ja", "Nein", "Auswahl A"} {
		t.Run(reported, func(t *testing.T) {
			got := decision(votingDTO{Decision: &reported})
			if got != reported {
				t.Errorf("decision() = %q, want %q", got, reported)
			}
		})
	}
}

func TestGroupByAffair(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}

	groups, err := c.GroupByAffair(vs)
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}
	if len(groups) == 0 {
		t.Fatal("expected at least one group")
	}

	for _, g := range groups {
		number := g[0].Affair.Number
		for _, v := range g {
			if v.Affair.Number != number {
				t.Errorf("group mixes affairs %q and %q", number, v.Affair.Number)
			}
			if v.Affair.Number == "" {
				t.Errorf("%s has no affair number; grouping would collapse", v.SourceID)
			}
		}
		// Votes within a group run in the order they were taken.
		for i := 1; i < len(g); i++ {
			if g[i-1].Sequence > g[i].Sequence {
				t.Errorf("group %q is not in chronological order", number)
			}
		}
	}

	// Groups are posted oldest first so a backlog drains chronologically.
	for i := 1; i < len(groups); i++ {
		if groups[i-1][0].Date.After(groups[i][0].Date) {
			t.Errorf("groups are not oldest-first at index %d", i)
		}
	}
}

func TestEnrichmentPopulatesFraktionBreakdown(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	groups, err := c.GroupByAffair(vs)
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}

	// Voting 104481 specifically: the research verified its breakdown against
	// the Kantonsrat's own publication, so it is the one worth pinning.
	v, ok := findVote(groups, "EBA24B53-B404-3BCB-9A1B-4E7E01C1ACAC")
	if !ok {
		t.Fatal("voting 104481 missing from the grouped output")
	}
	if len(v.MemberVotes) != 180 {
		t.Fatalf("got %d member votes, want 180 (the full chamber)", len(v.MemberVotes))
	}

	counts := voteformat.AggregateFraktionCounts(v)

	want := map[string]map[string]int{
		"SVP": {"Ja": 44, "Nein": 0, "Abwesend": 3},
		"SP":  {"Ja": 0, "Nein": 35, "Abwesend": 1},
		"FDP": {"Ja": 27, "Nein": 0, "Abwesend": 3},
		"AL":  {"Ja": 0, "Nein": 5, "Abwesend": 0},
	}
	for fraktion, expected := range want {
		fc, ok := counts[fraktion]
		if !ok {
			t.Errorf("no counts for %s; is the 'Fraktion ' prefix still being stripped?", fraktion)
			continue
		}
		for choice, n := range expected {
			if fc.Counts[choice] != n {
				t.Errorf("%s %s = %d, want %d", fraktion, choice, fc.Counts[choice], n)
			}
		}
	}

	// One member of this chamber has no faction. They must stay out of the
	// table without being dropped from the totals, which come from the source.
	total := 0
	for _, fc := range counts {
		for _, n := range fc.Counts {
			total += n
		}
	}
	if total != 179 {
		t.Errorf("faction table covers %d members, want 179 (180 less the unmapped one)", total)
	}

	// Totals are read from the source, never summed from the member list.
	if sum, _ := v.TotalRecorded(); sum != 180 {
		t.Errorf("reported totals sum to %d, want 180", sum)
	}
	if !votes.IsBreakdownComplete(v) {
		t.Error("a full member list against matching totals should pass the completeness gate")
	}
}

// The affair number is what groups votes, and it is null inline on a voting, so
// losing this call would collapse unrelated votes into one post.
func TestAffairNumberIsFetched(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	if got := vs[0].Affair.Number; !strings.HasPrefix(got, "#") {
		t.Errorf("before enrichment the number should be the id-based fallback, got %q", got)
	}

	groups, err := c.GroupByAffair(vs[:1])
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}
	if got := groups[0][0].Affair.Number; got != "207/2026" {
		t.Errorf("Affair.Number = %q, want 207/2026", got)
	}
	if got := groups[0][0].Affair.URL; !strings.Contains(got, "kantonsrat.zh.ch") {
		t.Errorf("Affair.URL = %q", got)
	}
}

// Posts link to the canton's own Geschäft permalink whether they cover one vote
// or five. The listing gives each vote a zh.recapp.ch deep link keyed by two
// opaque uuids; publishing that for a group of one would make the durability of
// a link depend on how many votes a sitting happened to hold.
func TestAffairURLReplacesBothURLs(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	if !strings.Contains(vs[0].SourceURL, "zh.recapp.ch") {
		t.Fatalf("precondition: listing SourceURL = %q, want the recapp link", vs[0].SourceURL)
	}

	groups, err := c.GroupByAffair(vs[:1])
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}

	got := groups[0][0]
	if !strings.Contains(got.SourceURL, "kantonsrat.zh.ch") {
		t.Errorf("SourceURL = %q, want the Geschäft permalink", got.SourceURL)
	}
	if got.GroupURL != got.SourceURL {
		t.Errorf("GroupURL = %q, SourceURL = %q; both should be the Geschäft permalink", got.GroupURL, got.SourceURL)
	}
}

// Grouping and the enrichment calls both key on the numeric API id. Looking it
// up per vote would double the request count for no benefit.
func TestVotingIDIsRememberedFromTheListing(t *testing.T) {
	c, rec := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	if _, err := c.GroupByAffair(vs[:1]); err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}

	if n := rec.countMatching("external_id="); n != 0 {
		t.Errorf("made %d external_id lookups; the id should come from the listing", n)
	}
}

func findVote(groups [][]votes.Vote, sourceID string) (votes.Vote, bool) {
	for _, g := range groups {
		for _, v := range g {
			if v.SourceID == sourceID {
				return v, true
			}
		}
	}
	return votes.Vote{}, false
}

func TestFraktionName(t *testing.T) {
	tests := map[string]string{
		"Fraktion SVP":           "SVP",
		"Fraktion Grünliberale":  "Grünliberale",
		"SP":                     "SP",
		"Die Mitte/EVP":          "Die Mitte/EVP",
		"":                       "",
		"  Fraktion Grüne  ":     "Grüne",
		"Fraktionslose Mitglied": "Fraktionslose Mitglied",
	}
	for in, want := range tests {
		if got := fraktionName(in); got != want {
			t.Errorf("fraktionName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToMemberVote_ChoiceVocabulary(t *testing.T) {
	// Mapping from the harmonised value rather than the display label is what
	// makes two bodies' tables line up: Kanton Zürich renders an abstention as
	// "Enthalten" where Stadt Zürich renders "Enthaltung".
	enthalten := "Enthalten"
	got := toMemberVote(voteDTO{Vote: "abstention", VoteDisplayDe: &enthalten})
	if got.Choice != "Enthaltung" {
		t.Errorf("Choice = %q, want Enthaltung", got.Choice)
	}

	// An unknown value keeps the member in the table under their own label
	// rather than dropping them and understating their faction.
	president := "Präsidium"
	got = toMemberVote(voteDTO{Vote: "president", VoteDisplayDe: &president})
	if got.Choice != "Präsidium" {
		t.Errorf("Choice = %q, want the display label as a fallback", got.Choice)
	}

	got = toMemberVote(voteDTO{Vote: "something-new"})
	if got.Choice != "something-new" {
		t.Errorf("Choice = %q, want the raw value as a last resort", got.Choice)
	}
}

func TestParseDate(t *testing.T) {
	tests := map[string]string{
		"2026-07-06T13:54:03":  "2026-07-06",
		"2026-07-06":           "2026-07-06",
		"2026-07-06T13:54:03Z": "2026-07-06",
		"":                     "",
		"garbage":              "",
	}
	for in, want := range tests {
		got := parseDate(in)
		var s string
		if !got.IsZero() {
			s = got.Format("2006-01-02")
		}
		if s != want {
			t.Errorf("parseDate(%q) = %q, want %q", in, s, want)
		}
	}
}

func TestGet_RetriesServerErrors(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "upstream hiccup", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0

	var resp votingsResponse
	if err := c.get("/votings/", nil, &resp); err != nil {
		t.Fatalf("get should have succeeded after retrying: %v", err)
	}
	if attempts != 3 {
		t.Errorf("made %d attempts, want 3", attempts)
	}
}

// A 4xx means the request itself is wrong. Retrying cannot fix it and only
// hammers an API that publishes no rate limit.
func TestGet_DoesNotRetryClientErrors(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "no such body", http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0

	var resp votingsResponse
	if err := c.get("/votings/", nil, &resp); err == nil {
		t.Fatal("expected an error")
	}
	if attempts != 1 {
		t.Errorf("made %d attempts, want 1", attempts)
	}
}

// The two sources hand over the same vote in different vocabularies: PARIS
// publishes German labels and bare faction names, OpenParlData publishes
// harmonised values under names like "Fraktion SVP". Each adapter normalises,
// and this pins that the two normalisations converge on identical output.
//
// It matters because both bodies post to the same account: a mismatch would
// split one faction across two rows, or render "Enthalten" and "Enthaltung" as
// separate columns, in adjacent posts in the same feed.
func TestFraktionBreakdownMatchesTheParisAdapter(t *testing.T) {
	// One vote, as each API delivers it.
	paris := []zurichapi.Stimmabgabe{
		{Vorname: "A", Name: "Eins", Fraktion: "SVP", Abstimmungsverhalten: "Ja"},
		{Vorname: "B", Name: "Zwei", Fraktion: "SVP", Abstimmungsverhalten: "Ja"},
		{Vorname: "C", Name: "Drei", Fraktion: "SVP", Abstimmungsverhalten: "Abwesend"},
		{Vorname: "D", Name: "Vier", Fraktion: "SP", Abstimmungsverhalten: "Nein"},
		{Vorname: "E", Name: "Fünf", Fraktion: "SP", Abstimmungsverhalten: "Enthaltung"},
		{Vorname: "F", Name: "Sechs", Abstimmungsverhalten: "Ja"}, // no faction
	}

	opd := []voteDTO{
		newVoteDTO("A Eins", "Fraktion SVP", "yes", "Ja"),
		newVoteDTO("B Zwei", "Fraktion SVP", "yes", "Ja"),
		newVoteDTO("C Drei", "Fraktion SVP", "absent", "Abwesend"),
		newVoteDTO("D Vier", "Fraktion SP", "no", "Nein"),
		// Kanton Zürich says "Enthalten" where Stadt Zürich says "Enthaltung";
		// mapping from the harmonised value is what reconciles them.
		newVoteDTO("E Fünf", "Fraktion SP", "abstention", "Enthalten"),
		newVoteDTO("F Sechs", "", "yes", "Ja"),
	}

	opdMembers := make([]votes.MemberVote, 0, len(opd))
	for _, m := range opd {
		opdMembers = append(opdMembers, toMemberVote(m))
	}

	parisOut := voteformat.FormatFraktionBreakdown(
		voteformat.AggregateFraktionCounts(votes.Vote{MemberVotes: zurichapi.ToMemberVotes(paris)}))
	opdOut := voteformat.FormatFraktionBreakdown(
		voteformat.AggregateFraktionCounts(votes.Vote{MemberVotes: opdMembers}))

	if parisOut != opdOut {
		t.Errorf("the same vote must render identically whichever source produced it:\n--- PARIS ---\n%s\n--- OpenParlData ---\n%s", parisOut, opdOut)
	}
	if !strings.Contains(parisOut, "Ja/Nein/Enth/Abw") {
		t.Errorf("expected the standard four columns, got:\n%s", parisOut)
	}
	// The member with no faction is omitted from both, not bucketed under "".
	if strings.Count(parisOut, "\n") != 2 {
		t.Errorf("expected a header and two faction rows, got:\n%s", parisOut)
	}
}

func newVoteDTO(name, group, vote, display string) voteDTO {
	d := voteDTO{Vote: vote, PersonFullname: &name, VoteDisplayDe: &display}
	if group != "" {
		d.PersonParliamentaryGroupNameDe = &group
	}
	return d
}

// Group completion runs after the age guard has already filtered the input, so
// anything it adds bypasses that guard. A Kantonsrat business matter routinely
// spans years — affair 313093 has votes from 2022 through 2026 — and pulling
// its old votes back in here would re-publish history that the guard exists to
// suppress. Completion must therefore stay within sitting days already in play.
func TestCompleteGroupsStaysWithinTheSittingDay(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}

	// One vote from the affair whose recorded listing spans several years.
	var seed votes.Vote
	for _, v := range vs {
		if v.Affair.ID == "313093" {
			seed = v
			break
		}
	}
	if seed.SourceID == "" {
		t.Fatal("fixture no longer contains a vote from affair 313093")
	}

	completed, err := c.completeGroups([]votes.Vote{seed})
	if err != nil {
		t.Fatalf("completeGroups: %v", err)
	}

	if len(completed) <= 1 {
		t.Error("completion should have pulled in the affair's other votes from the same sitting")
	}
	for _, v := range completed {
		if got := v.DateString(); got != seed.DateString() {
			t.Errorf("completion added a vote from %s; only %s was in play, and older votes have already passed the age guard",
				got, seed.DateString())
		}
	}
}

// A single affair can carry more votings than one page holds — the largest
// Kanton Zürich affair already has 90, and a budget debate could exceed 100. An
// unpaged call would silently drop the rest of a long debate from its post.
//
// The early exit depends on the listing being newest-first, so the request must
// set the sort explicitly rather than inherit whatever the API defaults to.
func TestCompleteGroupsPagesAndStopsAtOlderVotings(t *testing.T) {
	const (
		wantedDay = "2026-07-06"
		olderDay  = "2026-05-04"
		onWanted  = 150 // more than one page
	)

	var requested []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.RawQuery)

		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		// Newest-first: the wanted day, then a long tail of older votings.
		var all []map[string]any
		for i := range onWanted {
			all = append(all, votingJSON(i, wantedDay))
		}
		for i := range 400 {
			all = append(all, votingJSON(1000+i, olderDay))
		}

		end := min(offset+limit, len(all))
		if offset > len(all) {
			offset = len(all)
		}
		writeJSON(w, map[string]any{"data": all[offset:end]})
	}))
	defer srv.Close()

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0

	seed := votes.Vote{
		SourceID:     "seed",
		Jurisdiction: testJurisdiction.Key,
		Date:         parseDate(wantedDay),
		Affair:       votes.Affair{ID: "42", Number: "#42"},
	}

	completed, err := c.completeGroups([]votes.Vote{seed})
	if err != nil {
		t.Fatalf("completeGroups: %v", err)
	}

	if got := len(completed); got != onWanted+1 {
		t.Errorf("completed %d votes, want %d — a long debate must not be truncated at one page", got, onWanted+1)
	}
	for _, v := range completed {
		if v.DateString() != wantedDay {
			t.Errorf("pulled in a vote from %s; only %s was in play", v.DateString(), wantedDay)
		}
	}

	// Two pages cover the 150 wanted votings; the second contains older ones,
	// so paging must stop there rather than walking the whole affair.
	if len(requested) != 2 {
		t.Errorf("made %d requests, want 2 (stop as soon as results predate the day in play)", len(requested))
	}
	for _, q := range requested {
		if !strings.Contains(q, "sort_by=-date") {
			t.Errorf("request %q does not set an explicit sort; the early exit relies on newest-first", q)
		}
	}
}

func votingJSON(i int, day string) map[string]any {
	return map[string]any{
		"id":          i,
		"external_id": fmt.Sprintf("ext-%d", i),
		"date":        day + "T10:00:00",
		"affair_id":   42,
		"results_yes": 100,
		"results_no":  20,
		"title_de":    "Titel",
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// OpenParlData publishes timestamps in UTC with no zone marker, while the
// Kantonsrat's own archive lists them in local time: a vote the archive shows
// at 11:29 arrives here as 09:29. Posts carry that time so a reader can find
// the vote in the archive, so being an hour or two out is worse than showing
// nothing — it sends them to the wrong entry.
func TestDatesAreConvertedToLocalTime(t *testing.T) {
	tests := []struct {
		name, apiDate, wantLocal string
	}{
		// Verified against the archive's own segment timestamps.
		{"summer, CEST is UTC+2", "2026-06-15T09:29:12", "11:29"},
		{"winter, CET is UTC+1", "2026-02-23T16:16:46", "17:16"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDate(tc.apiDate)
			if got.Format("15:04") != tc.wantLocal {
				t.Errorf("parseDate(%q) renders %q, want %q", tc.apiDate, got.Format("15:04"), tc.wantLocal)
			}
			// The instant itself must not move; only the wall clock it is read in.
			if !got.UTC().Equal(mustParseUTC(t, tc.apiDate)) {
				t.Errorf("parseDate(%q) changed the instant to %v", tc.apiDate, got.UTC())
			}
		})
	}
}

// A vote taken late in a sitting must not be grouped under the previous day,
// which is what reading a local evening as UTC would do.
func TestLateVoteKeepsItsSittingDay(t *testing.T) {
	v := parseDate("2026-06-15T22:30:00") // 00:30 next day, Zurich
	if got := v.Format("2006-01-02 15:04"); got != "2026-06-16 00:30" {
		t.Errorf("got %q; the conversion should carry the vote into the next local day", got)
	}
}

func mustParseUTC(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// Votes are grouped on business number plus sitting day, so a vote with no
// business number would join every other such vote of that day into a single
// post claiming they are one matter. 47 Kanton Zürich votings have no
// affair_id, eight of them on one day.
func TestVotesWithoutAnAffairDoNotGroupTogether(t *testing.T) {
	c := New(testJurisdiction, "ZH")

	sameDay := func(externalID string) votes.Vote {
		return c.toVote(votingDTO{
			ExternalID: externalID,
			Date:       "2025-11-24T10:00:00",
			AffairID:   nil,
		})
	}

	a, b := sameDay("AAA-111"), sameDay("BBB-222")

	if a.Affair.Number == "" || b.Affair.Number == "" {
		t.Fatal("a vote with no affair must still carry a grouping number")
	}
	if a.Affair.Number == b.Affair.Number {
		t.Fatalf("both votes got grouping number %q; unrelated votes would be posted as one matter", a.Affair.Number)
	}

	if groups := votes.GroupByAffairAndDate([]votes.Vote{a, b}); len(groups) != 2 {
		t.Errorf("got %d group(s), want 2 — these votes have nothing to do with each other", len(groups))
	}
}

// The affair title and the voting title are filled in by different people and
// disagree on typography. Comparing them literally let «Verkehr» through as a
// subtitle against "Verkehr" as the headline, so every sub-vote of the
// Richtplan post reprinted the headline it already carried.
func TestSubtitleIsDroppedDespiteDifferentQuoteStyle(t *testing.T) {
	const affair = `Teilrevision 2022 des kantonalen Richtplans, Kapitel 4 "Verkehr"`

	tests := []struct {
		name   string
		voting string
		want   bool
	}{
		{"identical", affair, true},
		{"guillemets against straight quotes", `Teilrevision 2022 des kantonalen Richtplans, Kapitel 4 «Verkehr»`, true},
		{"curly against straight quotes", `Teilrevision 2022 des kantonalen Richtplans, Kapitel 4 “Verkehr”`, true},
		{"extra whitespace", `Teilrevision 2022 des  kantonalen Richtplans, Kapitel 4 "Verkehr" `, true},
		{"different case", `teilrevision 2022 des kantonalen richtplans, kapitel 4 "verkehr"`, true},
		// A genuinely different subtitle must survive: dropping it would lose
		// the only per-vote information the source ever supplies.
		{"different chapter", `Teilrevision 2022 des kantonalen Richtplans, Kapitel 5 "Verkehr"`, false},
		{"real subtitle", "Schlussabstimmung", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameHeadline(tc.voting, affair); got != tc.want {
				t.Errorf("sameHeadline(%q, affair) = %v, want %v", tc.voting, got, tc.want)
			}
		})
	}
}

// TestAffairsResponseAcceptsBareArray covers a shape the API actually serves.
//
// A voting that has an affair comes back in the documented {"data": [...]}
// envelope; a voting that has none comes back as a bare [], which is a correct
// empty answer in a different shape. Decoding only the envelope turned that into
// a decode error logged on every single run.
func TestAffairsResponseAcceptsBareArray(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"bare empty array", `[]`, 0},
		{"bare populated array", `[{"id":1,"number":"6040"}]`, 1},
		{"documented envelope", `{"data":[{"id":1,"number":"6040"}],"meta":{}}`, 1},
		{"envelope with no rows", `{"data":[],"meta":{}}`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp affairsResponse
			if err := json.Unmarshal([]byte(tt.body), &resp); err != nil {
				t.Fatalf("decoding %s: %v", tt.body, err)
			}
			if len(resp.Data) != tt.want {
				t.Errorf("got %d affairs, want %d", len(resp.Data), tt.want)
			}
		})
	}
}

// stubDetails is a DetailSource that answers from a fixed table and records
// what it was asked about.
type stubDetails struct {
	byVoting map[string]VoteDetail
	err      error
	calls    int
	asked    map[string]string
}

func (s *stubDetails) Lookup(voteURLs map[string]string) (map[string]VoteDetail, error) {
	s.calls++
	s.asked = voteURLs
	return s.byVoting, s.err
}

// TestGroupByAffairAppliesDetailTypes is the reason the DetailSource hook
// exists. The API served the whole 17.08.2026 sitting with a null type, and two
// of those votes were Ausgabenbremse votes decided against a threshold.
func TestGroupByAffairAppliesDetailTypes(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(2)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	// The fixture is a real listing, so start from the state that broke:
	// no type at all.
	for i := range vs {
		vs[i].Type = ""
	}

	details := &stubDetails{byVoting: map[string]VoteDetail{
		vs[0].SourceID: {Type: "Quorum", Decision: "angenommen"},
	}}
	c.WithDetails(details)

	groups, err := c.GroupByAffair(vs)
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}

	found, ok := findVote(groups, vs[0].SourceID)
	if !ok {
		t.Fatalf("vote %s vanished from the grouping", vs[0].SourceID)
	}
	if found.Type != "Quorum" {
		t.Errorf("Type = %q, want %q", found.Type, "Quorum")
	}
	if found.Decision != "angenommen" {
		t.Errorf("Decision = %q, want %q", found.Decision, "angenommen")
	}

	// A vote the source says nothing about keeps what the API gave it.
	untouched, ok := findVote(groups, vs[1].SourceID)
	if ok && untouched.Type != "" {
		t.Errorf("unmentioned vote got type %q, want it left alone", untouched.Type)
	}
}

// TestGroupByAffairDetailTypeOverridesTheAPI pins that the detail source wins a
// disagreement. The API's type_de is not merely incomplete for Kanton Zürich —
// in a 94-vote sample three votes typed "Quorum" were ordinary Abstimmungen and
// one typed "Normal" was a Quorumsabstimmung.
func TestGroupByAffairDetailTypeOverridesTheAPI(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(1)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	vs[0].Type = "Normal"

	c.WithDetails(&stubDetails{byVoting: map[string]VoteDetail{
		vs[0].SourceID: {Type: "Quorum"},
	}})

	groups, err := c.GroupByAffair(vs)
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}
	if found, ok := findVote(groups, vs[0].SourceID); !ok || found.Type != "Quorum" {
		t.Errorf("the API's type won; got type %q", found.Type)
	}
}

// TestGroupByAffairSurvivesADetailSourceFailure pins that the archive is
// enrichment. Losing it must degrade to "unknown type", which the posting
// pipeline already refuses to publish, rather than losing the run.
func TestGroupByAffairSurvivesADetailSourceFailure(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(1)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}

	c.WithDetails(&stubDetails{err: fmt.Errorf("archive unreachable")})

	if _, err := c.GroupByAffair(vs); err != nil {
		t.Fatalf("a failed detail lookup must not fail the fetch, got %v", err)
	}
}

// TestGroupByAffairAsksTheDetailSourceForTheVotesOwnURL guards the ordering the
// lookup depends on. Enrichment replaces Vote.SourceURL with the affair's page,
// so a lookup keyed off the vote as it ends up would identify nothing.
func TestGroupByAffairAsksTheDetailSourceForTheVotesOwnURL(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(1)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}
	own := vs[0].SourceURL
	if own == "" {
		t.Fatal("fixture vote has no source URL to check against")
	}

	details := &stubDetails{}
	c.WithDetails(details)

	groups, err := c.GroupByAffair(vs)
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}
	if got := details.asked[vs[0].SourceID]; got != own {
		t.Errorf("detail source was asked about %q, want the vote's own URL %q", got, own)
	}
	// And the vote really does end up pointing somewhere else, or this test
	// would pass for the wrong reason.
	if found, ok := findVote(groups, vs[0].SourceID); ok && found.SourceURL == own {
		t.Log("note: enrichment left SourceURL unchanged for this fixture")
	}
}

// TestDetailVerdictOnlyWhereItIsHonest is the guard against overstating what a
// vote achieved.
//
// A Vorlage that carries is adopted, so "angenommen" is true of it. An
// Einzelinitiative that carries is only *vorläufig unterstützt* and passes to
// the Regierungsrat — the Kantonsrat's own record for the 17.08.2026
// Sprungbeschwerde reads "Vorläufig unterstützt (79 Stimmen)" and still lists
// the business as pending, so a post saying "Angenommen" would tell a reader it
// had passed into law.
func TestDetailVerdictOnlyWhereItIsHonest(t *testing.T) {
	tests := []struct {
		affairType  string
		wantVerdict bool
	}{
		{"Vorlage", true},
		{"Rechenschaftsbericht", true},
		{"Geschäftsbericht", true},
		{"Tätigkeitsbericht", true},
		// A Ja here means provisional support, not adoption.
		{"Einzelinitiative", false},
		{"Parlamentarische Initiative", false},
		// Überwiesen rather than angenommen.
		{"Motion", false},
		{"Postulat", false},
		{"Wahl", false},
		// No affair at all: a procedural motion with nothing to state.
		{"", false},
		// A kind of business we have not classified stays silent.
		{"Behördeninitiative", false},
	}

	for _, tt := range tests {
		t.Run(tt.affairType, func(t *testing.T) {
			if got := affairStatesItsOutcome(tt.affairType); got != tt.wantVerdict {
				t.Errorf("affairStatesItsOutcome(%q) = %v, want %v",
					tt.affairType, got, tt.wantVerdict)
			}
		})
	}
}

// TestGroupByAffairWithholdsVerdictFromAnInitiative runs the gate through the
// real pipeline, so a future reordering that resolved details before the affair
// type was known would fail here rather than in production.
func TestGroupByAffairWithholdsVerdictFromAnInitiative(t *testing.T) {
	c, _ := newTestClient(t)

	vs, err := c.FetchRecent(12)
	if err != nil {
		t.Fatalf("FetchRecent: %v", err)
	}

	// Offer a verdict for every vote; the gate decides which may keep it.
	byVoting := make(map[string]VoteDetail, len(vs))
	for _, v := range vs {
		byVoting[v.SourceID] = VoteDetail{Type: "Normal", Decision: "angenommen"}
	}
	c.WithDetails(&stubDetails{byVoting: byVoting})

	groups, err := c.GroupByAffair(vs)
	if err != nil {
		t.Fatalf("GroupByAffair: %v", err)
	}

	var checkedWithheld, checkedKept bool
	for _, g := range groups {
		for _, v := range g {
			switch {
			case v.Affair.Type == "Wahl":
				// An election states no accept/reject outcome.
				if v.Decision != "" {
					t.Errorf("vote %s (%s): Decision = %q, want none",
						v.SourceID, v.Affair.Type, v.Decision)
				}
				checkedWithheld = true
			case affairStatesItsOutcome(v.Affair.Type):
				if v.Decision != "angenommen" {
					t.Errorf("vote %s (%s): Decision = %q, want %q",
						v.SourceID, v.Affair.Type, v.Decision, "angenommen")
				}
				checkedKept = true
			}
			// The type is never affected by the verdict gate.
			if v.Type != "Normal" {
				t.Errorf("vote %s: Type = %q, want the detail source's %q", v.SourceID, v.Type, "Normal")
			}
		}
	}
	if !checkedWithheld || !checkedKept {
		t.Fatalf("fixtures no longer cover both sides of the gate (withheld=%v kept=%v)",
			checkedWithheld, checkedKept)
	}
}
