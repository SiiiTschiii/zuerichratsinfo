package contacts

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Contact represents a council member with their social media accounts
type Contact struct {
	Name      string   `yaml:"name"`
	X         []string `yaml:"x,omitempty,flow"`         // X (Twitter) handles with @
	Facebook  []string `yaml:"facebook,omitempty,flow"`  // Full URLs
	Instagram []string `yaml:"instagram,omitempty,flow"` // Full URLs
	LinkedIn  []string `yaml:"linkedin,omitempty,flow"`  // Full URLs
	Bluesky   []string `yaml:"bluesky,omitempty,flow"`   // Full URLs
}

// ContactMapping contains the full mapping structure
type ContactMapping struct {
	Version  string    `yaml:"version"`
	Contacts []Contact `yaml:"contacts"`
}

// Mapper provides name-to-contact lookups
type Mapper struct {
	contacts map[string]Contact
}

// LoadContacts loads the contact mapping from a YAML file
func LoadContacts(filepath string) (*Mapper, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read contacts file: %w", err)
	}

	var mapping ContactMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("failed to parse contacts YAML: %w", err)
	}

	// Build lookup map
	contactMap := make(map[string]Contact)
	for _, contact := range mapping.Contacts {
		// Store by exact name
		contactMap[contact.Name] = contact
		
		// Also store normalized version (lowercase, trimmed)
		normalized := strings.ToLower(strings.TrimSpace(contact.Name))
		contactMap[normalized] = contact
	}

	return &Mapper{contacts: contactMap}, nil
}

// GetContact looks up a contact by name (case-insensitive)
func (m *Mapper) GetContact(name string) (Contact, bool) {
	// Try exact match first
	if contact, ok := m.contacts[name]; ok {
		return contact, true
	}
	
	// Try normalized match
	normalized := strings.ToLower(strings.TrimSpace(name))
	contact, ok := m.contacts[normalized]
	return contact, ok
}

// GetXHandle returns the first X (Twitter) handle for a name, if available
func (m *Mapper) GetXHandle(name string) string {
	contact, ok := m.GetContact(name)
	if !ok || len(contact.X) == 0 {
		return ""
	}
	return contact.X[0]
}

// GetXHandles returns all X (Twitter) handles for a name
func (m *Mapper) GetXHandles(name string) []string {
	contact, ok := m.GetContact(name)
	if !ok {
		return nil
	}
	return contact.X
}

// GetPlatformURL returns the first URL for a specific platform, if available
func (m *Mapper) GetPlatformURL(name, platform string) string {
	urls := m.GetPlatformURLs(name, platform)
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

// GetPlatformURLs returns all URLs for a specific platform
func (m *Mapper) GetPlatformURLs(name, platform string) []string {
	contact, ok := m.GetContact(name)
	if !ok {
		return nil
	}

	switch strings.ToLower(platform) {
	case "x", "twitter":
		return contact.X
	case "facebook":
		return contact.Facebook
	case "instagram":
		return contact.Instagram
	case "linkedin":
		return contact.LinkedIn
	case "bluesky":
		return contact.Bluesky
	default:
		return nil
	}
}

// HasPlatform checks if a contact has a specific platform configured
func (m *Mapper) HasPlatform(name, platform string) bool {
	return len(m.GetPlatformURLs(name, platform)) > 0
}
