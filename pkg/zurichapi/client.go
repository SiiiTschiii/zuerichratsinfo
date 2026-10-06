package zurichapi

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides methods to interact with Zurich city council APIs
type Client struct {
	httpClient *http.Client
	userAgent  string
	// backoffs holds the wait before each retry, so its length is the number
	// of retries after the first attempt.
	backoffs []time.Duration
}

// NewClient creates a new API client with default settings
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "ZurichRatsInfo/1.0 (Civic Tech Bot)",
		backoffs:  []time.Duration{2 * time.Second, 5 * time.Second},
	}
}

// makeRequest performs an HTTP GET request, retrying transient failures
// (network errors and 5xx responses) after the client's backoffs. The council's
// servers answer maintenance blips with a short 503; one attempt would turn
// each of those into a failed run. A 4xx is the caller's mistake and is not
// retried.
func (c *Client) makeRequest(url string) ([]byte, error) {
	body, retryable, err := c.attempt(url)
	for _, wait := range c.backoffs {
		if err == nil || !retryable {
			break
		}
		time.Sleep(wait)
		body, retryable, err = c.attempt(url)
	}
	return body, err
}

// attempt performs one GET and reports whether a failure is worth retrying.
func (c *Client) attempt(url string) (body []byte, retryable bool, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to avoid being blocked
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, resp.StatusCode >= 500, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, false, nil
}
