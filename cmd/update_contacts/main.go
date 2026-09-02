// Command update_contacts refreshes a jurisdiction's contacts.yaml from the
// body's own roster.
//
//	go run ./cmd/update_contacts                              # zurich-city
//	go run ./cmd/update_contacts -jurisdiction zurich-canton
//	go run ./cmd/update_contacts -jurisdiction zurich-canton -dry-run
//
// It is append-only by design. The file is a hand-curated mapping in which
// every handle was verified by a human, and a roster that briefly drops someone
// — a Nachrücken mid-processing, a source outage — must not be able to delete
// that work. Members who have left are removed by hand, deliberately.
//
// Party and Fraktion are printed for the run's benefit and never written: the
// parliament publishes them, so a second copy here could only go stale. See
// votes.Member.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/config"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
	"gopkg.in/yaml.v3"
)

// Contact mirrors the curated schema. Platform fields are declared in
// alphabetical order because that order is what gets marshalled, and
// cmd/validate_contacts requires it in the file.
type Contact struct {
	Name      string   `yaml:"name"`
	Bluesky   []string `yaml:"bluesky,omitempty"`
	Facebook  []string `yaml:"facebook,omitempty"`
	Instagram []string `yaml:"instagram,omitempty"`
	LinkedIn  []string `yaml:"linkedin,omitempty"`
	TikTok    []string `yaml:"tiktok,omitempty"`
	X         []string `yaml:"x,omitempty"`
}

type ContactMapping struct {
	Version  string    `yaml:"version"`
	Contacts []Contact `yaml:"contacts"`
}

func main() {
	jurisdiction := flag.String("jurisdiction", "zurich-city",
		"jurisdiction to refresh ("+strings.Join(config.JurisdictionKeys(), ", ")+")")
	dryRun := flag.Bool("dry-run", false, "report what would change without writing the file")
	flag.Parse()

	j, err := config.LookupJurisdiction(*jurisdiction)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	if j.NewMemberSource == nil {
		log.Fatalf("❌ %s publishes no member roster", j.Key)
	}

	path := contacts.PathFor(j.Key)

	existing, header, err := loadExisting(path)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	fmt.Printf("📋 %s: %d existing contacts\n", path, len(existing))

	fmt.Printf("📥 Fetching the %s roster...\n", j.Name)
	members, err := j.NewMemberSource().FetchMembers()
	if err != nil {
		log.Fatalf("❌ failed to fetch roster: %v", err)
	}
	fmt.Printf("🌐 %d members\n", len(members))

	merged, added, accounts := merge(existing, members)

	fmt.Printf("\n✅ %d contacts total, %d new, %d accounts added\n", len(merged), added, accounts)
	if *dryRun {
		fmt.Println("🔍 Dry run — nothing written.")
		return
	}
	if added == 0 && accounts == 0 {
		fmt.Println("💤 Nothing to write.")
		return
	}
	if err := save(path, header, merged); err != nil {
		log.Fatalf("❌ %v", err)
	}
	fmt.Printf("💾 Saved to %s\n", path)
}

// loadExisting reads the curated file, returning the contacts by name and the
// comment block above `version`.
//
// The header is carried through verbatim because it is the one part of the file
// no tool can regenerate: it says why this jurisdiction's mapping looks the way
// it does, and rewriting the file around it must not cost that.
func loadExisting(path string) (map[string]*Contact, string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		fmt.Printf("⚠️  %s does not exist yet, creating it\n", path)
		return map[string]*Contact{}, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", path, err)
	}

	var mapping ContactMapping
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return nil, "", fmt.Errorf("parsing %s: %w", path, err)
	}

	byName := make(map[string]*Contact, len(mapping.Contacts))
	for i := range mapping.Contacts {
		c := &mapping.Contacts[i]
		byName[contacts.NameKey(c.Name)] = c
	}
	return byName, leadingComments(string(data)), nil
}

// leadingComments returns the comment and blank lines that open a file.
func leadingComments(content string) string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			break
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

// merge folds the roster into the curated contacts, adding names that are new
// and accounts that are not already recorded. It reports how many of each.
//
// existing is keyed by contacts.NameKey rather than by name.
func merge(existing map[string]*Contact, members []votes.Member) ([]Contact, int, int) {
	added, accounts := 0, 0

	for _, m := range members {
		// Keyed on the person, not on the spelling: the curated file and the
		// source disagree on the order of the name parts often enough that
		// matching the string added a second entry for someone already in it.
		key := contacts.NameKey(m.Name)

		c, ok := existing[key]
		if !ok {
			c = &Contact{Name: m.Name}
			existing[key] = c
			added++
			fmt.Printf("➕ %s%s\n", m.Name, affiliation(m))
		}
		accounts += addAccounts(c, m.Accounts)
	}

	out := make([]Contact, 0, len(existing))
	for _, c := range existing {
		out = append(out, *c)
	}
	// The same comparator cmd/validate_contacts enforces on the file.
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, added, accounts
}

// addAccounts records the accounts the body itself publishes, keeping anything
// already in the file. A handle that differs from the one on record is added
// beside it rather than replacing it: both may be real, and deciding that is a
// human's call.
func addAccounts(c *Contact, published []votes.Account) int {
	added := 0

	for _, a := range published {
		field := platformField(c, a.Platform)
		if field == nil {
			continue
		}
		url := stripTracking(a.URL)
		if recorded(*field, url) {
			continue
		}
		*field = append(*field, url)
		added++
		fmt.Printf("   🔗 %s: %s %s\n", c.Name, a.Platform, url)
	}

	return added
}

// platformField addresses the slice a platform's URLs live in, or nil for a
// platform the schema has no column for.
func platformField(c *Contact, platform string) *[]string {
	switch platform {
	case "bluesky":
		return &c.Bluesky
	case "facebook":
		return &c.Facebook
	case "instagram":
		return &c.Instagram
	case "linkedin":
		return &c.LinkedIn
	case "tiktok":
		return &c.TikTok
	case "x":
		return &c.X
	default:
		return nil
	}
}

// recorded reports whether the file already has this account, comparing what
// the URLs point at rather than how they are spelled.
func recorded(existing []string, candidate string) bool {
	key := accountKey(candidate)
	for _, have := range existing {
		if accountKey(have) == key {
			return true
		}
	}
	return false
}

// accountKey identifies the account a URL points at, for comparison only —
// nothing is ever stored in this form.
//
// The curated file and PARIS write the same account differently in every way a
// URL can vary without changing: "facebook.com/attila.kipfer" against
// "www.facebook.com/attila.kipfer", "linkedin.com/in/alana-gerdes" against the
// same with a trailing slash. Each of those spellings, compared literally, adds
// a second copy of an account already on file — and then a third on the next
// refresh, because the copy never matches either.
func accountKey(raw string) string {
	u, err := url.Parse(strings.TrimSpace(stripTracking(raw)))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(raw))
	}

	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	// Country subdomains address the same profile: ch.linkedin.com and
	// linkedin.com differ only in which office serves the page.
	if i := strings.Index(host, ".linkedin.com"); i > 0 {
		host = "linkedin.com"
	}

	path := strings.TrimSuffix(strings.ToLower(u.EscapedPath()), "/")

	key := host + path
	if q := u.Query().Encode(); q != "" {
		key += "?" + strings.ToLower(q)
	}
	return key
}

// trackingParams are query parameters that say how a link was shared or in
// which language it was opened, rather than which account it points at.
//
// They have to go, because accounts are deduplicated by URL and PARIS serves
// the same account under several spellings: "instagram.com/perparim.avdili/"
// against the "…/?hl=de" already on file, "…?igsh=<token>" for a link someone
// copied out of the app. Kept, each refresh would record another copy of an
// account already there.
//
// It is a list of things to drop rather than a rule to drop every query,
// because some of these URLs are nothing without theirs: a Facebook profile is
// identified by "profile.php?id=100070693802425".
var trackingParams = map[string]bool{
	"hl":                true, // interface language
	"locale":            true, // interface language
	"originalSubdomain": true, // LinkedIn's country hint
	"igsh":              true, // Instagram share token
	"si":                true, // share token
	"fbclid":            true, // click id
}

// stripTracking removes those parameters and leaves everything else alone. A
// URL that will not parse comes back as it went in: this is tidying, and not
// worth failing a refresh over.
func stripTracking(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}

	q := u.Query()
	for key := range q {
		if trackingParams[key] || strings.HasPrefix(key, "utm_") {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()

	return strings.TrimSuffix(u.String(), "?")
}

// affiliation renders a member's party and Fraktion for the run's output, so a
// name arriving in the file can be recognised without a second lookup.
func affiliation(m votes.Member) string {
	switch {
	case m.Party == "" && m.Fraktion == "":
		return ""
	case m.Fraktion == "" || m.Fraktion == m.Party:
		return " (" + m.Party + ")"
	case m.Party == "":
		return " (Fraktion " + m.Fraktion + ")"
	default:
		return " (" + m.Party + ", Fraktion " + m.Fraktion + ")"
	}
}

// save writes the mapping back, preserving the file's own header.
//
// The two-space indent is not cosmetic: it is the shape the curated files are
// already in, and the default four would rewrite every line of a 900-line file
// the first time this runs, burying the actual change.
func save(path, header string, cs []Contact) error {
	var sb strings.Builder
	sb.WriteString(header)

	enc := yaml.NewEncoder(&sb)
	enc.SetIndent(2)
	if err := enc.Encode(ContactMapping{Version: "1.0", Contacts: cs}); err != nil {
		return fmt.Errorf("marshalling contacts: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("marshalling contacts: %w", err)
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
