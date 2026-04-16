package clients

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var FetchWeatherInfoFunc = fetchWeatherInfo

// FetchWeatherInfo fetches weather data from the Open-Meteo API
// for the given latitude and longitude coordinates.
func fetchWeatherInfo(lat, lng float64) (models.OpenMeteoResponse, error) {
	meteoURL := strings.NewReplacer(
		utility.LatPlaceholder, strconv.FormatFloat(lat, 'f', utility.FloatPrecision, utility.FloatBitSize),
		utility.LngPlaceholder, strconv.FormatFloat(lng, 'f', utility.FloatPrecision, utility.FloatBitSize),
	).Replace(utility.OpenMeteoAPIURL)

	resp, err := http.Get(meteoURL)
	if err != nil {
		return models.OpenMeteoResponse{}, fmt.Errorf("fetching weather: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return models.OpenMeteoResponse{}, fmt.Errorf("weather api returned %d: %s", resp.StatusCode, string(body))
	}

	var info models.OpenMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return models.OpenMeteoResponse{}, fmt.Errorf("decoding weather: %w", err)
	}
	return info, nil
}
