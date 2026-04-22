package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
	"gopkg.in/yaml.v3"
)

var (
	contactsPath  = flag.String("contacts", "data/contacts.yaml", "Path to contacts YAML file")
	overridesPath = flag.String("overrides", "data/email_overrides.yaml", "Path to email overrides YAML file")
	platform      = flag.String("platform", "", "Target platform (required): bluesky | instagram")
	preview       = flag.Bool("preview", false, "Preview: render all emails")
	outputFile    = flag.String("output", "", "Output file for preview (default: stdout)")
	testAddr      = flag.String("test", "", "Test mode: send all emails to this address")
	send          = flag.Bool("send", false, "Send emails to actual recipients")
	delay         = flag.Int("delay", 2, "Seconds between sends")
)

type platformConfig struct {
	Key         string
	DisplayName string
	Subject     string
	Body        func(platformURL string) string
}

var platformConfigs = map[string]platformConfig{
	"bluesky":   blueskyConfig,
	"instagram": instagramConfig,
}

var blueskyConfig = platformConfig{
	Key:         "bluesky",
	DisplayName: "Bluesky",
	Subject:     "zuerichratsinfo jetzt auch auf Bluesky",
	Body: func(url string) string {
		return fmt.Sprintf(`zuerichratsinfo ist jetzt auch auf Bluesky verfügbar:
👉 https://bsky.app/profile/zuerichratsinfo.bsky.social

Der Account publiziert die Abstimmungsresultate aus dem Gemeinderat auf X (https://x.com/zuerichratsinfo) und neu auch auf Bluesky, und markiert jeweils die Politikernnen und Politiker, welche die entsprechenden Vorstösse etc. eingereicht haben (wie dich: %s). Ziel ist es, politische Arbeit transparenter und für die Öffentlichkeit besser nachvollziehbar zu machen.

Wir arbeiten laufend daran, die Posts weiterzuentwickeln – zum Beispiel mit Statistiken, wie die einzelnen Fraktionen abgestimmt haben. Falls du Ideen oder Feedback hast, würde ich mich sehr darüber freuen!

Ich würde mich freuen, wenn du dem Account folgst. Und vielleicht hast du ja mal Lust einen Abstimmungspost mit deinen Followern zu teilen.
 
Weitere Informationen zum Projekt und eine Übersicht, wo alle GemeinderätInnen und StadträtInnen auf Social Media zu finden sind:
https://github.com/SiiiTschiii/zuerichratsinfo

Vielen Dank und liebe Grüsse
Christof
https://www.linkedin.com/in/christof-gerber/
`, url)
	},
}

var instagramConfig = platformConfig{
	Key:         "instagram",
	DisplayName: "Instagram",
	Subject:     "zuerichratsinfo jetzt auch auf Instagram",
	Body: func(url string) string {
		return fmt.Sprintf(`zuerichratsinfo ist jetzt auch auf Instagram verfügbar:
👉 https://www.instagram.com/zueriratsinfo

Der Account publiziert die Abstimmungsresultate aus dem Gemeinderat auf X (https://x.com/zuerichratsinfo), Bluesky (https://bsky.app/profile/zuerichratsinfo.bsky.social) und neu auch auf Instagram, und markiert jeweils die Politikerinnen und Politiker, welche die entsprechenden Vorstösse etc. eingereicht haben (wie dich: %s). Ziel ist es, politische Arbeit transparenter und für die Öffentlichkeit besser nachvollziehbar zu machen.

Wir arbeiten laufend daran, die Posts weiterzuentwickeln – zum Beispiel mit Statistiken, wie die einzelnen Fraktionen abgestimmt haben. Falls du Ideen oder Feedback hast, würde ich mich sehr darüber freuen!

Ich würde mich freuen, wenn du dem Account folgst. Und vielleicht hast du ja mal Lust einen Abstimmungspost mit deinen Followern zu teilen.

Weitere Informationen zum Projekt und eine Übersicht, wo alle GemeinderätInnen und StadträtInnen auf Social Media zu finden sind:
https://github.com/SiiiTschiii/zuerichratsinfo

Vielen Dank und liebe Grüsse
Christof
https://www.linkedin.com/in/christof-gerber/
`, url)
	},
}

type Recipient struct {
	Name        string
	Email       string
	Gender      string
	Salutation  string
	PlatformURL string
	Source      string
}

type Override struct {
	Name   string `yaml:"name"`
	Email  string `yaml:"email"`
	Gender string `yaml:"gender"`
}

type OverridesFile struct {
	Overrides []Override `yaml:"overrides"`
}

func main() {
	flag.Parse()

	if *send && *testAddr != "" {
		log.Fatal("Cannot use --send and --test at the same time")
	}

	cfg, ok := platformConfigs[strings.ToLower(*platform)]
	if !ok {
		log.Fatalf("--platform is required; supported: bluesky, instagram")
	}

	recipients := buildRecipientList(cfg)

	switch {
	case *preview:
		runPreview(cfg, recipients)
	case *testAddr != "":
		runSend(cfg, recipients, *testAddr)
	case *send:
		runSend(cfg, recipients, "")
	default:
		runVerify(cfg, recipients)
	}
}

func buildRecipientList(cfg platformConfig) []Recipient {
	mapper, err := contacts.LoadContacts(*contactsPath)
	if err != nil {
		log.Fatalf("Failed to load contacts: %v", err)
	}

	allContacts := mapper.GetAllContacts()
	var matchingContacts []contacts.Contact
	for _, c := range allContacts {
		if len(mapper.GetPlatformURLs(c.Name, cfg.Key)) > 0 {
			matchingContacts = append(matchingContacts, c)
		}
	}
	fmt.Fprintf(os.Stderr, "Found %d contacts with %s accounts\n", len(matchingContacts), cfg.DisplayName)

	fmt.Fprintf(os.Stderr, "Fetching contact data from API...\n")
	client := zurichapi.NewClient()
	apiKontakte, err := client.FetchAllKontakte()
	if err != nil {
		log.Fatalf("Failed to fetch API data: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Loaded %d contacts from API\n", len(apiKontakte))

	overrides := loadOverrides(*overridesPath)
	overrideMap := make(map[string]Override)
	for _, o := range overrides {
		overrideMap[strings.ToLower(o.Name)] = o
	}

	var recipients []Recipient

	for _, contact := range matchingContacts {
		urls := mapper.GetPlatformURLs(contact.Name, cfg.Key)
		if len(urls) == 0 {
			continue
		}
		platformURL := urls[0]
		nameLower := strings.ToLower(contact.Name)

		if o, ok := overrideMap[nameLower]; ok {
			recipients = append(recipients, Recipient{
				Name:        contact.Name,
				Email:       o.Email,
				Gender:      o.Gender,
				Salutation:  salutation(o.Gender),
				PlatformURL: platformURL,
				Source:      "override",
			})
			continue
		}

		email, gender, foundInAPI := findEmailForContact(contact.Name, apiKontakte)
		if !foundInAPI {
			fmt.Fprintf(os.Stderr, "  ⚠️  Not found in API (no override): %s\n", contact.Name)
			continue
		}
		if email == "" {
			fmt.Fprintf(os.Stderr, "  ⚠️  Found in API but no email (no override): %s\n", contact.Name)
			continue
		}

		recipients = append(recipients, Recipient{
			Name:        contact.Name,
			Email:       email,
			Gender:      gender,
			Salutation:  salutation(gender),
			PlatformURL: platformURL,
			Source:      "api",
		})
	}

	fmt.Fprintf(os.Stderr, "Total: %d recipients (%d from API, %d from overrides)\n",
		len(recipients),
		countBySource(recipients, "api"),
		countBySource(recipients, "override"))

	return recipients
}

func countBySource(recipients []Recipient, source string) int {
	n := 0
	for _, r := range recipients {
		if r.Source == source {
			n++
		}
	}
	return n
}

func salutation(gender string) string {
	if gender == "weiblich" {
		return "Liebe"
	}
	return "Lieber"
}

func loadOverrides(path string) []Override {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "No overrides file found at %s, continuing without overrides\n", path)
			return nil
		}
		log.Fatalf("Failed to read overrides file: %v", err)
	}
	var f OverridesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		log.Fatalf("Failed to parse overrides file: %v", err)
	}
	return f.Overrides
}

func findEmailForContact(name string, apiKontakte []zurichapi.Kontakt) (string, string, bool) {
	nameParts := strings.Fields(name)
	if len(nameParts) < 2 {
		return "", "", false
	}

	for _, kontakt := range apiKontakte {
		apiName := normalize(kontakt.Name)
		apiVorname := normalize(kontakt.Vorname)
		apiCombined := apiName + " " + apiVorname

		matchCount := 0
		for _, part := range nameParts {
			if strings.Contains(apiCombined, strings.ToLower(part)) {
				matchCount++
			}
		}

		if matchCount == len(nameParts) {
			email := ""
			if kontakt.EmailPrivat != "" {
				email = kontakt.EmailPrivat
			} else if kontakt.EmailGeschaeft != "" {
				email = kontakt.EmailGeschaeft
			}
			return email, kontakt.Geschlecht, true
		}
	}

	return "", "", false
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.ToLower(strings.TrimSpace(s))
}

// --- Verify mode ---

func runVerify(cfg platformConfig, recipients []Recipient) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "#\tName\tEmail\tGender\tSalutation\t%s URL\tSource\n", cfg.DisplayName)
	_, _ = fmt.Fprintf(w, "-\t----\t-----\t------\t----------\t-----------\t------\n")
	for i, r := range recipients {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i+1, r.Name, r.Email, r.Gender, r.Salutation, r.PlatformURL, r.Source)
	}
	if err := w.Flush(); err != nil {
		log.Fatalf("Failed to flush table: %v", err)
	}

	fmt.Printf("\nTotal: %d recipients\n", len(recipients))
	fmt.Printf("\n--- Email Template ---\n\n")
	fmt.Printf("Subject: %s\n\n", cfg.Subject)
	fmt.Printf("{Salutation} {Name}\n\n")
	fmt.Print(cfg.Body(fmt.Sprintf("{%sURL}", cfg.DisplayName)))
}

// --- Preview mode ---

func runPreview(cfg platformConfig, recipients []Recipient) {
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

	_, _ = fmt.Fprintf(output, "# %s\n\n---\n\n", cfg.Subject)

	for i, r := range recipients {
		_, _ = fmt.Fprintf(output, "## %d. %s\n\n", i+1, r.Name)
		_, _ = fmt.Fprintf(output, "**An:** %s\n\n", r.Email)
		_, _ = fmt.Fprintf(output, "%s %s\n\n", r.Salutation, r.Name)
		_, _ = fmt.Fprintf(output, "%s", cfg.Body(r.PlatformURL))
		_, _ = fmt.Fprintf(output, "\n---\n\n")
	}

	_, _ = fmt.Fprintf(output, "Total: %d emails\n", len(recipients))

	if *outputFile != "" {
		fmt.Fprintf(os.Stderr, "Wrote %d emails to %s\n", len(recipients), *outputFile)
	}
}

// --- Send mode ---

func runSend(cfg platformConfig, recipients []Recipient, testOverride string) {
	gmailAddr := os.Getenv("GMAIL_ADDRESS")
	gmailPass := os.Getenv("GMAIL_APP_PASSWORD")
	if gmailAddr == "" || gmailPass == "" {
		log.Fatal("GMAIL_ADDRESS and GMAIL_APP_PASSWORD environment variables must be set")
	}

	isTest := testOverride != ""
	if isTest {
		fmt.Fprintf(os.Stderr, "🧪 TEST MODE: All emails will be sent to %s\n\n", testOverride)
	} else {
		fmt.Fprintf(os.Stderr, "📧 SENDING %d emails for real\n\n", len(recipients))
	}

	var sent, failed int

	for i, r := range recipients {
		toAddr := r.Email
		if isTest {
			toAddr = testOverride
		}

		body := fmt.Sprintf("%s %s\n\n%s", r.Salutation, r.Name, cfg.Body(r.PlatformURL))
		msg := buildMIMEMessage(gmailAddr, toAddr, cfg.Subject, body)

		err := sendEmail(gmailAddr, gmailPass, toAddr, msg)

		if err != nil {
			fmt.Fprintf(os.Stderr, "[%d/%d] ❌ %s <%s>: %v\n", i+1, len(recipients), r.Name, toAddr, err)
			failed++
		} else {
			fmt.Fprintf(os.Stderr, "[%d/%d] ✅ %s <%s>\n", i+1, len(recipients), r.Name, toAddr)
			sent++
		}

		if i < len(recipients)-1 {
			time.Sleep(time.Duration(*delay) * time.Second)
		}
	}

	fmt.Fprintf(os.Stderr, "\nDone: %d sent, %d failed\n", sent, failed)
}

func buildMIMEMessage(from, to, subject, body string) []byte {
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n",
		from, to, subject)
	return []byte(headers + body)
}

func sendEmail(gmailAddr, gmailPass, to string, msg []byte) error {
	host := "smtp.gmail.com"
	addr := host + ":587"

	auth := smtp.PlainAuth("", gmailAddr, gmailPass, host)

	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	tlsConfig := &tls.Config{ServerName: host}
	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if err = conn.Mail(gmailAddr); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err = conn.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return conn.Quit()
}
