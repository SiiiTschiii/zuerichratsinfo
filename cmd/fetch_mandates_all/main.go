package main

import (
	"fmt"
	"log"

	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	client := zurichapi.NewClient()

	fmt.Println("📥 Fetching ALL active mandates (no filters)...")
	allMandates, err := client.GetActiveMandates()
	if err != nil {
		log.Fatalf("Failed to fetch mandates: %v", err)
	}

	fmt.Printf("✅ Found %d active mandates\n\n", len(allMandates))

	for _, m := range allMandates {
		fmt.Printf("%s %s - %s (%s)\n", m.Vorname, m.Name, m.Funktion, m.Partei)
	}
}
