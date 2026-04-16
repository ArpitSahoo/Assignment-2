package external

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"assignment-2/internal/utility"
)

// OpenAQAPIKey for external package
var OpenAQAPIKey = os.Getenv("OPENAQ_API_KEY")

func FetchCountryInfo(isoCode string) (utility.RestCountryResponse, error) {
	resp, err := http.Get(utility.RestCountriesApiUrl + isoCode)
	if err != nil {
		return utility.RestCountryResponse{}, fmt.Errorf("fetching country: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return utility.RestCountryResponse{}, fmt.Errorf("countries api returned %d: %s", resp.StatusCode, string(body))
	}

	var results []utility.RestCountryResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return utility.RestCountryResponse{}, fmt.Errorf("decoding country: %w", err)
	}
	if len(results) == 0 {
		return utility.RestCountryResponse{}, fmt.Errorf("no country found for code %s", isoCode)
	}
	return results[0], nil
}

func FetchWeatherInfo(lat, lng float64) (utility.OpenMeteoResponse, error) {
	meteoURL := strings.NewReplacer(
		"{lat}", strconv.FormatFloat(lat, 'f', 6, 64),
		"{lng}", strconv.FormatFloat(lng, 'f', 6, 64),
	).Replace(utility.OpenMeteoApiUrlBase)

	resp, err := http.Get(meteoURL)
	if err != nil {
		return utility.OpenMeteoResponse{}, fmt.Errorf("fetching weather: %w", err)
	}
	defer closeBody(resp.Body)

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

func FetchExchangeRate(targetCur []string, cur string) (map[string]float64, error) {
	if len(targetCur) == 0 {
		return nil, nil
	}
	if cur == "" {
		return nil, fmt.Errorf("base currency is empty")
	}

	curApiURL := strings.NewReplacer(
		"{cur}", strings.ToUpper(cur),
	).Replace(utility.CurrencyAPIURL)

	resp, err := http.Get(curApiURL)
	if err != nil {
		return nil, fmt.Errorf("fetching exchange rates: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("exchange rate api returned %d: %s", resp.StatusCode, string(body))
	}

	var exchangeRate utility.ExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&exchangeRate); err != nil {
		return nil, fmt.Errorf("decoding exchange rates: %w", err)
	}

	result := make(map[string]float64)
	for _, cur := range targetCur {
		if rate, ok := exchangeRate.Rates[strings.ToUpper(cur)]; ok {
			result[strings.ToUpper(cur)] = rate
		}
	}

	return result, nil
}

func FetchAirQualityInfo(isoCode, cap string) (pm10, pm25 float64, err error) {
	if OpenAQAPIKey == "" {
		return -1, -1, fmt.Errorf("missing OPENAQ_API_KEY")
	}

	city := url.QueryEscape(strings.TrimSpace(cap))
	if city == "" {
		return -1, -1, fmt.Errorf("missing capital")
	}

	lat, lng, err := FetchCapitalCoordinates(isoCode, city)
	if err != nil {
		return -1, -1, err
	}

	aq, err := FetchOpenAQLocations(isoCode, lat, lng)
	if err != nil {
		return -1, -1, err
	}

	return calculatePMAverages(aq)
}

func FetchCapitalCoordinates(isoCode, city string) (lat, lng float64, err error) {
	osmURL := strings.NewReplacer(
		"{cap}", city,
		"{isoCode}", strings.ToLower(isoCode),
	).Replace(utility.OSMURLBase)

	req, err := http.NewRequest(http.MethodGet, osmURL, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("creating OSM request: %w", err)
	}
	req.Header.Set("User-Agent", "assignment-2/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("fetching OSM: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("OSM returned %d: %s", resp.StatusCode, string(body))
	}

	var osmResults []utility.OSMResponse
	if err := json.NewDecoder(resp.Body).Decode(&osmResults); err != nil {
		return 0, 0, fmt.Errorf("decoding OSM: %w", err)
	}
	if len(osmResults) == 0 {
		return 0, 0, fmt.Errorf("no OSM results for capital %q", city)
	}

	return osmResults[0].CapLat, osmResults[0].CapLng, nil
}

func FetchOpenAQLocations(isoCode string, lat, lng float64) (utility.OpenAQResponse, error) {
	openAQURL := strings.NewReplacer(
		"{lat}", strconv.FormatFloat(lat, 'f', 6, 64),
		"{lng}", strconv.FormatFloat(lng, 'f', 6, 64),
		"{isoCode}", strings.ToUpper(isoCode),
	).Replace(utility.OpenAQURLBase)

	req, err := http.NewRequest(http.MethodGet, openAQURL, nil)
	if err != nil {
		return utility.OpenAQResponse{}, fmt.Errorf("creating OpenAQ request: %w", err)
	}
	req.Header.Set("X-API-Key", OpenAQAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return utility.OpenAQResponse{}, fmt.Errorf("fetching OpenAQ locations: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return utility.OpenAQResponse{}, fmt.Errorf("OpenAQ locations returned %d: %s", resp.StatusCode, string(body))
	}

	var aq utility.OpenAQResponse
	if err := json.NewDecoder(resp.Body).Decode(&aq); err != nil {
		return utility.OpenAQResponse{}, fmt.Errorf("decoding OpenAQ locations: %w", err)
	}

	return aq, nil
}

func FetchLatestPMValues(locationID, pm10SensorID, pm25SensorID int) (pm10, pm25 float64, err error) {
	latestURL := strings.NewReplacer(
		"{id}", strconv.Itoa(locationID),
	).Replace(utility.OpenAQLatestURL)

	req, err := http.NewRequest(http.MethodGet, latestURL, nil)
	if err != nil {
		return -1, -1, fmt.Errorf("creating OpenAQ latest request: %w", err)
	}
	req.Header.Set("X-API-Key", OpenAQAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, -1, fmt.Errorf("fetching OpenAQ latest: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return -1, -1, fmt.Errorf("OpenAQ latest returned %d: %s", resp.StatusCode, string(body))
	}

	var latest utility.OpenAQLatestResponse
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

func calculatePMAverages(aq utility.OpenAQResponse) (pm10, pm25 float64, err error) {
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

		locValuePM10, locValuePM25, err := FetchLatestPMValues(location.ID, locPM10, locPM25)
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

	pm10 = calculateAQMean(pm10Values)
	pm25 = calculateAQMean(pm25Values)

	return pm10, pm25, nil
}

func calculateAQMean(values []float64) float64 {
	if len(values) == 0 {
		return -1
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// closeBody helper
func closeBody(body io.ReadCloser) {
	if err := body.Close(); err != nil {
		log.Printf("error closing body: %v", err)
	}
}
