package handlers

import (
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

// TODO add comments
// TODO os.Getenv eller sånn .env fil ting

var OpenAQAPIKey = os.Getenv("OPENAQ_API_KEY")

func getRegistrationByID(ctx context.Context, client *firestore.Client, id string) (utility.StoredRegistration, error) {
	doc, err := client.Collection(utility.RegistrationsCollection).Doc(id).Get(ctx)
	if err != nil {
		return utility.StoredRegistration{}, err
	}

	var reg utility.StoredRegistration
	if err := doc.DataTo(&reg); err != nil {
		return utility.StoredRegistration{}, err
	}

	reg.ID = doc.Ref.ID
	return reg, nil
}

func (h *Handler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	registrationID := strings.TrimSpace(r.PathValue("id"))
	if registrationID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing registration id")
		return
	}

	ctx := firestoreContext(r)

	reg, err := getRegistrationByID(ctx, h.Client, registrationID)
	if err != nil {
		log.Printf("registration %s not found: %v", registrationID, err)
		writeJSONError(w, http.StatusNotFound, "registration not found")
		return
	}

	if reg.IsoCode == "" {
		writeJSONError(w, http.StatusInternalServerError, "registration missing isoCode")
		return
	}

	country, err := fetchCountryInfo(reg.IsoCode)
	if err != nil {
		log.Printf("country fetch error for reg %s: %v", registrationID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch country information")
		return
	}

	if len(country.Latlng) < 2 {
		writeJSONError(w, http.StatusInternalServerError, "country coordinates unavailable")
		return
	}

	resp := newDashboardResponse(country)

	populateCountryFeatures(&resp, reg, country)

	if err := populateWeatherFeatures(&resp, reg, country.Latlng[0], country.Latlng[1]); err != nil {
		log.Printf("weather fetch failed for reg %s: %v", registrationID, err)
	}

	if err := populateAirQualityFeature(&resp, reg, country); err != nil {
		log.Printf("air quality fetch error for reg %s: %v", registrationID, err)
	}

	//TODO add currency - List gjennom target og hent exchange rate - se Assignement 1?

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode dashboard response: %v", err)
	}
}

// TODO flytt til utils hør me gruppe?
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

func closeBody(body io.ReadCloser) {
	if err := body.Close(); err != nil {
		log.Printf("error closing body: %v", err)
	}
}

func newDashboardResponse(country utility.RestCountryInfo) utility.DashboardResponse {
	return utility.DashboardResponse{
		Country:       country.Name.Common,
		ISOCode:       country.ISOCode,
		Features:      utility.DashboardFeatures{},
		LastRetrieval: time.Now().Format("20060102 15:04"),
	}
}

func populateCountryFeatures(resp *utility.DashboardResponse, reg utility.StoredRegistration, country utility.RestCountryInfo) {
	lat := country.Latlng[0]
	lng := country.Latlng[1]

	if reg.Features.Capital && len(country.Capital) > 0 {
		capital := country.Capital[0]
		resp.Features.Capital = &capital
	}

	if reg.Features.Coordinates {
		resp.Features.Coordinates = &utility.Coordinates{
			Latitude:  lat,
			Longitude: lng,
		}
	}

	if reg.Features.Population {
		pop := int64(country.Population)
		resp.Features.Population = &pop
	}

	if reg.Features.Area {
		area := country.Area
		resp.Features.Area = &area
	}
}

func populateWeatherFeatures(resp *utility.DashboardResponse, reg utility.StoredRegistration, lat, lng float64) error {
	if !reg.Features.Temperature && !reg.Features.Precipitation {
		return nil
	}

	weather, err := fetchWeatherInfo(lat, lng)
	if err != nil {
		return err
	}

	if reg.Features.Temperature {
		temp := meanValue(weather.Hourly.Temperature2M)
		resp.Features.Temperature = &temp
	}

	if reg.Features.Precipitation {
		precipitation := meanValue(weather.Hourly.Precipitation)
		resp.Features.Precipitation = &precipitation
	}

	return nil
}

func populateAirQualityFeature(resp *utility.DashboardResponse, reg utility.StoredRegistration, country utility.RestCountryInfo) error {
	if !reg.Features.AirQuality {
		return nil
	}

	if len(country.Capital) == 0 {
		resp.Features.AirQuality = &utility.AirQuality{
			PM25:  -1,
			PM10:  -1,
			Level: "unknown",
		}
		return nil
	}

	pm10, pm25, err := fetchAirQualityInfo(country.ISOCode, country.Capital[0])
	if err != nil {
		return err
	}

	resp.Features.AirQuality = &utility.AirQuality{
		PM25:  pm25,
		PM10:  pm10,
		Level: airQualityLevel(pm25),
	}
	return nil
}

func meanValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func fetchCountryInfo(isoCode string) (utility.RestCountryInfo, error) {
	resp, err := http.Get(utility.RestCountriesApiUrl + isoCode)
	if err != nil {
		return utility.RestCountryInfo{}, fmt.Errorf("fetching country: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return utility.RestCountryInfo{}, fmt.Errorf("countries api returned %d: %s", resp.StatusCode, string(body))
	}

	var results []utility.RestCountryInfo
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return utility.RestCountryInfo{}, fmt.Errorf("decoding country: %w", err)
	}
	if len(results) == 0 {
		return utility.RestCountryInfo{}, fmt.Errorf("no country found for code %s", isoCode)
	}
	return results[0], nil
}

func fetchWeatherInfo(lat, lng float64) (utility.OpenMeteoInfo, error) {
	meteoURL := strings.NewReplacer(
		"{lat}", strconv.FormatFloat(lat, 'f', 6, 64),
		"{lng}", strconv.FormatFloat(lng, 'f', 6, 64),
	).Replace(utility.OpenMeteoApiUrlBase)

	resp, err := http.Get(meteoURL)
	if err != nil {
		return utility.OpenMeteoInfo{}, fmt.Errorf("fetching weather: %w", err)
	}
	defer closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return utility.OpenMeteoInfo{}, fmt.Errorf("weather api returned %d: %s", resp.StatusCode, string(body))
	}

	var info utility.OpenMeteoInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return utility.OpenMeteoInfo{}, fmt.Errorf("decoding weather: %w", err)
	}
	return info, nil
}

func fetchAirQualityInfo(isoCode, cap string) (pm10, pm25 float64, err error) {
	if OpenAQAPIKey == "" {
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

func fetchCapitalCoordinates(isoCode, city string) (lat, lng float64, err error) {
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

func fetchOpenAQLocations(isoCode string, lat, lng float64) (utility.OpenAQResponse, error) {
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

	pm10 = calculateAQMean(pm10Values)
	pm25 = calculateAQMean(pm25Values)

	return pm10, pm25, nil
}

func calculateAQMean(values []float64) float64 {
	if len(values) == 0 {
		return -1
	}
	return meanValue(values)
}

func fetchLatestPMValues(locationID, pm10SensorID, pm25SensorID int) (pm10, pm25 float64, err error) {
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

func airQualityLevel(pm25 float64) string {
	if pm25 < 0 {
		return "unknown"
	}

	switch {
	case pm25 <= 12.0:
		return "Good"
	case pm25 <= 35.4:
		return "Moderate"
	case pm25 <= 55.4:
		return "Unhealthy for Sensitive Groups"
	case pm25 <= 150.4:
		return "Unhealthy"
	case pm25 <= 250.4:
		return "Very Unhealthy"
	default:
		return "Hazardous"
	}
}
