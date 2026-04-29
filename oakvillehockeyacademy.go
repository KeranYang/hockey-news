package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type OakvilleHockeyAcademyScraper struct{}

func (s *OakvilleHockeyAcademyScraper) Name() string    { return "Oakville Hockey Academy" }
func (s *OakvilleHockeyAcademyScraper) SiteURL() string { return "https://oakvillehockeyacademy.com" }
func (s *OakvilleHockeyAcademyScraper) PageURLs() []string {
	return []string{
		"https://oakvillehockeyacademy.com/camps/",
		"https://oakvillehockeyacademy.com/leagues/",
		"https://oakvillehockeyacademy.com/programs/",
	}
}

// FetchPage uses curl to bypass Cloudflare TLS fingerprinting, which blocks Go's HTTP client.
func (s *OakvilleHockeyAcademyScraper) FetchPage(url string) ([]byte, error) {
	return curlFetch(url)
}

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
