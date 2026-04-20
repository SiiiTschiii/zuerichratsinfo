package igapi

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageHoster_PagesBaseURL(t *testing.T) {
	h := NewImageHoster("myorg", "myrepo", "token")
	expected := "https://myorg.github.io/myrepo"
	if got := h.PagesBaseURL(); got != expected {
		t.Errorf("PagesBaseURL() = %q, want %q", got, expected)
	}
}

func TestImageHoster_UploadImages(t *testing.T) {
	var requests []requestCapture

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, requestCapture{
			method: r.Method,
			path:   r.URL.Path,
			body:   string(body),
		})

		if r.Method == http.MethodGet {
			// getFileSHA: file not found (new file)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// PUT: create file
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"content":{"sha":"abc123"}}`))
	}))
	defer server.Close()

	h := NewImageHoster("testorg", "testrepo", "ghp_test")
	h.SetAPIBase(server.URL)

	images := [][]byte{
		{0xFF, 0xD8, 0xFF, 0xE0}, // fake JPEG
		{0xFF, 0xD8, 0xFF, 0xE1}, // another fake JPEG
	}
	names := []string{"img_0.jpg", "img_1.jpg"}

	urls, err := h.UploadImages(images, names)
	if err != nil {
		t.Fatalf("UploadImages: %v", err)
	}

	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}

	expectedURL0 := "https://testorg.github.io/testrepo/ig-images/img_0.jpg"
	if urls[0] != expectedURL0 {
		t.Errorf("urls[0] = %q, want %q", urls[0], expectedURL0)
	}

	// Verify PUT requests were made with correct content
	putCount := 0
	for _, req := range requests {
		if req.method == http.MethodPut {
			putCount++
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(req.body), &payload); err != nil {
				t.Fatalf("parse PUT body: %v", err)
			}
			if payload["branch"] != "gh-pages" {
				t.Errorf("expected branch=gh-pages, got %v", payload["branch"])
			}
			// Verify content is base64 encoded
			contentStr, ok := payload["content"].(string)
			if !ok {
				t.Error("expected content to be a string")
			}
			if _, err := base64.StdEncoding.DecodeString(contentStr); err != nil {
				t.Errorf("content is not valid base64: %v", err)
			}
		}
	}
	if putCount != 2 {
		t.Errorf("expected 2 PUT requests, got %d", putCount)
	}
}

func TestImageHoster_UploadImages_MismatchedLengths(t *testing.T) {
	h := NewImageHoster("org", "repo", "token")
	_, err := h.UploadImages([][]byte{{1}}, []string{"a.jpg", "b.jpg"})
	if err == nil {
		t.Fatal("expected error for mismatched lengths")
	}
}

func TestImageHoster_CleanupImages(t *testing.T) {
	var requests []requestCapture

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, requestCapture{
			method: r.Method,
			path:   r.URL.Path,
			body:   string(body),
		})

		if r.Method == http.MethodGet {
			// getFileSHA: file exists
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"sha":"sha_to_delete"}`))
			return
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	h := NewImageHoster("testorg", "testrepo", "ghp_test")
	h.SetAPIBase(server.URL)

	err := h.CleanupImages([]string{"img_0.jpg", "img_1.jpg"})
	if err != nil {
		t.Fatalf("CleanupImages: %v", err)
	}

	// Verify DELETE requests include the correct SHA
	deleteCount := 0
	for _, req := range requests {
		if req.method == http.MethodDelete {
			deleteCount++
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(req.body), &payload); err != nil {
				t.Fatalf("parse DELETE body: %v", err)
			}
			if payload["sha"] != "sha_to_delete" {
				t.Errorf("expected sha=sha_to_delete, got %v", payload["sha"])
			}
			if payload["branch"] != "gh-pages" {
				t.Errorf("expected branch=gh-pages, got %v", payload["branch"])
			}
		}
	}
	if deleteCount != 2 {
		t.Errorf("expected 2 DELETE requests, got %d", deleteCount)
	}
}

func TestImageHoster_UploadImages_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer server.Close()

	h := NewImageHoster("org", "repo", "bad_token")
	h.SetAPIBase(server.URL)

	_, err := h.UploadImages([][]byte{{1}}, []string{"img.jpg"})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("error should mention 'forbidden': %v", err)
	}
}

func TestImageHoster_UploadExistingFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// File exists — return SHA
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"sha":"existing_sha"}`))
			return
		}
		if r.Method == http.MethodPut {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			if err := json.Unmarshal(body, &payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			// Verify SHA is included for update
			if payload["sha"] != "existing_sha" {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"message":"sha mismatch"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"content":{"sha":"new_sha"}}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	h := NewImageHoster("org", "repo", "token")
	h.SetAPIBase(server.URL)

	urls, err := h.UploadImages([][]byte{{1}}, []string{"img.jpg"})
	if err != nil {
		t.Fatalf("UploadImages (existing file): %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 URL, got %d", len(urls))
	}
}

func TestImageHoster_SetHeaders(t *testing.T) {
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	h := NewImageHoster("org", "repo", "ghp_mytoken")
	h.SetAPIBase(server.URL)

	// Trigger a request (getFileSHA will make a GET)
	_, _ = h.getFileSHA("test/path")

	if got := capturedHeaders.Get("Authorization"); got != "Bearer ghp_mytoken" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer ghp_mytoken")
	}
	if got := capturedHeaders.Get("X-GitHub-Api-Version"); got != "2022-11-28" {
		t.Errorf("X-GitHub-Api-Version = %q, want %q", got, "2022-11-28")
	}
}

type requestCapture struct {
	method string
	path   string
	body   string
}
