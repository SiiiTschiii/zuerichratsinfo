// Package main fetches contacts from Zurich API and merges them into contacts.yaml
// Usage: go run cmd/update_contacts/main.go
//
// This script:
// 1. Loads existing data/contacts.yaml
// 2. Fetches contacts from Zurich API
// 3. For each contact:
//   - If new: add with all their accounts
//   - If exists: merge accounts (warn on conflicts)
//
// 4. Saves updated contacts.yaml (append-only)
package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
	"gopkg.in/yaml.v3"
)

type Contact struct {
	Name      string   `yaml:"name"`
	X         []string `yaml:"x,omitempty,flow"`
	Facebook  []string `yaml:"facebook,omitempty,flow"`
	Instagram []string `yaml:"instagram,omitempty,flow"`
	LinkedIn  []string `yaml:"linkedin,omitempty,flow"`
	Bluesky   []string `yaml:"bluesky,omitempty,flow"`
}

type ContactMapping struct {
	Version  string    `yaml:"version"`
	Contacts []Contact `yaml:"contacts"`
}

const contactsFile = "data/contacts.yaml"

func main() {
	fmt.Println("📥 Fetching contacts from Zurich API...")

	// Step 1: Load existing contacts.yaml
	existingContacts := loadExistingContacts()
	fmt.Printf("📋 Loaded %d existing contacts\n", len(existingContacts))

	// Step 2: Fetch contacts from API
	apiContacts := fetchContactsFromAPI()
	fmt.Printf("🌐 Fetched %d contacts from API\n", len(apiContacts))

	// Step 3: Merge contacts
	updated, added, warnings := mergeContacts(existingContacts, apiContacts)

	// Step 4: Save updated contacts.yaml
	saveContacts(updated)

	// Summary
	fmt.Println("\n✅ Update complete!")
	fmt.Printf("   - Total contacts: %d\n", len(updated))
	fmt.Printf("   - New contacts added: %d\n", added)
	fmt.Printf("   - Warnings: %d\n", warnings)
}

func loadExistingContacts() map[string]*Contact {
	contacts := make(map[string]*Contact)

	data, err := os.ReadFile(contactsFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("⚠️  No existing contacts.yaml found, will create new one")
			return contacts
		}
		log.Fatalf("Failed to read contacts file: %v", err)
	}

	var mapping ContactMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		log.Fatalf("Failed to parse contacts YAML: %v", err)
	}

	for i := range mapping.Contacts {
		contact := &mapping.Contacts[i]
		contacts[contact.Name] = contact
	}

	return contacts
}

func fetchContactsFromAPI() []Contact {
	client := zurichapi.NewClient()

	apiContacts, err := client.FetchAllKontakte()
	if err != nil {
		log.Fatalf("Failed to fetch contacts: %v", err)
	}

	var contacts []Contact
	for _, apiContact := range apiContacts {
		name := strings.ReplaceAll(apiContact.NameVorname, "\u00a0", " ")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		contact := Contact{Name: name}
		hasAccount := false

		for _, sm := range apiContact.SozialeMedien.Kommunikation {
			url := strings.TrimSpace(sm.Adresse)
			if url == "" {
				continue
			}

			hasAccount = true
			platform := strings.ToLower(strings.TrimSpace(sm.Typ))

			switch platform {
			case "x", "twitter":
				// Normalize twitter.com to x.com
				url = strings.ReplaceAll(url, "twitter.com", "x.com")
				contact.X = append(contact.X, url)
			case "facebook":
				contact.Facebook = append(contact.Facebook, url)
			case "instagram":
				contact.Instagram = append(contact.Instagram, url)
			case "linkedin":
				contact.LinkedIn = append(contact.LinkedIn, url)
			case "bluesky":
				contact.Bluesky = append(contact.Bluesky, url)
			}
		}

		// Only add contacts with at least one social media account
		if hasAccount {
			contacts = append(contacts, contact)
		}
	}

	return contacts
}

func mergeContacts(existing map[string]*Contact, apiContacts []Contact) ([]Contact, int, int) {
	added := 0
	warnings := 0

	for _, apiContact := range apiContacts {
		existingContact, exists := existing[apiContact.Name]

		if !exists {
			// New contact - add it
			existing[apiContact.Name] = &apiContact
			added++
			fmt.Printf("➕ New: %s\n", apiContact.Name)
			continue
		}

		// Contact exists - merge accounts
		merged := mergeAccounts(existingContact, &apiContact)
		warnings += merged
	}

	// Convert map back to slice and sort alphabetically by name
	result := make([]Contact, 0, len(existing))
	for _, contact := range existing {
		result = append(result, *contact)
	}

	// Sort by name for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, added, warnings
}

func mergeAccounts(existing, new *Contact) int {
	warnings := 0

	// Helper function to merge a platform's accounts
	mergePlatform := func(platformName, emoji string, existingAccounts, newAccounts *[]string) {
		for _, newAccount := range *newAccounts {
			found := false
			for _, existingAccount := range *existingAccounts {
				if existingAccount == newAccount {
					found = true
					break
				}
			}

			if !found {
				// Check if it might be a conflicting account (same platform, different URL)
				if len(*existingAccounts) > 0 {
					// We have existing accounts but this is a new one - could be additional or conflict
					// For now, treat as additional account
					*existingAccounts = append(*existingAccounts, newAccount)
					fmt.Printf("   %s Added additional %s for %s: %s\n", emoji, platformName, existing.Name, newAccount)
				} else {
					// First account for this platform
					*existingAccounts = append(*existingAccounts, newAccount)
					fmt.Printf("   %s Added %s for %s: %s\n", emoji, platformName, existing.Name, newAccount)
				}
			}
		}
	}

	// Merge each platform
	mergePlatform("X", "📱", &existing.X, &new.X)
	mergePlatform("Facebook", "📘", &existing.Facebook, &new.Facebook)
	mergePlatform("Instagram", "📷", &existing.Instagram, &new.Instagram)
	mergePlatform("LinkedIn", "💼", &existing.LinkedIn, &new.LinkedIn)
	mergePlatform("Bluesky", "🦋", &existing.Bluesky, &new.Bluesky)

	return warnings
}

func saveContacts(contacts []Contact) {
	mapping := ContactMapping{
		Version:  "1.0",
		Contacts: contacts,
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&mapping)
	if err != nil {
		log.Fatalf("Failed to marshal YAML: %v", err)
	}

	// Convert to VS Code-compatible formatting (2 spaces, dash at list level)
	formattedYAML := formatYAMLForVSCode(string(yamlData))

	// Build output with header
	output := "# Contact to Social Media Mapping\n"
	output += "# This file maps names of politicians to their social media accounts\n"
	output += fmt.Sprintf("# Last updated: %s\n\n", time.Now().Format("2006-01-02"))
	output += formattedYAML

	if err := os.WriteFile(contactsFile, []byte(output), 0644); err != nil {
		log.Fatalf("Failed to write contacts file: %v", err)
	}

	fmt.Printf("\n💾 Saved to %s\n", contactsFile)
}

// formatYAMLForVSCode converts Go YAML style to VS Code style
// Go marshaler uses: 4 spaces for list items, 6 spaces for fields
// VS Code wants: 2 spaces for list items, 4 spaces for fields
// Also adds blank lines between contacts for better readability
func formatYAMLForVSCode(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	var result []string
	var previousWasContact bool

	for _, line := range lines {
		if len(line) == 0 {
			result = append(result, line)
			continue
		}

		// Count leading spaces
		leadingSpaces := 0
		for _, ch := range line {
			if ch == ' ' {
				leadingSpaces++
			} else {
				break
			}
		}

		// Convert indentation based on the pattern
		if leadingSpaces > 0 {
			var newSpaces int
			content := strings.TrimLeft(line, " ")

			// List item line (starts with -)
			if strings.HasPrefix(content, "- ") {
				// 4 spaces → 2 spaces, 8 spaces → 4 spaces, etc.
				newSpaces = leadingSpaces / 2

				// Add blank line before each contact (except the first one)
				if strings.HasPrefix(content, "- name:") && previousWasContact {
					result = append(result, "")
				}
				previousWasContact = strings.HasPrefix(content, "- name:")
			} else {
				// Field line (has a colon)
				// 6 spaces → 4 spaces, 10 spaces → 6 spaces, etc.
				// Pattern: subtract 2 from original
				newSpaces = leadingSpaces - 2
			}

			line = strings.Repeat(" ", newSpaces) + content
		}

		// Convert single quotes to double quotes for consistency
		line = strings.ReplaceAll(line, "['", "[\"")
		line = strings.ReplaceAll(line, "']", "\"]")
		line = strings.ReplaceAll(line, "', '", "\", \"")

		// Add quotes around name values to avoid whitespace issues
		if strings.Contains(line, "name: ") {
			// Extract name value (everything after "name: ")
			parts := strings.SplitN(line, "name: ", 2)
			if len(parts) == 2 {
				nameValue := strings.TrimSpace(parts[1])
				// Only add quotes if not already quoted
				if !strings.HasPrefix(nameValue, "\"") && nameValue != "" {
					line = parts[0] + "name: \"" + nameValue + "\""
				}
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
