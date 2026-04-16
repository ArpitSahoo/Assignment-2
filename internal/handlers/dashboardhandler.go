package handlers

import (
	"assignment-2/internal/clients"
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

// getRegistrationByID retrieves a stored registration from Firestore by its document ID.
// Returns the populated StoredRegistration with its ID set, or an error if not found.
func getRegistrationByID(ctx context.Context, client *firestore.Client, id string) (models.StoredRegistration, error) {
	doc, err := client.Collection(utility.RegistrationsCollection).Doc(id).Get(ctx)
	if err != nil {
		return models.StoredRegistration{}, err
	}

	var reg models.StoredRegistration
	if err := doc.DataTo(&reg); err != nil {
		return models.StoredRegistration{}, err
	}

	reg.ID = doc.Ref.ID
	return reg, nil
}

// DashboardHandler handles GET requests to the dashboard endpoint.
// It fetches registration data, then aggregates country info, weather, air quality,
// and exchange rates based on the features enabled in the registration.
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

	country, err := clients.FetchCountryInfoFunc(reg.IsoCode)
	if err != nil {
		log.Printf("country fetch error for reg %s: %v", registrationID, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch country information")
		return
	}

	if len(country.Latlng) < utility.MinCountryCoordinates {
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

	if err := populateExchangeRateFeature(&resp, reg, country); err != nil {
		log.Printf("exchange rate fetch failed for reg %s: %v", registrationID, err)
	}

	h.triggerLifecycleWebhooks(ctx, "INVOKE", reg.IsoCode)
	h.triggerThresholdWebhooks(ctx, reg.IsoCode, resp)

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode dashboard response: %v", err)
	}
}

// populateCountryFeatures fills in country-related fields on the dashboard response
// based on which features are enabled in the registration.
func populateCountryFeatures(resp *models.DashboardResponse, reg models.StoredRegistration, country models.RestCountryResponse) {
	lat := country.Latlng[0]
	lng := country.Latlng[1]

	if reg.Features.Capital && len(country.Capital) > 0 {
		capital := country.Capital[0]
		resp.Features.Capital = &capital
	}

	if reg.Features.Coordinates {
		resp.Features.Coordinates = &models.Coordinates{
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

// populateWeatherFeatures fetches weather data and populates temperature and/or
// precipitation on the dashboard response, if those features are enabled in the registration.
func populateWeatherFeatures(resp *models.DashboardResponse, reg models.StoredRegistration, lat, lng float64) error {
	if !reg.Features.Temperature && !reg.Features.Precipitation {
		return nil
	}

	weather, err := clients.FetchWeatherInfoFunc(lat, lng)
	if err != nil {
		return err
	}

	if reg.Features.Temperature {
		temp := models.MeanValue(weather.Hourly.Temperature2M)
		resp.Features.Temperature = &temp
	}

	if reg.Features.Precipitation {
		precipitation := models.MeanValue(weather.Hourly.Precipitation)
		resp.Features.Precipitation = &precipitation
	}

	return nil
}

// populateAirQualityFeature fetches air quality data for the country's capital and
// populates PM2.5, PM10, and an air quality level string on the dashboard response.
// Sets all values to -1 with level "unknown" if no capital is available.
func populateAirQualityFeature(resp *models.DashboardResponse, reg models.StoredRegistration, country models.RestCountryResponse) error {
	if !reg.Features.AirQuality {
		return nil
	}

	if len(country.Capital) == 0 {
		resp.Features.AirQuality = &models.AirQuality{
			PM25:  utility.UnknownAirQualityValue,
			PM10:  utility.UnknownAirQualityValue,
			Level: utility.AirQualityUnknown,
		}
		return nil
	}

	pm10, pm25, err := clients.FetchAirQualityInfoFunc(country.ISOCode, country.Capital[0])
	if err != nil {
		return err
	}

	resp.Features.AirQuality = &models.AirQuality{
		PM25:  pm25,
		PM10:  pm10,
		Level: airQualityLevel(pm25),
	}
	return nil
}

// populateExchangeRateFeature fetches exchange rates for the target currencies specified
// in the registration, using the country's own currency as the base rate.
// Skips fetching entirely if no target currencies are registered.
func populateExchangeRateFeature(resp *models.DashboardResponse, reg models.StoredRegistration, country models.RestCountryResponse) error {
	if len(reg.Features.TargetCurrencies) == 0 {
		return nil
	}

	var baseCurrency string
	for code := range country.Currencies {
		baseCurrency = code
		break
	}

	currency, err := clients.FetchExchangeRateFunc(reg.Features.TargetCurrencies, baseCurrency)
	if err != nil {
		return err
	}

	resp.Features.TargetCurrencies = currency
	return nil
}

// airQualityLevel maps a PM2.5 concentration to a human-readable air quality
// Returns "unknown" if pm25 is negative.
func airQualityLevel(pm25 float64) string {
	switch {
	case pm25 <= utility.UnknownAirQualityValue:
		return utility.AirQualityUnknown
	case pm25 <= utility.Pm25GoodMax:
		return utility.AirQualityGood
	case pm25 <= utility.Pm25ModerateMax:
		return utility.AirQualityModerate
	case pm25 <= utility.Pm25SensitiveGroupsMax:
		return utility.AirQualitySensitiveGroups
	case pm25 <= utility.Pm25UnhealthyMax:
		return utility.AirQualityUnhealthy
	case pm25 <= utility.Pm25VeryUnhealthyMax:
		return utility.AirQualityVeryUnhealthy
	default:
		return utility.AirQualityHazardous
	}
}

// writeJSONError writes a JSON-encoded error response with the given HTTP status code
// and message to the response writer.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

// newDashboardResponse creates a new DashboardResponse initialised with the country
// name, ISO code, and the current time as the last retrieval timestamp.
func newDashboardResponse(country models.RestCountryResponse) models.DashboardResponse {
	return models.DashboardResponse{
		Country:       country.Name.Common,
		ISOCode:       country.ISOCode,
		Features:      models.DashboardFeatures{},
		LastRetrieval: time.Now().Format("20060102 15:04"),
	}
}
