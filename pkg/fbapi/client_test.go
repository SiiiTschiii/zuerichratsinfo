package fbapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadPhoto(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/page_1/photos" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		// An unpublished upload is the whole point: a published one would post
		// every carousel card to the Page on its own.
		if got := r.FormValue("published"); got != "false" {
			t.Errorf("published = %q, want false", got)
		}
		if got := r.FormValue("access_token"); got != "tok123" {
			t.Errorf("unexpected access_token: %q", got)
		}
		file, header, err := r.FormFile("source")
		if err != nil {
			t.Fatalf("no source part: %v", err)
		}
		defer func() { _ = file.Close() }()
		data, _ := io.ReadAll(file)
		if string(data) != "jpeg-bytes" {
			t.Errorf("uploaded %q, want the JPEG bytes", data)
		}
		if got := header.Header.Get("Content-Type"); got != "image/jpeg" {
			t.Errorf("part content type = %q, want image/jpeg", got)
		}

		_, _ = w.Write([]byte(`{"id":"photo_7"}`))
	}))
	defer server.Close()

	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	id, err := client.UploadPhoto([]byte("jpeg-bytes"), "card_0.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "photo_7" {
		t.Errorf("expected photo_7, got %s", id)
	}
}

func TestPublishPhotoPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/page_1/feed" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("message"); got != "🗳️ Hallo\n\nWelt" {
			t.Errorf("unexpected message: %q", got)
		}
		// Order is the carousel's order, so it is part of the contract.
		if got := r.FormValue("attached_media[0]"); got != `{"media_fbid":"p1"}` {
			t.Errorf("attached_media[0] = %q", got)
		}
		if got := r.FormValue("attached_media[1]"); got != `{"media_fbid":"p2"}` {
			t.Errorf("attached_media[1] = %q", got)
		}
		if got := r.FormValue("access_token"); got != "tok123" {
			t.Errorf("unexpected access_token: %q", got)
		}

		_, _ = w.Write([]byte(`{"id":"page_1_post_9"}`))
	}))
	defer server.Close()

	client := NewClient("page_1", "tok123")
	client.SetAPIBase(server.URL)

	id, err := client.PublishPhotoPost("🗳️ Hallo\n\nWelt", []string{"p1", "p2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "page_1_post_9" {
		t.Errorf("expected page_1_post_9, got %s", id)
	}
}

func TestErrors(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"forbidden": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"message":"(#200) requires pages_manage_posts"}}`))
		},
		"missing id": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{}`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(handler)
			defer server.Close()
			client := NewClient("page_1", "tok123")
			client.SetAPIBase(server.URL)

			if _, err := client.UploadPhoto([]byte("x"), "a.jpg"); err == nil {
				t.Error("UploadPhoto: expected error")
			}
			if _, err := client.PublishPhotoPost("text", []string{"p1"}); err == nil {
				t.Error("PublishPhotoPost: expected error")
			}
		})
	}
}
