package fbapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAccountURL(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want Account
	}{
		{"https://www.facebook.com/fdpstadtzuerich/", Account{Username: "fdpstadtzuerich"}},
		{"https://www.facebook.com/alice.kohli", Account{Username: "alice.kohli"}},
		{"facebook.com/ALzuerich/", Account{}}, // no scheme: not a URL host
		{"https://www.facebook.com/profile.php?id=100070693802425", Account{ID: "100070693802425"}},
		{"https://m.facebook.com/1063164230212573", Account{ID: "1063164230212573"}},
	} {
		got, err := ParseAccountURL(tc.raw)
		if tc.want == (Account{}) {
			if err == nil {
				t.Errorf("ParseAccountURL(%q) = %+v, want an error", tc.raw, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("ParseAccountURL(%q) = %+v, %v; want %+v", tc.raw, got, err, tc.want)
		}
	}

	for _, bad := range []string{
		"https://www.instagram.com/alice.kohli",     // another site
		"https://www.facebook.com/profile.php",      // no id
		"https://www.facebook.com/profile.php?id=x", // id not numeric
		"https://www.facebook.com/groups/abc/posts", // more than one account
		"https://www.facebook.com/",                 // no account
	} {
		if got, err := ParseAccountURL(bad); err == nil {
			t.Errorf("ParseAccountURL(%q) = %+v, want an error", bad, got)
		}
	}
}

func TestMention(t *testing.T) {
	if got := Mention("123"); got != "@[123]" {
		t.Errorf("Mention = %q", got)
	}
}

func TestLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fdpstadtzuerich" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("fields"); got != "id,name" {
			t.Errorf("fields = %q", got)
		}
		if got := r.URL.Query().Get("access_token"); got != "tok123" {
			t.Errorf("access_token = %q", got)
		}
		_, _ = w.Write([]byte(`{"id":"555","name":"FDP Stadt Zürich"}`))
	}))
	defer server.Close()
	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	id, name, err := client.Lookup("fdpstadtzuerich")
	if err != nil || id != "555" || name != "FDP Stadt Zürich" {
		t.Errorf("Lookup = %q, %q, %v", id, name, err)
	}
}

// Meta answers a lookup it will not serve with an error, which must reach the
// caller: that refusal is the answer the experiment is after.
func TestLookup_Refused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"(#10) requires Page Public Content Access"}}`))
	}))
	defer server.Close()
	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	if _, _, err := client.Lookup("fdpstadtzuerich"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMessageTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/page_1_post_9" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("fields"); got != "message_tags" {
			t.Errorf("fields = %q", got)
		}
		_, _ = w.Write([]byte(`{"id":"page_1_post_9","message_tags":[{"id":"555","name":"FDP"},{"id":"777","name":"Alice"}]}`))
	}))
	defer server.Close()
	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	ids, err := client.MessageTags("page_1_post_9")
	if err != nil || len(ids) != 2 || ids[0] != "555" || ids[1] != "777" {
		t.Errorf("MessageTags = %v, %v", ids, err)
	}
}

// A post whose mentions Facebook did not accept has no message_tags at all.
func TestMessageTags_None(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"page_1_post_9"}`))
	}))
	defer server.Close()
	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	ids, err := client.MessageTags("page_1_post_9")
	if err != nil || len(ids) != 0 {
		t.Errorf("MessageTags = %v, %v; want none", ids, err)
	}
}

func TestPublishTextPost_SendsNoAttachedMedia(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("message"); got != "hello" {
			t.Errorf("message = %q", got)
		}
		for k := range r.PostForm {
			if len(k) >= 14 && k[:14] == "attached_media" {
				t.Errorf("unexpected field %q", k)
			}
		}
		_, _ = w.Write([]byte(`{"id":"page_1_post_1"}`))
	}))
	defer server.Close()
	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	if id, err := client.PublishTextPost("hello"); err != nil || id != "page_1_post_1" {
		t.Errorf("PublishTextPost = %q, %v", id, err)
	}
}

func TestDeletePost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/page_1_post_1" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	if err := client.DeletePost("page_1_post_1"); err != nil {
		t.Errorf("DeletePost: %v", err)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer failing.Close()
	client.SetAPIBase(failing.URL)
	if err := client.DeletePost("page_1_post_1"); err == nil {
		t.Error("DeletePost: expected error for a failed delete")
	}
}
