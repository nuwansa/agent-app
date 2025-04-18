package tools

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// CityForecast holds the forecast information for a single city.
type CityForecast struct {
	City        string `json:"city"`
	MaxTempC    string `json:"max_temperature_c"`
	MinTempC    string `json:"min_temperature_c"`
	MaxRH       string `json:"max_relative_humidity_percent"`
	MinRH       string `json:"min_relative_humidity_percent"`
	WeatherDesc string `json:"weather_description"`
}

// ForecastData wraps both the summary text and the list of city forecasts.
type ForecastData struct {
	Summary       string         `json:"weather_summary"`
	CityForecasts []CityForecast `json:"city_forecasts"`
}
type ReadWeatherInput struct {
}

func GetWeather(ctx context.Context, input ReadWeatherInput) (ForecastData, error) {
	resp, err := http.Get("https://meteo.gov.lk/index.php?lang=en")
	if err != nil {
		log.Fatalf("Error fetching the page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Failed to load page. Status: %d %s\n", resp.StatusCode, resp.Status)
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Error parsing HTML: %v", err)
	}

	// We'll store the final results here.
	var data ForecastData

	// Approach:
	//  1) Find *all* div.article_anywhere
	//  2) For each, see if it contains the phrase "Weather Forecast for Main Cities"
	//     If yes, that's likely the container we want.
	//  3) Parse the summary above that table and parse the table rows.

	var found bool

	doc.Find("div.article_anywhere").EachWithBreak(func(i int, container *goquery.Selection) bool {
		// Does this container's text mention "Weather Forecast for Main Cities"?
		// If yes, we treat this container as the "City Forecasts" block.
		if strings.Contains(container.Text(), "Weather Forecast for Main Cities") {
			// Mark we found the correct container
			found = true

			// ---- EXTRACT SUMMARY TEXT ----
			var summaryBuilder strings.Builder
			foundTable := false

			container.Children().Each(func(ci int, s *goquery.Selection) {
				nodeName := goquery.NodeName(s)
				if nodeName == "table" {
					foundTable = true
					return
				}
				if nodeName == "p" && !foundTable {
					text := strings.TrimSpace(s.Text())
					if text != "" {
						if summaryBuilder.Len() > 0 {
							summaryBuilder.WriteString("\n\n") // blank line between paragraphs
						}
						summaryBuilder.WriteString(text)
					}
				}
			})

			data.Summary = summaryBuilder.String()

			// ---- EXTRACT CITY FORECAST TABLE ROWS ----
			var forecasts []CityForecast

			// The city forecast table has style="border: none; border-collapse: collapse;"
			// We'll search inside this container for that table, then parse rows.

			container.Find(`table[style="border: none; border-collapse: collapse;"] tbody tr`).Each(func(i int, tr *goquery.Selection) {
				cells := tr.Find("td")
				if cells.Length() < 6 {
					return
				}
				// Check if it's a header row
				if i == 0 {
					headerText := strings.ToLower(strings.TrimSpace(cells.Eq(0).Text()))
					if strings.Contains(headerText, "city") {
						return
					}
				}

				city := strings.TrimSpace(cells.Eq(0).Text())
				maxTempC := strings.TrimSpace(cells.Eq(1).Text())
				minTempC := strings.TrimSpace(cells.Eq(2).Text())
				maxRH := strings.TrimSpace(cells.Eq(3).Text())
				minRH := strings.TrimSpace(cells.Eq(4).Text())
				weatherDesc := strings.TrimSpace(cells.Eq(5).Text())

				forecasts = append(forecasts, CityForecast{
					City:        city,
					MaxTempC:    maxTempC,
					MinTempC:    minTempC,
					MaxRH:       maxRH,
					MinRH:       minRH,
					WeatherDesc: weatherDesc,
				})
			})

			data.CityForecasts = forecasts

			// We can break the EachWithBreak by returning false from the callback
			return false
		}
		return true // keep searching
	})

	if !found {
		log.Println("Could not find the city forecast container. Page structure may have changed.")
	}

	return data, nil
	//// Output the results in JSON
	//jsonBytes, err := json.MarshalIndent(data, "", "  ")
	//if err != nil {
	//	log.Fatalf("Error marshaling JSON: %v", err)
	//}
	//fmt.Println(string(jsonBytes))
}
