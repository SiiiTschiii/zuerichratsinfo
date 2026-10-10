package linkedinapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(srv *httptest.Server) *Client {
	c := New("tok", "urn:li:person:abc")
	c.host = srv.URL
	return c
}

func TestSharePost_LinkShare(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/ugcPosts" || r.Header.Get("Authorization") != "Bearer tok" ||
			r.Header.Get("X-Restli-Protocol-Version") != "2.0.0" {
			t.Errorf("bad request: %s %v", r.URL.Path, r.Header)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		w.Header().Set("X-RestLi-Id", "urn:li:share:42")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	id, err := testClient(srv).SharePost("hello", "https://example.ch/a")
	if err != nil || id != "urn:li:share:42" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	body, _ := json.Marshal(got)
	for _, want := range []string{`"author":"urn:li:person:abc"`, `"shareMediaCategory":"ARTICLE"`,
		`"originalUrl":"https://example.ch/a"`, `"text":"hello"`, `"PUBLIC"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("payload missing %s: %s", want, body)
		}
	}
}

func TestSharePost_TextOnly(t *testing.T) {
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	if _, err := testClient(srv).SharePost("hello", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, `"shareMediaCategory":"NONE"`) || strings.Contains(raw, "media\"") {
		t.Errorf("unexpected payload: %s", raw)
	}
}

func TestSharePost_ExpiredToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"EXPIRED_ACCESS_TOKEN"}`))
	}))
	defer srv.Close()
	_, err := testClient(srv).SharePost("x", "")
	if err == nil || !strings.Contains(err.Error(), "expired") || !strings.Contains(err.Error(), "401") {
		t.Errorf("got %v", err)
	}
}
