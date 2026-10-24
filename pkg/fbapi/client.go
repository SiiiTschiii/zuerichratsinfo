// Package fbapi is a minimal Facebook Pages API client: it publishes a post
// with photos to a Page feed.
package fbapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
		// Photos travel in the request body, so the limit has to allow for a
		// carousel's worth of JPEGs on a slow runner link, not just a JSON call.
		httpClient: &http.Client{Timeout: 120 * time.Second},
		apiBase:    DefaultGraphAPIBase,
	}
}

// SetAPIBase overrides the Graph API base URL (for testing).
func (c *Client) SetAPIBase(base string) {
	c.apiBase = base
}

// idResponse is the response from the photo and feed endpoints.
type idResponse struct {
	ID string `json:"id"`
}

// UploadPhoto uploads a JPEG to the Page without publishing it, and returns the
// photo ID to attach to a post with PublishPhotoPost. caption, when not empty,
// is the photo's own text, shown with it when a reader opens it from the post.
//
// Meta holds an unpublished photo for about 24 hours and then deletes it, so a
// photo whose post never happens needs no cleanup.
func (c *Client) UploadPhoto(jpeg []byte, filename, caption string) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// The part is named "source", which is what people have always used for a
	// file upload; the docs say the name does not matter.
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="source"; filename=%q`, filename))
	header.Set("Content-Type", "image/jpeg")
	part, err := w.CreatePart(header)
	if err != nil {
		return "", fmt.Errorf("building photo upload: %w", err)
	}
	if _, err := part.Write(jpeg); err != nil {
		return "", fmt.Errorf("building photo upload: %w", err)
	}
	fields := map[string]string{"published": "false", "access_token": c.accessToken}
	if caption != "" {
		fields["caption"] = caption
	}
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return "", fmt.Errorf("building photo upload: %w", err)
		}
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("building photo upload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/%s/photos", c.apiBase, c.pageID)
	resp, err := c.httpClient.Post(endpoint, w.FormDataContentType(), &body)
	if err != nil {
		return "", fmt.Errorf("uploading photo: %w", err)
	}
	return readID(resp, "upload photo")
}

// PublishPhotoPost publishes message to the Page feed with the given uploaded
// photos attached, in order, and returns the post ID ("{page-id}_{post-id}").
func (c *Client) PublishPhotoPost(message string, photoIDs []string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/feed", c.apiBase, c.pageID)

	params := url.Values{}
	params.Set("message", message)
	for i, id := range photoIDs {
		media, err := json.Marshal(map[string]string{"media_fbid": id})
		if err != nil {
			return "", fmt.Errorf("encoding attached media: %w", err)
		}
		params.Set(fmt.Sprintf("attached_media[%d]", i), string(media))
	}
	params.Set("access_token", c.accessToken)

	resp, err := c.httpClient.PostForm(endpoint, params)
	if err != nil {
		return "", fmt.Errorf("publishing page post: %w", err)
	}
	return readID(resp, "publish page post")
}

// readID reads an {"id": ...} response, closing the body.
func readID(resp *http.Response, what string) (string, error) {
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned status %d: %s", what, resp.StatusCode, string(body))
	}

	var result idResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing %s response: %w", what, err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("%s response carried no id: %s", what, string(body))
	}
	return result.ID, nil
}

// PostURL returns the public URL of a post from the ID PublishPhotoPost returned.
func PostURL(postID string) string {
	return "https://www.facebook.com/" + postID
}
