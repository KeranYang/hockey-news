package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type OakvilleRangersScraper struct{}

func (s *OakvilleRangersScraper) Name() string    { return "Oakville Rangers" }
func (s *OakvilleRangersScraper) SiteURL() string { return "https://oakvillerangers.ca" }

func (s *OakvilleRangersScraper) FetchArticles(client *http.Client, since time.Time) ([]Article, error) {
	articlesURL := s.SiteURL() + "/Articles/"

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
	doc.Find(".carousel-item").Each(func(_ int, sel *goquery.Selection) {
		dateStr := strings.TrimSpace(sel.Find(".submitted-date").Text())
		titleEl := sel.Find(".title a")
		title := strings.TrimSpace(titleEl.Text())
		href, exists := titleEl.Attr("href")

		if !exists || dateStr == "" || title == "" {
			return
		}

		date, err := time.Parse("Jan 02, 2006", dateStr)
		if err != nil {
			return
		}

		if date.Before(since) || !isRelevant(title) {
			return
		}

		article := Article{
			Source: s.Name(),
			Title:  title,
			URL:    s.SiteURL() + href,
			Date:   date,
		}

		// Fetch the og:description from the article page as the summary
		if err := s.fetchSummary(client, &article); err != nil {
			// Non-fatal: article is still included without a summary
		}

		articles = append(articles, article)
	})

	return articles, nil
}

func (s *OakvilleRangersScraper) fetchSummary(client *http.Client, article *Article) error {
	resp, err := fetchWithUA(client, article.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	if desc, exists := doc.Find(`meta[property="og:description"]`).Attr("content"); exists && desc != "" {
		article.Summary = desc
		return nil
	}

	article.Summary = strings.TrimSpace(doc.Find(".article-details").Text())
	return nil
}
