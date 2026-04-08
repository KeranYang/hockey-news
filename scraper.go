package main

import (
	"net/http"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Article holds the data for a single news article from any site.
type Article struct {
	Source  string
	Title   string
	URL     string
	Date    time.Time
	Summary string
}

// Scraper is the interface every site must implement.
// To add a new site, create a new file (e.g. somesite.go) with a struct
// that implements these three methods, then register it in main.go.
type Scraper interface {
	// Name returns a human-readable site name used in the email digest.
	Name() string
	// SiteURL returns the site's homepage URL, used in the email footer.
	SiteURL() string
	// FetchArticles returns relevant articles published since the given time.
	FetchArticles(client *http.Client, since time.Time) ([]Article, error)
}

// fetchWithUA performs a GET request with a browser-like User-Agent to avoid 403s.
func fetchWithUA(client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return client.Do(req)
}

// isRelevant returns true for articles relevant to a parent of a U8 (age 7) player:
//   - Explicitly mentions U7 or U8
//   - Mentions House League (the level U8 plays at)
//   - Mentions registration
//   - General org-wide news (no older age group mentioned)
func isRelevant(title string) bool {
	upper := strings.ToUpper(title)

	if strings.Contains(upper, "U8") || strings.Contains(upper, "U7") {
		return true
	}
	if strings.Contains(upper, "HOUSE LEAGUE") {
		return true
	}
	if strings.Contains(upper, "REGISTR") {
		return true
	}

	// Exclude articles clearly targeting older age groups
	for _, ag := range []string{"U9", "U10", "U11", "U12", "U13", "U14", "U15", "U16", "U17", "U18", "AAA", "ADVANCED", "REP "} {
		if strings.Contains(upper, ag) {
			return false
		}
	}

	// No specific age group → general org news
	return true
}
