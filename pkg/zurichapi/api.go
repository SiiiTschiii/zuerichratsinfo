package zurichapi

import (
	"encoding/xml"
	"fmt"
)

const (
	// Base URLs for the Zurich city council APIs
	GeschaeftBaseURL = "https://www.gemeinderat-zuerich.ch/api/geschaeft"
	KontaktBaseURL   = "https://www.gemeinderat-zuerich.ch/api/kontakt"
)

// FetchLatestGeschaeft fetches the most recent geschaeft from the Zurich council API
func (c *Client) FetchLatestGeschaeft() (*Geschaeft, error) {
	// Fetch recent geschaeft (council business) from 2024 onwards, sorted by date descending
	url := GeschaeftBaseURL + "/searchdetails?q=beginn_start%20%3E%20%222024-01-01%2000:00:00%22%20sortBy%20beginn_start/sort.descending&l=de-CH&s=1&m=1"

	body, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}

	var resp GeschaeftSearchResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	if len(resp.Hits) == 0 {
		return nil, fmt.Errorf("no geschaeft found in API response")
	}

	return &resp.Hits[0].Geschaeft, nil
}

// FetchAllKontakte fetches all contacts from the Zurich council API
func (c *Client) FetchAllKontakte() ([]Kontakt, error) {
	url := KontaktBaseURL + "/searchdetails?q=seq>0&l=de-CH"

	body, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}

	var resp KontaktSearchResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	contacts := make([]Kontakt, len(resp.Hits))
	for i, hit := range resp.Hits {
		contacts[i] = hit.Kontakt
	}

	return contacts, nil
}