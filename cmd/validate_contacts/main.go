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

	errors = append(errors, checkAccountShape(string(data))...)

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

// accountFields are the only keys an account may carry. Anything else is a
// typo, and a typo in this file is dangerous in one specific direction: the
// custom unmarshaller ignores what it does not recognise, so `verifed: true`
// would decode as unverified and quietly stop tagging someone.
var accountFields = map[string]bool{"url": true, "verified": true, "confidence": true}

// checkAccountShape walks the file's nodes to enforce what the parsed value can
// no longer tell us apart.
//
// Both `- https://…` and `- url: …` with no `verified` decode to the same
// unverified Account, so by the time validateContactPlatforms sees it the
// omission is invisible. Whether a handle may be published is the one thing
// this file has to state out loud, and this is what makes it say so.
func checkAccountShape(content string) []ValidationError {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil || len(root.Content) == 0 {
		return nil // the decode above already reported this
	}

	var errors []ValidationError
	forEachAccount(&root, func(name, platform string, account *yaml.Node) {
		if account.Kind == yaml.ScalarNode {
			errors = append(errors, ValidationError{
				ContactName: name, Platform: platform, URL: account.Value,
				Message: "Account is written as a bare URL. Give it an explicit flag: " +
					"`- url: <URL>` followed by `verified: true` (a human confirmed this account) " +
					"or `verified: false` (a candidate, never posted).",
			})
			return
		}
		if account.Kind != yaml.MappingNode {
			return
		}

		seen := map[string]bool{}
		url := ""
		for i := 0; i+1 < len(account.Content); i += 2 {
			key := account.Content[i].Value
			seen[key] = true
			if key == "url" {
				url = account.Content[i+1].Value
			}
			if !accountFields[key] {
				errors = append(errors, ValidationError{
					ContactName: name, Platform: platform,
					Message: fmt.Sprintf("Unknown account field %q (want url, verified or confidence). "+
						"A misspelt `verified` decodes as false and silently stops the account being posted.", key),
				})
			}
		}
		if !seen["verified"] {
			errors = append(errors, ValidationError{
				ContactName: name, Platform: platform, URL: url,
				Message: "Account does not say whether it may be published. Add `verified: true` " +
					"(a human confirmed this account) or `verified: false` (a candidate, never posted).",
			})
		}
	})
	return errors
}

// forEachAccount visits every account node under every contact's platform keys.
func forEachAccount(root *yaml.Node, visit func(name, platform string, account *yaml.Node)) {
	doc := root.Content[0]
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value != "contacts" {
			continue
		}
		for _, contact := range doc.Content[i+1].Content {
			name := ""
			for j := 0; j+1 < len(contact.Content); j += 2 {
				if contact.Content[j].Value == "name" {
					name = contact.Content[j+1].Value
				}
			}
			for j := 0; j+1 < len(contact.Content); j += 2 {
				platform := contact.Content[j].Value
				if !supportedPlatforms[platform] {
					continue
				}
				for _, account := range contact.Content[j+1].Content {
					visit(name, platform, account)
				}
			}
		}
	}
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
