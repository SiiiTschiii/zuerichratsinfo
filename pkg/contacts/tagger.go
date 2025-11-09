package contacts

import (
	"fmt"
	"regexp"
	"strings"
)

// XHandleTag represents a name and its X handle for tagging
type XHandleTag struct {
	Name   string
	Handle string
}

// ExtractXHandleFromURL extracts the @handle from an X/Twitter URL
// Examples:
//   - "https://x.com/MoritzBoegli" -> "@MoritzBoegli"
//   - "https://twitter.com/thuritch" -> "@thuritch"
//   - "www.x.com/pascallamprecht" -> "@pascallamprecht"
func ExtractXHandleFromURL(url string) string {
	// Remove common prefixes
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	
	// Handle both x.com and twitter.com
	if strings.HasPrefix(url, "x.com/") {
		handle := strings.TrimPrefix(url, "x.com/")
		return "@" + strings.TrimSpace(handle)
	}
	if strings.HasPrefix(url, "twitter.com/") {
		handle := strings.TrimPrefix(url, "twitter.com/")
		return "@" + strings.TrimSpace(handle)
	}
	
	return ""
}

// generateNameVariants creates name permutations for flexible matching
// For "Bögli Moritz" -> ["Bögli Moritz", "Moritz Bögli"]
// For "Garcia Nuñez David" -> ["Garcia Nuñez David", "David Garcia Nuñez"]
func generateNameVariants(name string) []string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) < 2 {
		return []string{strings.TrimSpace(name)}
	}
	
	// Join parts to create normalized original
	normalized := strings.Join(parts, " ")
	variants := []string{normalized}
	
	// Simple reversal: last word becomes first, everything else follows
	// "A B C" -> "C A B"
	reversed := parts[len(parts)-1] + " " + strings.Join(parts[:len(parts)-1], " ")
	variants = append(variants, reversed)
	
	return variants
}

// TagXHandlesInText finds all contacts with X accounts in the text and adds their @handle
// Returns the modified text with inline tags like: "Moritz Bögli @MoritzBoegli"
// Uses greedy matching to prefer longer names (e.g., "David Garcia Nuñez" over "David Garcia")
func (m *Mapper) TagXHandlesInText(text string) string {
	// Collect all contacts with X accounts and their handles
	var taggableContacts []XHandleTag
	
	for _, contact := range m.getAllContacts() {
		if len(contact.X) == 0 {
			continue
		}
		
		handle := ExtractXHandleFromURL(contact.X[0])
		if handle == "" {
			continue
		}
		
		// Generate name variants for flexible matching
		variants := generateNameVariants(contact.Name)
		for _, variant := range variants {
			taggableContacts = append(taggableContacts, XHandleTag{
				Name:   variant,
				Handle: handle,
			})
		}
	}
	
	// Sort by name length (descending) to match longer names first
	// This prevents "David Garcia" from matching before "David Garcia Nuñez"
	for i := 0; i < len(taggableContacts); i++ {
		for j := i + 1; j < len(taggableContacts); j++ {
			if len(taggableContacts[i].Name) < len(taggableContacts[j].Name) {
				taggableContacts[i], taggableContacts[j] = taggableContacts[j], taggableContacts[i]
			}
		}
	}
	
	// Collect all matches first (position in original text, name, handle)
	type match struct {
		start  int
		end    int
		name   string
		handle string
	}
	var matches []match
	
	for _, tag := range taggableContacts {
		// Create a regex that matches the name with word boundaries
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(tag.Name))
		re := regexp.MustCompile(pattern)
		
		// Find all matches
		indices := re.FindAllStringIndex(text, -1)
		for _, idx := range indices {
			matches = append(matches, match{
				start:  idx[0],
				end:    idx[1],
				name:   tag.Name,
				handle: tag.Handle,
			})
		}
	}
	
	// Remove overlapping matches (keep longest/first)
	var filtered []match
	for _, m1 := range matches {
		overlaps := false
		for _, m2 := range filtered {
			// Check if m1 overlaps with m2
			if (m1.start >= m2.start && m1.start < m2.end) ||
				(m1.end > m2.start && m1.end <= m2.end) ||
				(m1.start <= m2.start && m1.end >= m2.end) {
				overlaps = true
				break
			}
		}
		if !overlaps {
			filtered = append(filtered, m1)
		}
	}
	
	// Sort by position (descending) so we can work backwards
	// Working backwards means indices stay valid as we insert
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[i].start < filtered[j].start {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}
	
	// Apply tags by working backwards through the text
	result := text
	for _, m := range filtered {
		// Insert handle after the name
		result = result[:m.end] + " " + m.handle + result[m.end:]
	}
	
	return result
}

// getAllContacts returns all contacts (helper method for internal use)
func (m *Mapper) getAllContacts() []Contact {
	seen := make(map[string]bool)
	var contacts []Contact
	
	for _, contact := range m.contacts {
		// Use name as unique key (since we store both original and normalized)
		if !seen[contact.Name] {
			seen[contact.Name] = true
			contacts = append(contacts, contact)
		}
	}
	
	return contacts
}
