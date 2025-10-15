package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// postTweet posts a tweet using X API v2 with OAuth 1.0a User Context
func postTweet(apiKey, apiSecret, accessToken, accessSecret, message string) error {
	// X API v2 endpoint for creating tweets
	apiURL := "https://api.x.com/2/tweets"

	// Create the tweet payload
	payload := map[string]interface{}{
		"text": message,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tweet payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Generate OAuth 1.0a authorization header
	authHeader := generateOAuthHeader(apiURL, "POST", apiKey, apiSecret, accessToken, accessSecret)
	req.Header.Set("Authorization", authHeader)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("X API returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ Tweet posted successfully!\nResponse: %s\n", string(body))
	return nil
}

// generateOAuthHeader generates the OAuth 1.0a authorization header
func generateOAuthHeader(apiURL, method, apiKey, apiSecret, accessToken, accessSecret string) string {
	// OAuth parameters
	params := map[string]string{
		"oauth_consumer_key":     apiKey,
		"oauth_nonce":            generateNonce(),
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        strconv.FormatInt(time.Now().Unix(), 10),
		"oauth_token":            accessToken,
		"oauth_version":          "1.0",
	}

	// Generate signature
	signature := generateSignature(method, apiURL, params, apiSecret, accessSecret)
	params["oauth_signature"] = signature

	// Build authorization header
	var authParts []string
	for key, value := range params {
		authParts = append(authParts, fmt.Sprintf(`%s="%s"`, key, percentEncode(value)))
	}

	// Sort for consistency
	sort.Strings(authParts)

	return "OAuth " + strings.Join(authParts, ", ")
}

// generateSignature generates the OAuth signature
func generateSignature(method, apiURL string, params map[string]string, apiSecret, accessSecret string) string {
	// Create parameter string
	var paramPairs []string
	for key, value := range params {
		paramPairs = append(paramPairs, percentEncode(key)+"="+percentEncode(value))
	}

	// Sort parameters
	sort.Strings(paramPairs)
	paramString := strings.Join(paramPairs, "&")

	// Create signature base string
	baseString := method + "&" + percentEncode(apiURL) + "&" + percentEncode(paramString)

	// Create signing key
	signingKey := percentEncode(apiSecret) + "&" + percentEncode(accessSecret)

	// Generate HMAC-SHA1 signature
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(baseString))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

// generateNonce generates a random nonce for OAuth
func generateNonce() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// percentEncode encodes a string per RFC 3986
func percentEncode(s string) string {
	return url.QueryEscape(s)
}
