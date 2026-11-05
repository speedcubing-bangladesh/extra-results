package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const wcaAPIBase = "https://www.worldcubeassociation.org/api/v0"

type WCIF struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Persons  []Person `json:"persons"`
	Events   []Event  `json:"events"`
	Schedule Schedule `json:"schedule"`
}

type Schedule struct {
	StartDate string `json:"startDate"`
}

type Person struct {
	RegistrantID *int          `json:"registrantId"`
	Name         string        `json:"name"`
	WCAID        string        `json:"wcaId"`
	Registration *Registration `json:"registration"`
}

type Registration struct {
	Status      string `json:"status"`
	IsCompeting bool   `json:"isCompeting"`
}

type Event struct {
	ID     string  `json:"id"`
	Rounds []Round `json:"rounds"`
}

type Round struct {
	ID      string   `json:"id"`
	Results []Result `json:"results"`
}

type Result struct {
	PersonID int  `json:"personId"`
	Ranking  *int `json:"ranking"`
	Best     int  `json:"best"`
	Average  int  `json:"average"`
}

func (p Person) isCompetitor() bool {
	return p.RegistrantID != nil &&
		p.Registration != nil &&
		p.Registration.Status == "accepted" &&
		p.Registration.IsCompeting
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func getJSON(endpoint string, notFound string, v any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "extra-results")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatalf("failed to close the resp body. %v", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return errors.New(notFound)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("WCA API returned %s: %s", resp.Status, body)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func fetchWCIF(compID string) (*WCIF, error) {
	endpoint := fmt.Sprintf("%s/competitions/%s/wcif/", wcaAPIBase, url.PathEscape(compID))
	var w WCIF
	if err := getJSON(endpoint, fmt.Sprintf("competition %q not found (or not publicly visible)", compID), &w); err != nil {
		return nil, fmt.Errorf("fetching WCIF: %w", err)
	}
	return &w, nil
}

type PersonCompetition struct {
	ID        string `json:"id"`
	StartDate string `json:"start_date"`
}

func fetchPersonCompetitions(wcaID string) ([]PersonCompetition, error) {
	endpoint := fmt.Sprintf("%s/persons/%s/competitions", wcaAPIBase, url.PathEscape(wcaID))
	var comps []PersonCompetition
	if err := getJSON(endpoint, fmt.Sprintf("person %s not found", wcaID), &comps); err != nil {
		return nil, fmt.Errorf("fetching competitions of %s: %w", wcaID, err)
	}
	return comps, nil
}
