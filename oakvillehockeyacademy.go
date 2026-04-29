package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var ohaPages = []string{
	"https://oakvillehockeyacademy.com/camps/",
	"https://oakvillehockeyacademy.com/leagues/",
	"https://oakvillehockeyacademy.com/programs/",
}

type OakvilleHockeyAcademyScraper struct{}

func (s *OakvilleHockeyAcademyScraper) Name() string    { return "Oakville Hockey Academy" }
func (s *OakvilleHockeyAcademyScraper) SiteURL() string { return "https://oakvillehockeyacademy.com" }

func (s *OakvilleHockeyAcademyScraper) FetchArticles(client *http.Client, since time.Time) ([]Article, error) {
	var all []Article
	for _, pageURL := range ohaPages {
		body, err := curlFetch(pageURL)
		if err != nil {
			log.Printf("Warning: skipping %s: %v", pageURL, err)
			continue
		}
		articles, err := extractArticles(client, body, s.SiteURL(), s.Name(), since)
		if err != nil {
			log.Printf("Warning: extracting from %s: %v", pageURL, err)
			continue
		}
		all = append(all, articles...)
	}
	return all, nil
}

// curlBin returns the path to curl, preferring the user's PATH.
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
