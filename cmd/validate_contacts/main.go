package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Contact represents a council member with their social media accounts
type Contact struct {
	Name      string   `yaml:"name"`
	X         []string `yaml:"x,omitempty"`
	Facebook  []string `yaml:"facebook,omitempty"`
	Instagram []string `yaml:"instagram,omitempty"`
	LinkedIn  []string `yaml:"linkedin,omitempty"`
	Bluesky   []string `yaml:"bluesky,omitempty"`
	TikTok    []string `yaml:"tiktok,omitempty"`
}

// ContactMapping contains the full mapping structure
type ContactMapping struct {
	Version  string    `yaml:"version"`
	Contacts []Contact `yaml:"contacts"`
}

var (
	supportedPlatforms = map[string]bool{
		"x":         true,
		"facebook":  true,
		"instagram": true,
		"linkedin":  true,
		"bluesky":   true,
		"tiktok":    true,
	}

	platformDomains = map[string][]string{
		"x":         {"x.com", "twitter.com"},
		"facebook":  {"facebook.com", "www.facebook.com"},
		"instagram": {"instagram.com", "www.instagram.com"},
		"linkedin":  {"linkedin.com", "www.linkedin.com"},
		"bluesky":   {"bsky.app", "web-cdn.bsky.app"},
		"tiktok":    {"tiktok.com", "www.tiktok.com"},
	}
)

type ValidationError struct {
	ContactName string
	Platform    string
	URL         string
	Message     string
}

func (e ValidationError) String() string {
	if e.URL != "" {
		return fmt.Sprintf("Contact '%s', platform '%s', URL '%s': %s", e.ContactName, e.Platform, e.URL, e.Message)
	}
	if e.Platform != "" {
		return fmt.Sprintf("Contact '%s', platform '%s': %s", e.ContactName, e.Platform, e.Message)
	}
	return fmt.Sprintf("Contact '%s': %s", e.ContactName, e.Message)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path-to-contacts.yaml>\n", os.Args[0])
		os.Exit(1)
	}

	filepath := os.Args[1]
	
	errors := validateContactsFile(filepath)
	
	if len(errors) > 0 {
		fmt.Printf("❌ Validation failed with %d error(s):\n\n", len(errors))
		for i, err := range errors {
			fmt.Printf("%d. %s\n", i+1, err.String())
		}
		os.Exit(1)
	}

	fmt.Println("✅ Validation successful! contacts.yaml is valid.")
	os.Exit(0)
}

func validateContactsFile(filepath string) []ValidationError {
	var errors []ValidationError

	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		errors = append(errors, ValidationError{
			Message: fmt.Sprintf("Failed to read file: %v", err),
		})
		return errors
	}

	// Validate YAML syntax with strict mode to catch unknown fields
	var mapping ContactMapping
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // This will error on unknown fields
	if err := decoder.Decode(&mapping); err != nil {
		errors = append(errors, ValidationError{
			Message: fmt.Sprintf("Invalid YAML: %v", err),
		})
		return errors
	}

	// Validate structure
	if mapping.Version == "" {
		errors = append(errors, ValidationError{
			Message: "Missing 'version' field",
		})
	}

	if len(mapping.Contacts) == 0 {
		errors = append(errors, ValidationError{
			Message: "No contacts found in file",
		})
		return errors
	}

	// Validate each contact
	seenNames := make(map[string]bool)
	for i, contact := range mapping.Contacts {
		// Validate name
		if contact.Name == "" {
			errors = append(errors, ValidationError{
				Message: fmt.Sprintf("Contact at index %d has no name", i),
			})
			continue
		}

		// Check for duplicate names
		if seenNames[contact.Name] {
			errors = append(errors, ValidationError{
				ContactName: contact.Name,
				Message:     "Duplicate contact name",
			})
		}
		seenNames[contact.Name] = true

		// Validate platforms and URLs
		errors = append(errors, validateContactPlatforms(contact)...)
	}

	return errors
}

func validateContactPlatforms(contact Contact) []ValidationError {
	var errors []ValidationError

	// Check each platform
	platforms := map[string][]string{
		"x":         contact.X,
		"facebook":  contact.Facebook,
		"instagram": contact.Instagram,
		"linkedin":  contact.LinkedIn,
		"bluesky":   contact.Bluesky,
		"tiktok":    contact.TikTok,
	}

	for platform, urls := range platforms {
		// Validate platform is supported
		if !supportedPlatforms[platform] {
			errors = append(errors, ValidationError{
				ContactName: contact.Name,
				Platform:    platform,
				Message:     "Unsupported platform",
			})
			continue
		}

		// Validate each URL
		for _, urlStr := range urls {
			if err := validateURL(urlStr, platform); err != nil {
				errors = append(errors, ValidationError{
					ContactName: contact.Name,
					Platform:    platform,
					URL:         urlStr,
					Message:     err.Error(),
				})
			}
		}
	}

	// Check if contact has at least one platform (warn if empty, but don't error)
	hasAnyPlatform := false
	for _, urls := range platforms {
		if len(urls) > 0 {
			hasAnyPlatform = true
			break
		}
	}
	if !hasAnyPlatform {
		// This is just informational, not an error
		// fmt.Printf("Info: Contact '%s' has no social media platforms\n", contact.Name)
	}

	return errors
}

func validateURL(urlStr, platform string) error {
	// Check if URL is empty
	if strings.TrimSpace(urlStr) == "" {
		return fmt.Errorf("empty URL")
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	// Check hostname matches platform
	hostname := strings.ToLower(parsedURL.Hostname())
	allowedDomains := platformDomains[platform]
	
	validDomain := false
	for _, domain := range allowedDomains {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			validDomain = true
			break
		}
	}

	if !validDomain {
		return fmt.Errorf("URL domain '%s' does not match platform '%s' (expected one of: %v)", 
			hostname, platform, allowedDomains)
	}

	return nil
}
