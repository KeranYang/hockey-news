package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)


// Sitemaps to watch: camps, leagues, and programs are the content types on this site.
var ohaSitemaps = []string{
	"https://oakvillehockeyacademy.com/camp-sitemap.xml",
	"https://oakvillehockeyacademy.com/league-sitemap.xml",
	"https://oakvillehockeyacademy.com/program-sitemap.xml",
}

type OakvilleHockeyAcademyScraper struct{}

func (s *OakvilleHockeyAcademyScraper) Name() string    { return "Oakville Hockey Academy" }
func (s *OakvilleHockeyAcademyScraper) SiteURL() string { return "https://oakvillehockeyacademy.com" }

// curlBin returns the path to curl, preferring the user's PATH (which may
// point to an OpenSSL-based curl with a TLS fingerprint that passes Cloudflare)
// over system hardcoded paths.
func curlBin() (string, error) {
	if bin, err := exec.LookPath("curl"); err == nil {
		return bin, nil
	}
	for _, path := range []string{"/usr/bin/curl", "/usr/local/bin/curl"} {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("curl not found")
}

// curlFetch uses the system curl to bypass Cloudflare TLS fingerprinting.
// oakvillehockeyacademy.com blocks Go's HTTP client via TLS fingerprinting.
// We intentionally omit -A so curl uses its own default User-Agent — sending a
// Chrome UA with curl's TLS fingerprint triggers Cloudflare's bot detection.
// -f makes curl exit non-zero on HTTP 4xx/5xx so we get a real error instead of HTML.
func curlFetch(url string, extraHeaders ...string) ([]byte, error) {
	bin, err := curlBin()
	if err != nil {
		return nil, fmt.Errorf("curl not found: %w", err)
	}

	args := []string{"-sLf", "--max-time", "30"}
	for _, h := range extraHeaders {
		args = append(args, "-H", h)
	}
	args = append(args, url)

	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("curl HTTP error for %s (exit %d)", url, exitErr.ExitCode())
		}
		return nil, fmt.Errorf("curl %s: %w", url, err)
	}
	return out, nil
}

func (s *OakvilleHockeyAcademyScraper) FetchArticles(client *http.Client, since time.Time) ([]Article, error) {
	// Collect URLs updated since the cutoff across all sitemaps
	type candidate struct{ url, lastmod string }
	var candidates []candidate

	for i, sitemapURL := range ohaSitemaps {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		entries, err := s.parseSitemap(client, sitemapURL)
		if err != nil {
			log.Printf("Warning: skipping %s: %v", sitemapURL, err)
			continue
		}
		for _, e := range entries {
			candidates = append(candidates, candidate{e.url, e.lastmod})
		}
	}

	var articles []Article
	for _, c := range candidates {
		lastmod, err := time.Parse(time.RFC3339, c.lastmod)
		if err != nil {
			continue
		}
		if lastmod.Before(since) {
			continue
		}

		// Skip index pages (e.g. /camps/, /leagues/, /programs/)
		trimmed := strings.TrimSuffix(c.url, "/")
		parts := strings.Split(trimmed, "/")
		if len(parts) <= 4 { // e.g. https://oakvillehockeyacademy.com/camps/
			continue
		}

		title, summary, err := s.fetchMeta(client, c.url)
		if err != nil || title == "" {
			continue
		}

		if !isRelevant(title) {
			continue
		}

		articles = append(articles, Article{
			Source:  s.Name(),
			Title:   title,
			URL:     c.url,
			Date:    lastmod,
			Summary: summary,
		})
	}

	return articles, nil
}

// parseSitemap fetches an XML sitemap and returns (url, lastmod) pairs.
func (s *OakvilleHockeyAcademyScraper) parseSitemap(_ *http.Client, sitemapURL string) ([]struct{ url, lastmod string }, error) {
	body, err := curlFetch(sitemapURL, "Accept: application/xml, text/xml, */*")
	if err != nil {
		return nil, err
	}

	type urlEntry struct {
		Loc     string `xml:"loc"`
		Lastmod string `xml:"lastmod"`
	}
	type urlset struct {
		URLs []urlEntry `xml:"url"`
	}

	var result urlset
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return nil, err
	}

	out := make([]struct{ url, lastmod string }, 0, len(result.URLs))
	for _, u := range result.URLs {
		out = append(out, struct{ url, lastmod string }{u.Loc, u.Lastmod})
	}
	return out, nil
}

// fetchMeta fetches og:title and og:description from an individual page.
func (s *OakvilleHockeyAcademyScraper) fetchMeta(_ *http.Client, pageURL string) (title, summary string, err error) {
	body, err := curlFetch(pageURL)
	if err != nil {
		return "", "", err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}

	title, _ = doc.Find(`meta[property="og:title"]`).Attr("content")
	// Strip site name suffix (e.g. " - Oakville Hockey Academy")
	if i := strings.LastIndex(title, " - "); i != -1 {
		title = title[:i]
	}
	title = strings.TrimSpace(title)

	summary, _ = doc.Find(`meta[property="og:description"]`).Attr("content")
	if summary == "" {
		summary, _ = doc.Find(`meta[name="description"]`).Attr("content")
	}

	return title, strings.TrimSpace(summary), nil
}
