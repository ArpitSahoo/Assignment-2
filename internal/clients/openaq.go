package clients

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var FetchAirQualityInfoFunc = FetchAirQualityInfo

// openAQAPIKey is the API key for the OpenAQ air quality service,
// loaded from the OPENAQ_API_KEY environment variable.
var openAQAPIKey = os.Getenv(utility.OpenAQAPIKey)

// FetchAirQualityInfo fetches PM10 and PM25 air quality averages for a country's capital.
// It resolves the capital's coordinates via OSM, queries OpenAQ for nearby sensors,
// and then computes the mean PM values across up to a small number of locations. uses context
// so callers can cancel/time out the entire operation.
// Returns -1 for both values if data is unavailable.
func FetchAirQualityInfo(ctx context.Context, isoCode, cap string) (pm10, pm25 float64, err error) {
	if openAQAPIKey == "" {
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, fmt.Errorf("missing OPENAQ_API_KEY")
	}

	city := strings.TrimSpace(cap)
	if city == "" {
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, fmt.Errorf("missing capital")
	}
	// Ensure city is URL-escaped where needed, FetchCapitalCoordinates expects a raw city string.
	lat, lng, err := FetchCapitalCoordinates(ctx, isoCode, city)
	if err != nil {
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, err
	}

	aq, err := fetchOpenAQLocations(ctx, lat, lng)
	if err != nil {
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, err
	}

	return calculatePMAverages(ctx, aq)
}

// fetchOpenAQLocations queries the OpenAQ API for air quality monitoring locations
// near the given coordinates within the specified country. Uses ctx on the HTTP requests.
func fetchOpenAQLocations(ctx context.Context, lat, lng float64) (models.OpenAQResponse, error) {
	openAQURL := strings.NewReplacer(
		utility.LatPlaceholder, strconv.FormatFloat(lat, 'f', 6, 64),
		utility.LngPlaceholder, strconv.FormatFloat(lng, 'f', 6, 64),
	).Replace(utility.OpenAQURL)

	// Use the context on the request so cancellations/timeouts are respected.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openAQURL, nil)
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
// and returns the most recent PM10 and PM2.5 values by matching against the provided sensor IDs.
// Uses ctx on the HTTP request.
func fetchLatestPMValues(ctx context.Context, locationID, pm10SensorID, pm25SensorID int) (pm10, pm25 float64, err error) {
	latestURL := strings.NewReplacer(
		utility.OpenAQIDPlaceholder, strconv.Itoa(locationID),
	).Replace(utility.OpenAQLatestURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, fmt.Errorf("creating OpenAQ latest request: %w", err)
	}
	req.Header.Set("X-API-Key", openAQAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, fmt.Errorf("fetching OpenAQ latest: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, fmt.Errorf("OpenAQ latest returned %d: %s", resp.StatusCode, string(body))
	}

	var latest models.OpenAQLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, fmt.Errorf("decoding OpenAQ latest: %w", err)
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

// calculatePMAverages iterates over up to a small number of OpenAQ locations, fetches their latest
// PM10 and PM25 sensor readings (using ctx), and returns the mean of each across all locations.
// Returns -1 for either value if no valid readings are found.
func calculatePMAverages(ctx context.Context, aq models.OpenAQResponse) (pm10, pm25 float64, err error) {
	if len(aq.Results) == 0 {
		return utility.UnknownAirQualityValue, utility.UnknownAirQualityValue, nil
	}

	var pm10Values []float64
	var pm25Values []float64

	maxLocations := utility.MaxOpenAQLocations
	for i, location := range aq.Results {
		if i >= maxLocations {
			break
		}

		var locPM10, locPM25 int

		for _, sensor := range location.Sensors {
			switch sensor.Parameter.ID {
			case utility.Pm10ParameterID:
				locPM10 = sensor.ID
			case utility.Pm25ParameterID:
				locPM25 = sensor.ID
			}
		}

		locValuePM10, locValuePM25, err := fetchLatestPMValues(ctx, location.ID, locPM10, locPM25)
		if err != nil {
			// log and continue — don't fail the whole aggregation just because one location failed
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
