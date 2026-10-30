package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/siiitschiii/parlinfo/pkg/zurichapi"
)

func main() {
	client := zurichapi.NewClient()

	fmt.Println("📥 Fetching active Gemeinderat mandates...")

	mandates, err := client.FetchActiveGemeinderatMandates()
	if err != nil {
		log.Fatalf("Failed to fetch mandates: %v", err)
	}

	fmt.Printf("✅ Found %d active mandates\n\n", len(mandates))

	// Sort by name for consistent output
	sort.Slice(mandates, func(i, j int) bool {
		if mandates[i].Vorname != mandates[j].Vorname {
			return mandates[i].Vorname < mandates[j].Vorname
		}
		return mandates[i].Name < mandates[j].Name
	})

	// Print all mandates
	for _, m := range mandates {
		fmt.Printf("%s %s - %s (%s)\n", m.Vorname, m.Name, m.Funktion, m.Partei)
	}
}
