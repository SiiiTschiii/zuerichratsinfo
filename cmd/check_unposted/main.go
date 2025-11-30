package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/siiitschiii/zuerichratsinfo/pkg/contacts"
	"github.com/siiitschiii/zuerichratsinfo/pkg/votelog"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Get number of votes to check from command line argument, or environment, or default
	maxVotesToCheck := 50
	if len(os.Args) > 1 {
		// Command line argument takes priority
		if n, err := strconv.Atoi(os.Args[1]); err == nil && n > 0 {
			maxVotesToCheck = n
		} else {
			log.Fatalf("Invalid argument: please provide a positive number")
		}
	} else {
		// Fall back to environment variable
		maxVotesToCheck = getEnvInt("MAX_VOTES_TO_CHECK", 50)
	}
	
	maxPostsPerRun := getEnvInt("X_MAX_POSTS_PER_RUN", 10)

	// Load contacts for X handle tagging
	contactsPath := filepath.Join("data", "contacts.yaml")
	contactMapper, err := contacts.LoadContacts(contactsPath)
	if err != nil {
		log.Printf("Warning: Could not load contacts for tagging: %v", err)
		contactMapper = nil // Continue without tagging
	}

	// Load the vote log for X platform
	voteLog, err := votelog.Load(votelog.PlatformX)
	if err != nil {
		log.Fatalf("Error loading vote log: %v", err)
	}
	fmt.Printf("📊 Loaded vote log: %d votes already posted\n\n", voteLog.Count())

	// Fetch recent votes from Zurich council API
	client := zurichapi.NewClient()
	abstimmungen, err := client.FetchRecentAbstimmungen(maxVotesToCheck)
	if err != nil {
		log.Fatalf("Error fetching abstimmungen: %v", err)
	}
	
	if len(abstimmungen) == 0 {
		log.Fatal("No abstimmungen found")
	}

	// Filter out already posted votes
	var unpostedVotes []zurichapi.Abstimmung
	for _, vote := range abstimmungen {
		if !voteLog.IsPosted(vote.OBJGUID) {
			unpostedVotes = append(unpostedVotes, vote)
		}
	}

	fmt.Printf("📝 Found %d unposted votes out of %d recent votes\n\n", len(unpostedVotes), len(abstimmungen))

	if len(unpostedVotes) == 0 {
		fmt.Println("✨ No new votes to post!")
		return
	}

	// Group votes by business matter and date
	voteGroups, err := client.GroupAbstimmungenByGeschaeft(unpostedVotes)
	if err != nil {
		log.Fatalf("Error grouping votes: %v", err)
	}
	fmt.Printf("📦 Grouped into %d post(s)\n\n", len(voteGroups))

	// Limit number of posts
	// Count individual votes, but always include complete groups
	groupsToPost := voteGroups
	if len(voteGroups) > 0 {
		voteCount := 0
		groupLimit := len(voteGroups)
		for i, group := range voteGroups {
			if voteCount+len(group) > maxPostsPerRun {
				groupLimit = i
				break
			}
			voteCount += len(group)
		}
		if groupLimit < len(voteGroups) {
			fmt.Printf("⚠️  Would limit to %d groups (%d votes) per run (found %d groups with %d total votes)\n\n", 
				groupLimit, voteCount, len(voteGroups), len(unpostedVotes))
			groupsToPost = voteGroups[:groupLimit]
		}
	}

	fmt.Printf("🚀 Would post these %d groups:\n\n", len(groupsToPost))
	
	for i, group := range groupsToPost {
		message := zurichapi.FormatVoteGroupPost(group, contactMapper)
		fmt.Printf("─────────────────────────────────────────────────────────────────\n")
		fmt.Printf("[%d/%d] Group with %d vote(s)\n", i+1, len(groupsToPost), len(group))
		fmt.Printf("Business: %s\n", group[0].GeschaeftGrNr)
		fmt.Printf("Date: %s\n", group[0].SitzungDatum[:10])
		fmt.Printf("Vote IDs: ")
		for j, vote := range group {
			if j > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", vote.OBJGUID[:8]+"...")
		}
		fmt.Printf("\n\n%s\n", message)
		fmt.Printf("\nCharacter count: %d\n", len(message))
	}
	fmt.Printf("─────────────────────────────────────────────────────────────────\n")
}

// getEnvInt gets an integer from environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}
