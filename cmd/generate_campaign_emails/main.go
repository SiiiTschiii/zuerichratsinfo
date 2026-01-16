package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

var (
	platform     = flag.String("platform", "x", "Social media platform (x, instagram, facebook, linkedin, bluesky, tiktok)")
	contactsPath = flag.String("contacts", "data/contacts.yaml", "Path to contacts YAML file")
	outputFile   = flag.String("output", "", "Output file (default: stdout)")
)

// Stadtrat names for role detection
var stadtraete = map[string]bool{
	"Corine Mauch": true, "Daniel Leupi": true, "Karin Rykart": true,
	"Andreas Hauri": true, "Simone Brander": true, "André Odermatt": true,
	"Raphael Golta": true, "Michael Baumer": true, "Filippo Leutenegger": true,
}

type ContactInfo struct {
	Name        string
	PlatformURL string
	Email       string
	Gender      string
	IsStadtrat  bool
}

func main() {
	flag.Parse()

	// Normalize platform name
	*platform = strings.ToLower(strings.TrimSpace(*platform))

	// Validate platform
	validPlatforms := map[string]bool{
		"x": true, "twitter": true, "instagram": true, "facebook": true,
		"linkedin": true, "bluesky": true, "tiktok": true,
	}
	if !validPlatforms[*platform] {
		log.Fatalf("Invalid platform: %s. Valid platforms: x, instagram, facebook, linkedin, bluesky, tiktok", *platform)
	}

	// Normalize twitter to x
	if *platform == "twitter" {
		*platform = "x"
	}

	// Load contacts
	mapper, err := contacts.LoadContacts(*contactsPath)
	if err != nil {
		log.Fatalf("Failed to load contacts: %v", err)
	}

	// Get contacts for the specified platform
	platformContacts := getContactsForPlatform(mapper, *platform)

	fmt.Fprintf(os.Stderr, "Found %d contacts with %s accounts\n", len(platformContacts), *platform)

	// Fetch API data for email lookups
	fmt.Fprintf(os.Stderr, "Fetching contact data from API...\n")
	client := zurichapi.NewClient()
	apiKontakte, err := client.FetchAllKontakte()
	if err != nil {
		log.Fatalf("Failed to fetch API data: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Loaded %d contacts from API\n", len(apiKontakte))

	// Find email addresses and additional info
	contactsWithEmails := []ContactInfo{}
	for _, contact := range platformContacts {
		email, gender := findEmailForContact(contact.Name, apiKontakte)
		if email != "" {
			contactsWithEmails = append(contactsWithEmails, ContactInfo{
				Name:        contact.Name,
				PlatformURL: getPlatformURL(contact, *platform),
				Email:       email,
				Gender:      gender,
				IsStadtrat:  stadtraete[contact.Name],
			})
		}
	}

	fmt.Fprintf(os.Stderr, "Found %d contacts with emails\n", len(contactsWithEmails))

	// Generate emails
	output := os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", err)
			}
		}()
		output = f
	}

	if err := generateEmails(output, contactsWithEmails, *platform); err != nil {
		log.Fatalf("Failed to generate emails: %v", err)
	}
}

func getContactsForPlatform(mapper *contacts.Mapper, platform string) []contacts.Contact {
	allContacts := mapper.GetAllContacts()
	var result []contacts.Contact

	for _, contact := range allContacts {
		hasAccount := false
		switch platform {
		case "x":
			hasAccount = len(contact.X) > 0
		case "instagram":
			hasAccount = len(contact.Instagram) > 0
		case "facebook":
			hasAccount = len(contact.Facebook) > 0
		case "linkedin":
			hasAccount = len(contact.LinkedIn) > 0
		case "bluesky":
			hasAccount = len(contact.Bluesky) > 0
		case "tiktok":
			hasAccount = len(contact.TikTok) > 0
		}

		if hasAccount {
			result = append(result, contact)
		}
	}

	return result
}

func getPlatformURL(contact contacts.Contact, platform string) string {
	switch platform {
	case "x":
		if len(contact.X) > 0 {
			return contact.X[0]
		}
	case "instagram":
		if len(contact.Instagram) > 0 {
			return contact.Instagram[0]
		}
	case "facebook":
		if len(contact.Facebook) > 0 {
			return contact.Facebook[0]
		}
	case "linkedin":
		if len(contact.LinkedIn) > 0 {
			return contact.LinkedIn[0]
		}
	case "bluesky":
		if len(contact.Bluesky) > 0 {
			return contact.Bluesky[0]
		}
	case "tiktok":
		if len(contact.TikTok) > 0 {
			return contact.TikTok[0]
		}
	}
	return ""
}

func findEmailForContact(name string, apiKontakte []zurichapi.Kontakt) (string, string) {
	// Split the name into parts for flexible matching
	nameParts := strings.Fields(name)
	if len(nameParts) < 2 {
		return "", ""
	}

	for _, kontakt := range apiKontakte {
		// API returns "Lastname Firstname" or "Firstname Lastname" format
		// Check if all name parts appear in the API contact (NameVorname field)
		apiNameFull := strings.ReplaceAll(kontakt.NameVorname, "\u00a0", " ")
		apiNameFull = strings.TrimSpace(apiNameFull)
		
		matchCount := 0
		for _, part := range nameParts {
			if strings.Contains(apiNameFull, part) {
				matchCount++
			}
		}

		// If all parts match, we found the contact
		if matchCount == len(nameParts) {
			email := ""
			if kontakt.EmailPrivat != "" {
				email = kontakt.EmailPrivat
			} else if kontakt.EmailGeschaeft != "" {
				email = kontakt.EmailGeschaeft
			}
			return email, kontakt.Geschlecht
		}
	}

	return "", ""
}

func generateEmails(output *os.File, contactsWithEmails []ContactInfo, platform string) error {
	platformNames := map[string]string{
		"x":         "X (Twitter)",
		"instagram": "Instagram",
		"facebook":  "Facebook",
		"linkedin":  "LinkedIn",
		"bluesky":   "Bluesky",
		"tiktok":    "TikTok",
	}

	platformName := platformNames[platform]

	w := func(format string, args ...interface{}) {
		if _, err := fmt.Fprintf(output, format, args...); err != nil {
			log.Fatalf("Failed to write output: %v", err)
		}
	}

	w("Transparenz im Gemeinderat: zuerichratsinfo auf Social Media\n")
	w("\n")
	w("Platform: %s\n", platformName)
	w("\n")
	w("---\n")
	w("\n")

	for i, contact := range contactsWithEmails {
		var roleGreeting, anrede string

		if contact.Gender == "weiblich" {
			anrede = "Liebe"
			if contact.IsStadtrat {
				roleGreeting = "Liebe Stadträtin"
			} else {
				roleGreeting = "Liebe Gemeinderätin"
			}
		} else {
			anrede = "Lieber"
			if contact.IsStadtrat {
				roleGreeting = "Lieber Stadtrat"
			} else {
				roleGreeting = "Lieber Gemeinderat"
			}
		}

		w("## %d. %s\n", i+1, contact.Name)
		w("\n")
		w("**An:** %s\n", contact.Email)
		w("\n")
		w("%s\n", roleGreeting)
		w("%s %s\n", anrede, contact.Name)
		w("\n")
		w("In Vorbereitung auf den kommenden Gemeinderatswahlkampf hat mich interessiert, wie die Arbeit im Gemeinderat konkret abläuft und insbesondere, was tatsächlich entschieden wird.\n")
		w("\n")
		w("Daraus ist das Projekt zuerichratsinfo entstanden:\n")
		w("👉 https://x.com/zuerichratsinfo\n")
		w("\n")
		w("Der Account publiziert die Abstimmungsresultate aus dem Gemeinderat auf X (Twitter) und markiert jeweils die Politikerinnen und Politiker, welche die entsprechenden Vorstösse etc. eingereicht haben (wie dich: %s). Ziel ist es, politische Arbeit transparenter und für die Öffentlichkeit besser nachvollziehbar zu machen. Mein Ziel ist, @zuerichratsinfo auf weitere Social Media Plattformen zu erweitern wie zum Beispiel Facebook, TikTok, Bluesky oder Instagram – überall wo die Gemeinderätinnen und Gemeinderäte und ihre Wählerinnen und Wähler unterwegs sind.\n", contact.PlatformURL)
		w("\n")
		w("Ich würde mich freuen, wenn du dem Account folgst und mir Feedback gibst, ob du darin einen Mehrwert für dich und deine Wählerinnen und Wähler siehst, insbesondere im Hinblick auf den Wahlkampf.\n")
		w("\n")
		w("Weitere Informationen zum Projekt und eine Übersicht, wo alle deine Kolleginnen und Kollegen im Gemeinderat auf Social Media zu finden sind:\n")
		w("https://github.com/SiiiTschiii/zuerichratsinfo\n")
		w("\n")
		w("Ich wünsche dir einen erfolgreichen Wahlkampf!\n")
		w("\n")
		w("Vielen Dank und liebe Grüsse\n")
		w("Christof\n")
		w("https://www.linkedin.com/in/christof-gerber/\n")
		w("\n")
		w("---\n")
		w("\n")
	}

	w("\nTotal: %d personalized emails generated\n", len(contactsWithEmails))
	return nil
}
