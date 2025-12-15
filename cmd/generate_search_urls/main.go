package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ContactsFile struct {
	Version  string    `yaml:"version"`
	Contacts []Contact `yaml:"contacts"`
}

type Contact struct {
	Name      string   `yaml:"name"`
	X         []string `yaml:"x,omitempty"`
	Facebook  []string `yaml:"facebook,omitempty"`
	Instagram []string `yaml:"instagram,omitempty"`
	LinkedIn  []string `yaml:"linkedin,omitempty"`
	TikTok    []string `yaml:"tiktok,omitempty"`
	Bluesky   []string `yaml:"bluesky,omitempty"`
}

func main() {
	contactsPath := "data/contacts.yaml"
	data, err := os.ReadFile(contactsPath)
	if err != nil {
		log.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	var contactsFile ContactsFile
	if err := yaml.Unmarshal(data, &contactsFile); err != nil {
		log.Fatalf("Failed to parse contacts.yaml: %v", err)
	}

	fmt.Println("# Social Media Search URLs")
	fmt.Println("# Copy these URLs to find social media accounts")
	fmt.Println()
	fmt.Println("**Note:** Instagram doesn't support direct search links from external browsers. Search for names directly within the Instagram app or website.")
	fmt.Println()

	missingCount := 0
	for _, contact := range contactsFile.Contacts {
		// Skip if already has all platforms
		hasX := len(contact.X) > 0
		hasFacebook := len(contact.Facebook) > 0
		hasInstagram := len(contact.Instagram) > 0
		hasLinkedIn := len(contact.LinkedIn) > 0
		hasTikTok := len(contact.TikTok) > 0
		hasBluesky := len(contact.Bluesky) > 0

		if hasX && hasFacebook && hasInstagram && hasLinkedIn && hasTikTok && hasBluesky {
			continue
		}

		missingCount++
		fmt.Printf("## %s\n", contact.Name)

		// Parse name for search
		parts := strings.Fields(contact.Name)
		searchName := strings.Join(parts, " ")

		if !hasX {
			xSearch := url.QueryEscape(searchName)
			fmt.Printf("- X/Twitter: https://x.com/search?q=%s&src=typed_query&f=user\n", xSearch)
		}

		if !hasInstagram {
			fmt.Println("- Instagram: https://www.instagram.com/")
		}

		if !hasFacebook {
			fbSearch := url.QueryEscape(searchName + " Zürich")
			fmt.Printf("- Facebook: https://www.facebook.com/search/top?q=%s\n", fbSearch)
		}

		if !hasLinkedIn {
			linkedInSearch := url.QueryEscape(searchName)
			fmt.Printf("- LinkedIn: https://www.linkedin.com/search/results/all/?keywords=%s\n", linkedInSearch)
		}

		if !hasTikTok {
			tiktokSearch := url.QueryEscape(searchName)
			fmt.Printf("- TikTok: https://www.tiktok.com/search?q=%s\n", tiktokSearch)
		}

		if !hasBluesky {
			bskySearch := url.QueryEscape(searchName)
			fmt.Printf("- Bluesky: https://bsky.app/search?q=%s\n", bskySearch)
		}

		// Google search as fallback
		googleSearch := url.QueryEscape(searchName + " Gemeinderat Zürich social media")
		fmt.Printf("- Google: https://www.google.com/search?q=%s\n", googleSearch)

		fmt.Println()
	}

	fmt.Printf("\n---\n")
	fmt.Printf("Found %d people missing social media accounts\n", missingCount)
	fmt.Println("\n💡 Tips:")
	fmt.Println("- Search for the person's name + 'Gemeinderat' or their party name")
	fmt.Println("- Check their official Gemeinderat profile page for social media links")
	fmt.Println("- Gemeinderat profiles: https://www.gemeinderat-zuerich.ch/ratsmitglieder")
}
