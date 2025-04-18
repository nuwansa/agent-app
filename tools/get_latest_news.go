package tools

import (
	"context"
	"github.com/gocolly/colly/v2"
	"log"
	"strings"
)

// NewsItem represents the structure of a news item
type NewsItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	NewsDate    string `json:"news_date"`
	ImageURL    string `json:"image_url"`
	NewsURL     string `json:"news_url"`
}

type NewsInput struct {
}

func GetLatestNews(ctx context.Context, input NewsInput) ([]NewsItem, error) {
	// Initialize the collector
	c := colly.NewCollector(
		colly.AllowedDomains("www.adaderana.lk"),
	)

	// Slice to store news items
	var newsItems []NewsItem

	// Callback for each news item found
	c.OnHTML(".news-story", func(e *colly.HTMLElement) {
		// Extract title and remove duplicate parts
		title := e.ChildText("h2.hidden-xs a")
		title = cleanTitle(title)

		// Extract description
		description := e.ChildText("p")

		// Extract news date and remove '|' character
		newsDate := strings.TrimSpace(strings.Split(e.ChildText(".comments span"), "|")[1])

		// Extract image URL
		imageURL := e.ChildAttr(".thumb-image a img", "src")

		// Extract news URL
		newsURL := e.ChildAttr("h2.hidden-xs a", "href")

		// Store the news item in the slice
		newsItems = append(newsItems, NewsItem{
			Title:       title,
			Description: description,
			NewsDate:    newsDate,
			ImageURL:    imageURL,
			NewsURL:     newsURL,
		})
	})

	// On request error
	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	// Start scraping
	err := c.Visit("https://www.adaderana.lk/hot-news/")
	if err != nil {
		log.Fatal(err)
	}
	return newsItems, nil
}

// cleanTitle removes duplicate trailing parts in the title
func cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	words := strings.Fields(title)
	// Detect duplicate sequence from the end
	for i := 1; i <= len(words)/2; i++ {
		if strings.Join(words[len(words)-i:], " ") == strings.Join(words[len(words)-2*i:len(words)-i], " ") {
			return strings.Join(words[:len(words)-i], " ")
		}
	}
	return title
}
