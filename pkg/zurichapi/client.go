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
}

// NewClient creates a new API client with default settings
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "ZurichRatsInfo/1.0 (Civic Tech Bot)",
	}
}

// makeRequest performs an HTTP GET request with appropriate headers
func (c *Client) makeRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to avoid being blocked
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}