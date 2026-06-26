// Package geschaeftposting provides functionality for posting newly submitted
// Geschäfte (Motionen, Postulate, etc.) to social media platforms.
package geschaeftposting

import (
	"log"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/geschaeftposting/geschaeftformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

// PrepareGeschaefte fetches recent Geschäfte, filters to postable types that have
// not yet been posted, and returns them sorted oldest-first (to post in submission
// order). maxAgeDays limits how old a Geschaeft's Beginn date can be (0 = no limit).
func PrepareGeschaefte(
	client *zurichapi.Client,
	geschaeftLog *votelog.VoteLog,
	maxToFetch int,
	maxAgeDays int,
) ([]zurichapi.Geschaeft, error) {
	all, err := client.FetchRecentGeschaefte(maxToFetch)
	if err != nil {
		return nil, err
	}

	var result []zurichapi.Geschaeft
	for _, g := range all {
		// Only post supported types
		if !geschaeftformat.IsPostable(g.Geschaeftsart) {
			continue
		}

		// Skip already posted Geschäfte
		if geschaeftLog.IsPosted(g.OBJGUID) {
			continue
		}

		// Skip Geschäfte older than maxAgeDays
		if maxAgeDays > 0 && g.Beginn.Start != "" {
			dt, err := time.Parse("2006-01-02 15:04:05", g.Beginn.Start)
			if err != nil {
				dt, err = time.Parse("2006-01-02", g.Beginn.Start[:10])
			}
			if err == nil {
				cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
				if dt.Before(cutoff) {
					log.Printf("⚠️  Skipping old Geschaeft %s (Beginn %s, older than %d days)",
						g.GRNr, g.Beginn.Start[:10], maxAgeDays)
					continue
				}
			}
		}

		result = append(result, g)
	}

	// Reverse to post oldest-first (API returns newest-first)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}
