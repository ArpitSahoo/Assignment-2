package handlers

import (
	"assignment-2/internal/utility"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDashboardHandler(t *testing.T) *Handler {
	t.Helper()

	clearFirestoreEmulator(t)

	return &Handler{
		Client: newTestFirestoreClient(t),
	}
}

func createDashboardThroughHandler(t *testing.T, h *Handler, body string) utility.DashboardResponse {
	t.Helper()
	created := createRegistrationThroughHandler(t, h, body)

	req := httptest.NewRequest(http.MethodGet, utility.DashboardPath+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()

	h.DashboardHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())

	var resp utility.DashboardResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp), "failed decoding dashboard response")
	return resp
}

func TestDashboardSuccess_TableDriven(t *testing.T) {
	origCountry := fetchCountryInfoFunc
	origWeather := fetchWeatherInfoFunc
	origAQ := fetchAirQualityInfoFunc
	origRates := fetchExchangeRateFunc
	defer func() {
		fetchCountryInfoFunc = origCountry
		fetchWeatherInfoFunc = origWeather
		fetchAirQualityInfoFunc = origAQ
		fetchExchangeRateFunc = origRates
	}()

	tests := []struct {
		name       string
		createBody string

		stubCountry utility.RestCountryResponse
		stubWeather utility.OpenMeteoResponse
		stubPM10    float64
		stubPM25    float64
		stubRates   map[string]float64

		wantBody utility.DashboardResponse
	}{
		{
			name: "all features enabled",
			createBody: `{
                "country": "Norway",
                "isoCode": "NO",
                "features": {
                	"temperature": true,
					"precipitation": true,
                    "airQuality": true,
                    "capital": true,
					"coordinates": true,
					"population": true,
					"area": true,
					"targetCurrencies": ["EUR", "USD", "SEK"]
                }
            }`,
			stubCountry: utility.RestCountryResponse{
				ISOCode:    "NO",
				Capital:    []string{"Oslo"},
				Latlng:     []float64{59.91, 10.75},
				Population: 5500000,
				Area:       385207,
				Currencies: map[string]struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				}{
					"NOK": {
						Name:   "Norwegian krone",
						Symbol: "kr",
					},
				},
				Name: struct {
					Common string `json:"common"`
				}{
					Common: "Norway",
				},
			},
			stubWeather: utility.OpenMeteoResponse{
				Hourly: struct {
					Temperature2M []float64 `json:"temperature_2m"`
					Precipitation []float64 `json:"precipitation"`
				}{
					Temperature2M: []float64{10, 12},
					Precipitation: []float64{2, 4},
				},
			},
			stubPM10: 20,
			stubPM25: 8,
			stubRates: map[string]float64{
				"EUR": 0.88,
				"USD": 1.15,
				"SEK": 10.4,
			},
			wantBody: utility.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: utility.DashboardFeatures{
					Capital: func() *string {
						s := "Oslo"
						return &s
					}(),
					Temperature: func() *float64 {
						v := 11.0
						return &v
					}(),
					Precipitation: func() *float64 {
						v := 3.0
						return &v
					}(),
					AirQuality: &utility.AirQuality{
						PM25:  8,
						PM10:  20,
						Level: "Good",
					},
					Coordinates: &utility.Coordinates{
						Latitude:  59.91,
						Longitude: 10.75,
					},
					Population: func() *int64 {
						v := int64(5500000)
						return &v
					}(),
					Area: func() *float64 {
						v := 385207.0
						return &v
					}(),
					TargetCurrencies: map[string]float64{
						"EUR": 0.88,
						"USD": 1.15,
						"SEK": 10.4,
					},
				},
			},
		},
		{
			name: "only temperature and precipitation enabled",
			createBody: `{
                "country": "Norway",
                "isoCode": "NO",
                "features": {
                	"temperature": true,
					"precipitation": true,
                    "airQuality": false,
                    "capital": false,
					"coordinates": false,
					"population": false,
					"area": false,
					"targetCurrencies": []
                }
            }`,
			stubCountry: utility.RestCountryResponse{
				ISOCode:    "NO",
				Capital:    []string{"Oslo"},
				Latlng:     []float64{59.91, 10.75},
				Population: 5500000,
				Area:       385207,
				Currencies: map[string]struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				}{
					"NOK": {
						Name:   "Norwegian krone",
						Symbol: "kr",
					},
				},
				Name: struct {
					Common string `json:"common"`
				}{
					Common: "Norway",
				},
			},
			stubWeather: utility.OpenMeteoResponse{
				Hourly: struct {
					Temperature2M []float64 `json:"temperature_2m"`
					Precipitation []float64 `json:"precipitation"`
				}{
					Temperature2M: []float64{10.0, 12.0},
					Precipitation: []float64{2.0, 4.0},
				},
			},
			stubPM10:  0,
			stubPM25:  0,
			stubRates: map[string]float64{},
			wantBody: utility.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: utility.DashboardFeatures{
					Temperature: func() *float64 {
						v := 11.0
						return &v
					}(),
					Precipitation: func() *float64 {
						v := 3.0
						return &v
					}(),
				},
			},
		},
		{
			name: "only capital, coordinates, population and area enabled",
			createBody: `{
                "country": "Norway",
                "isoCode": "NO",
                "features": {
                	"temperature": false,
					"precipitation": false,
                    "airQuality": false,
                    "capital": true,
					"coordinates": true,
					"population": true,
					"area": true,
					"targetCurrencies": []
                }
            }`,
			stubCountry: utility.RestCountryResponse{
				ISOCode:    "NO",
				Capital:    []string{"Oslo"},
				Latlng:     []float64{59.91, 10.75},
				Population: 5500000,
				Area:       385207,
				Currencies: map[string]struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				}{
					"NOK": {
						Name:   "Norwegian krone",
						Symbol: "kr",
					},
				},
				Name: struct {
					Common string `json:"common"`
				}{
					Common: "Norway",
				},
			},
			stubWeather: utility.OpenMeteoResponse{
				Hourly: struct {
					Temperature2M []float64 `json:"temperature_2m"`
					Precipitation []float64 `json:"precipitation"`
				}{
					Temperature2M: []float64{10.0, 12.0},
					Precipitation: []float64{2.0, 4.0},
				},
			},
			stubPM10:  0,
			stubPM25:  0,
			stubRates: map[string]float64{},
			wantBody: utility.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: utility.DashboardFeatures{
					Capital: func() *string {
						c := "Oslo"
						return &c
					}(),
					Coordinates: &utility.Coordinates{
						Latitude:  59.91,
						Longitude: 10.75,
					},
					Population: func() *int64 {
						v := int64(5500000)
						return &v
					}(),
					Area: func() *float64 {
						v := 385207.0
						return &v
					}(),
				},
			},
		},
		{
			name: "air quality enabled with no capital available",
			createBody: `{
        		"country": "Norway",
        		"isoCode": "NO",
        		"features": {
        			"temperature": false,
					"precipitation": false,
            		"airQuality": true,
            		"capital": false,
					"coordinates": false,
					"population": false,
					"area": false,
					"targetCurrencies": []
        		}
    		}`,
			stubCountry: utility.RestCountryResponse{
				ISOCode:    "NO",
				Capital:    []string{},
				Latlng:     []float64{59.91, 10.75},
				Population: 5500000,
				Area:       385207,
				Currencies: map[string]struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				}{
					"NOK": {
						Name:   "Norwegian krone",
						Symbol: "kr",
					},
				},
				Name: struct {
					Common string `json:"common"`
				}{
					Common: "Norway",
				},
			},
			stubWeather: utility.OpenMeteoResponse{},
			stubPM10:    0,
			stubPM25:    0,
			stubRates:   map[string]float64{},
			wantBody: utility.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: utility.DashboardFeatures{
					AirQuality: &utility.AirQuality{
						PM25:  -1,
						PM10:  -1,
						Level: "unknown",
					},
				},
			},
		},
		{
			name: "target currencies enabled only",
			createBody: `{
        		"country": "Norway",
        		"isoCode": "NO",
        		"features": {
            		"temperature": false,
            		"precipitation": false,
            		"airQuality": false,
            		"capital": false,
            		"coordinates": false,
            		"population": false,
            		"area": false,
            		"targetCurrencies": ["EUR", "USD", "SEK"]
        		}
    		}`,
			stubCountry: utility.RestCountryResponse{
				ISOCode:    "NO",
				Capital:    []string{"Oslo"},
				Latlng:     []float64{59.91, 10.75},
				Population: 5500000,
				Area:       385207,
				Currencies: map[string]struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				}{
					"NOK": {
						Name:   "Norwegian krone",
						Symbol: "kr",
					},
				},
				Name: struct {
					Common string `json:"common"`
				}{
					Common: "Norway",
				},
			},
			stubWeather: utility.OpenMeteoResponse{},
			stubPM10:    0,
			stubPM25:    0,
			stubRates: map[string]float64{
				"EUR": 0.88,
				"USD": 1.15,
				"SEK": 10.4,
			},
			wantBody: utility.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: utility.DashboardFeatures{
					TargetCurrencies: map[string]float64{
						"EUR": 0.88,
						"USD": 1.15,
						"SEK": 10.4,
					},
				},
			},
		},
		{
			name: "no features enabled",
			createBody: `{
        		"country": "Norway",
        		"isoCode": "NO",
        		"features": {
            		"temperature": false,
            		"precipitation": false,
            		"airQuality": false,
            		"capital": false,
            		"coordinates": false,
            		"population": false,
            		"area": false,
            		"targetCurrencies": []
        		}
    		}`,
			stubCountry: utility.RestCountryResponse{
				ISOCode:    "NO",
				Capital:    []string{"Oslo"},
				Latlng:     []float64{59.91, 10.75},
				Population: 5500000,
				Area:       385207,
				Currencies: map[string]struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				}{
					"NOK": {
						Name:   "Norwegian krone",
						Symbol: "kr",
					},
				},
				Name: struct {
					Common string `json:"common"`
				}{
					Common: "Norway",
				},
			},
			stubWeather: utility.OpenMeteoResponse{},
			stubPM10:    0,
			stubPM25:    0,
			stubRates: map[string]float64{
				"EUR": 0.88,
				"USD": 1.15,
				"SEK": 10.4,
			},
			wantBody: utility.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestDashboardHandler(t)

			fetchCountryInfoFunc = func(isoCode string) (utility.RestCountryResponse, error) {
				return tt.stubCountry, nil
			}

			fetchWeatherInfoFunc = func(lat, lng float64) (utility.OpenMeteoResponse, error) {
				return tt.stubWeather, nil
			}

			fetchAirQualityInfoFunc = func(isoCode, cap string) (float64, float64, error) {
				return tt.stubPM10, tt.stubPM25, nil
			}

			fetchExchangeRateFunc = func(targetCur []string, curr string) (map[string]float64, error) {
				return tt.stubRates, nil
			}

			got := createDashboardThroughHandler(t, h, tt.createBody)

			assert.Equal(t, tt.wantBody.Country, got.Country)
			assert.Equal(t, tt.wantBody.ISOCode, got.ISOCode)
			assert.NotEmpty(t, got.LastRetrieval)

			assert.Equal(t, tt.wantBody.Features.Capital, got.Features.Capital)
			assert.Equal(t, tt.wantBody.Features.Temperature, got.Features.Temperature)
			assert.Equal(t, tt.wantBody.Features.Precipitation, got.Features.Precipitation)
			assert.Equal(t, tt.wantBody.Features.AirQuality, got.Features.AirQuality)
			assert.Equal(t, tt.wantBody.Features.Coordinates, got.Features.Coordinates)
			assert.Equal(t, tt.wantBody.Features.Population, got.Features.Population)
			assert.Equal(t, tt.wantBody.Features.Area, got.Features.Area)
			assert.Equal(t, tt.wantBody.Features.TargetCurrencies, got.Features.TargetCurrencies)
		})
	}
}
