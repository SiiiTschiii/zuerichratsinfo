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

// Account is one social media account in the mapping.
//
// It exists so a handle can sit in the file before anyone has confirmed it
// belongs to the person named beside it. Curating 180 cantonal members is weeks
// of one-by-one checking, and the alternative to holding candidates here was
// holding them somewhere the tooling could not see — which meant the work could
// not start until it was finished.
//
// Verified is the whole point: only a verified account is ever posted. See
// Contact.Verified, which is the single gate every reader goes through.
type Account struct {
	URL string `yaml:"url"`

	// Verified marks a handle a human has confirmed belongs to this person —
	// opened the profile and recognised them. It is not a guess that scored
	// well. Tagging the wrong account puts a real person's handle next to a
	// vote they did not cast, and this flag is what stands between a search
	// result and that happening.
	Verified bool `yaml:"verified"`

	// Confidence is scaffolding on an unverified candidate, and says only how
	// well a search result's own page title matched the person: "high",
	// "medium" or "low". It is an ordering aid for whoever works through the
	// list, never evidence of identity, and nothing reads it at post time.
	Confidence string `yaml:"confidence,omitempty"`
}

// UnmarshalYAML accepts both shapes the file uses.
//
// A bare string is a verified account. That is the form every handle in the
// mapping had before candidates existed, and each one got there because a human
// put it there — so reading it as verified preserves exactly what the file
// already meant, and spares ~360 entries a "verified: true" line that would say
// nothing.
//
// The mapping form is for everything else, and Verified defaults to false in
// it. The defaults therefore fall the safe way round: to publish a handle you
// have to say so, and forgetting the flag on a candidate leaves it silent.
func (a *Account) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		var url string
		if err := value.Decode(&url); err != nil {
			return err
		}
		*a = Account{URL: url, Verified: true}
		return nil
	}

	// A named alias, so decoding the mapping does not re-enter this method.
	type account Account
	var raw account
	if err := value.Decode(&raw); err != nil {
		return err
	}
	*a = Account(raw)
	return nil
}

// MarshalYAML writes back the shape the account came in as, so a verified
// handle stays one readable line and only candidates carry their metadata.
func (a Account) MarshalYAML() (any, error) {
	if a.Verified && a.Confidence == "" {
		return a.URL, nil
	}
	type account Account
	return account(a), nil
}

// VerifiedAccounts builds accounts that are already confirmed, which is what
// code constructing a mapping in memory — tests, fixtures — almost always
// means.
func VerifiedAccounts(urls ...string) []Account {
	out := make([]Account, 0, len(urls))
	for _, u := range urls {
		out = append(out, Account{URL: u, Verified: true})
	}
	return out
}

// Contact represents a council member with their social media accounts
// Fields are declared alphabetically because that order is what gets
// marshalled, and cmd/validate_contacts requires it in the file.
type Contact struct {
	Name      string    `yaml:"name"`
	Bluesky   []Account `yaml:"bluesky,omitempty"`
	Facebook  []Account `yaml:"facebook,omitempty"`
	Instagram []Account `yaml:"instagram,omitempty"`
	LinkedIn  []Account `yaml:"linkedin,omitempty"`
	TikTok    []Account `yaml:"tiktok,omitempty"`
	X         []Account `yaml:"x,omitempty"`
}

// Platforms are the platform keys the mapping supports, in the order the file
// carries them.
var Platforms = []string{"bluesky", "facebook", "instagram", "linkedin", "tiktok", "x"}

// Accounts returns every account recorded for a platform, verified or not.
//
// Callers that put a handle in front of readers must use Verified instead. This
// one is for the curation tools, which exist precisely to show what has not
// been confirmed yet.
func (c Contact) Accounts(platform string) []Account {
	switch strings.ToLower(platform) {
	case "x", "twitter":
		return c.X
	case "facebook":
		return c.Facebook
	case "instagram":
		return c.Instagram
	case "linkedin":
		return c.LinkedIn
	case "bluesky":
		return c.Bluesky
	case "tiktok":
		return c.TikTok
	default:
		return nil
	}
}

// Verified returns the URLs on a platform that a human has confirmed.
//
// This is the gate. Every path that can reach a published post — the tagger,
// the Mapper accessors, outreach — reads through here, so an unverified handle
// cannot be posted by forgetting to check a flag somewhere.
func (c Contact) Verified(platform string) []string {
	var out []string
	for _, a := range c.Accounts(platform) {
		if a.Verified && strings.TrimSpace(a.URL) != "" {
			out = append(out, a.URL)
		}
	}
	return out
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

// appendNew adds the accounts not already recorded, keeping the existing order.
//
// When the same URL arrives twice and either copy is verified, the merged one
// is verified: a confirmation is a fact about the account, and the file that
// happens to be read second should not be able to take it back.
func appendNew(existing, incoming []Account) []Account {
	for _, add := range incoming {
		found := false
		for i, have := range existing {
			if have.URL != add.URL {
				continue
			}
			found = true
			if add.Verified && !existing[i].Verified {
				existing[i].Verified = true
				existing[i].Confidence = ""
			}
			break
		}
		if !found {
			existing = append(existing, add)
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

// GetXHandle returns the first verified X (Twitter) handle for a name.
func (m *Mapper) GetXHandle(name string) string {
	handles := m.GetXHandles(name)
	if len(handles) == 0 {
		return ""
	}
	return handles[0]
}

// GetXHandles returns the verified X (Twitter) handles for a name.
func (m *Mapper) GetXHandles(name string) []string {
	return m.GetPlatformURLs(name, "x")
}

// GetPlatformURL returns the first URL for a specific platform, if available
func (m *Mapper) GetPlatformURL(name, platform string) string {
	urls := m.GetPlatformURLs(name, platform)
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

// GetPlatformURLs returns the verified URLs for a specific platform.
//
// Unverified candidates are deliberately invisible here: everything downstream
// of this method eventually reaches a reader.
func (m *Mapper) GetPlatformURLs(name, platform string) []string {
	contact, ok := m.GetContact(name)
	if !ok {
		return nil
	}
	return contact.Verified(platform)
}

// HasPlatform checks if a contact has a specific platform configured
func (m *Mapper) HasPlatform(name, platform string) bool {
	return len(m.GetPlatformURLs(name, platform)) > 0
}

// GetAllContacts returns all contacts without duplicates
func (m *Mapper) GetAllContacts() []Contact {
	return m.allContacts
}
