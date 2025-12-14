package zurichapi

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
)

const (
	// Base URLs for the Zurich city council APIs
	GeschaeftBaseURL       = "https://www.gemeinderat-zuerich.ch/api/geschaeft"
	KontaktBaseURL         = "https://www.gemeinderat-zuerich.ch/api/kontakt"
	AbstimmungBaseURL      = "https://www.gemeinderat-zuerich.ch/api/abstimmung"
	BehoerdenmandatBaseURL = "https://www.gemeinderat-zuerich.ch/api/behoerdenmandat"
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

// FetchRecentAbstimmungen fetches the n most recent abstimmungen (votes) from the Zurich council API
func (c *Client) FetchRecentAbstimmungen(limit int) ([]Abstimmung, error) {
	url := fmt.Sprintf("%s/searchdetails?q=seq%%3E0%%20sortBy%%20seq/sort.descending&l=de-CH&s=1&m=%d",
		AbstimmungBaseURL, limit)

	body, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}

	var resp AbstimmungSearchResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	abstimmungen := make([]Abstimmung, len(resp.Hits))
	for i, hit := range resp.Hits {
		abstimmungen[i] = hit.Abstimmung
	}

	return abstimmungen, nil
}

// FetchAbstimmungenForTraktandum fetches all votes for a specific Traktandum by its GUID
func (c *Client) FetchAbstimmungenForTraktandum(traktandumGuid string) ([]Abstimmung, error) {
	// Query the Abstimmung API for this specific TraktandumGuid
	// Using "traktandumguid any" syntax (text fields require "any" relation)
	url := fmt.Sprintf("%s/searchdetails?q=traktandumguid%%20any%%20%%22%s%%22&l=de-CH",
		AbstimmungBaseURL,
		traktandumGuid)

	body, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}

	var resp AbstimmungSearchResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	// Extract all Abstimmungen from hits
	var abstimmungen []Abstimmung
	for _, hit := range resp.Hits {
		abstimmungen = append(abstimmungen, hit.Abstimmung)
	}

	return abstimmungen, nil
}

// GetActiveMandates fetches active mandates from the Gemeinderat
// A mandate is considered active if its End date is "9999-12-31 00:00:00" or later
// excludeFunktionen is an optional list of Funktion values to exclude from results
func (c *Client) GetActiveMandates(excludeFunktionen ...string) ([]Behoerdenmandat, error) {
	// Query for active Gemeinderat mandates using API search fields
	// dauer_end >= "9999-12-31 00:00:00" filters for active mandates
	url := BehoerdenmandatBaseURL + "/searchdetails?q=gremium%20adj%20Gemeinderat%20and%20dauer_end%20%3E%3D%20%229999-12-31%2000:00:00%22&l=de-CH"

	body, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}

	// Strip the namespace declaration which causes issues with Go's XML parser
	body = bytes.ReplaceAll(body, []byte(` xmlns="http://www.cmiag.ch/cdws/Behoerdenmandat"`), []byte(""))

	var resp BehoerdenmandatSearchResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	// Build a set of excluded functions for efficient lookup
	excludeSet := make(map[string]bool)
	for _, f := range excludeFunktionen {
		excludeSet[f] = true
	}

	// Filter out excluded functions
	var activeMandates []Behoerdenmandat
	for _, hit := range resp.Hits {
		mandat := hit.Behoerdenmandat

		// Skip if this function is in the exclude list
		if excludeSet[mandat.Funktion] {
			continue
		}

		activeMandates = append(activeMandates, mandat)
	}

	return activeMandates, nil
}

// FetchActiveGemeinderatMandates fetches all active Gemeinderat member mandates
// This is a convenience function that excludes Stadtrat, Stimmenzählende, and Ratssekretariat roles
func (c *Client) FetchActiveGemeinderatMandates() ([]Behoerdenmandat, error) {
	return c.GetActiveMandates(
		"Stadtrat",
		"Stimmenzählende",
		"Ratssekretariat",
		"Präsidium Stadtrat",
		"Mitglied Stadtrat",
	)
}

// GroupAbstimmungenByGeschaeft groups abstimmungen by their business matter (GeschaeftGrNr)
// and voting session date (SitzungDatum). Returns a slice of vote groups.
// This method also ensures that if the last vote belongs to a multi-vote Geschäft,
// all votes from that Geschäft are fetched and included in the group.
func (c *Client) GroupAbstimmungenByGeschaeft(votes []Abstimmung) ([][]Abstimmung, error) {
	if len(votes) == 0 {
		return nil, nil
	}

	// First, ensure the last vote's group is complete if needed
	completeVotes, err := c.ensureCompleteGroupIfNeeded(votes)
	if err != nil {
		return nil, err
	}

	// Build a map keyed by "GeschaeftGrNr|SitzungDatum"
	groupMap := make(map[string][]Abstimmung)

	for _, vote := range completeVotes {
		// Extract just the date part (YYYY-MM-DD) from SitzungDatum
		date := vote.SitzungDatum
		if len(date) > 10 {
			date = date[:10]
		}

		key := vote.GeschaeftGrNr + "|" + date
		groupMap[key] = append(groupMap[key], vote)
	}

	// Sort votes within each group by SEQ (ascending) to preserve Sitzung chronological order
	for key := range groupMap {
		votes := groupMap[key]
		sort.Slice(votes, func(i, j int) bool {
			seqI, _ := strconv.Atoi(votes[i].SEQ)
			seqJ, _ := strconv.Atoi(votes[j].SEQ)
			return seqI < seqJ
		})
		groupMap[key] = votes
	}

	// Convert map to slice of groups, preserving the order of first occurrence
	seen := make(map[string]bool)
	var groups [][]Abstimmung

	for _, vote := range completeVotes {
		date := vote.SitzungDatum
		if len(date) > 10 {
			date = date[:10]
		}
		key := vote.GeschaeftGrNr + "|" + date

		if !seen[key] {
			seen[key] = true
			groups = append(groups, groupMap[key])
		}
	}

	// Sort groups by their minimum SEQ value (oldest first)
	sort.Slice(groups, func(i, j int) bool {
		minSeqI, _ := strconv.Atoi(groups[i][0].SEQ) // First vote in group is already sorted to be oldest
		minSeqJ, _ := strconv.Atoi(groups[j][0].SEQ)
		return minSeqI < minSeqJ
	})

	return groups, nil
}

// ensureCompleteGroupIfNeeded checks if the last vote belongs to a multi-vote Geschäft
// and fetches any missing votes from that Geschäft/date if needed
func (c *Client) ensureCompleteGroupIfNeeded(votes []Abstimmung) ([]Abstimmung, error) {
	if len(votes) == 0 {
		return votes, nil
	}

	lastVote := votes[len(votes)-1]

	// Fetch all votes for this Traktandum using the TraktandumGuid
	allVotesForTraktandum, err := c.FetchAbstimmungenForTraktandum(lastVote.TraktandumGuid)
	if err != nil {
		// If we can't fetch, just return what we have
		return votes, nil
	}

	// Count how many votes we already have for this Traktandum
	existingIDs := make(map[string]bool)
	for _, v := range votes {
		existingIDs[v.OBJGUID] = true
	}

	// Append missing votes from the same Traktandum
	for _, v := range allVotesForTraktandum {
		if !existingIDs[v.OBJGUID] {
			votes = append(votes, v)
		}
	}

	return votes, nil
}
