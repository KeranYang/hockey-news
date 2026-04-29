package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

var anthropicClient *anthropic.Client

var (
	reHead   = regexp.MustCompile(`(?i)<head[\s\S]*?</head\s*>`)
	reScript = regexp.MustCompile(`(?i)<script[\s\S]*?</script\s*>`)
	reStyle  = regexp.MustCompile(`(?i)<style[\s\S]*?</style\s*>`)
)

const maxHTMLBytes = 80_000

func stripNonContent(html string) string {
	html = reHead.ReplaceAllString(html, "")
	html = reScript.ReplaceAllString(html, "")
	html = reStyle.ReplaceAllString(html, "")
	if len(html) > maxHTMLBytes {
		html = html[:maxHTMLBytes]
	}
	return html
}

type rawArticle struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Date    string `json:"date"`
	Summary string `json:"summary"`
}

var dateFmts = []string{
	"2006-01-02",
	"January 2, 2006",
	"Jan 2, 2006",
	"Jan 02, 2006",
}

// extractArticles passes page HTML to Claude, which extracts and filters articles.
// Relevance filtering (U8 Oakville hockey parent) is baked into the prompt.
func extractArticles(client *http.Client, html []byte, baseURL, sourceName string, since time.Time) ([]Article, error) {
	cleaned := stripNonContent(string(html))

	prompt := fmt.Sprintf(`Extract articles or programs from this hockey website page.
Source: %s
Base URL: %s
Return only items relevant to a parent of a 7-year-old U8 house league hockey player in Oakville, Ontario (e.g. registrations, tryouts, schedules, ice time, camps, programs, leagues, news).
Return only items published or updated on or after %s. If an item has no visible date, include it with "date": "".

Return a JSON array. Each object must have:
- "title": item title
- "url": full absolute URL — copy href attribute values exactly; prepend base URL for relative paths (e.g. /foo -> %s/foo)
- "date": YYYY-MM-DD if a date is visible on the page, otherwise ""
- "summary": 1–2 sentence description

Return ONLY the JSON array, no other text. If nothing matches, return [].

HTML:
%s`, sourceName, baseURL, since.Format("2006-01-02"), baseURL, cleaned)

	msg, err := anthropicClient.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     "claude-haiku-4-5",
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM extraction: %w", err)
	}

	var text string
	for _, block := range msg.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			text = strings.TrimSpace(tb.Text)
			break
		}
	}

	// Strip markdown code fences if present.
	if after, ok := strings.CutPrefix(text, "```json"); ok {
		text = after
	} else if after, ok := strings.CutPrefix(text, "```"); ok {
		text = after
	}
	if idx := strings.LastIndex(text, "```"); idx >= 0 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)

	var raw []rawArticle
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("parsing LLM response: %w", err)
	}

	var articles []Article
	for _, r := range raw {
		if r.Title == "" || r.URL == "" {
			continue
		}
		var date time.Time
		for _, f := range dateFmts {
			if d, err := time.Parse(f, r.Date); err == nil {
				date = d
				break
			}
		}
		if date.IsZero() {
			date = time.Now().Truncate(24 * time.Hour)
		}
		if !validateURL(client, r.URL) {
			log.Printf("Warning: dropping %q — URL returns 404: %s", r.Title, r.URL)
			continue
		}
		articles = append(articles, Article{
			Source:  sourceName,
			Title:   r.Title,
			URL:     r.URL,
			Date:    date,
			Summary: r.Summary,
		})
	}
	return articles, nil
}

// validateURL returns false only when the URL is syntactically invalid or explicitly 404s.
// Non-404 HTTP errors and network errors are treated as "URL likely exists."
func validateURL(client *http.Client, rawURL string) bool {
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return true // network/TLS error — trust the URL
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusMethodNotAllowed {
		// Server doesn't support HEAD; fall back to GET.
		req2, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return true
		}
		req2.Header.Set("User-Agent", userAgent)
		resp2, err := client.Do(req2)
		if err != nil {
			return true
		}
		resp2.Body.Close()
		return resp2.StatusCode != http.StatusNotFound
	}
	return resp.StatusCode != http.StatusNotFound
}
