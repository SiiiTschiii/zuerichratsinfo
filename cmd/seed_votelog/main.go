// seed_votelog marks a jurisdiction's existing votes as already posted.
//
// A new jurisdiction starts with an empty vote log, and an empty log means
// "nothing was ever posted" — so every one of its historical votes reads as
// unposted. Kanton Zürich has 2,626 of them. The only thing standing between
// that and a mass-post is the age guard, which is a single config value; resting
// a live account on one number is not a plan.
//
// Seeding removes the question. Run it before enabling a jurisdiction, so the
// log records the history as posted and the bot starts from the next real vote.
//
//	# See what would be seeded, and what would be left to post.
//	go run ./cmd/seed_votelog -jurisdiction zurich-canton -n 200
//
//	# Write the logs.
//	go run ./cmd/seed_votelog -jurisdiction zurich-canton -n 200 -write
//
// The written files must then be committed to the state-log branch, which is
// where the workflow reads them from.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/siiitschiii/zuerichratsinfo/pkg/config"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votes"
)

// seedPlatforms covers every platform the bot posts to. A log seeded for only
// some of them would leave the others to publish the backlog.
var seedPlatforms = []votelog.Platform{
	votelog.PlatformX,
	votelog.PlatformBluesky,
	votelog.PlatformInstagram,
}

func main() {
	jurisdictionKey := flag.String("jurisdiction", "", "jurisdiction to seed (required)")
	fetchLimit := flag.Int("n", 200, "how many recent votes to fetch and mark as posted")
	keepDays := flag.Int("keep-days", 0, "leave votes from the last N days unseeded, so they still post (0 = seed everything fetched)")
	write := flag.Bool("write", false, "write the vote logs; without this the command only reports")
	flag.Parse()

	if *jurisdictionKey == "" {
		log.Fatalf("-jurisdiction is required. Available: %v", config.JurisdictionKeys())
	}

	jurisdiction, err := config.LookupJurisdiction(*jurisdictionKey)
	if err != nil {
		log.Fatalf("%v. Available: %v", err, config.JurisdictionKeys())
	}

	fmt.Printf("Seeding %s (%s)\n", jurisdiction.Name, jurisdiction.Key)
	fmt.Printf("Fetching the %d most recent votes…\n", *fetchLimit)

	fetched, err := jurisdiction.NewSource().FetchRecent(*fetchLimit)
	if err != nil {
		log.Fatalf("Error fetching votes: %v", err)
	}
	fmt.Printf("Fetched %d votes.\n\n", len(fetched))

	toSeed, toPost := split(fetched, *keepDays)

	report("Would be marked as posted", toSeed)
	report("Would remain postable", toPost)

	// The age guard applies on top of the log, so it also bounds what a first
	// run can publish. Showing both makes the real exposure obvious.
	fmt.Printf("\nAge guard for %s: %d days.\n", jurisdiction.Key, jurisdiction.MaxAgeDays)
	fmt.Printf("After seeding, a first run would consider %d vote(s), of which %d are inside the age guard.\n",
		len(toPost), countWithinAge(toPost, jurisdiction.MaxAgeDays))

	if !*write {
		fmt.Println("\nDry run — nothing written. Re-run with -write once the numbers above look right.")
		return
	}

	fmt.Println()
	for _, platform := range seedPlatforms {
		vl, err := votelog.Load(jurisdiction.Key, platform)
		if err != nil {
			log.Fatalf("Error loading %s/%s vote log: %v", jurisdiction.Key, platform, err)
		}
		before := vl.Count()

		for _, v := range toSeed {
			vl.MarkAsPosted(v.SourceID)
		}
		if err := vl.Save(); err != nil {
			log.Fatalf("Error saving %s/%s vote log: %v", jurisdiction.Key, platform, err)
		}

		fmt.Printf("✅ %s: %d → %d entries (%s)\n",
			platform, before, vl.Count(), votelog.LogFilePath(jurisdiction.Key, platform))
	}

	fmt.Println("\nCommit these files to the state-log branch before enabling the jurisdiction.")
}

// split divides fetched votes into those to record as posted and those to leave
// postable. keepDays exists so a launch can deliberately start with the last
// sitting rather than with silence.
func split(fetched []votes.Vote, keepDays int) (seed, post []votes.Vote) {
	if keepDays <= 0 {
		return fetched, nil
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for _, v := range fetched {
		if !v.Date.IsZero() && v.Date.After(cutoff) {
			post = append(post, v)
			continue
		}
		seed = append(seed, v)
	}
	return seed, post
}

func report(label string, vs []votes.Vote) {
	fmt.Printf("%s: %d vote(s)", label, len(vs))
	if len(vs) == 0 {
		fmt.Println()
		return
	}
	fmt.Printf(" (%s … %s)\n", oldest(vs), newest(vs))
}

func countWithinAge(vs []votes.Vote, maxAgeDays int) int {
	if maxAgeDays <= 0 {
		return len(vs)
	}
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	n := 0
	for _, v := range vs {
		if v.Date.IsZero() || !v.Date.Before(cutoff) {
			n++
		}
	}
	return n
}

func oldest(vs []votes.Vote) string {
	var out time.Time
	for _, v := range vs {
		if !v.Date.IsZero() && (out.IsZero() || v.Date.Before(out)) {
			out = v.Date
		}
	}
	return dateOrUnknown(out)
}

func newest(vs []votes.Vote) string {
	var out time.Time
	for _, v := range vs {
		if v.Date.After(out) {
			out = v.Date
		}
	}
	return dateOrUnknown(out)
}

func dateOrUnknown(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Format("2006-01-02")
}
