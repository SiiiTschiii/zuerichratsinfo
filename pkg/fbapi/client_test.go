package fbapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublishPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/page_1/feed" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("message"); got != "🗳️ Hallo\n\nWelt" {
			t.Errorf("unexpected message: %q", got)
		}
		if got := r.FormValue("link"); got != "https://example.com/vote" {
			t.Errorf("unexpected link: %q", got)
		}
		if got := r.FormValue("access_token"); got != "tok123" {
			t.Errorf("unexpected access_token: %q", got)
		}

		_, _ = w.Write([]byte(`{"id":"page_1_post_9"}`))
	}))
	defer server.Close()

	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	id, err := client.PublishPost("🗳️ Hallo\n\nWelt", "https://example.com/vote")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "page_1_post_9" {
		t.Errorf("expected page_1_post_9, got %s", id)
	}
}

// Without a link the parameter must be absent, not empty: an empty link is a
// Graph API error rather than "no preview".
func TestPublishPost_OmitsEmptyLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if _, ok := r.PostForm["link"]; ok {
			t.Errorf("link parameter sent although empty")
		}
		_, _ = w.Write([]byte(`{"id":"p_1"}`))
	}))
	defer server.Close()

	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	if _, err := client.PublishPost("text", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishPost_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"(#200) requires pages_manage_posts"}}`))
	}))
	defer server.Close()

	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	if _, err := client.PublishPost("text", ""); err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestPublishPost_MissingID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	if _, err := client.PublishPost("text", ""); err == nil {
		t.Fatal("expected error for a response without an id")
	}
}
