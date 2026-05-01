package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Article holds the data for a single news item from any site.
type Article struct {
	Source  string
	Title   string
	URL     string
	Date    time.Time
	Summary string
}

// httpClient is the shared HTTP client, initialised in main.
var httpClient *http.Client

// pageHashes maps page URLs to the SHA256 of their last-seen stripped HTML.
var pageHashes map[string]string

// Scraper is the interface every site must implement.
// PageURLs returns the listing/news pages to watch.
// FetchPage retrieves the raw HTML for one URL — embed HTTPFetcher for the default
// Go HTTP client, or override when special fetch behaviour is needed (e.g. curl).
//
// To add a new site: create a new .go file, implement this interface, embed
// HTTPFetcher if the standard client suffices, then register in main.go.
type Scraper interface {
	Name() string
	SiteURL() string
	PageURLs() []string
	FetchPage(url string) ([]byte, error)
}

// HTTPFetcher implements FetchPage using the standard Go HTTP client.
// Embed this in scrapers that do not need special fetch behaviour.
type HTTPFetcher struct{}

func (HTTPFetcher) FetchPage(url string) ([]byte, error) {
	resp, err := fetchWithUA(httpClient, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// fetchWithUA performs a GET with a browser-like User-Agent to avoid 403s.
func fetchWithUA(client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return client.Do(req)
}

// fetchArticles is the shared implementation used by all scrapers.
// It fetches each page from PageURLs, extracts articles via LLM, and returns
// combined results. Partial results are returned alongside any error so the
// caller can still process what succeeded.
func fetchArticles(s Scraper, since time.Time) ([]Article, error) {
	var all []Article
	var failedURLs []string
	for _, pageURL := range s.PageURLs() {
		body, err := s.FetchPage(pageURL)
		if err != nil {
			log.Printf("Error: %s: fetching %s: %v", s.Name(), pageURL, err)
			failedURLs = append(failedURLs, pageURL)
			continue
		}
		stripped := stripNonContent(string(body))
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(stripped)))
		if pageHashes[pageURL] == hash {
			log.Printf("%s: %s unchanged, skipping LLM", s.Name(), pageURL)
			continue
		}
		articles, err := extractArticles(httpClient, body, s.SiteURL(), s.Name(), since)
		if err != nil {
			log.Printf("Error: %s: extracting from %s: %v", s.Name(), pageURL, err)
			failedURLs = append(failedURLs, pageURL)
			continue
		}
		pageHashes[pageURL] = hash
		all = append(all, articles...)
	}
	if len(failedURLs) > 0 {
		return all, fmt.Errorf("failed to process: %s", strings.Join(failedURLs, ", "))
	}
	return all, nil
}
