package clients

import (
	"assignment-2/internal/utility"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// FetchWeatherInfo fetches weather data from the Open-Meteo API
// for the given latitude and longitude coordinates.
func FetchWeatherInfo(lat, lng float64) (utility.OpenMeteoResponse, error) {
	meteoURL := strings.NewReplacer(
		"{lat}", strconv.FormatFloat(lat, 'f', 6, 64),
		"{lng}", strconv.FormatFloat(lng, 'f', 6, 64),
	).Replace(utility.OpenMeteoApiUrlBase)

	resp, err := http.Get(meteoURL)
	if err != nil {
		return utility.OpenMeteoResponse{}, fmt.Errorf("fetching weather: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return utility.OpenMeteoResponse{}, fmt.Errorf("weather api returned %d: %s", resp.StatusCode, string(body))
	}

	var info utility.OpenMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return utility.OpenMeteoResponse{}, fmt.Errorf("decoding weather: %w", err)
	}
	return info, nil
}
