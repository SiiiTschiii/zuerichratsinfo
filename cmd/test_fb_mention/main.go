// test_fb_mention posts one test message per account to a Facebook Page and
// reads back whether Facebook turned the mention into a tag.
//
// It answers a question the Pages API documentation does not: with the access
// this app has, can a post mention another Page, or a person's profile? The
// bot tags politicians and parties on every other platform, and a mention that
// publishes as literal "@[123]" text would be worse than none.
//
// Usage (the Page must be a test Page, since the posts are public):
//
//	ZURICH_FB_ACCESS_TOKEN=<token of that Page> go run ./cmd/test_fb_mention \
//	    -page-id <test page id> \
//	    -party-url https://www.facebook.com/fdpstadtzuerich/ \
//	    -contact-url https://www.facebook.com/profile.php?id=<id>
//
// A mention notifies the account it names, so name only accounts that expect
// the test. At least one of -party-url and -contact-url is required.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/siiitschiii/zuerichratsinfo/pkg/config"
	"github.com/siiitschiii/zuerichratsinfo/pkg/fbapi"
)

func main() {
	channelKey := flag.String("channel", config.DefaultChannelKey, fmt.Sprintf("channel whose token to use %v", config.ChannelKeys()))
	pageID := flag.String("page-id", "", "ID of the test Page to post to (required; never read from the environment, so production cannot be hit by a leftover variable)")
	partyURL := flag.String("party-url", "", "facebook.com URL of a party Page to mention")
	contactURL := flag.String("contact-url", "", "facebook.com URL of a contact to mention")
	del := flag.Bool("delete", false, "delete the test posts again after reading them back")
	flag.Parse()

	channel, err := config.LookupChannel(*channelKey)
	if err != nil {
		log.Fatal(err)
	}
	token := channel.Env("FB_ACCESS_TOKEN")
	switch {
	case *pageID == "":
		log.Fatal("-page-id is required: the ID of a test Page")
	case token == "":
		log.Fatalf("%sFB_ACCESS_TOKEN is not set: it must be a Page token for the test Page", channel.EnvPrefix())
	case *partyURL == "" && *contactURL == "":
		log.Fatal("give at least one of -party-url and -contact-url")
	}

	client := fbapi.NewClient(*pageID, token)

	failed := false
	for _, target := range []struct{ kind, url string }{
		{"party", *partyURL},
		{"contact", *contactURL},
	} {
		if target.url == "" {
			continue
		}
		fmt.Printf("\n━━━ %s: %s ━━━\n", target.kind, target.url)
		if !run(client, target.kind, target.url, *del) {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// run resolves one account, mentions it in a post, and reports whether the
// mention was accepted. It returns false when something went wrong that is not
// itself the experiment's answer.
func run(client *fbapi.Client, kind, rawURL string, del bool) (ok bool) {
	account, err := fbapi.ParseAccountURL(rawURL)
	if err != nil {
		log.Printf("❌ %v", err)
		return false
	}

	id, name := account.ID, ""
	if id == "" {
		// The vanity name needs a lookup, which Meta may refuse for another
		// account. That refusal is a result, not a failure of the tool.
		id, name, err = client.Lookup(account.Username)
		if err != nil {
			fmt.Printf("🔎 Lookup refused or failed: %v\n", err)
			fmt.Println("   → without a numeric ID the bot cannot write this mention; the contacts file would have to carry IDs.")
			return true
		}
		fmt.Printf("🔎 Resolved %q to ID %s (%s)\n", account.Username, id, name)
	} else {
		fmt.Printf("🔎 The URL carries the ID %s, so no lookup is needed\n", id)
	}

	message := fmt.Sprintf("Mention-Test (%s): %s\n\nTestpost, wird wieder gelöscht.", kind, fbapi.Mention(id))
	postID, err := client.PublishTextPost(message)
	if err != nil {
		log.Printf("❌ Publishing failed: %v", err)
		return false
	}
	fmt.Printf("✅ Post created: %s\n", fbapi.PostURL(postID))

	// From here the post is public, so it is removed however the rest goes: a
	// read-back that fails must not leave it behind. A failed delete turns the
	// result into a failure, and a result that already failed stays one.
	if del {
		defer func() {
			if err := client.DeletePost(postID); err != nil {
				log.Printf("❌ Deleting the post failed: %v", err)
				ok = false
				return
			}
			fmt.Println("🗑️  Post deleted")
		}()
	}

	tags, err := client.MessageTags(postID)
	if err != nil {
		log.Printf("❌ Reading the tags back failed: %v", err)
		return false
	}
	if contains(tags, id) {
		fmt.Printf("🏷️  Facebook accepted the mention (tags: %s)\n", strings.Join(tags, ", "))
	} else {
		fmt.Printf("🚫 Facebook did NOT accept the mention (tags: %v) — it publishes as plain text\n", tags)
	}
	return true
}

func contains(ids []string, id string) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}
	return false
}
