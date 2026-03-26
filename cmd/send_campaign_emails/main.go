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
	preview       = flag.Bool("preview", false, "Preview: render all emails")
	outputFile    = flag.String("output", "", "Output file for preview (default: stdout)")
	testAddr      = flag.String("test", "", "Test mode: send all emails to this address")
	send          = flag.Bool("send", false, "Send emails to actual recipients")
	delay         = flag.Int("delay", 2, "Seconds between sends")
)

const emailSubject = "zuerichratsinfo jetzt auch auf Bluesky"

type Recipient struct {
	Name       string
	Email      string
	Gender     string
	Salutation string
	BlueskyURL string
	Source     string
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

	recipients := buildRecipientList()

	switch {
	case *preview:
		runPreview(recipients)
	case *testAddr != "":
		runSend(recipients, *testAddr)
	case *send:
		runSend(recipients, "")
	default:
		runVerify(recipients)
	}
}

func buildRecipientList() []Recipient {
	mapper, err := contacts.LoadContacts(*contactsPath)
	if err != nil {
		log.Fatalf("Failed to load contacts: %v", err)
	}

	allContacts := mapper.GetAllContacts()
	var bskyContacts []contacts.Contact
	for _, c := range allContacts {
		if len(c.Bluesky) > 0 {
			bskyContacts = append(bskyContacts, c)
		}
	}
	fmt.Fprintf(os.Stderr, "Found %d contacts with Bluesky accounts\n", len(bskyContacts))

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

	for _, contact := range bskyContacts {
		bskyURL := contact.Bluesky[0]
		nameLower := strings.ToLower(contact.Name)

		if o, ok := overrideMap[nameLower]; ok {
			recipients = append(recipients, Recipient{
				Name:       contact.Name,
				Email:      o.Email,
				Gender:     o.Gender,
				Salutation: salutation(o.Gender),
				BlueskyURL: bskyURL,
				Source:     "override",
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
			Name:       contact.Name,
			Email:      email,
			Gender:     gender,
			Salutation: salutation(gender),
			BlueskyURL: bskyURL,
			Source:     "api",
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

func runVerify(recipients []Recipient) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "#\tName\tEmail\tGender\tSalutation\tBluesky URL\tSource\n")
	_, _ = fmt.Fprintf(w, "-\t----\t-----\t------\t----------\t-----------\t------\n")
	for i, r := range recipients {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i+1, r.Name, r.Email, r.Gender, r.Salutation, r.BlueskyURL, r.Source)
	}
	if err := w.Flush(); err != nil {
		log.Fatalf("Failed to flush table: %v", err)
	}

	fmt.Printf("\nTotal: %d recipients\n", len(recipients))
	fmt.Printf("\n--- Email Template ---\n\n")
	fmt.Printf("Subject: %s\n\n", emailSubject)
	fmt.Printf("{Salutation} {Name}\n\n")
	fmt.Print(emailTemplatePreview())
}

func emailTemplatePreview() string {
	return `zuerichratsinfo ist jetzt auch auf Bluesky verfügbar:
👉 https://bsky.app/profile/zuerichratsinfo.bsky.social

Der Account publiziert die Abstimmungsresultate aus dem Gemeinderat auf X (https://x.com/zuerichratsinfo) und neu auch auf Bluesky, und markiert jeweils die Politikerinnen und Politiker, welche die entsprechenden Vorstösse etc. eingereicht haben (wie dich: {BlueskyURL}). Ziel ist es, politische Arbeit transparenter und für die Öffentlichkeit besser nachvollziehbar zu machen.

Wir arbeiten laufend daran, die Posts weiterzuentwickeln – zum Beispiel mit Statistiken, wie die einzelnen Fraktionen abgestimmt haben. Falls du Ideen oder Feedback hast, würde ich mich sehr darüber freuen!

Ich würde mich freuen, wenn du dem Account folgst.

Weitere Informationen zum Projekt und eine Übersicht, wo alle deine Kolleginnen und Kollegen im Gemeinderat auf Social Media zu finden sind:
https://github.com/SiiiTschiii/zuerichratsinfo

Vielen Dank und liebe Grüsse
Christof
https://www.linkedin.com/in/christof-gerber/
`
}

// --- Preview mode ---

func runPreview(recipients []Recipient) {
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

	_, _ = fmt.Fprintf(output, "# %s\n\n---\n\n", emailSubject)

	for i, r := range recipients {
		_, _ = fmt.Fprintf(output, "## %d. %s\n\n", i+1, r.Name)
		_, _ = fmt.Fprintf(output, "**An:** %s\n\n", r.Email)
		_, _ = fmt.Fprintf(output, "%s %s\n\n", r.Salutation, r.Name)
		_, _ = fmt.Fprintf(output, "%s", generateEmailBody(r))
		_, _ = fmt.Fprintf(output, "\n---\n\n")
	}

	_, _ = fmt.Fprintf(output, "Total: %d emails\n", len(recipients))

	if *outputFile != "" {
		fmt.Fprintf(os.Stderr, "Wrote %d emails to %s\n", len(recipients), *outputFile)
	}
}

func generateEmailBody(r Recipient) string {
	return fmt.Sprintf(`zuerichratsinfo ist jetzt auch auf Bluesky verfügbar:
👉 https://bsky.app/profile/zuerichratsinfo.bsky.social

Der Account publiziert die Abstimmungsresultate aus dem Gemeinderat auf X (https://x.com/zuerichratsinfo) und neu auch auf Bluesky, und markiert jeweils die Politikerinnen und Politiker, welche die entsprechenden Vorstösse etc. eingereicht haben (wie dich: %s). Ziel ist es, politische Arbeit transparenter und für die Öffentlichkeit besser nachvollziehbar zu machen.

Wir arbeiten laufend daran, die Posts weiterzuentwickeln – zum Beispiel mit Statistiken, wie die einzelnen Fraktionen abgestimmt haben. Falls du Ideen oder Feedback hast, würde ich mich sehr darüber freuen!

Ich würde mich freuen, wenn du dem Account folgst. Und vielleicht hast du ja mal Lust einen Abstimmungspost mit deinen Followern zu teilen.

Weitere Informationen zum Projekt und eine Übersicht, wo alle deine Kolleginnen und Kollegen im Gemeinderat auf Social Media zu finden sind:
https://github.com/SiiiTschiii/zuerichratsinfo

Vielen Dank und liebe Grüsse
Christof
https://www.linkedin.com/in/christof-gerber/
`, r.BlueskyURL)
}

func generateFullEmail(r Recipient) string {
	return fmt.Sprintf("%s %s\n\n%s", r.Salutation, r.Name, generateEmailBody(r))
}

// --- Send mode ---

func runSend(recipients []Recipient, testOverride string) {
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

		body := generateFullEmail(r)
		msg := buildMIMEMessage(gmailAddr, toAddr, emailSubject, body)

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
