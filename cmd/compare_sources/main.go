// compare_sources diffs the same Stadt Zürich votes as served by PARIS and by
// OpenParlData.
//
// It exists because the Kanton Zürich adapter cannot be validated directly:
// there is no second source to check it against. Stadt Zürich is served by
// both, so running the OpenParlData adapter against it — where PARIS is
// canonical and known-good — is what makes the adapter trustworthy before it is
// pointed at a body nobody can cross-check.
//
// It makes live calls to both APIs and is therefore run by hand, never in CI.
//
//	go run ./cmd/compare_sources -n 40
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/openparldata"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// cityBodyKey is Stadt Zürich's body in OpenParlData.
const cityBodyKey = "261"

func main() {
	n := flag.Int("n", 40, "number of recent votes to compare")
	verbose := flag.Bool("v", false, "print every vote, not just mismatches")
	flag.Parse()

	fmt.Printf("Comparing the %d most recent Stadt Zürich votes: PARIS vs OpenParlData (body_key=%s)\n\n", *n, cityBodyKey)

	paris, err := zurichapi.NewClient().FetchRecent(*n)
	if err != nil {
		log.Fatalf("PARIS: %v", err)
	}

	opd, err := openparldata.New(zurichapi.Jurisdiction, cityBodyKey).FetchRecent(*n)
	if err != nil {
		log.Fatalf("OpenParlData: %v", err)
	}

	fmt.Printf("PARIS returned %d votes, OpenParlData %d\n", len(paris), len(opd))

	opdByID := make(map[string]votes.Vote, len(opd))
	for _, v := range opd {
		// Both sources key on the same identifier, which is what makes the
		// comparison possible at all: OpenParlData re-serves PARIS, so its
		// external_id is PARIS' OBJ_GUID.
		opdByID[strings.ToLower(v.SourceID)] = v
	}

	var compared, missing int
	var conflicts []string

	for _, p := range paris {
		o, ok := opdByID[strings.ToLower(p.SourceID)]
		if !ok {
			missing++
			if *verbose {
				fmt.Printf("· %s not in OpenParlData yet (harvest lag)\n", p.SourceID)
			}
			continue
		}
		compared++

		diffs := diff(p, o)
		if len(diffs) > 0 {
			conflicts = append(conflicts, fmt.Sprintf("✗ %s (%s, %s)\n    %s",
				p.SourceID, p.Affair.Number, p.DateString(), strings.Join(diffs, "\n    ")))
		} else if *verbose {
			fmt.Printf("✓ %s (%s) %s\n", p.SourceID, p.Affair.Number, truncate(p.Title, 60))
		}
	}

	fmt.Printf("\nCompared %d votes; %d present in PARIS but not yet in OpenParlData.\n", compared, missing)

	if len(conflicts) == 0 {
		fmt.Println("✅ No conflicting totals, decisions or titles.")
		return
	}

	sort.Strings(conflicts)
	fmt.Printf("\n❌ %d vote(s) disagree:\n\n%s\n", len(conflicts), strings.Join(conflicts, "\n"))
	os.Exit(1)
}

// diff reports substantive disagreements only.
//
// A vote missing from one source is harvest lag, not a conflict — but the same
// vote carrying different numbers in the two sources means one of them is
// wrong, and that is what would put an incorrect result in front of readers.
func diff(paris, opd votes.Vote) []string {
	var out []string

	cmpCount := func(name string, a, b *int) {
		switch {
		case a == nil && b == nil:
		case a == nil || b == nil:
			out = append(out, fmt.Sprintf("%s: PARIS=%s OpenParlData=%s", name, countStr(a), countStr(b)))
		case *a != *b:
			out = append(out, fmt.Sprintf("%s: PARIS=%d OpenParlData=%d", name, *a, *b))
		}
	}

	cmpCount("Ja", paris.Yes, opd.Yes)
	cmpCount("Nein", paris.No, opd.No)
	cmpCount("Enthaltung", paris.Abstention, opd.Abstention)
	cmpCount("Abwesend", paris.Absent, opd.Absent)

	// PARIS says "angenommen"/"abgelehnt", OpenParlData "Ja"/"Nein". Comparing
	// the rendered outcome rather than the raw label is the only comparison
	// that means anything, and it is also what a reader would see.
	if pd, od := outcome(paris.Decision), outcome(opd.Decision); pd != od {
		out = append(out, fmt.Sprintf("decision: PARIS=%q(%s) OpenParlData=%q(%s)",
			paris.Decision, pd, opd.Decision, od))
	}

	if paris.DateString() != opd.DateString() {
		out = append(out, fmt.Sprintf("date: PARIS=%s OpenParlData=%s", paris.DateString(), opd.DateString()))
	}

	if !titlesAgree(paris, opd) {
		out = append(out, fmt.Sprintf("title: PARIS=%q OpenParlData=%q",
			truncate(paris.Title, 70), truncate(opd.Title, 70)))
	}

	return out
}

// titlesAgree is lenient by design. PARIS picks between an agenda-item title
// and a business title; OpenParlData only has the business title. Requiring
// equality would report a difference in editorial convention as a data error.
func titlesAgree(paris, opd votes.Vote) bool {
	candidates := []string{paris.Title, paris.Affair.Title}
	for _, c := range candidates {
		if normalise(c) == normalise(opd.Title) || normalise(c) == normalise(opd.Affair.Title) {
			return true
		}
	}
	return false
}

func normalise(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func outcome(decision string) string {
	d := strings.ToLower(strings.TrimSpace(decision))
	switch {
	case strings.HasPrefix(d, "auswahl"):
		return "auswahl"
	case strings.Contains(d, "angenommen") || d == "ja":
		return "accepted"
	case d == "":
		return "unknown"
	default:
		return "rejected"
	}
}

func countStr(c *int) string {
	if c == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *c)
}

func truncate(s string, n int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
