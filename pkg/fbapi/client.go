// Package fbapi is a minimal Facebook Pages API client: it publishes a text
// post with a link to a Page feed.
package fbapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DefaultGraphAPIBase is the base URL for the Graph API. It pins the same
// version as igapi, because both are driven by one Meta app and one Page token.
const DefaultGraphAPIBase = "https://graph.facebook.com/v25.0"

// Client publishes to one Facebook Page.
type Client struct {
	pageID      string
	accessToken string
	httpClient  *http.Client
	apiBase     string // base URL for the Graph API (overridable for testing)
}

// NewClient creates a client for a Page. The token must be a Page access token
// carrying pages_manage_posts; a user token posts as the user, not the Page.
func NewClient(pageID, accessToken string) *Client {
	return &Client{
		pageID:      pageID,
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		apiBase:     DefaultGraphAPIBase,
	}
}

// SetAPIBase overrides the Graph API base URL (for testing).
func (c *Client) SetAPIBase(base string) {
	c.apiBase = base
}

// idResponse is the response from POST /{page-id}/feed.
type idResponse struct {
	ID string `json:"id"`
}

// PublishPost publishes message to the Page feed and returns the post ID
// ("{page-id}_{post-id}").
//
// link is optional. When set, Facebook scrapes it and attaches a preview card
// to the post; the URL does not have to appear in the message for that.
func (c *Client) PublishPost(message, link string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/feed", c.apiBase, c.pageID)

	params := url.Values{}
	params.Set("message", message)
	if link != "" {
		params.Set("link", link)
	}
	params.Set("access_token", c.accessToken)

	resp, err := c.httpClient.PostForm(endpoint, params)
	if err != nil {
		return "", fmt.Errorf("publishing page post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("publish page post returned status %d: %s", resp.StatusCode, string(body))
	}

	var result idResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing page post response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("page post response carried no id: %s", string(body))
	}

	return result.ID, nil
}

// PostURL returns the public URL of a post from the ID PublishPost returned.
func PostURL(postID string) string {
	return "https://www.facebook.com/" + postID
}
