package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type OakvilleRangersScraper struct{}

func (s *OakvilleRangersScraper) Name() string    { return "Oakville Rangers" }
func (s *OakvilleRangersScraper) SiteURL() string { return "https://oakvillerangers.ca" }

func (s *OakvilleRangersScraper) FetchArticles(client *http.Client, since time.Time) ([]Article, error) {
	resp, err := fetchWithUA(client, s.SiteURL()+"/Articles/")
	if err != nil {
		return nil, fmt.Errorf("fetching articles page: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading articles page: %w", err)
	}
	return extractArticles(client, body, s.SiteURL(), s.Name(), since)
}
