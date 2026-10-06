package zurichapi

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(retries int) *Client {
	c := NewClient()
	c.backoffs = make([]time.Duration, retries)
	return c
}

func TestMakeRequest_RetriesTransient503(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body, err := testClient(2).makeRequest(srv.URL)
	if err != nil || string(body) != "ok" {
		t.Fatalf("got body=%q err=%v, want ok after retries", body, err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestMakeRequest_GivesUpAfterBackoffs(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if _, err := testClient(2).makeRequest(srv.URL); err == nil {
		t.Fatal("expected error from a persistent 503")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3 (1 attempt + 2 retries)", calls.Load())
	}
}

func TestMakeRequest_DoesNotRetry4xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := testClient(2).makeRequest(srv.URL); err == nil {
		t.Fatal("expected error from a 404")
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}
