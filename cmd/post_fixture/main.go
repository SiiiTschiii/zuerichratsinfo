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
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/instagram"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

type namedPlatform struct {
	name string
	plat platforms.Platform
}

// platformCredentials holds all platform credentials loaded from env.
type platformCredentials struct {
	xAPIKey      string
	xAPISecret   string
	xAccessToken string
	xAccessSecret string
	xEnabled     bool

	bskyHandle   string
	bskyPassword string
	bskyEnabled  bool

	igUserID      string
	igAccessToken string
	githubToken   string
	igRepoOwner   string
	igRepoName    string
	igEnabled     bool
}

func main() {
	fixture := flag.String("fixture", "all", "fixture name from AllFixtures() keys, or 'all'")
	platform := flag.String("platform", "all", "platform to post to: x, bluesky, instagram, or all")
	contactsFile := flag.String("contacts", filepath.Join("data", "contacts_test.yaml"), "contacts YAML file (default: test contacts with fake handles)")
	maxChars := flag.Int("x-max-chars", x.DefaultMaxChars, "per-post character limit for X (280 for free accounts, 2000 for Premium)")
	flag.Parse()

	creds := loadCredentials()
	validatePlatform(*platform, creds)

	// Load contacts for tagging
	contactMapper, err := contacts.LoadContacts(*contactsFile)
	if err != nil {
		log.Printf("Warning: Could not load contacts for tagging: %v", err)
		contactMapper = nil
	}

	fixtures := resolveFixtures(*fixture)
	plats := buildPlatforms(*platform, creds, contactMapper, *maxChars)

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

// loadCredentials reads platform credentials from environment variables.
func loadCredentials() platformCredentials {
	c := platformCredentials{
		xAPIKey:       os.Getenv("X_API_KEY"),
		xAPISecret:    os.Getenv("X_API_SECRET"),
		xAccessToken:  os.Getenv("X_ACCESS_TOKEN"),
		xAccessSecret: os.Getenv("X_ACCESS_SECRET"),
		bskyHandle:    os.Getenv("BLUESKY_HANDLE"),
		bskyPassword:  os.Getenv("BLUESKY_PASSWORD"),
		igUserID:      os.Getenv("IG_USER_ID"),
		igAccessToken: os.Getenv("IG_ACCESS_TOKEN"),
		githubToken:   os.Getenv("GITHUB_TOKEN"),
		igRepoOwner:   os.Getenv("IG_REPO_OWNER"),
		igRepoName:    os.Getenv("IG_REPO_NAME"),
	}
	c.xEnabled = c.xAPIKey != "" && c.xAPISecret != "" && c.xAccessToken != "" && c.xAccessSecret != ""
	c.bskyEnabled = c.bskyHandle != "" && c.bskyPassword != ""
	c.igEnabled = c.igUserID != "" && c.igAccessToken != "" && c.githubToken != "" && c.igRepoOwner != "" && c.igRepoName != ""
	return c
}

// validatePlatform checks that the selected platform has valid credentials configured.
func validatePlatform(platform string, creds platformCredentials) {
	if !creds.xEnabled && !creds.bskyEnabled && platform != "instagram" {
		log.Fatal("No platform credentials configured. Set X_API_KEY/X_API_SECRET/X_ACCESS_TOKEN/X_ACCESS_SECRET for X, or BLUESKY_HANDLE/BLUESKY_PASSWORD for Bluesky. Instagram (stub mode without IG_USER_ID/IG_ACCESS_TOKEN, or real mode with credentials) is available via -platform instagram.")
	}

	if platform == "x" && !creds.xEnabled {
		log.Fatal("X credentials required but not set")
	}
	if platform == "bluesky" && !creds.bskyEnabled {
		log.Fatal("Bluesky credentials required but not set")
	}
	if platform != "all" && platform != "x" && platform != "bluesky" && platform != "instagram" {
		log.Fatalf("Unknown platform %q — use x, bluesky, instagram, or all", platform)
	}
}

// resolveFixtures loads the requested fixture(s) from the test fixture map.
func resolveFixtures(fixture string) map[string][]zurichapi.Abstimmung {
	allFixtures := testfixtures.AllFixtures()

	if fixture == "all" {
		return allFixtures
	}

	votes, ok := allFixtures[fixture]
	if !ok {
		var names []string
		for k := range allFixtures {
			names = append(names, k)
		}
		sort.Strings(names)
		log.Fatalf("Unknown fixture %q. Available: %v", fixture, names)
	}
	return map[string][]zurichapi.Abstimmung{fixture: votes}
}

// buildPlatforms constructs the list of platforms to post to based on flags and credentials.
func buildPlatforms(platform string, creds platformCredentials, contactMapper *contacts.Mapper, maxChars int) []namedPlatform {
	var plats []namedPlatform

	if (platform == "all" || platform == "x") && creds.xEnabled {
		xPlat := x.NewXPlatform(creds.xAPIKey, creds.xAPISecret, creds.xAccessToken, creds.xAccessSecret, contactMapper, 100)
		xPlat.SetMaxChars(maxChars)
		plats = append(plats, namedPlatform{name: "X", plat: xPlat})
	}
	if (platform == "all" || platform == "bluesky") && creds.bskyEnabled {
		plats = append(plats, namedPlatform{
			name: "Bluesky",
			plat: bluesky.NewBlueskyPlatform(creds.bskyHandle, creds.bskyPassword, 100, contactMapper),
		})
	}
	if platform == "all" || platform == "instagram" {
		var igPlat *instagram.InstagramPlatform
		if creds.igEnabled {
			igPlat = instagram.NewInstagramPlatformWithCredentials(
				creds.igUserID, creds.igAccessToken, creds.githubToken,
				creds.igRepoOwner, creds.igRepoName, 100,
			)
			fmt.Println("📷 Instagram: real mode (credentials configured)")
		} else {
			igPlat = instagram.NewInstagramPlatform(100)
			fmt.Println("📷 Instagram: stub mode (no credentials)")
		}
		igPlat.SetContactMapper(contactMapper)
		plats = append(plats, namedPlatform{name: "Instagram", plat: igPlat})
	}

	return plats
}
