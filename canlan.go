package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type CanlanOakvilleScraper struct{}

func (s *CanlanOakvilleScraper) Name() string    { return "Canlan Sports Oakville" }
func (s *CanlanOakvilleScraper) SiteURL() string { return "https://www.canlansports.com/locations/ca/on/oakville/" }

func (s *CanlanOakvilleScraper) FetchArticles(client *http.Client, since time.Time) ([]Article, error) {
	newsURL := "https://www.canlansports.com/news/"

	resp, err := fetchWithUA(client, newsURL)
	if err != nil {
		return nil, fmt.Errorf("fetching news page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing news page: %w", err)
	}

	var articles []Article
	doc.Find(".card.card-news").Each(func(_ int, sel *goquery.Selection) {
		// URL is in a span's href attribute
		href, exists := sel.Find("span[href]").Attr("href")
		if !exists || href == "" {
			return
		}

		title := strings.TrimSpace(sel.Find("h3").Text())
		if title == "" {
			return
		}

		// Date format: "February 4, 2026." (trailing period)
		dateStr := strings.TrimSuffix(strings.TrimSpace(sel.Find("p.content-date").Text()), ".")
		date, err := time.Parse("January 2, 2006", dateStr)
		if err != nil {
			return
		}

		if date.Before(since) {
			return
		}

		// Summary is the first <p> that isn't the date
		summary := ""
		sel.Find(".content-news-slider p").Each(func(_ int, p *goquery.Selection) {
			if !p.HasClass("content-date") && summary == "" {
				summary = strings.TrimSpace(p.Text())
			}
		})

		if !isRelevant(title, summary) {
			return
		}

		articles = append(articles, Article{
			Source:  s.Name(),
			Title:   title,
			URL:     href,
			Date:    date,
			Summary: summary,
		})
	})

	return articles, nil
}
