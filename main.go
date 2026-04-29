package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/joho/godotenv"
)

// ── Sites ────────────────────────────────────────────────────────────────────
// To add a new site, create a new file implementing the Scraper interface
// and append an instance here.
var scrapers = []Scraper{
	&OakvilleRangersScraper{},
	&CanlanOakvilleScraper{},
	&OakvilleHockeyAcademyScraper{},
}

// ── Config ───────────────────────────────────────────────────────────────────

type EmailConfig struct {
	SMTPHost string
	SMTPPort string
	User     string
	Password string
	From     string
	To       []string // comma-separated in .env: EMAIL_TO=a@x.com,b@x.com
}

func loadConfig() EmailConfig {
	// In CI, variables are injected via secrets so a missing .env is fine.
	// Locally, require the file so misconfiguration is caught early.
	if err := godotenv.Load(); err != nil && os.Getenv("CI") == "" {
		log.Fatal("Error loading .env file: ", err)
	}
	return EmailConfig{
		SMTPHost: "smtp.gmail.com",
		SMTPPort: "587",
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_USER"),
		To:       splitTrimmed(os.Getenv("EMAIL_TO"), ","),
	}
}

func splitTrimmed(s, sep string) []string {
	parts := strings.Split(s, sep)
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

// ── Email ────────────────────────────────────────────────────────────────────

var alertTmpl = template.Must(template.New("alert").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    body { font-family: Arial, sans-serif; color: #333; max-width: 700px; margin: 0 auto; padding: 20px; }
    h1 { color: #00508A; border-bottom: 3px solid #C22033; padding-bottom: 10px; }
    .article { margin-bottom: 24px; border-left: 4px solid #00508A; padding-left: 14px; }
    .article h3 { margin: 0 0 4px 0; font-size: 16px; }
    .article h3 a { color: #00508A; text-decoration: none; }
    .article h3 a:hover { text-decoration: underline; }
    .article .meta { color: #888; font-size: 13px; margin-bottom: 6px; }
    .article p { margin: 0; line-height: 1.6; }
    .footer { margin-top: 40px; font-size: 12px; color: #aaa; border-top: 1px solid #eee; padding-top: 12px; }
  </style>
</head>
<body>
  <h1>What's new at the rink</h1>
  <p>
    These just went up in the last hour across the Oakville hockey sites — figured you'd want to know sooner rather than later.
    Lucas probably just wants to know if there's ice time, but here we are.
  </p>

  {{range .Articles}}
  <div class="article">
    <h3><a href="{{.URL}}">{{.Title}}</a></h3>
    <div class="meta">{{.Source}} &middot; {{.Date.Format "January 2, 2006"}}</div>
    {{if .Summary}}<p>{{.Summary}}</p>{{end}}
  </div>
  {{end}}

  <div class="footer">
    HockeyNews &mdash; built by Keran Yang to keep Lucas on the ice and off the waitlist. &copy; 2026.
  </div>
</body>
</html>
`))

func buildAlertEmail(articles []Article) (string, error) {
	data := struct{ Articles []Article }{Articles: articles}
	var buf bytes.Buffer
	if err := alertTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func sendEmail(cfg EmailConfig, subject, htmlBody string) error {
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.SMTPHost)
	header := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		cfg.From, strings.Join(cfg.To, ", "), subject,
	)
	addr := cfg.SMTPHost + ":" + cfg.SMTPPort
	return smtp.SendMail(addr, auth, cfg.From, cfg.To, []byte(header+htmlBody))
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	cfg := loadConfig()
	ac := anthropic.NewClient()
	anthropicClient = &ac
	httpClient = newHTTPClient()
	since := time.Now().AddDate(0, 0, -7).Truncate(24 * time.Hour)

	seen := loadSeen()

	log.Printf("Fetching articles since %s", since.Format("2006-01-02"))

	var newArticles []Article
	var failed []string
	for _, s := range scrapers {
		log.Printf("Scraping %s...", s.Name())
		articles, err := fetchArticles(s, since)
		if err != nil {
			log.Printf("Error: %s: %v", s.Name(), err)
			failed = append(failed, s.Name())
		}
		for _, a := range articles {
			if seen[a.URL] {
				continue
			}
			seen[a.URL] = true
			newArticles = append(newArticles, a)
		}
	}

	saveSeen(seen)

	if len(failed) > 0 {
		log.Fatalf("Scraping failed for: %s", strings.Join(failed, ", "))
	}

	if len(newArticles) == 0 {
		log.Println("No new relevant articles found.")
		return
	}

	log.Printf("Found %d new relevant article(s) — sending alert", len(newArticles))

	htmlBody, err := buildAlertEmail(newArticles)
	if err != nil {
		log.Fatalf("Failed to build email: %v", err)
	}

	subject := fmt.Sprintf("just posted: %d new hockey article(s) in the last hour", len(newArticles))
	if len(newArticles) == 1 {
		subject = fmt.Sprintf("just posted: %s", newArticles[0].Title)
	}

	log.Printf("Sending alert to %s", strings.Join(cfg.To, ", "))
	if err := sendEmail(cfg, subject, htmlBody); err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	log.Println("Alert sent successfully!")
}
