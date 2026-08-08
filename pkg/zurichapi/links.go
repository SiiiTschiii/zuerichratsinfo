package zurichapi

import (
	"fmt"
	"regexp"
	"strings"
)

// antragOnlyRegex matches Traktandum titles that are just "Antrag XXX" or
// "Anträge XXX bis YYY", optionally followed by "zu Dispositivziffer X"
// (e.g. "Antrag 1 zu Dispositivziffer 1a"). Such titles carry no information,
// so the Geschäft title is used instead.
// The dot after the number is optional because the API sometimes omits it
// (e.g. "Antrag 1" vs "Antrag 007.").
var antragOnlyRegex = regexp.MustCompile(`^(\d+/\d+\s+)?Antr(a|ä)ge?\s+\d+\.?(\s*(bis|–|-)\s*\d+\.?)?(\s+zu\s+Dispositivziffer\s+\S+)?$`)

// SelectBestTitle chooses between the Traktandum and Geschäft titles,
// falling back to the Geschäft when the Traktandum title is a generic "Antrag N."
func SelectBestTitle(traktandumTitel, geschaeftTitel string) string {
	if IsGenericAntragTitle(traktandumTitel) {
		return geschaeftTitel
	}
	return traktandumTitel
}

// IsGenericAntragTitle reports whether a Traktandum title is just a generic
// "Antrag XXX" pattern.
func IsGenericAntragTitle(traktandumTitel string) bool {
	cleaned := strings.TrimSpace(traktandumTitel)
	cleaned = strings.ReplaceAll(cleaned, "\r\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	return antragOnlyRegex.MatchString(cleaned)
}

// VoteLink is the public page for a single vote.
func VoteLink(objGUID string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/abstimmungen/detail.php?aid=%s", objGUID)
}

// TraktandumLink is the agenda-item page, showing every vote taken on one
// business matter within a session.
func TraktandumLink(sitzungGuid, traktandumGuid string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/sitzungen/sitzung/?gid=%s#%s", sitzungGuid, traktandumGuid)
}

// GeschaeftLink is the business-matter page, showing everything filed under one
// Geschäft (e.g. budget 2025/391).
func GeschaeftLink(geschaeftGuid string) string {
	return fmt.Sprintf("https://www.gemeinderat-zuerich.ch/geschaefte/detail.php?gid=%s", geschaeftGuid)
}
