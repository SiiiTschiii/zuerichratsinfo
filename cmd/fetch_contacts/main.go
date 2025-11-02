package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/siiitschiii/zurichratsinfo/pkg/zurichapi"
)

func main() {
	client := zurichapi.NewClient()

	contacts, err := client.FetchAllKontakte()
	if err != nil {
		log.Fatalf("failed to fetch contacts: %v", err)
	}

	for _, contact := range contacts {
		name := strings.ReplaceAll(contact.NameVorname, "\u00a0", " ")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		fmt.Println(name)

		// Print social media if available
		for _, sm := range contact.SozialeMedien.Kommunikation {
			if strings.TrimSpace(sm.Adresse) != "" {
				fmt.Printf("  - %s: %s\n", sm.Typ, sm.Adresse)
			}
		}
	}
}