package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type CanlanOakvilleScraper struct{}

func (s *CanlanOakvilleScraper) Name() string    { return "Canlan Sports Oakville" }
func (s *CanlanOakvilleScraper) SiteURL() string { return "https://www.canlansports.com" }

func (s *CanlanOakvilleScraper) FetchArticles(client *http.Client, since time.Time) ([]Article, error) {
	resp, err := fetchWithUA(client, "https://www.canlansports.com/news/")
	if err != nil {
		return nil, fmt.Errorf("fetching news page: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading news page: %w", err)
	}
	return extractArticles(client, body, s.SiteURL(), s.Name(), since)
}
