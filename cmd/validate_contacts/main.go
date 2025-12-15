package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Contact represents a council member with their social media accounts
type Contact struct {
	Name      string   `yaml:"name"`
	Bluesky   []string `yaml:"bluesky,omitempty"`
	Facebook  []string `yaml:"facebook,omitempty"`
	Instagram []string `yaml:"instagram,omitempty"`
	LinkedIn  []string `yaml:"linkedin,omitempty"`
	TikTok    []string `yaml:"tiktok,omitempty"`
	X         []string `yaml:"x,omitempty"`
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
	sortFlag := flag.Bool("sort", false, "Sort platforms alphabetically and write back to file")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-sort] <path-to-contacts.yaml>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -sort    Sort platforms alphabetically and rewrite the file\n")
		os.Exit(1)
	}

	filepath := flag.Arg(0)

	// If sorting, validate without order check, then sort
	if *sortFlag {
		errors := validateContactsFile(filepath, true) // skip order check
		if len(errors) > 0 {
			fmt.Printf("❌ Validation failed with %d error(s):\n\n", len(errors))
			for i, err := range errors {
				fmt.Printf("%d. %s\n", i+1, err.String())
			}
			os.Exit(1)
		}
		
		if err := sortAndWriteContacts(filepath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to sort and write file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Platforms sorted alphabetically and file updated.")
		os.Exit(0)
	}

	// Normal validation including order check
	errors := validateContactsFile(filepath, false)

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

func validateContactsFile(filepath string, skipOrderCheck bool) []ValidationError {
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
		
		// Check if platforms are in alphabetical order (unless skipping)
		if !skipOrderCheck {
			if orderErr := checkPlatformOrderInFile(filepath, contact.Name); orderErr != nil {
				errors = append(errors, *orderErr)
			}
		}
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
	// Note: hasAnyPlatform is computed but not currently used for validation
	_ = hasAnyPlatform

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

func checkPlatformOrderInFile(filepath string, contactName string) *ValidationError {
	// Read the file and parse line by line to get actual field order
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil // Skip check if can't read file
	}
	
	lines := strings.Split(string(data), "\n")
	inContact := false
	platformsInFile := []string{}
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're at the contact we're looking for
		if strings.HasPrefix(trimmed, "- name:") || strings.HasPrefix(trimmed, "name:") {
			// Extract name from the line
			namePart := strings.TrimPrefix(trimmed, "- name:")
			namePart = strings.TrimPrefix(namePart, "name:")
			namePart = strings.TrimSpace(namePart)
			namePart = strings.Trim(namePart, "\"'")
			
			if namePart == contactName {
				inContact = true
				platformsInFile = []string{}
			} else if inContact {
				// We've moved to the next contact
				break
			}
		} else if inContact && strings.Contains(trimmed, ":") {
			// Check if this is a platform field
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				fieldName := strings.TrimSpace(parts[0])
				// Check if it's one of our platform fields
				if fieldName == "x" || fieldName == "facebook" || fieldName == "instagram" ||
					fieldName == "linkedin" || fieldName == "bluesky" || fieldName == "tiktok" {
					platformsInFile = append(platformsInFile, fieldName)
				}
			}
		}
	}
	
	if len(platformsInFile) == 0 {
		return nil // No platforms to check
	}
	
	// Create a sorted copy
	sortedPlatforms := make([]string, len(platformsInFile))
	copy(sortedPlatforms, platformsInFile)
	sort.Strings(sortedPlatforms)
	
	// Compare
	for i := range platformsInFile {
		if platformsInFile[i] != sortedPlatforms[i] {
			return &ValidationError{
				ContactName: contactName,
				Message:     "Platforms are not in alphabetical order. Run with -sort flag to fix: go run cmd/validate_contacts/main.go -sort data/contacts.yaml",
			}
		}
	}
	
	return nil
}

func sortAndWriteContacts(filepath string) error {
	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Parse YAML
	var mapping ContactMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return fmt.Errorf("failed to parse YAML: %v", err)
	}

	// No need to sort - struct fields are already in alphabetical order
	// The YAML marshaler will output them in struct field order

	// Marshal back to YAML
	output, err := yaml.Marshal(&mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	// Write back to file
	if err := os.WriteFile(filepath, output, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}
