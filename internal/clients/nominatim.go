package clients

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// fetchCapitalCoordinates looks up the latitude and longitude of a capital city
// using the OpenStreetMap Nominatim API, filtered by ISO country code.
func fetchCapitalCoordinates(isoCode, city string) (lat, lng float64, err error) {
	osmURL := strings.NewReplacer(
		utility.CapitalPlaceholder, city,
		utility.ISOCodePlaceholder, strings.ToLower(isoCode),
	).Replace(utility.OSMURL)

	req, err := http.NewRequest(http.MethodGet, osmURL, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("creating OSM request: %w", err)
	}
	req.Header.Set(utility.HeaderUserAgent, utility.UserAgentValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("fetching OSM: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("OSM returned %d: %s", resp.StatusCode, string(body))
	}

	var osmResults []models.OSMResponse
	if err := json.NewDecoder(resp.Body).Decode(&osmResults); err != nil {
		return 0, 0, fmt.Errorf("decoding OSM: %w", err)
	}
	if len(osmResults) == 0 {
		return 0, 0, fmt.Errorf("no OSM results for capital %q", city)
	}

	return osmResults[0].CapLat, osmResults[0].CapLng, nil
}
