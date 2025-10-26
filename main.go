package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cgerber/zurichratsinfo/pkg/zurichapi"
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

	// Create API client and fetch latest geschaeft from Zurich council API
	client := zurichapi.NewClient()
	geschaeft, err := client.FetchLatestGeschaeft()
	if err != nil {
		log.Fatalf("Error fetching latest geschaeft: %v", err)
	}

	// Format tweet message
	message := zurichapi.FormatGeschaeftTweet(geschaeft)
	// Add timestamp for uniqueness during development
	timestamp := fmt.Sprintf("\n[dev %s]", time.Now().Format("2006-01-02 15:04:05"))
	message += timestamp
	fmt.Printf("Tweet to post:\n%s\n\n", message)

	// Post to X
	err = postTweet(apiKey, apiSecret, accessToken, accessSecret, message)
	if err != nil {
		log.Fatalf("Error posting tweet: %v", err)
	}

	fmt.Println("Successfully posted tweet!")
}
