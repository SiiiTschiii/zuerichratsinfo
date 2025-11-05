package main

import (
	"fmt"
	"log"
	"os"

	"github.com/siiitschiii/zuerichratsinfo/pkg/xapi"
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi"
)

func main() {
	// Load configuration from environment
	apiKey := os.Getenv("X_API_KEY")
	apiSecret := os.Getenv("X_API_SECRET")
	accessToken := os.Getenv("X_ACCESS_TOKEN")
	accessSecret := os.Getenv("X_ACCESS_SECRET")

	if apiKey == "" || apiSecret == "" || accessToken == "" || accessSecret == "" {
		log.Fatal("Missing X API credentials. Please set X_API_KEY, X_API_SECRET, X_ACCESS_TOKEN, and X_ACCESS_SECRET environment variables.")
	}

	// Create API client and fetch latest abstimmung from Zurich council API
	client := zurichapi.NewClient()
	abstimmungen, err := client.FetchRecentAbstimmungen(1)
	if err != nil {
		log.Fatalf("Error fetching latest abstimmung: %v", err)
	}
	
	if len(abstimmungen) == 0 {
		log.Fatal("No abstimmungen found")
	}
	
	abstimmung := &abstimmungen[0]

	// Format tweet message
	message := zurichapi.FormatAbstimmungTweet(abstimmung)
	
	// Truncate to 280 characters if needed (count runes, not bytes, for Unicode)
	if len([]rune(message)) > 280 {
		runes := []rune(message)
		message = string(runes[:277]) + "..."
	}
	
	fmt.Printf("Tweet to post:\n%s\n\n", message)
	fmt.Printf("Character count: %d\n\n", len([]rune(message)))

	// Post to X
	err = xapi.PostTweet(apiKey, apiSecret, accessToken, accessSecret, message)
	if err != nil {
		log.Fatalf("Error posting tweet: %v", err)
	}

	fmt.Println("Successfully posted tweet!")
}
