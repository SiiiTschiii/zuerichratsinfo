package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

type Contact struct {
	Name      string   `yaml:"name"`
	X         []string `yaml:"x,omitempty"`
	Instagram []string `yaml:"instagram,omitempty"`
	Facebook  []string `yaml:"facebook,omitempty"`
	LinkedIn  []string `yaml:"linkedin,omitempty"`
	TikTok    []string `yaml:"tiktok,omitempty"`
	Bluesky   []string `yaml:"bluesky,omitempty"`
}

type Contacts struct {
	Version  string    `yaml:"version"`
	Contacts []Contact `yaml:"contacts"`
}

type PlatformStat struct {
	Name  string
	Count int
}

func main() {
	// Read contacts.yaml
	data, err := os.ReadFile("data/contacts.yaml")
	if err != nil {
		log.Fatalf("Error reading contacts.yaml: %v", err)
	}

	var contacts Contacts
	err = yaml.Unmarshal(data, &contacts)
	if err != nil {
		log.Fatalf("Error parsing contacts.yaml: %v", err)
	}

	// Count platforms
	stats := map[string]int{
		"X (Twitter)": 0,
		"Instagram":   0,
		"Facebook":    0,
		"LinkedIn":    0,
		"TikTok":      0,
		"Bluesky":     0,
	}

	for _, contact := range contacts.Contacts {
		if len(contact.X) > 0 {
			stats["X (Twitter)"]++
		}
		if len(contact.Instagram) > 0 {
			stats["Instagram"]++
		}
		if len(contact.Facebook) > 0 {
			stats["Facebook"]++
		}
		if len(contact.LinkedIn) > 0 {
			stats["LinkedIn"]++
		}
		if len(contact.TikTok) > 0 {
			stats["TikTok"]++
		}
		if len(contact.Bluesky) > 0 {
			stats["Bluesky"]++
		}
	}

	// Convert to slice and sort by count (descending)
	var platformStats []PlatformStat
	for name, count := range stats {
		platformStats = append(platformStats, PlatformStat{Name: name, Count: count})
	}
	sort.Slice(platformStats, func(i, j int) bool {
		return platformStats[i].Count > platformStats[j].Count
	})

	// Find max count for scaling
	maxCount := 0
	for _, stat := range platformStats {
		if stat.Count > maxCount {
			maxCount = stat.Count
		}
	}

	// Update README.md
	updateREADME(stats, len(contacts.Contacts))
}

func updateREADME(stats map[string]int, totalContacts int) {
	// Read README.md
	readmeData, err := os.ReadFile("README.md")
	if err != nil {
		log.Printf("Warning: Could not read README.md: %v", err)
		return
	}

	readme := string(readmeData)
	originalReadme := readme

	// Create a map for easier lookup with platform names as they appear in the table
	platformMap := map[string]int{
		"LinkedIn":    stats["LinkedIn"],
		"Facebook":    stats["Facebook"],
		"Instagram":   stats["Instagram"],
		"X (Twitter)": stats["X (Twitter)"],
		"Bluesky":     stats["Bluesky"],
		"TikTok":      stats["TikTok"],
	}

	// Update each platform's count in the table
	for platform, count := range platformMap {
		// Match the platform row and replace the Gemeinderäte column (3rd column)
		// Pattern: | Platform | Status | OLD_NUMBER | Account |
		pattern := regexp.MustCompile(fmt.Sprintf(`(\| %s\s+\| [^|]+ \|) \d+(\s+\|)`, regexp.QuoteMeta(platform)))
		readme = pattern.ReplaceAllString(readme, fmt.Sprintf("${1} %d${2}", count))
	}

	// Update the total contacts count in the footer
	footerPattern := regexp.MustCompile(`Out of \d+ total contacts`)
	readme = footerPattern.ReplaceAllString(readme, fmt.Sprintf("Out of %d total contacts", totalContacts))

	// Check if there were any changes
	if readme == originalReadme {
		fmt.Fprintf(os.Stderr, "\n✓ README.md is already up to date (no changes needed)\n\n")
		return
	}

	// Write back to README.md
	err = os.WriteFile("README.md", []byte(readme), 0644)
	if err != nil {
		log.Printf("Warning: Could not write README.md: %v", err)
		return
	}

	// Sort platforms by count for the output message
	type platformCount struct {
		name  string
		count int
	}
	var sorted []platformCount
	for name, count := range platformMap {
		sorted = append(sorted, platformCount{name, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	fmt.Fprintf(os.Stderr, "\n✓ Updated README.md with platform statistics:\n")
	for _, p := range sorted {
		fmt.Fprintf(os.Stderr, "  %s: %d\n", p.name, p.count)
	}
	fmt.Fprintf(os.Stderr, "  Total contacts: %d\n\n", totalContacts)
}

