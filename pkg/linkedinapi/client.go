// Package linkedinapi is a minimal client for Share on LinkedIn, the Consumer
// API product that lets an app publish text posts as the member who authorised
// it (scope w_member_social).
package linkedinapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultHost = "https://api.linkedin.com"

// MaxChars is the commentary limit of a share.
const MaxChars = 3000

// Client posts on behalf of one member.
type Client struct {
	AccessToken string
	// AuthorURN is the member the token belongs to, "urn:li:person:<id>".
	AuthorURN string

	host string
	http *http.Client
}

// New creates a client for the given token and member.
func New(accessToken, authorURN string) *Client {
	return &Client{
		AccessToken: accessToken,
		AuthorURN:   authorURN,
		host:        defaultHost,
		http:        &http.Client{Timeout: 15 * time.Second},
	}
}

// SharePost creates a public post and returns its URN ("urn:li:share:…").
//
// A non-empty articleURL makes it a link share, which LinkedIn renders with a
// preview card; the URL must also appear in text if readers are to see it in
// the commentary itself. An empty one posts text only.
func (c *Client) SharePost(text, articleURL string) (string, error) {
	content := map[string]any{
		"shareCommentary":    map[string]string{"text": text},
		"shareMediaCategory": "NONE",
	}
	if articleURL != "" {
		content["shareMediaCategory"] = "ARTICLE"
		content["media"] = []map[string]string{{"status": "READY", "originalUrl": articleURL}}
	}
	payload := map[string]any{
		"author":         c.AuthorURN,
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]any{
			"com.linkedin.ugc.ShareContent": content,
		},
		"visibility": map[string]string{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal share: %w", err)
	}

	req, err := http.NewRequest("POST", c.host+"/v2/ugcPosts", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create share request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to post share: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("linkedin ugcPosts returned status %d: %s%s",
			resp.StatusCode, strings.TrimSpace(string(respBody)), tokenHint(resp.StatusCode))
	}

	id := resp.Header.Get("X-RestLi-Id")
	if id == "" {
		var parsed struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(respBody, &parsed)
		id = parsed.ID
	}
	return id, nil
}

// PostURL is the public address of a share URN.
func PostURL(urn string) string {
	return "https://www.linkedin.com/feed/update/" + urn
}

// tokenHint names the usual cause of a 401. Share on LinkedIn tokens last 60
// days and cannot be refreshed, so this is a scheduled failure rather than a
// surprising one, and the log line should say so.
func tokenHint(status int) string {
	if status != http.StatusUnauthorized {
		return ""
	}
	return " (the access token has probably expired: Share on LinkedIn tokens last 60 days and cannot be refreshed; authorise again and replace it)"
}
