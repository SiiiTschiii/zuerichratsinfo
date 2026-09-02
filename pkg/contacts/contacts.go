package contacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	TikTok    []string `yaml:"tiktok,omitempty,flow"`    // Full URLs
}

// ContactMapping contains the full mapping structure
type ContactMapping struct {
	Version  string    `yaml:"version"`
	Contacts []Contact `yaml:"contacts"`
}

// Mapper provides name-to-contact lookups
type Mapper struct {
	contacts    map[string]Contact
	allContacts []Contact
}

// PathFor returns where a jurisdiction's curated contacts live.
func PathFor(jurisdiction string) string {
	return filepath.Join("data", jurisdiction, "contacts.yaml")
}

// LoadContacts loads the contact mapping from a YAML file
func LoadContacts(path string) (*Mapper, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read contacts file: %w", err)
	}

	var mapping ContactMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("failed to parse contacts YAML: %w", err)
	}

	return newMapper(mapping.Contacts), nil
}

// LoadContactFiles merges several contacts files into one mapper, which is what
// a channel serving more than one jurisdiction needs.
//
// A missing file is not an error: a jurisdiction ships before its ~180 members
// are curated, and the correct behaviour then is to post without tagging rather
// than not to post. A name in more than one file keeps the accounts from all of
// them — see mergeByName.
func LoadContactFiles(paths ...string) (*Mapper, error) {
	var all []Contact
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read contacts file %s: %w", path, err)
		}

		var mapping ContactMapping
		if err := yaml.Unmarshal(data, &mapping); err != nil {
			return nil, fmt.Errorf("failed to parse contacts YAML %s: %w", path, err)
		}
		all = append(all, mapping.Contacts...)
	}
	return newMapper(all), nil
}

// mergeByName folds contacts that share a name into one, keeping every account.
//
// Dual mandates make this necessary rather than tidy: someone sits in the
// Gemeinderat and the Kantonsrat at once, so they appear in both files — once
// with the handles verified for them, and once as a bare name in a roster still
// being curated. Letting the last file win would take a curated politician's
// tagging away, silently, because a second file happened to list them.
//
// Order is preserved: the first appearance fixes a contact's position, and
// later files only add to it.
func mergeByName(cs []Contact) []Contact {
	merged := make([]Contact, 0, len(cs))
	at := make(map[string]int, len(cs))

	for _, c := range cs {
		key := NameKey(c.Name)
		i, seen := at[key]
		if !seen {
			at[key] = len(merged)
			merged = append(merged, c)
			continue
		}

		into := &merged[i]
		into.X = appendNew(into.X, c.X)
		into.Facebook = appendNew(into.Facebook, c.Facebook)
		into.Instagram = appendNew(into.Instagram, c.Instagram)
		into.LinkedIn = appendNew(into.LinkedIn, c.LinkedIn)
		into.Bluesky = appendNew(into.Bluesky, c.Bluesky)
		into.TikTok = appendNew(into.TikTok, c.TikTok)
	}

	return merged
}

// appendNew adds the URLs not already recorded, keeping the existing order.
func appendNew(existing, incoming []string) []string {
	for _, url := range incoming {
		found := false
		for _, have := range existing {
			if have == url {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, url)
		}
	}
	return existing
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// NameKey is the identity of a person across files and sources, insensitive to
// case, spacing and the order of the name parts.
//
// The order has to be ignored because the same politician is written both ways
// in practice: PARIS serves "Bögli Moritz" while its own vote titles and the
// curated mapping say "Moritz Bögli". Keying on the exact string quietly
// created a second entry for eight sitting Gemeinderäte, which no validation
// catches — two entries with different names are not duplicates to a checker
// that compares strings, and only one of them ends up carrying the handles.
//
// Sorting the parts, rather than trying to guess which one is the surname, is
// what makes it work for "David Garcia Nuñez" too, where the split is
// ambiguous. Two different politicians whose names are anagrams by whole word
// would collide; there are none, and the alternative collides on real people.
func NameKey(name string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(name)))
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func newMapper(cs []Contact) *Mapper {
	cs = mergeByName(cs)

	contactMap := make(map[string]Contact, len(cs)*2)
	for _, contact := range cs {
		// Store by exact name
		contactMap[contact.Name] = contact

		// Also store normalized version (lowercase, trimmed)
		contactMap[normalizeName(contact.Name)] = contact
	}

	return &Mapper{
		contacts:    contactMap,
		allContacts: cs,
	}
}

// GetContact looks up a contact by name (case-insensitive)
func (m *Mapper) GetContact(name string) (Contact, bool) {
	// Try exact match first
	if contact, ok := m.contacts[name]; ok {
		return contact, true
	}

	// Try normalized match
	contact, ok := m.contacts[normalizeName(name)]
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
	case "tiktok":
		return contact.TikTok
	default:
		return nil
	}
}

// HasPlatform checks if a contact has a specific platform configured
func (m *Mapper) HasPlatform(name, platform string) bool {
	return len(m.GetPlatformURLs(name, platform)) > 0
}

// GetAllContacts returns all contacts without duplicates
func (m *Mapper) GetAllContacts() []Contact {
	return m.allContacts
}
