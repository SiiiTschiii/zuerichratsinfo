package contacts

import (
	"strings"
	"testing"
)

func TestValidURLs(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		platform  string
		wantError bool
		errorMsg  string
	}{
		// Valid URLs
		{
			name:      "valid x.com URL",
			url:       "https://x.com/username",
			platform:  "x",
			wantError: false,
		},
		{
			name:      "valid twitter.com URL (legacy)",
			url:       "https://twitter.com/username",
			platform:  "x",
			wantError: false,
		},
		{
			name:      "valid facebook URL",
			url:       "https://www.facebook.com/username",
			platform:  "facebook",
			wantError: false,
		},
		{
			name:      "valid instagram URL",
			url:       "https://www.instagram.com/username/",
			platform:  "instagram",
			wantError: false,
		},
		{
			name:      "valid linkedin URL",
			url:       "https://www.linkedin.com/in/username",
			platform:  "linkedin",
			wantError: false,
		},
		{
			name:      "valid bluesky URL",
			url:       "https://bsky.app/profile/username",
			platform:  "bluesky",
			wantError: false,
		},
		{
			name:      "valid tiktok URL",
			url:       "https://www.tiktok.com/@username",
			platform:  "tiktok",
			wantError: false,
		},
		{
			name:      "valid http URL (not just https)",
			url:       "http://x.com/username",
			platform:  "x",
			wantError: false,
		},

		// Invalid URLs - missing scheme
		{
			name:      "URL without scheme",
			url:       "www.x.com/username",
			platform:  "x",
			wantError: true,
			errorMsg:  "URL must use http or https scheme",
		},
		{
			name:      "URL with just domain",
			url:       "x.com/username",
			platform:  "x",
			wantError: true,
			errorMsg:  "URL must use http or https scheme",
		},

		// Invalid URLs - wrong domain
		{
			name:      "wrong domain for platform",
			url:       "https://twitter.com/username",
			platform:  "facebook",
			wantError: true,
			errorMsg:  "does not match platform",
		},
		{
			name:      "completely wrong domain",
			url:       "https://example.com/username",
			platform:  "x",
			wantError: true,
			errorMsg:  "does not match platform",
		},

		// Invalid URLs - empty
		{
			name:      "empty URL",
			url:       "",
			platform:  "x",
			wantError: true,
			errorMsg:  "empty URL",
		},
		{
			name:      "whitespace only URL",
			url:       "   ",
			platform:  "x",
			wantError: true,
			errorMsg:  "empty URL",
		},

		// Invalid URLs - bad format
		{
			name:      "invalid URL format",
			url:       "ht!tp://x.com/user",
			platform:  "x",
			wantError: true,
		},

		// Edge cases - subdomains
		{
			name:      "subdomain for linkedin",
			url:       "https://ch.linkedin.com/in/username",
			platform:  "linkedin",
			wantError: false,
		},
		{
			name:      "bluesky web-cdn subdomain",
			url:       "https://web-cdn.bsky.app/profile/username",
			platform:  "bluesky",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.platform, tt.url)

			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.wantError && tt.errorMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}
