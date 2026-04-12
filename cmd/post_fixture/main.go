package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/bluesky"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	fixture := flag.String("fixture", "all", "fixture name from AllFixtures() keys, or 'all'")
	platform := flag.String("platform", "all", "platform to post to: x, bluesky, or all")
	contactsFile := flag.String("contacts", filepath.Join("data", "contacts_test.yaml"), "contacts YAML file (default: test contacts with fake handles)")
	maxChars := flag.Int("x-max-chars", x.DefaultMaxChars, "per-post character limit for X (280 for free accounts, 2000 for Premium)")
	flag.Parse()

	// Load credentials from env
	apiKey := os.Getenv("X_API_KEY")
	apiSecret := os.Getenv("X_API_SECRET")
	accessToken := os.Getenv("X_ACCESS_TOKEN")
	accessSecret := os.Getenv("X_ACCESS_SECRET")
	xEnabled := apiKey != "" && apiSecret != "" && accessToken != "" && accessSecret != ""

	bskyHandle := os.Getenv("BLUESKY_HANDLE")
	bskyPassword := os.Getenv("BLUESKY_PASSWORD")
	bskyEnabled := bskyHandle != "" && bskyPassword != ""

	if !xEnabled && !bskyEnabled {
		log.Fatal("No platform credentials configured. Set X_API_KEY/X_API_SECRET/X_ACCESS_TOKEN/X_ACCESS_SECRET for X, or BLUESKY_HANDLE/BLUESKY_PASSWORD for Bluesky.")
	}

	// Filter platforms based on flag
	if *platform == "x" && !xEnabled {
		log.Fatal("X credentials required but not set")
	}
	if *platform == "bluesky" && !bskyEnabled {
		log.Fatal("Bluesky credentials required but not set")
	}
	if *platform != "all" && *platform != "x" && *platform != "bluesky" {
		log.Fatalf("Unknown platform %q — use x, bluesky, or all", *platform)
	}

	// Load contacts for tagging
	contactMapper, err := contacts.LoadContacts(*contactsFile)
	if err != nil {
		log.Printf("Warning: Could not load contacts for tagging: %v", err)
		contactMapper = nil
	}

	// Resolve fixtures
	allFixtures := testfixtures.AllFixtures()
	var fixtures map[string][]zurichapi.Abstimmung

	if *fixture == "all" {
		fixtures = allFixtures
	} else {
		votes, ok := allFixtures[*fixture]
		if !ok {
			var names []string
			for k := range allFixtures {
				names = append(names, k)
			}
			sort.Strings(names)
			log.Fatalf("Unknown fixture %q. Available: %v", *fixture, names)
		}
		fixtures = map[string][]zurichapi.Abstimmung{*fixture: votes}
	}

	// Build platform list
	type namedPlatform struct {
		name string
		plat platforms.Platform
	}
	var plats []namedPlatform

	if (*platform == "all" || *platform == "x") && xEnabled {
		xPlat := x.NewXPlatform(apiKey, apiSecret, accessToken, accessSecret, contactMapper, 100)
		xPlat.SetMaxChars(*maxChars)
		plats = append(plats, namedPlatform{
			name: "X",
			plat: xPlat,
		})
	}
	if (*platform == "all" || *platform == "bluesky") && bskyEnabled {
		plats = append(plats, namedPlatform{
			name: "Bluesky",
			plat: bluesky.NewBlueskyPlatform(bskyHandle, bskyPassword, 100, contactMapper),
		})
	}

	// Post each fixture to each platform
	fixtureNames := make([]string, 0, len(fixtures))
	for k := range fixtures {
		fixtureNames = append(fixtureNames, k)
	}
	sort.Strings(fixtureNames)

	for _, name := range fixtureNames {
		votes := fixtures[name]
		for _, p := range plats {
			fmt.Printf("\n━━━ %s / %s ━━━\n", p.name, name)

			content, err := p.plat.Format(votes)
			if err != nil {
				log.Printf("Format error (%s / %s): %v", p.name, name, err)
				continue
			}

			fmt.Printf("Preview:\n%s\n\n", content.String())

			_, err = p.plat.Post(content)
			if err != nil {
				log.Printf("Post error (%s / %s): %v", p.name, name, err)
				continue
			}

			fmt.Printf("Posted %s / %s successfully\n", p.name, name)
		}
	}
}
