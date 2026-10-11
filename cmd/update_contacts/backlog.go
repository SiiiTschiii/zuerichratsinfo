package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/config"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// backlog returns up to n people to look for accounts for next: sitting
// members with no account of any kind, never-searched first and then the least
// recently searched, in file order within a tie.
//
// Ordering by when someone was last searched is what makes the batches rotate. A
// search that finds nothing leaves the person without accounts, so by file order
// alone the same few would head the list every time and the rest would never be
// reached.
//
// sitting is nil when the source cannot say who sits, in which case everyone
// counts. Party and Fraktion accounts are never included.
func backlog(cs []Contact, sitting []string, n int) []Contact {
	sits := make(map[string]bool, len(sitting))
	for _, name := range sitting {
		sits[contacts.NameKey(name)] = true
	}

	var due []Contact
	for _, c := range cs {
		if c.IsOrganization() || c.HasAccounts() {
			continue
		}
		if sitting != nil && !sits[contacts.NameKey(c.Name)] {
			continue
		}
		due = append(due, c)
	}

	// ISO dates sort as text, and "" (never searched) sorts first.
	sort.SliceStable(due, func(i, j int) bool { return due[i].Searched < due[j].Searched })

	if len(due) > n {
		due = due[:n]
	}
	return due
}

// validateBacklog rejects a -backlog that was given but is not a positive
// number. Without it, 0 or a negative value reads as the flag being absent and
// the run falls through to a roster refresh, which writes the file, when the
// caller asked only to see a list.
func validateBacklog(set bool, n int) error {
	if set && n <= 0 {
		return fmt.Errorf("-backlog must be a positive number, got %d", n)
	}
	return nil
}

// runBacklog prints the next batch, one person per line.
func runBacklog(j config.Jurisdiction, cs []Contact, n int) error {
	var sitting []string
	if lister, ok := j.NewMemberSource().(votes.SittingLister); ok {
		names, err := lister.SittingNames()
		if err != nil {
			return fmt.Errorf("failed to fetch who sits: %w", err)
		}
		sitting = names
	}

	for _, c := range backlog(cs, sitting, n) {
		last := "never searched"
		if c.Searched != "" {
			last = "searched " + c.Searched
		}
		fmt.Printf("%s (%s)\n", c.Name, last)
	}
	return nil
}

// markSearched records that the named people were searched for on date. It
// refuses a name that is not on file, or that is a party account, rather than
// skipping it: a typo here would quietly leave someone at the head of the queue.
func markSearched(existing map[string]*Contact, names []string, date string) error {
	if _, err := time.Parse(contacts.SearchedDateLayout, date); err != nil {
		return fmt.Errorf("date %q must be YYYY-MM-DD", date)
	}
	if len(names) == 0 {
		return fmt.Errorf("no names given")
	}

	var unknown []string
	for _, name := range names {
		if c, ok := existing[contacts.NameKey(name)]; !ok || c.IsOrganization() {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("not a person on file: %s", strings.Join(unknown, ", "))
	}

	for _, name := range names {
		existing[contacts.NameKey(name)].Searched = date
	}
	return nil
}

// runMarkSearched marks the people and writes the file.
func runMarkSearched(path, header string, existing map[string]*Contact, names []string, date string, dryRun bool) error {
	if err := markSearched(existing, names, date); err != nil {
		return err
	}
	fmt.Printf("📝 %d marked searched on %s\n", len(names), date)
	if dryRun {
		fmt.Println("🔍 Dry run — nothing written.")
		return nil
	}

	merged, _, _ := merge(existing, nil)
	return save(path, header, merged)
}
