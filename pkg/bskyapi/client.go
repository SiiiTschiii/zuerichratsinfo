package bskyapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultPDSHost = "https://bsky.social"

// Session holds the authenticated session data
type Session struct {
	DID             string `json:"did"`
	Handle          string `json:"handle"`
	AccessJwt       string `json:"accessJwt"`
	RefreshJwt      string `json:"refreshJwt"`
	ServiceEndpoint string // resolved from didDoc
}

// sessionResponse is the raw API response from createSession
type sessionResponse struct {
	DID        string `json:"did"`
	Handle     string `json:"handle"`
	AccessJwt  string `json:"accessJwt"`
	RefreshJwt string `json:"refreshJwt"`
	DidDoc     struct {
		Service []struct {
			ID              string `json:"id"`
			Type            string `json:"type"`
			ServiceEndpoint string `json:"serviceEndpoint"`
		} `json:"service"`
	} `json:"didDoc"`
}

// Facet represents a rich text annotation (link, mention, etc.)
type Facet struct {
	Index    FacetIndex     `json:"index"`
	Features []FacetFeature `json:"features"`
}

// FacetIndex represents the byte range of a facet in the text
type FacetIndex struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

// FacetFeature represents a facet feature (link, mention, tag)
type FacetFeature struct {
	Type string `json:"$type"`
	URI  string `json:"uri,omitempty"`
	DID  string `json:"did,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

// CreateSession authenticates with the Bluesky AT Protocol and returns a session
func CreateSession(handle, password string) (*Session, error) {
	url := defaultPDSHost + "/xrpc/com.atproto.server.createSession"

	payload := map[string]string{
		"identifier": handle,
		"password":   password,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bluesky createSession returned status %d: %s", resp.StatusCode, string(body))
	}

	var sessResp sessionResponse
	if err := json.Unmarshal(body, &sessResp); err != nil {
		return nil, fmt.Errorf("failed to parse session response: %w", err)
	}

	// Resolve the PDS service endpoint from the didDoc
	serviceEndpoint := defaultPDSHost
	for _, svc := range sessResp.DidDoc.Service {
		if svc.Type == "AtprotoPersonalDataServer" {
			serviceEndpoint = svc.ServiceEndpoint
			break
		}
	}

	return &Session{
		DID:             sessResp.DID,
		Handle:          sessResp.Handle,
		AccessJwt:       sessResp.AccessJwt,
		RefreshJwt:      sessResp.RefreshJwt,
		ServiceEndpoint: serviceEndpoint,
	}, nil
}

// CreateRecord creates a post on Bluesky
// text is the post content, facets are optional rich text annotations (links, mentions)
func CreateRecord(session *Session, text string, facets []Facet) error {
	url := session.ServiceEndpoint + "/xrpc/com.atproto.repo.createRecord"

	// Build the record
	record := map[string]interface{}{
		"$type":     "app.bsky.feed.post",
		"text":      text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}

	if len(facets) > 0 {
		record["facets"] = facets
	}

	payload := map[string]interface{}{
		"repo":       session.DID,
		"collection": "app.bsky.feed.post",
		"record":     record,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal post payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create post request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bluesky createRecord returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ Bluesky post created successfully!\nResponse: %s\n", string(body))
	return nil
}

// LinkFacet creates a facet for a URL link in the post text.
// byteStart and byteEnd are the byte offsets of the link text in the UTF-8 encoded post.
func LinkFacet(byteStart, byteEnd int, uri string) Facet {
	return Facet{
		Index: FacetIndex{
			ByteStart: byteStart,
			ByteEnd:   byteEnd,
		},
		Features: []FacetFeature{
			{
				Type: "app.bsky.richtext.facet#link",
				URI:  uri,
			},
		},
	}
}
