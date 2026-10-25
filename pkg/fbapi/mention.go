package fbapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Mention returns the markup that mentions a Page or profile in a post message,
// by numeric ID. Facebook turns it into a link and notifies the account, but
// only where the app is allowed to mention it; elsewhere the text is left as
// typed, which is why MessageTags exists.
func Mention(id string) string {
	return "@[" + id + "]"
}

// Account is what ParseAccountURL makes of a facebook.com URL.
type Account struct {
	// ID is the numeric ID, known without a lookup only for a
	// profile.php?id=… URL or a numeric path.
	ID string
	// Username is the vanity name in the path, which Lookup turns into an ID.
	Username string
}

// ParseAccountURL reads the account a facebook.com URL names, in the shapes the
// contacts files hold: facebook.com/fdpstadtzuerich/, facebook.com/alice.kohli,
// facebook.com/profile.php?id=100070693802425.
func ParseAccountURL(raw string) (Account, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return Account{}, fmt.Errorf("%q is not a URL", raw)
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if host != "facebook.com" && host != "m.facebook.com" {
		return Account{}, fmt.Errorf("%q is not a facebook.com URL", raw)
	}

	path := strings.Trim(u.Path, "/")
	if path == "profile.php" {
		id := u.Query().Get("id")
		if !isNumeric(id) {
			return Account{}, fmt.Errorf("%q has no numeric id", raw)
		}
		return Account{ID: id}, nil
	}
	if path == "" || strings.Contains(path, "/") {
		return Account{}, fmt.Errorf("%q does not name one account", raw)
	}
	if isNumeric(path) {
		return Account{ID: path}, nil
	}
	return Account{Username: path}, nil
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Lookup resolves a Page username to its numeric ID and name. It reads another
// account's public data, which Meta may allow only with a feature this app has
// not been granted; the error then says so, and is itself a finding.
func (c *Client) Lookup(username string) (id, name string, err error) {
	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.get(url.PathEscape(username), url.Values{"fields": {"id,name"}}, &out); err != nil {
		return "", "", fmt.Errorf("looking up %q: %w", username, err)
	}
	if out.ID == "" {
		return "", "", fmt.Errorf("looking up %q: response carried no id", username)
	}
	return out.ID, out.Name, nil
}

// MessageTags returns the IDs Facebook recognised as mentions in a post's
// message. A mention it did not accept is absent, whatever the message says.
func (c *Client) MessageTags(postID string) ([]string, error) {
	var out struct {
		MessageTags []struct {
			ID string `json:"id"`
		} `json:"message_tags"`
	}
	if err := c.get(url.PathEscape(postID), url.Values{"fields": {"message_tags"}}, &out); err != nil {
		return nil, fmt.Errorf("reading tags of %s: %w", postID, err)
	}
	ids := make([]string, len(out.MessageTags))
	for i, t := range out.MessageTags {
		ids[i] = t.ID
	}
	return ids, nil
}

// PublishTextPost publishes a post with a message and nothing attached.
func (c *Client) PublishTextPost(message string) (string, error) {
	return c.PublishPhotoPost(message, nil)
}

// DeletePost removes a post from the Page.
func (c *Client) DeletePost(postID string) error {
	req, err := c.newRequest(http.MethodDelete, url.PathEscape(postID), nil)
	if err != nil {
		return fmt.Errorf("deleting %s: %w", postID, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting %s: %w", postID, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deleting %s returned status %d: %s", postID, resp.StatusCode, string(body))
	}
	return nil
}

// newRequest builds a request that carries the Page token in an Authorization
// header and not in the URL. Go's HTTP client puts the URL into the error it
// returns for a failed connection or a timeout, and the commands print those
// errors, so a token in the URL would end up in a terminal or a log.
func (c *Client) newRequest(method, path string, params url.Values) (*http.Request, error) {
	u := fmt.Sprintf("%s/%s", c.apiBase, path)
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	return req, nil
}

// get reads a JSON object from the Graph API with the Page token.
func (c *Client) get(path string, params url.Values, out any) error {
	req, err := c.newRequest(http.MethodGet, path, params)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	return nil
}
