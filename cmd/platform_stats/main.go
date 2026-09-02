package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	contactsPkg "github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
)

type PlatformStat struct {
	Name  string
	Count int
}

func main() {
	// Every jurisdiction's file, because the table counts politicians the bot
	// can tag and it can tag them wherever they sit.
	files, err := filepath.Glob(filepath.Join("data", "*", "contacts.yaml"))
	if err != nil || len(files) == 0 {
		log.Fatalf("Error finding contacts files: %v", err)
	}
	sort.Strings(files)

	mapper, err := contactsPkg.LoadContactFiles(files...)
	if err != nil {
		log.Fatalf("Error loading contacts: %v", err)
	}
	all := mapper.GetAllContacts()

	// Count platforms
	stats := map[string]int{
		"X (Twitter)": 0,
		"Instagram":   0,
		"Facebook":    0,
		"LinkedIn":    0,
		"TikTok":      0,
		"Bluesky":     0,
	}

	// Verified only: an unverified candidate is a lead, not an account we can
	// tag, and the table's own column calls them verified.
	label := map[string]string{
		"x": "X (Twitter)", "instagram": "Instagram", "facebook": "Facebook",
		"linkedin": "LinkedIn", "tiktok": "TikTok", "bluesky": "Bluesky",
	}
	curated := 0
	for _, contact := range all {
		counted := false
		for _, platform := range contactsPkg.Platforms {
			if len(contact.Verified(platform)) > 0 {
				stats[label[platform]]++
				counted = true
			}
		}
		if counted {
			curated++
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
	updateREADME(stats, curated)
}

func updateREADME(stats map[string]int, totalContacts int) {
	// Read README.md
	readmeData, err := os.ReadFile("README.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not read README.md: %v\n", err)
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
	// Try to match and replace, but don't fail if it doesn't match
	platformsUpdated := 0
	for platform, count := range platformMap {
		// Pattern: | Platform     | Status (with emoji) | NUMBER | Account |
		pattern := regexp.MustCompile(fmt.Sprintf(`(\|\s*%s\s*\|[^|]+\|)\s*\d+\s*(\|)`, regexp.QuoteMeta(platform)))
		newReadme := pattern.ReplaceAllString(readme, fmt.Sprintf("${1} %d ${2}", count))
		if newReadme != readme {
			platformsUpdated++
			readme = newReadme
		}
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
		fmt.Fprintf(os.Stderr, "Warning: Could not write README.md: %v\n", err)
		return
	}

	// Report what was updated
	fmt.Fprintf(os.Stderr, "\n✓ Updated README.md with platform statistics (%d platforms updated)\n", platformsUpdated)
	for platform, count := range platformMap {
		fmt.Fprintf(os.Stderr, "  %s: %d\n", platform, count)
	}
	fmt.Fprintf(os.Stderr, "  Total contacts: %d\n\n", totalContacts)
}
