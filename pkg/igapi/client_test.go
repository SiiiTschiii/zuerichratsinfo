package igapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateMediaContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("image_url") != "https://example.com/img.jpg" {
			t.Errorf("unexpected image_url: %s", r.FormValue("image_url"))
		}
		if r.FormValue("is_carousel_item") != "true" {
			t.Errorf("expected is_carousel_item=true, got %s", r.FormValue("is_carousel_item"))
		}
		if r.FormValue("access_token") != "tok123" {
			t.Errorf("unexpected access_token: %s", r.FormValue("access_token"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"container_1"}`))
	}))
	defer server.Close()

	client := NewClient("ig_user_1", "tok123")
	client.SetAPIBase(server.URL)

	id, err := client.CreateMediaContainer("https://example.com/img.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "container_1" {
		t.Errorf("expected container_1, got %s", id)
	}
}

func TestCreateMediaContainer_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer server.Close()

	client := NewClient("ig_user_1", "tok123")
	client.SetAPIBase(server.URL)

	_, err := client.CreateMediaContainer("https://example.com/img.jpg")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestCreateCarouselContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("media_type") != "CAROUSEL" {
			t.Errorf("expected media_type=CAROUSEL, got %s", r.FormValue("media_type"))
		}

		var children []string
		if err := json.Unmarshal([]byte(r.FormValue("children")), &children); err != nil {
			t.Fatalf("parsing children: %v", err)
		}
		if len(children) != 2 || children[0] != "c1" || children[1] != "c2" {
			t.Errorf("unexpected children: %v", children)
		}

		if r.FormValue("caption") != "test caption" {
			t.Errorf("unexpected caption: %s", r.FormValue("caption"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"carousel_1"}`))
	}))
	defer server.Close()

	client := NewClient("ig_user_1", "tok123")
	client.SetAPIBase(server.URL)

	id, err := client.CreateCarouselContainer([]string{"c1", "c2"}, "test caption")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "carousel_1" {
		t.Errorf("expected carousel_1, got %s", id)
	}
}

func TestPublishContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("creation_id") != "carousel_1" {
			t.Errorf("unexpected creation_id: %s", r.FormValue("creation_id"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"media_1"}`))
	}))
	defer server.Close()

	client := NewClient("ig_user_1", "tok123")
	client.SetAPIBase(server.URL)

	id, err := client.PublishContainer("carousel_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "media_1" {
		t.Errorf("expected media_1, got %s", id)
	}
}

func TestPollContainerStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("fields") != "status_code" {
			t.Errorf("expected fields=status_code, got %s", r.URL.Query().Get("fields"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status_code":"FINISHED"}`))
	}))
	defer server.Close()

	client := NewClient("ig_user_1", "tok123")
	client.SetAPIBase(server.URL)

	status, err := client.PollContainerStatus("container_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "FINISHED" {
		t.Errorf("expected FINISHED, got %s", status)
	}
}

func TestPollContainerStatus_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer server.Close()

	client := NewClient("ig_user_1", "tok123")
	client.SetAPIBase(server.URL)

	_, err := client.PollContainerStatus("container_1")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

// TestRoundTrip_FullCarouselFlow tests the complete flow: create items → create carousel → publish
func TestRoundTrip_FullCarouselFlow(t *testing.T) {
	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step++
		body, _ := io.ReadAll(r.Body)
		_ = body // consume body for POST requests with form encoding

		w.WriteHeader(http.StatusOK)
		switch {
		case step <= 3: // 3 CreateMediaContainer calls
			_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"child_%d"}`, step)))
		case step == 4: // CreateCarouselContainer
			_, _ = w.Write([]byte(`{"id":"carousel_99"}`))
		case step == 5: // PublishContainer
			_, _ = w.Write([]byte(`{"id":"media_99"}`))
		}
	}))
	defer server.Close()

	client := NewClient("ig_user_1", "tok123")
	client.SetAPIBase(server.URL)

	// Step 1: Create child containers
	childIDs := make([]string, 3)
	for i := range childIDs {
		id, err := client.CreateMediaContainer("https://example.com/img.jpg")
		if err != nil {
			t.Fatalf("CreateMediaContainer %d: %v", i, err)
		}
		childIDs[i] = id
	}

	// Step 2: Create carousel container
	carouselID, err := client.CreateCarouselContainer(childIDs, "caption")
	if err != nil {
		t.Fatalf("CreateCarouselContainer: %v", err)
	}
	if carouselID != "carousel_99" {
		t.Errorf("expected carousel_99, got %s", carouselID)
	}

	// Step 3: Publish
	mediaID, err := client.PublishContainer(carouselID)
	if err != nil {
		t.Fatalf("PublishContainer: %v", err)
	}
	if mediaID != "media_99" {
		t.Errorf("expected media_99, got %s", mediaID)
	}
}


