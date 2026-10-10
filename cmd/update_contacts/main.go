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

// The schema lives in pkg/contacts, so what this writes is what the bot reads.
type (
	Account        = contacts.Account
	Contact        = contacts.Contact
	ContactMapping = contacts.ContactMapping
)

func main() {
	jurisdiction := flag.String("jurisdiction", "zurich-city",
		"jurisdiction to refresh ("+strings.Join(config.JurisdictionKeys(), ", ")+")")
	dryRun := flag.Bool("dry-run", false, "report what would change without writing the file")
	candidate := flag.Bool("add-candidate", false, "record one unverified candidate account for a person already on file, instead of refreshing the roster")
	candName := flag.String("name", "", "with -add-candidate: the person, as named in the file")
	candPlatform := flag.String("platform", "", "with -add-candidate: "+strings.Join(contacts.Platforms, ", "))
	candURL := flag.String("url", "", "with -add-candidate: the profile URL")
	candConfidence := flag.String("confidence", "", "with -add-candidate: how well the profile matched the person: high, medium or low")
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

	if *candidate {
		if err := runAddCandidate(path, header, existing, *candName, *candPlatform, *candURL, *candConfidence, *dryRun); err != nil {
			log.Fatalf("❌ %v", err)
		}
		return
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
	reportDeparted(j, merged)
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

// validConfidence are the levels a candidate may carry, matching what
// cmd/validate_contacts accepts.
var validConfidence = map[string]bool{"high": true, "medium": true, "low": true}

// runAddCandidate records one candidate and writes the file, or says why not.
func runAddCandidate(path, header string, existing map[string]*Contact, name, platform, rawURL, confidence string, dryRun bool) error {
	added, err := addCandidate(existing, name, platform, rawURL, confidence)
	if err != nil {
		return err
	}
	if !added {
		fmt.Printf("💤 %s: %s already on file, nothing to add\n", name, rawURL)
		return nil
	}

	// A checklist line, so the pull request can ask the reviewer to open it.
	fmt.Printf("- [ ] %s — %s candidate (%s): %s\n", name, platform, confidence, stripTracking(rawURL))
	if dryRun {
		fmt.Println("🔍 Dry run — nothing written.")
		return nil
	}

	merged, _, _ := merge(existing, nil)
	return save(path, header, merged)
}

// addCandidate appends an unverified account to a person already on file.
//
// It is the only way a handle that nobody has confirmed reaches the file, and it
// cannot write a verified one: Verified is never set here, and an account
// already on file is left exactly as it is, so a candidate cannot demote or
// overwrite a confirmed handle either. Confirming is a human's edit.
func addCandidate(existing map[string]*Contact, name, platform, rawURL, confidence string) (bool, error) {
	c, ok := existing[contacts.NameKey(name)]
	if !ok {
		return false, fmt.Errorf("%q is not on file: candidates attach to people already in it", name)
	}
	if c.IsOrganization() {
		return false, fmt.Errorf("%q is a party or Fraktion account, not a person", name)
	}
	field := platformField(c, platform)
	if field == nil {
		return false, fmt.Errorf("unknown platform %q, want one of %s", platform, strings.Join(contacts.Platforms, ", "))
	}
	if !validConfidence[confidence] {
		return false, fmt.Errorf("confidence %q must be high, medium or low", confidence)
	}
	// The rule CI applies to the file, so a candidate cannot be saved only for
	// cmd/validate_contacts to reject the rewritten file.
	if err := contacts.ValidateURL(platform, rawURL); err != nil {
		return false, fmt.Errorf("url %q: %w", rawURL, err)
	}

	url := stripTracking(rawURL)
	key := accountKey(url)
	for _, have := range *field {
		if accountKey(have.URL) == key {
			return false, nil
		}
	}

	*field = append(*field, Account{URL: url, Confidence: confidence})
	return true, nil
}

// reportDeparted prints the people on file who no longer sit, for a human to
// decide about. It never changes the file: the mapping is append-only.
func reportDeparted(j config.Jurisdiction, cs []Contact) {
	lister, ok := j.NewMemberSource().(votes.SittingLister)
	if !ok {
		return
	}
	sitting, err := lister.SittingNames()
	if err != nil {
		log.Fatalf("❌ failed to fetch who sits: %v", err)
	}

	gone := departed(cs, sitting)
	fmt.Printf("\n👋 %d on file who no longer sit (left office, or the source spells the name differently):\n", len(gone))
	for _, name := range gone {
		fmt.Printf("   - %s\n", name)
	}
}

// departed returns the curated people the source no longer lists as sitting.
// Party and Fraktion accounts are skipped: no roster lists them, and they are
// not members.
func departed(cs []Contact, sitting []string) []string {
	sits := make(map[string]bool, len(sitting))
	for _, name := range sitting {
		sits[contacts.NameKey(name)] = true
	}

	var gone []string
	for _, c := range cs {
		if c.IsOrganization() || sits[contacts.NameKey(c.Name)] {
			continue
		}
		gone = append(gone, c.Name)
	}
	return gone
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

		// The same account may already be on file as a candidate. The
		// parliament publishing it is the confirmation that candidate was
		// waiting for, so promote it rather than skipping past it — otherwise a
		// harvested guess permanently shadows the authoritative record.
		if promoted, found := promote(*field, url); found {
			if promoted {
				added++
				fmt.Printf("   ✅ %s: %s confirmed by %s\n", c.Name, a.Platform, url)
			}
			continue
		}
		// Verified on arrival: these are the accounts the parliament publishes
		// for its own members, which is a stronger claim than any search result
		// and the same one the file has always recorded for them.
		*field = append(*field, Account{URL: url, Verified: true})
		added++
		fmt.Printf("   🔗 %s: %s %s\n", c.Name, a.Platform, url)
	}

	return added
}

// platformField addresses the slice a platform's accounts live in, or nil for a
// platform the schema has no column for.
func platformField(c *Contact, platform string) *[]Account {
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
// promote marks an account already on file as verified, reporting whether it
// changed anything and whether the account was there at all.
//
// A confirmation is a fact about the account: once the body publishes it, the
// confidence score that ranked it as a guess is spent and comes off with it.
func promote(existing []Account, candidate string) (changed, found bool) {
	key := accountKey(candidate)
	for i, have := range existing {
		if accountKey(have.URL) != key {
			continue
		}
		if existing[i].Verified {
			return false, true
		}
		existing[i].Verified = true
		existing[i].Confidence = ""
		return true, true
	}
	return false, false
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
	// X's old name: both spellings are valid for the platform and reach the same
	// profile.
	if host == "twitter.com" {
		host = "x.com"
	}
	// Bluesky's CDN host serves the same profile as the canonical one.
	host = strings.TrimPrefix(host, "web-cdn.")
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
