package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Test valid YAML parsing
func TestValidYAML(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErrors  int
	}{
		{
			name: "valid yaml with all fields",
			yamlContent: `version: "1.0"
contacts:
  - name: "John Doe"
    x: ["https://x.com/johndoe"]
    facebook: ["https://www.facebook.com/johndoe"]
    instagram: ["https://www.instagram.com/johndoe"]
    linkedin: ["https://www.linkedin.com/in/johndoe"]
    bluesky: ["https://bsky.app/profile/johndoe"]
    tiktok: ["https://www.tiktok.com/@johndoe"]
`,
			wantErrors: 0,
		},
		{
			name: "valid yaml with minimal fields",
			yamlContent: `version: "1.0"
contacts:
  - name: "Jane Smith"
    x: ["https://x.com/janesmith"]
`,
			wantErrors: 0,
		},
		{
			name: "invalid yaml - malformed",
			yamlContent: `version: "1.0"
contacts:
  - name: "Test"
    x: [https://x.com/test
`,
			wantErrors: 1, // Should fail YAML parsing
		},
		{
			name: "invalid yaml - wrong indentation",
			yamlContent: `version: "1.0"
contacts:
- name: "Test"
  x: ["https://x.com/test"]
    facebook: ["https://facebook.com/test"]
`,
			wantErrors: 1, // Should fail YAML parsing
		},
		{
			name: "missing version field",
			yamlContent: `contacts:
  - name: "Test"
    x: ["https://x.com/test"]
`,
			wantErrors: 1, // Missing version
		},
		{
			name: "empty contacts list",
			yamlContent: `version: "1.0"
contacts: []
`,
			wantErrors: 1, // No contacts found
		},
		{
			name: "contact without name",
			yamlContent: `version: "1.0"
contacts:
  - x: ["https://x.com/test"]
`,
			wantErrors: 1, // Contact has no name
		},
		{
			name: "duplicate contact names",
			yamlContent: `version: "1.0"
contacts:
  - name: "John Doe"
    x: ["https://x.com/johndoe"]
  - name: "John Doe"
    facebook: ["https://facebook.com/johndoe"]
`,
			wantErrors: 1, // Duplicate name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test-contacts.yaml")

			if err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Validate
			errors := validateContactsFile(tmpFile)

			if len(errors) != tt.wantErrors {
				t.Errorf("Expected %d errors, got %d", tt.wantErrors, len(errors))
				for i, err := range errors {
					t.Logf("Error %d: %s", i+1, err.String())
				}
			}
		})
	}
}

// Test supported platforms
func TestSupportedPlatforms(t *testing.T) {
	tests := []struct {
		name          string
		contact       Contact
		wantErrors    int
		errorContains string
	}{
		{
			name: "all supported platforms",
			contact: Contact{
				Name:      "Test User",
				X:         []string{"https://x.com/test"},
				Facebook:  []string{"https://facebook.com/test"},
				Instagram: []string{"https://instagram.com/test"},
				LinkedIn:  []string{"https://linkedin.com/in/test"},
				Bluesky:   []string{"https://bsky.app/profile/test"},
				TikTok:    []string{"https://tiktok.com/@test"},
			},
			wantErrors: 0,
		},
		{
			name: "contact with no platforms",
			contact: Contact{
				Name: "Test User",
			},
			wantErrors: 0, // This is allowed
		},
		{
			name: "platform with multiple URLs",
			contact: Contact{
				Name: "Test User",
				X:    []string{"https://x.com/test1", "https://x.com/test2"},
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateContactPlatforms(tt.contact)

			if len(errors) != tt.wantErrors {
				t.Errorf("Expected %d errors, got %d", tt.wantErrors, len(errors))
				for i, err := range errors {
					t.Logf("Error %d: %s", i+1, err.String())
				}
			}

			if tt.errorContains != "" && len(errors) > 0 {
				found := false
				for _, err := range errors {
					if contains(err.String(), tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error to contain '%s', but didn't find it", tt.errorContains)
				}
			}
		})
	}
}

// Test valid URLs
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
			err := validateURL(tt.url, tt.platform)

			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.wantError && tt.errorMsg != "" && err != nil {
				if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

// Test full file validation with URL issues
func TestURLValidationInFile(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErrors  int
	}{
		{
			name: "all valid URLs",
			yamlContent: `version: "1.0"
contacts:
  - name: "User One"
    x: ["https://x.com/user1"]
    facebook: ["https://facebook.com/user1"]
  - name: "User Two"
    instagram: ["https://instagram.com/user2"]
    linkedin: ["https://linkedin.com/in/user2"]
`,
			wantErrors: 0,
		},
		{
			name: "multiple URL errors",
			yamlContent: `version: "1.0"
contacts:
  - name: "User One"
    x: ["www.x.com/user1"]
    facebook: ["https://twitter.com/user1"]
  - name: "User Two"
    instagram: [""]
`,
			wantErrors: 3, // Missing scheme, wrong domain, empty URL
		},
		{
			name: "mixed valid and invalid URLs in same contact",
			yamlContent: `version: "1.0"
contacts:
  - name: "User One"
    x: ["https://x.com/valid", "www.x.com/invalid"]
    facebook: ["https://facebook.com/valid"]
`,
			wantErrors: 1, // One invalid x URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test-contacts.yaml")

			if err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			errors := validateContactsFile(tmpFile)

			if len(errors) != tt.wantErrors {
				t.Errorf("Expected %d errors, got %d", tt.wantErrors, len(errors))
				for i, err := range errors {
					t.Logf("Error %d: %s", i+1, err.String())
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
