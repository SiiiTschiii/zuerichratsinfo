package urlshorten

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ShortenURL shortens a URL using the is.gd service
// Returns the shortened URL or the original URL if shortening fails
func ShortenURL(longURL string) string {
	// Try to shorten, but if it fails, return the original URL
	shortURL, err := shortenWithIsGd(longURL)
	if err != nil {
		// Fallback to original URL
		return longURL
	}
	return shortURL
}

// shortenWithIsGd shortens a URL using is.gd API
func shortenWithIsGd(longURL string) (string, error) {
	// is.gd API endpoint
	apiURL := "https://is.gd/create.php"

	// Build request URL
	params := url.Values{}
	params.Add("format", "json")
	params.Add("url", longURL)

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make request
	resp, err := client.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var result struct {
		ShortURL     string `json:"shorturl"`
		ErrorCode    int    `json:"errorcode"`
		ErrorMessage string `json:"errormessage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if result.ErrorCode != 0 {
		return "", fmt.Errorf("is.gd error: %s", result.ErrorMessage)
	}

	return result.ShortURL, nil
}
