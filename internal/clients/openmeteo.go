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
	"strconv"
	"strings"
)

var FetchWeatherInfoFunc = FetchWeatherInfo

// FetchWeatherInfo fetches weather data from the Open-Meteo API for the given
// latitude and longitude coordinates. Uses ctx on the HTTP request.
func FetchWeatherInfo(ctx context.Context, lat, lng float64) (models.OpenMeteoResponse, error) {
	meteoURL := strings.NewReplacer(
		utility.LatPlaceholder, strconv.FormatFloat(lat, 'f', 6, 64),
		utility.LngPlaceholder, strconv.FormatFloat(lng, 'f', 6, 64),
	).Replace(utility.OpenMeteoAPIURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meteoURL, nil)
	if err != nil {
		return models.OpenMeteoResponse{}, fmt.Errorf("creating weather request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
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
