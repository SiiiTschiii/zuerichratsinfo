package openparldata

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// bulletinServer serves the two routes the Bulletin lookup walks, from bodies
// keyed by path. A path with no body answers 404, which is also how a fetch
// failure is simulated.
func bulletinServer(t *testing.T, bodies map[string]string) (*Client, *recorder) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.requests = append(rec.requests, r.URL.String())
		body, ok := bodies[r.URL.Path]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c := New(testJurisdiction, "ZH")
	c.SetBaseURL(srv.URL)
	c.retryDelay = 0
	return c, rec
}

// The shapes of the real responses for affair 337845 and meeting 33709, with an
// older sitting of the same business added: a Kantonsrat business is on the
// agenda of sittings years apart, and only the one the vote was taken in has
// the right Bulletin.
const (
	meetingsTwoSittings = `{"data":[
		{"id":30001,"begin_date":"2024-03-11T08:15:00"},
		{"id":33709,"begin_date":"2026-09-14T08:15:00"}]}`
	docsWithBulletin = `{"data":[
		{"category_de":"Traktandenliste","url":"https://example.test/traktandenliste.pdf"},
		{"category_de":"Bulletin","url":"https://example.test/bulletin-2026-09-14.pdf"},
		{"category_de":"Vorschau","url":"https://example.test/vorschau.pdf"}]}`
	docsOlderSitting = `{"data":[
		{"category_de":"Bulletin","url":"https://example.test/bulletin-2024-03-11.pdf"}]}`
)

var sittingDay = time.Date(2026, 9, 14, 8, 19, 39, 0, time.Local)

func TestBulletinURL(t *testing.T) {
	tests := []struct {
		name   string
		bodies map[string]string
		want   string
	}{
		{
			name: "the sitting on the vote's day, and its Bulletin rather than the first document",
			bodies: map[string]string{
				"/affairs/337845/meetings": meetingsTwoSittings,
				"/meetings/33709/docs":     docsWithBulletin,
				"/meetings/30001/docs":     docsOlderSitting,
			},
			want: "https://example.test/bulletin-2026-09-14.pdf",
		},
		{
			name: "no sitting on the vote's day",
			bodies: map[string]string{
				"/affairs/337845/meetings": `{"data":[{"id":30001,"begin_date":"2024-03-11T08:15:00"}]}`,
				"/meetings/30001/docs":     docsOlderSitting,
			},
			want: "",
		},
		{
			name: "the sitting has no Bulletin filed yet",
			bodies: map[string]string{
				"/affairs/337845/meetings": meetingsTwoSittings,
				"/meetings/33709/docs":     `{"data":[{"category_de":"Traktandenliste","url":"https://example.test/traktandenliste.pdf"}]}`,
			},
			want: "",
		},
		{
			name:   "the meetings call fails",
			bodies: map[string]string{},
			want:   "",
		},
		{
			name: "the documents call fails",
			bodies: map[string]string{
				"/affairs/337845/meetings": meetingsTwoSittings,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := bulletinServer(t, tt.bodies)
			if got := c.bulletinURL(337845, sittingDay); got != tt.want {
				t.Errorf("bulletinURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestApplyBulletinsOnlyAsksForElectionRollCalls pins that the lookup is
// confined to the votes whose post needs it. Every other vote keeps the sitting
// link and costs no extra request.
func TestApplyBulletinsOnlyAsksForElectionRollCalls(t *testing.T) {
	c, rec := bulletinServer(t, map[string]string{
		"/affairs/337845/meetings": meetingsTwoSittings,
		"/meetings/33709/docs":     docsWithBulletin,
	})

	vs := []votes.Vote{
		{
			SourceID: "roll-call",
			Date:     sittingDay,
			Type:     votes.AttendanceType,
			Affair:   votes.Affair{ID: "337845", Type: votes.WahlAffairType},
		},
		{
			SourceID: "ordinary-vote",
			Date:     sittingDay,
			Type:     "Normal",
			Affair:   votes.Affair{ID: "91382", Type: "Motion"},
		},
		{
			// The routine roll call that opens a sitting belongs to no
			// election business, so it must not trigger a lookup either.
			SourceID: "opening-roll-call",
			Date:     sittingDay,
			Type:     votes.AttendanceType,
		},
	}

	c.applyBulletins(vs)

	if vs[0].BulletinURL != "https://example.test/bulletin-2026-09-14.pdf" {
		t.Errorf("election roll call: BulletinURL = %q", vs[0].BulletinURL)
	}
	for _, v := range vs[1:] {
		if v.BulletinURL != "" {
			t.Errorf("%s: BulletinURL = %q, want none", v.SourceID, v.BulletinURL)
		}
	}
	for _, req := range rec.requests {
		if !strings.Contains(req, "/affairs/337845/") && !strings.Contains(req, "/meetings/33709/") {
			t.Errorf("unexpected request %s", req)
		}
	}
}
