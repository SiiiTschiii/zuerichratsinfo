package contacts

import (
	"fmt"
	"net/url"
	"strings"
)

var platformDomains = map[string][]string{
	"x":         {"x.com", "twitter.com"},
	"facebook":  {"facebook.com", "www.facebook.com"},
	"instagram": {"instagram.com", "www.instagram.com"},
	"linkedin":  {"linkedin.com", "www.linkedin.com"},
	"bluesky":   {"bsky.app", "web-cdn.bsky.app"},
	"tiktok":    {"tiktok.com", "www.tiktok.com"},
}

// ValidateURL reports whether raw is a parseable http(s) URL on a domain the
// platform uses.
//
// It lives here, rather than in cmd/validate_contacts, because the tools that
// write the file have to hold a new account to the rule CI will apply to it: a
// check that only the validator knows lets a writer save a file the validator
// then rejects.
func ValidateURL(platform, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme, got: %s", parsed.Scheme)
	}

	hostname := strings.ToLower(parsed.Hostname())
	allowed := platformDomains[platform]
	for _, domain := range allowed {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			return nil
		}
	}

	return fmt.Errorf("URL domain '%s' does not match platform '%s' (expected one of: %v)",
		hostname, platform, allowed)
}
