package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"gopkg.in/yaml.v3"
)

// The schema lives in pkg/contacts, so the validator checks the same types the
// bot reads rather than a copy that can drift away from them.
type (
	Account        = contacts.Account
	Contact        = contacts.Contact
	ContactMapping = contacts.ContactMapping
)

// validConfidence are the values a candidate may carry. It is an ordering aid
// for whoever works through the list — never evidence of identity — so an
// invented level would only mislead the person doing the verifying.
var validConfidence = map[string]bool{"high": true, "medium": true, "low": true}

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
		fmt.Println("✅ Contacts and platforms sorted alphabetically and file updated.")
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

// sortHint names the exact command that fixes an ordering complaint, for the
// file actually being validated — there is more than one.
func sortHint(filepath string) string {
	return "go run cmd/validate_contacts/main.go -sort " + filepath
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

	// Check contacts are sorted alphabetically by name
	if !skipOrderCheck {
		for i := 1; i < len(mapping.Contacts); i++ {
			prev := mapping.Contacts[i-1].Name
			curr := mapping.Contacts[i].Name
			if strings.ToLower(curr) < strings.ToLower(prev) {
				errors = append(errors, ValidationError{
					ContactName: curr,
					Message:     fmt.Sprintf("Contact is out of alphabetical order (comes after '%s'). Run with -sort flag to fix: %s", prev, sortHint(filepath)),
				})
			}
		}
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

		// Check if name contains URL patterns (signs of corrupted YAML)
		if strings.Contains(contact.Name, "http://") || strings.Contains(contact.Name, "https://") {
			errors = append(errors, ValidationError{
				ContactName: contact.Name,
				Message:     "Name field contains URL - this indicates corrupted YAML structure. Check if platform data was incorrectly merged into the name field.",
			})
		}

		// Check if name contains platform field patterns (e.g., "Name facebook: url")
		for platform := range supportedPlatforms {
			if strings.Contains(strings.ToLower(contact.Name), platform+":") {
				errors = append(errors, ValidationError{
					ContactName: contact.Name,
					Message:     fmt.Sprintf("Name field contains platform data ('%s:') - this indicates corrupted YAML structure", platform),
				})
				break
			}
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

		// Check for empty platforms and platform order in the file
		if orderErr := checkPlatformOrderInFile(filepath, contact.Name, skipOrderCheck); orderErr != nil {
			errors = append(errors, *orderErr)
		}
	}

	return errors
}

func validateContactPlatforms(contact Contact) []ValidationError {
	var errors []ValidationError

	for _, platform := range contacts.Platforms {
		if !supportedPlatforms[platform] {
			errors = append(errors, ValidationError{
				ContactName: contact.Name,
				Platform:    platform,
				Message:     "Unsupported platform",
			})
			continue
		}

		for _, account := range contact.Accounts(platform) {
			if err := validateURL(account.URL, platform); err != nil {
				errors = append(errors, ValidationError{
					ContactName: contact.Name,
					Platform:    platform,
					URL:         account.URL,
					Message:     err.Error(),
				})
			}
			if c := account.Confidence; c != "" && !validConfidence[strings.ToLower(c)] {
				errors = append(errors, ValidationError{
					ContactName: contact.Name,
					Platform:    platform,
					URL:         account.URL,
					Message: fmt.Sprintf("Unknown confidence %q (want high, medium or low). "+
						"Confidence records how well a search result matched the person; "+
						"it is not evidence of identity.", c),
				})
			}
			// A verified account has been confirmed by a human, so the score
			// that once ranked it as a guess is spent. Leaving it behind reads
			// as though the confirmation were itself only likely.
			if account.Verified && account.Confidence != "" {
				errors = append(errors, ValidationError{
					ContactName: contact.Name,
					Platform:    platform,
					URL:         account.URL,
					Message:     "Verified accounts must not keep a confidence score — drop it once you have confirmed the account.",
				})
			}
		}
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

func checkPlatformOrderInFile(filepath string, contactName string, skipOrderCheck bool) *ValidationError {
	// Read the file and parse line by line to get actual field order
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil // Skip check if can't read file
	}

	lines := strings.Split(string(data), "\n")
	inContact := false
	platformsInFile := []string{}
	emptyPlatforms := []string{}

	for i, line := range lines {
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
				emptyPlatforms = []string{}
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

					// Check if the platform has an empty value
					valueAfterColon := strings.TrimSpace(parts[1])
					if valueAfterColon == "" {
						// Check the next line to see if it has an array element
						hasValue := false
						if i+1 < len(lines) {
							nextLine := strings.TrimSpace(lines[i+1])
							// Check if next line starts with "- " (array element)
							if strings.HasPrefix(nextLine, "- ") {
								hasValue = true
							}
						}

						if !hasValue {
							emptyPlatforms = append(emptyPlatforms, fieldName)
						}
					}
				}
			}
		}
	}

	// Always check for empty platforms (even when skipOrderCheck is true)
	if len(emptyPlatforms) > 0 {
		return &ValidationError{
			ContactName: contactName,
			Message:     fmt.Sprintf("Platform(s) without values: %s. Remove the platform field or add a value.", strings.Join(emptyPlatforms, ", ")),
		}
	}

	if len(platformsInFile) == 0 || skipOrderCheck {
		return nil // No platforms to check or skipping order check
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
				Message:     "Platforms are not in alphabetical order. Run with -sort flag to fix: " + sortHint(filepath),
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

	// Sort contacts alphabetically by name (case-insensitive)
	sort.Slice(mapping.Contacts, func(i, j int) bool {
		return strings.ToLower(mapping.Contacts[i].Name) < strings.ToLower(mapping.Contacts[j].Name)
	})

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
