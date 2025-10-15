package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

// Geschaeft represents a council business/matter from the Zurich city council
type Geschaeft struct {
	GRNr             string `xml:"GRNr"`
	Titel            string `xml:"Titel"`
	Geschaeftsart    string `xml:"Geschaeftsart"`
	Geschaeftsstatus string `xml:"Geschaeftsstatus"`
	Beginn           struct {
		Start string `xml:"Start"`
		Text  string `xml:"Text"`
	} `xml:"Beginn"`
	Erstunterzeichner struct {
		KontaktGremium struct {
			Name   string `xml:"Name"`
			Partei string `xml:"Partei"`
		} `xml:"KontaktGremium"`
	} `xml:"Erstunterzeichner"`
}

// SearchDetailResponse represents the XML response from the API
type SearchDetailResponse struct {
	XMLName  xml.Name `xml:"SearchDetailResponse"`
	NumHits  int      `xml:"numHits,attr"`
	Hits     []Hit    `xml:"Hit"`
}

// Hit represents a single result in the search response
type Hit struct {
	Geschaeft Geschaeft `xml:"Geschaeft"`
}

// fetchLatestVote fetches the most recent geschaeft from the Zurich council API
func fetchLatestVote() (*Geschaeft, error) {
	// Fetch recent geschaeft (council business) from 2024 onwards, sorted by date descending
	url := "https://www.gemeinderat-zuerich.ch/api/geschaeft/searchdetails?q=beginn_start%20%3E%20%222024-01-01%2000:00:00%22%20sortBy%20beginn_start/sort.descending&l=de-CH&s=1&m=1"

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add user agent to avoid being blocked
	req.Header.Set("User-Agent", "ZurichRatsInfo/1.0 (Civic Tech Bot)")
	req.Header.Set("Accept", "application/xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp SearchDetailResponse
	if err := xml.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	if len(apiResp.Hits) == 0 {
		return nil, fmt.Errorf("no geschaeft found in API response")
	}

	return &apiResp.Hits[0].Geschaeft, nil
}

// formatVoteTweet formats a geschaeft into a tweet message
func formatVoteTweet(geschaeft *Geschaeft) string {
	dateStr := geschaeft.Beginn.Text
	if dateStr == "" {
		dateStr = "Datum unbekannt"
	}

	// Format: GR Nr, Type, Title, Author
	author := ""
	if geschaeft.Erstunterzeichner.KontaktGremium.Name != "" {
		author = fmt.Sprintf(" von %s", geschaeft.Erstunterzeichner.KontaktGremium.Name)
		if geschaeft.Erstunterzeichner.KontaktGremium.Partei != "" {
			author += fmt.Sprintf(" (%s)", geschaeft.Erstunterzeichner.KontaktGremium.Partei)
		}
	}

	message := fmt.Sprintf("🏛️ Neues Geschäft im Gemeinderat Zürich\n\n📋 %s: %s\n📅 %s%s\n\n%s",
		geschaeft.GRNr,
		geschaeft.Geschaeftsart,
		dateStr,
		author,
		geschaeft.Titel)

	// X has a 280 character limit
	if len(message) > 280 {
		message = message[:277] + "..."
	}

	return message
}
