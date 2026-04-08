package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"time"

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
	To       string
}

func loadConfig() EmailConfig {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file: ", err)
	}
	return EmailConfig{
		SMTPHost: "smtp.gmail.com",
		SMTPPort: "587",
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_USER"),
		To:       os.Getenv("EMAIL_TO"),
	}
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

// ── Email ────────────────────────────────────────────────────────────────────

type SiteSection struct {
	Name     string
	SiteURL  string
	Articles []Article
}

var emailTmpl = template.Must(template.New("email").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    body { font-family: Arial, sans-serif; color: #333; max-width: 700px; margin: 0 auto; padding: 20px; }
    h1 { color: #00508A; border-bottom: 3px solid #C22033; padding-bottom: 10px; }
    h2.site { color: #C22033; margin-top: 32px; font-size: 20px; }
    .article { margin-bottom: 24px; border-left: 4px solid #00508A; padding-left: 14px; }
    .article h3 { margin: 0 0 4px 0; font-size: 16px; }
    .article h3 a { color: #00508A; text-decoration: none; }
    .article h3 a:hover { text-decoration: underline; }
    .article .meta { color: #888; font-size: 13px; margin-bottom: 6px; }
    .article p { margin: 0; line-height: 1.6; }
    .empty { color: #888; font-style: italic; }
    .footer { margin-top: 40px; font-size: 12px; color: #aaa; border-top: 1px solid #eee; padding-top: 12px; }
  </style>
</head>
<body>
  <h1>Hockey Weekly Digest — {{.WeekOf}}</h1>

  {{range .Sections}}
  <h2 class="site"><a href="{{.SiteURL}}">{{.Name}}</a></h2>
  {{if .Articles}}
    {{range .Articles}}
    <div class="article">
      <h3><a href="{{.URL}}">{{.Title}}</a></h3>
      <div class="meta">{{.Date.Format "January 2, 2006"}}</div>
      {{if .Summary}}<p>{{.Summary}}</p>{{end}}
    </div>
    {{end}}
  {{else}}
    <p class="empty">No relevant articles this week.</p>
  {{end}}
  {{end}}

  <div class="footer">
    This digest was automatically generated for a U8 hockey parent.
  </div>
</body>
</html>
`))

func buildEmailHTML(sections []SiteSection) (string, error) {
	data := struct {
		WeekOf   string
		Sections []SiteSection
	}{
		WeekOf:   time.Now().AddDate(0, 0, -7).Format("January 2") + " – " + time.Now().Format("January 2, 2006"),
		Sections: sections,
	}

	var buf bytes.Buffer
	if err := emailTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func sendEmail(cfg EmailConfig, subject, htmlBody string) error {
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.SMTPHost)
	header := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		cfg.From, cfg.To, subject,
	)
	addr := cfg.SMTPHost + ":" + cfg.SMTPPort
	return smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, []byte(header+htmlBody))
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	cfg := loadConfig()
	client := newHTTPClient()
	since := time.Now().AddDate(0, 0, -7).Truncate(24 * time.Hour)

	log.Printf("Fetching articles since %s", since.Format("2006-01-02"))

	var sections []SiteSection
	for _, s := range scrapers {
		log.Printf("Scraping %s...", s.Name())
		articles, err := s.FetchArticles(client, since)
		if err != nil {
			log.Printf("Warning: failed to scrape %s: %v", s.Name(), err)
			articles = nil
		}
		log.Printf("  → %d relevant articles", len(articles))
		sections = append(sections, SiteSection{
			Name:     s.Name(),
			SiteURL:  s.SiteURL(),
			Articles: articles,
		})
	}

	htmlBody, err := buildEmailHTML(sections)
	if err != nil {
		log.Fatalf("Failed to build email: %v", err)
	}

	subject := fmt.Sprintf("Hockey U8 Weekly Digest — %s", time.Now().Format("Jan 2, 2006"))
	log.Printf("Sending email to %s", cfg.To)
	if err := sendEmail(cfg, subject, htmlBody); err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	log.Println("Email sent successfully!")
}
