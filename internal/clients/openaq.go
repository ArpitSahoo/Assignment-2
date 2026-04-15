package clients

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var FetchAirQualityInfoFunc = fetchAirQualityInfo

// openAQAPIKey is the API key for the OpenAQ air quality service,
// loaded from the OPENAQ_API_KEY environment variable.
var openAQAPIKey = os.Getenv("OPENAQ_API_KEY")

// FetchAirQualityInfo fetches PM10 and PM25 air quality averages for a country's capital.
// It first resolves the capital's coordinates via OSM, then queries OpenAQ for nearby sensors.
// Returns -1 for both values if data is unavailable.
func fetchAirQualityInfo(isoCode, cap string) (pm10, pm25 float64, err error) {
	if openAQAPIKey == "" {
		return -1, -1, fmt.Errorf("missing OPENAQ_API_KEY")
	}

	city := url.QueryEscape(strings.TrimSpace(cap))
	if city == "" {
		return -1, -1, fmt.Errorf("missing capital")
	}

	lat, lng, err := fetchCapitalCoordinates(isoCode, city)
	if err != nil {
		return -1, -1, err
	}

	aq, err := fetchOpenAQLocations(isoCode, lat, lng)
	if err != nil {
		return -1, -1, err
	}

	return calculatePMAverages(aq)
}

// fetchOpenAQLocations queries the OpenAQ API for air quality monitoring locations
// near the given coordinates within the specified country.
func fetchOpenAQLocations(isoCode string, lat, lng float64) (models.OpenAQResponse, error) {
	openAQURL := strings.NewReplacer(
		"{lat}", strconv.FormatFloat(lat, 'f', 6, 64),
		"{lng}", strconv.FormatFloat(lng, 'f', 6, 64),
		"{isoCode}", strings.ToUpper(isoCode),
	).Replace(utility.OpenAQURLBase)

	req, err := http.NewRequest(http.MethodGet, openAQURL, nil)
	if err != nil {
		return models.OpenAQResponse{}, fmt.Errorf("creating OpenAQ request: %w", err)
	}
	req.Header.Set("X-API-Key", openAQAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.OpenAQResponse{}, fmt.Errorf("fetching OpenAQ locations: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return models.OpenAQResponse{}, fmt.Errorf("OpenAQ locations returned %d: %s", resp.StatusCode, string(body))
	}

	var aq models.OpenAQResponse
	if err := json.NewDecoder(resp.Body).Decode(&aq); err != nil {
		return models.OpenAQResponse{}, fmt.Errorf("decoding OpenAQ locations: %w", err)
	}

	return aq, nil
}

// fetchLatestPMValues fetches the latest sensor readings for a given OpenAQ location
// and returns the most recent PM10 and PM25 values by matching against the provided sensor IDs.
// Returns -1 for any value whose sensor ID is not found in the response.
func fetchLatestPMValues(locationID, pm10SensorID, pm25SensorID int) (pm10, pm25 float64, err error) {
	latestURL := strings.NewReplacer(
		"{id}", strconv.Itoa(locationID),
	).Replace(utility.OpenAQLatestURL)

	req, err := http.NewRequest(http.MethodGet, latestURL, nil)
	if err != nil {
		return -1, -1, fmt.Errorf("creating OpenAQ latest request: %w", err)
	}
	req.Header.Set("X-API-Key", openAQAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, -1, fmt.Errorf("fetching OpenAQ latest: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return -1, -1, fmt.Errorf("OpenAQ latest returned %d: %s", resp.StatusCode, string(body))
	}

	var latest models.OpenAQLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return -1, -1, fmt.Errorf("decoding OpenAQ latest: %w", err)
	}

	pm10, pm25 = -1, -1
	for _, result := range latest.Results {
		switch result.SensorsID {
		case pm10SensorID:
			pm10 = result.Value
		case pm25SensorID:
			pm25 = result.Value
		}
	}

	return pm10, pm25, nil
}

// calculatePMAverages iterates over up to 5 OpenAQ locations, fetches their latest
// PM10 and PM25 sensor readings, and returns the mean of each across all locations.
// Returns -1 for either value if no valid readings are found.
func calculatePMAverages(aq models.OpenAQResponse) (pm10, pm25 float64, err error) {
	if len(aq.Results) == 0 {
		return -1, -1, nil
	}

	var pm10Values []float64
	var pm25Values []float64

	maxLocations := 5
	for i, location := range aq.Results {
		if i >= maxLocations {
			break
		}

		var locPM10, locPM25 int

		for _, sensor := range location.Sensors {
			switch sensor.Parameter.ID {
			case 1:
				locPM10 = sensor.ID
			case 2:
				locPM25 = sensor.ID
			}
		}

		locValuePM10, locValuePM25, err := fetchLatestPMValues(location.ID, locPM10, locPM25)
		if err != nil {
			log.Printf("fetching latest PM values failed for location %d: %v", location.ID, err)
			continue
		}

		if locValuePM10 >= 0 {
			pm10Values = append(pm10Values, locValuePM10)
		}
		if locValuePM25 >= 0 {
			pm25Values = append(pm25Values, locValuePM25)
		}
	}

	pm10 = models.MeanValue(pm10Values)
	pm25 = models.MeanValue(pm25Values)

	return pm10, pm25, nil
}
