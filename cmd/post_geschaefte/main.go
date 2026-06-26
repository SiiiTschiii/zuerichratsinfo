// Command post_geschaefte fetches recently submitted Geschäfte (Motionen, Postulate, etc.)
// and posts new ones to the configured social media platforms.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/bskyapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/geschaeftposting"
	bskyfmt "github.com/siiitschiii/zuerichratsinfo/pkg/geschaeftposting/platforms/bluesky"
	xfmt "github.com/siiitschiii/zuerichratsinfo/pkg/geschaeftposting/platforms/x"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting/voteformat"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "preview posts without publishing")
	platform := flag.String("platform", "all", "platform to post to: bluesky, x, or all")
	maxFetch := flag.Int("max-fetch", 50, "max number of recent Geschäfte to fetch")
	maxAge := flag.Int("max-age-days", 14, "skip Geschäfte older than N days (0=no limit)")
	contactsFile := flag.String("contacts", filepath.Join("data", "contacts.yaml"), "contacts YAML file")
	maxChars := flag.Int("x-max-chars", xfmt.DefaultMaxChars, "per-post character limit for X")
	flag.Parse()

	validatePlatform(*platform)

	// Load contacts for handle tagging
	contactMapper, err := contacts.LoadContacts(*contactsFile)
	if err != nil {
		log.Printf("Warning: could not load contacts (%v) — handle tagging disabled", err)
		contactMapper = nil
	}

	client := zurichapi.NewClient()

	// Determine which platforms are enabled
	postBluesky := *platform == "all" || *platform == "bluesky"
	postX := *platform == "all" || *platform == "x"

	// Load per-platform logs
	var bskyLog, xLog *votelog.VoteLog
	if postBluesky {
		bskyLog, err = votelog.LoadGeschaeftLog(votelog.PlatformBluesky)
		if err != nil {
			log.Fatalf("Failed to load Bluesky geschaeft log: %v", err)
		}
	}
	if postX {
		xLog, err = votelog.LoadGeschaeftLog(votelog.PlatformX)
		if err != nil {
			log.Fatalf("Failed to load X geschaeft log: %v", err)
		}
	}

	// Use the Bluesky log as the primary filter (or X log if Bluesky is disabled)
	filterLog := bskyLog
	if filterLog == nil {
		filterLog = xLog
	}
	if filterLog == nil {
		filterLog = votelog.NewEmptyGeschaeftLog(votelog.PlatformBluesky)
	}

	// Prepare Geschäfte to post
	geschaefte, err := geschaeftposting.PrepareGeschaefte(client, filterLog, *maxFetch, *maxAge)
	if err != nil {
		log.Fatalf("Failed to prepare Geschäfte: %v", err)
	}

	if len(geschaefte) == 0 {
		fmt.Println("✅ No new Geschäfte to post.")
		return
	}

	fmt.Printf("📋 Found %d new Geschäfte to post\n", len(geschaefte))

	// Authenticate Bluesky lazily (only if needed)
	var bskySession *bskyapi.Session
	bskyHandle := os.Getenv("BLUESKY_HANDLE")
	bskyPassword := os.Getenv("BLUESKY_PASSWORD")
	bskyEnabled := postBluesky && bskyHandle != "" && bskyPassword != ""

	// Load X credentials
	xAPIKey := os.Getenv("X_API_KEY")
	xAPISecret := os.Getenv("X_API_SECRET")
	xAccessToken := os.Getenv("X_ACCESS_TOKEN")
	xAccessSecret := os.Getenv("X_ACCESS_SECRET")
	xEnabled := postX && xAPIKey != "" && xAPISecret != "" && xAccessToken != "" && xAccessSecret != ""

	if !bskyEnabled && postBluesky {
		log.Println("Warning: Bluesky credentials not set — skipping Bluesky")
	}
	if !xEnabled && postX {
		log.Println("Warning: X credentials not set — skipping X")
	}

	for _, g := range geschaefte {
		link := voteformat.GenerateGeschaeftLink(g.OBJGUID)
		fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("📋 %s (%s) — %s\n", g.GRNr, g.Geschaeftsart, g.Beginn.Start[:10])
		fmt.Printf("   %s\n", link)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		// --- Bluesky ---
		if bskyEnabled || *dryRun && postBluesky {
			thread := bskyfmt.FormatGeschaeftThread(g, contactMapper)
			fmt.Println("\n🟦 Bluesky:")
			for i, post := range thread {
				if i == 0 {
					fmt.Println(post.Text)
				} else {
					fmt.Printf("  ↳ Reply %d:\n%s\n", i, post.Text)
				}
			}

			if !*dryRun && bskyEnabled {
				if bskySession == nil {
					bskySession, err = bskyapi.CreateSession(bskyHandle, bskyPassword)
					if err != nil {
						log.Printf("❌ Bluesky auth failed: %v", err)
						bskyEnabled = false
					} else {
						fmt.Printf("🔑 Bluesky: authenticated as %s\n", bskySession.Handle)
					}
				}
				if bskyEnabled && bskySession != nil {
					if err := postBlueskyThread(bskySession, thread); err != nil {
						log.Printf("❌ Bluesky post failed for %s: %v", g.GRNr, err)
					} else {
						bskyLog.MarkAsPosted(g.OBJGUID)
						if err := bskyLog.Save(); err != nil {
							log.Printf("Warning: failed to save Bluesky log: %v", err)
						}
						fmt.Printf("✅ Bluesky: posted %s\n", g.GRNr)
					}
				}
			} else if *dryRun {
				fmt.Println("  [dry-run: not posted]")
			}
		}

		// --- X ---
		if xEnabled || *dryRun && postX {
			thread := xfmt.FormatGeschaeftThread(g, contactMapper, *maxChars)
			fmt.Println("\n🐦 X:")
			for i, post := range thread {
				if i == 0 {
					fmt.Println(post.Text)
				} else {
					fmt.Printf("  ↳ Reply %d:\n%s\n", i, post.Text)
				}
			}

			if !*dryRun && xEnabled {
				// X posting via xapi is not yet wired up for Geschäfte.
				// Posting must be done manually until xapi integration is added.
				log.Printf("ℹ️  X posting for Geschäfte not yet wired to xapi; post manually")
				// Do NOT mark as posted — the item was not actually published.
			} else if *dryRun {
				fmt.Println("  [dry-run: not posted]")
			}
		}
	}

	fmt.Printf("\n✅ Done. Processed %d Geschäfte.\n", len(geschaefte))
}

// postBlueskyThread posts a 2-post thread (root + reply) to Bluesky.
func postBlueskyThread(session *bskyapi.Session, thread []*bskyfmt.GeschaeftPost) error {
	if len(thread) == 0 {
		return nil
	}

	// Post root
	root := thread[0]
	rootRef, err := bskyapi.CreateRecord(session, root.Text, root.Facets, nil)
	if err != nil {
		return fmt.Errorf("failed to post root: %w", err)
	}
	uriParts := strings.Split(rootRef.URI, "/")
	fmt.Printf("   ✅ Root: https://bsky.app/profile/%s/post/%s\n",
		session.Handle, uriParts[len(uriParts)-1])

	// Post replies
	parentRef := rootRef
	for i, reply := range thread[1:] {
		replyRef := &bskyapi.ReplyRef{
			Root:   *rootRef,
			Parent: *parentRef,
		}
		ref, err := bskyapi.CreateRecord(session, reply.Text, reply.Facets, replyRef)
		if err != nil {
			return fmt.Errorf("failed to post reply %d: %w", i+1, err)
		}
		fmt.Printf("   ↳ Reply %d: %s\n", i+1, ref.URI)
		parentRef = ref
	}

	return nil
}

// validatePlatform checks the -platform flag is one of the supported values.
func validatePlatform(p string) {
	switch p {
	case "all", "bluesky", "x":
		// OK
	default:
		log.Fatalf("Unknown platform %q — use bluesky, x, or all", p)
	}
}
