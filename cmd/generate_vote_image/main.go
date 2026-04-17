package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/siiitschiii/zuerichratsinfo/pkg/imagegen"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/testfixtures"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	fixture := flag.String("fixture", "all", "fixture name from AllFixtures() keys, or 'all'")
	outDir := flag.String("out", "out/images", "output directory for generated JPEGs")
	flag.Parse()

	allFixtures := testfixtures.AllFixtures()
	var fixtures map[string][]zurichapi.Abstimmung

	if *fixture == "all" {
		fixtures = allFixtures
	} else {
		votes, ok := allFixtures[*fixture]
		if !ok {
			log.Fatalf("Unknown fixture %q. Available: %v", *fixture, testfixtures.FixtureNames)
		}
		fixtures = map[string][]zurichapi.Abstimmung{*fixture: votes}
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("Creating output directory: %v", err)
	}

	// Use definition order from FixtureNames
	var fixtureNames []string
	if *fixture == "all" {
		fixtureNames = testfixtures.FixtureNames
	} else {
		fixtureNames = []string{*fixture}
	}

	for idx, name := range fixtureNames {
		votes := fixtures[name]
		images, err := imagegen.GenerateCarousel(votes)
		if err != nil {
			log.Printf("Error generating %s: %v", name, err)
			continue
		}

		for i, imgData := range images {
			filename := fmt.Sprintf("%02d_%s_%d.jpg", idx, name, i)
			path := filepath.Join(*outDir, filename)
			if err := os.WriteFile(path, imgData, 0o644); err != nil {
				log.Printf("Error writing %s: %v", path, err)
				continue
			}
			fmt.Printf("Wrote %s (%d bytes)\n", path, len(imgData))
		}
	}
}
