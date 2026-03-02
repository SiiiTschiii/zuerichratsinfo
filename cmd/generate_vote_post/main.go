package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/bluesky"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	numVotes := flag.Int("n", 1, "number of vote groups to preview")
	platform := flag.String("platform", "", "platform to preview: x, bluesky (default: all)")
	flag.Parse()

	showX := *platform == "" || strings.EqualFold(*platform, "x")
	showBluesky := *platform == "" || strings.EqualFold(*platform, "bluesky") || strings.EqualFold(*platform, "bsky")

	if !showX && !showBluesky {
		log.Fatalf("Unknown platform %q. Use: x, bluesky", *platform)
	}

	// Create API client
	client := zurichapi.NewClient()

	// Load contacts for X handle tagging
	var contactMapper *contacts.Mapper
	if showX {
		contactsPath := filepath.Join("data", "contacts.yaml")
		var err error
		contactMapper, err = contacts.LoadContacts(contactsPath)
		if err != nil {
			log.Printf("Warning: Could not load contacts for tagging: %v", err)
			contactMapper = nil
		}
	}

	// Use empty vote log (show all votes, not just unposted)
	emptyLog := votelog.NewEmpty(votelog.PlatformX)

	// Prepare votes (same logic as main.go)
	groups, err := voteposting.PrepareVoteGroups(client, emptyLog, *numVotes)
	if err != nil {
		log.Fatalf("Error preparing votes: %v", err)
	}

	if len(groups) == 0 {
		log.Fatal("No votes found")
	}

	if showX {
		xPlatform := x.NewXPlatform("", "", "", "", contactMapper, *numVotes)
		fmt.Println("━━━ X/Twitter ━━━")
		_, err = voteposting.PostToPlatform(groups, xPlatform, emptyLog, true)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	if showBluesky {
		bskyPlatform := bluesky.NewBlueskyPlatform("", "", *numVotes)
		if showX {
			fmt.Println()
		}
		fmt.Println("━━━ Bluesky ━━━")
		bskyLog := votelog.NewEmpty(votelog.PlatformBluesky)
		_, err = voteposting.PostToPlatform(groups, bskyPlatform, bskyLog, true)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}
