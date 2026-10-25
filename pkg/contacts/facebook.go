package contacts

import (
	"net/url"
	"regexp"
	"strings"
)

// facebookIDPath matches the paths that carry a numeric ID after a name:
// /people/Anna-Graff/100070693802425/ and /pages/FDP/123456789.
var facebookIDPath = regexp.MustCompile(`^(?:people|pages)/[^/]+/(\d+)$`)

// ExtractFacebookMentionFromURL returns the markup that mentions the account a
// facebook.com URL names, "@[id]", or "" when the URL carries no numeric ID.
//
// A Facebook mention is written with the numeric ID, and the contacts files
// hold the URLs people copied from the address bar, which mostly name an
// account by its vanity name. Those cannot be mentioned until an ID has been
// found for them, so they are left untagged rather than guessed at:
//   - "https://www.facebook.com/profile.php?id=100070693802425" -> "@[100070693802425]"
//   - "https://www.facebook.com/1063164230212573"               -> "@[1063164230212573]"
//   - "https://www.facebook.com/people/Anna-Graff/1000706/"      -> "@[1000706]"
//   - "https://www.facebook.com/fdpstadtzuerich/"                -> ""
func ExtractFacebookMentionFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	host = strings.TrimPrefix(host, "m.")
	if host != "facebook.com" {
		return ""
	}

	path := strings.Trim(u.Path, "/")
	if path == "profile.php" {
		if id := u.Query().Get("id"); isDigits(id) {
			return "@[" + id + "]"
		}
		return ""
	}
	if isDigits(path) {
		return "@[" + path + "]"
	}
	if m := facebookIDPath.FindStringSubmatch(path); m != nil {
		return "@[" + m[1] + "]"
	}
	return ""
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// TagFacebookHandlesInText replaces each contact name that has a mentionable
// Facebook account with the mention. The mention renders as the account's own
// name, so it takes the place of the name instead of following it.
func (m *Mapper) TagFacebookHandlesInText(text string) string {
	var taggable []HandleTag
	for _, contact := range m.getAllContacts() {
		// A contact can hold several accounts, and the one with an ID need not
		// be the first listed: take the first that has one.
		mention := ""
		for _, url := range contact.Verified("facebook") {
			if mention = ExtractFacebookMentionFromURL(url); mention != "" {
				break
			}
		}
		if mention == "" {
			continue
		}
		for _, variant := range generateNameVariants(contact.Name) {
			taggable = append(taggable, HandleTag{Name: variant, Handle: mention})
		}
	}
	return replaceNamesInText(text, taggable)
}

// TagFacebookFraktionen tags the parties of a Fraktion phrase with their
// Facebook Pages, after the phrase as on the other platforms.
func (m *Mapper) TagFacebookFraktionen(text, jurisdiction string) string {
	return m.tagFraktionen(text, "facebook", jurisdiction, ExtractFacebookMentionFromURL)
}
