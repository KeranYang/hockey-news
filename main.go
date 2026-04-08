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

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
)

const (
	baseURL    = "https://oakvillerangers.ca"
	articlesURL = baseURL + "/Articles/"
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type Article struct {
	Title   string
	URL     string
	Date    time.Time
	Summary string
}

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

func fetchWithUA(client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return client.Do(req)
}

// isRelevant returns true for articles that a parent of a U8 player should care about:
//   - Explicitly about U7 or U8
//   - About House League (the level U8 plays at)
//   - General org-wide news (no specific older age group mentioned)
func isRelevant(title string) bool {
	upper := strings.ToUpper(title)

	// Always include U7/U8 specific news
	if strings.Contains(upper, "U8") || strings.Contains(upper, "U7") {
		return true
	}

	// Include house league news
	if strings.Contains(upper, "HOUSE LEAGUE") {
		return true
	}

	// Include registration news
	if strings.Contains(upper, "REGISTR") {
		return true
	}

	// Exclude articles clearly targeting older age groups
	olderGroups := []string{"U9", "U10", "U11", "U12", "U13", "U14", "U15", "U16", "U17", "U18", "AAA", "ADVANCED", "REP "}
	for _, ag := range olderGroups {
		if strings.Contains(upper, ag) {
			return false
		}
	}

	// No specific age group mentioned → general org news, include it
	return true
}

// fetchArticleList scrapes the /Articles/ listing and returns articles published since `since`.
func fetchArticleList(client *http.Client, since time.Time) ([]Article, error) {
	resp, err := fetchWithUA(client, articlesURL)
	if err != nil {
		return nil, fmt.Errorf("fetching articles page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing articles page: %w", err)
	}

	var articles []Article
	doc.Find(".carousel-item").Each(func(_ int, s *goquery.Selection) {
		dateStr := strings.TrimSpace(s.Find(".submitted-date").Text())
		titleEl := s.Find(".title a")
		title := strings.TrimSpace(titleEl.Text())
		href, exists := titleEl.Attr("href")

		if !exists || dateStr == "" || title == "" {
			return
		}

		date, err := time.Parse("Jan 02, 2006", dateStr)
		if err != nil {
			return
		}

		// Only keep articles from within the requested window
		if date.Before(since) {
			return
		}

		if !isRelevant(title) {
			return
		}

		articles = append(articles, Article{
			Title: title,
			URL:   baseURL + href,
			Date:  date,
		})
	})

	return articles, nil
}

// fetchArticleSummary fetches the og:description from an individual article page.
func fetchArticleSummary(client *http.Client, article *Article) error {
	resp, err := fetchWithUA(client, article.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	// Try og:description first (most concise summary)
	if desc, exists := doc.Find(`meta[property="og:description"]`).Attr("content"); exists && desc != "" {
		article.Summary = desc
		return nil
	}

	// Fall back to .article-details text
	article.Summary = strings.TrimSpace(doc.Find(".article-details").Text())
	return nil
}

var emailTmpl = template.Must(template.New("email").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    body { font-family: Arial, sans-serif; color: #333; max-width: 700px; margin: 0 auto; padding: 20px; }
    h1 { color: #00508A; border-bottom: 3px solid #C22033; padding-bottom: 10px; }
    .article { margin-bottom: 28px; border-left: 4px solid #00508A; padding-left: 14px; }
    .article h2 { margin: 0 0 4px 0; font-size: 18px; }
    .article h2 a { color: #00508A; text-decoration: none; }
    .article h2 a:hover { text-decoration: underline; }
    .article .meta { color: #888; font-size: 13px; margin-bottom: 8px; }
    .article p { margin: 0; line-height: 1.6; }
    .footer { margin-top: 40px; font-size: 12px; color: #aaa; border-top: 1px solid #eee; padding-top: 12px; }
  </style>
</head>
<body>
  <h1>Oakville Rangers U8 — Weekly News Digest</h1>
  <p>Here are the latest U8 updates from <a href="https://oakvillerangers.ca">oakvillerangers.ca</a> for the week of {{.WeekOf}}.</p>

  {{if .Articles}}
    {{range .Articles}}
    <div class="article">
      <h2><a href="{{.URL}}">{{.Title}}</a></h2>
      <div class="meta">{{.Date.Format "January 2, 2006"}}</div>
      {{if .Summary}}<p>{{.Summary}}</p>{{end}}
    </div>
    {{end}}
  {{else}}
    <p>No new articles were published this week.</p>
  {{end}}

  <div class="footer">
    This digest was automatically generated. View all news at
    <a href="https://oakvillerangers.ca/Articles/">oakvillerangers.ca/Articles/</a>.
  </div>
</body>
</html>
`))

func buildEmailHTML(articles []Article) (string, error) {
	data := struct {
		WeekOf   string
		Articles []Article
	}{
		WeekOf:   time.Now().AddDate(0, 0, -7).Format("January 2") + " – " + time.Now().Format("January 2, 2006"),
		Articles: articles,
	}

	var buf bytes.Buffer
	if err := emailTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func sendEmail(cfg EmailConfig, subject, htmlBody string) error {
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.SMTPHost)

	// Compose MIME message
	header := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		cfg.From, cfg.To, subject,
	)
	msg := []byte(header + htmlBody)

	addr := cfg.SMTPHost + ":" + cfg.SMTPPort
	return smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, msg)
}

func main() {
	cfg := loadConfig()
	client := newHTTPClient()

	since := time.Now().AddDate(0, 0, -7).Truncate(24 * time.Hour)
	log.Printf("Fetching articles since %s", since.Format("2006-01-02"))

	articles, err := fetchArticleList(client, since)
	if err != nil {
		log.Fatalf("Failed to fetch article list: %v", err)
	}
	log.Printf("Found %d articles in the past week", len(articles))

	for i := range articles {
		log.Printf("Fetching summary for: %s", articles[i].Title)
		if err := fetchArticleSummary(client, &articles[i]); err != nil {
			log.Printf("Warning: could not fetch summary for %q: %v", articles[i].Title, err)
		}
	}

	htmlBody, err := buildEmailHTML(articles)
	if err != nil {
		log.Fatalf("Failed to build email: %v", err)
	}

	subject := fmt.Sprintf("Oakville Rangers U8 Weekly Digest — %s", time.Now().Format("Jan 2, 2006"))
	log.Printf("Sending email to %s", cfg.To)
	if err := sendEmail(cfg, subject, htmlBody); err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	log.Println("Email sent successfully!")
}
