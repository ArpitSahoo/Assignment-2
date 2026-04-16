package handlers

import (
	"assignment-2/internal/clients"
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"fmt"
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
		API:    &clients.RawClient{},
	}
}

func createDashboardThroughHandler(t *testing.T, h *Handler, body string) models.DashboardResponse {
	t.Helper()
	created := createRegistrationThroughHandler(t, h, body)

	req := httptest.NewRequest(http.MethodGet, utility.DashboardPath+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()

	h.DashboardHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())

	var resp models.DashboardResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp), "failed decoding dashboard response")
	return resp
}

func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()

	writeJSONError(w, http.StatusBadRequest, "missing registration id")

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var got map[string]string
	err := json.NewDecoder(w.Body).Decode(&got)
	require.NoError(t, err)

	assert.Equal(t, "missing registration id", got["error"])
}

func TestNewDashboardResponseSuccess(t *testing.T) {
	country := models.RestCountryResponse{
		ISOCode: "NO",
		Name: struct {
			Common string `json:"common"`
		}{
			Common: "Norway",
		},
	}

	got := newDashboardResponse(country)

	assert.Equal(t, "Norway", got.Country)
	assert.Equal(t, "NO", got.ISOCode)
	assert.NotEmpty(t, got.LastRetrieval)
	assert.Equal(t, models.DashboardFeatures{}, got.Features)
}

func TestPopulateCountryFeaturesSuccess(t *testing.T) {
	resp := models.DashboardResponse{}

	reg := models.StoredRegistration{
		Features: models.RegistrationFeatures{
			Capital:     true,
			Coordinates: true,
			Population:  true,
			Area:        true,
		},
	}

	country := models.RestCountryResponse{
		Capital:    []string{"Oslo"},
		Latlng:     []float64{59.91, 10.75},
		Population: 5500000,
		Area:       385207,
	}

	(&Handler{}).populateCountryFeatures(&resp, reg, country)

	require.NotNil(t, resp.Features.Capital)
	assert.Equal(t, "Oslo", *resp.Features.Capital)

	require.NotNil(t, resp.Features.Coordinates)
	assert.Equal(t, 59.91, resp.Features.Coordinates.Latitude)
	assert.Equal(t, 10.75, resp.Features.Coordinates.Longitude)

	require.NotNil(t, resp.Features.Population)
	assert.Equal(t, int64(5500000), *resp.Features.Population)

	require.NotNil(t, resp.Features.Area)
	assert.Equal(t, 385207.0, *resp.Features.Area)
}

func TestGetRegistrationByIDDataToFailure(t *testing.T) {
	h := newTestDashboardHandler(t)
	ctx := context.Background()

	id := "bad-doc-001"

	_, err := h.Client.Collection(utility.RegistrationsCollection).Doc(id).Set(ctx, map[string]any{
		"country":  "Norway",
		"isoCode":  123,
		"features": "not-an-object",
	})
	require.NoError(t, err)

	got, errGet := getRegistrationByID(ctx, h.Client, id)

	require.Error(t, errGet)
	assert.Equal(t, models.StoredRegistration{}, got)
}

func TestGetRegistrationByIDNotFoundError(t *testing.T) {
	h := newTestDashboardHandler(t)
	ctx := context.Background()

	got, err := getRegistrationByID(ctx, h.Client, "missing-id")

	require.Error(t, err)
	assert.Equal(t, models.StoredRegistration{}, got)
}

func TestDashboardSuccess_TableDriven(t *testing.T) {
	origCountry := clients.FetchCountryInfoFunc
	origWeather := clients.FetchWeatherInfoFunc
	origAQ := clients.FetchAirQualityInfoFunc
	origRates := clients.FetchExchangeRateFunc
	defer func() {
		clients.FetchCountryInfoFunc = origCountry
		clients.FetchWeatherInfoFunc = origWeather
		clients.FetchAirQualityInfoFunc = origAQ
		clients.FetchExchangeRateFunc = origRates
	}()

	tests := []struct {
		name       string
		createBody string

		stubCountry models.RestCountryResponse
		stubWeather models.OpenMeteoResponse
		stubPM10    float64
		stubPM25    float64
		stubRates   map[string]float64

		wantBody models.DashboardResponse
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
			stubCountry: models.RestCountryResponse{
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
			stubWeather: models.OpenMeteoResponse{
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
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: models.DashboardFeatures{
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
					AirQuality: &models.AirQuality{
						PM25:  8,
						PM10:  20,
						Level: "Good",
					},
					Coordinates: &models.Coordinates{
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
			stubCountry: models.RestCountryResponse{
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
			stubWeather: models.OpenMeteoResponse{
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
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: models.DashboardFeatures{
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
			stubCountry: models.RestCountryResponse{
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
			stubWeather: models.OpenMeteoResponse{
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
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: models.DashboardFeatures{
					Capital: func() *string {
						c := "Oslo"
						return &c
					}(),
					Coordinates: &models.Coordinates{
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
			stubCountry: models.RestCountryResponse{
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
			stubWeather: models.OpenMeteoResponse{},
			stubPM10:    0,
			stubPM25:    0,
			stubRates:   map[string]float64{},
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: models.DashboardFeatures{
					AirQuality: &models.AirQuality{
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
			stubCountry: models.RestCountryResponse{
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
			stubWeather: models.OpenMeteoResponse{},
			stubPM10:    0,
			stubPM25:    0,
			stubRates: map[string]float64{
				"EUR": 0.88,
				"USD": 1.15,
				"SEK": 10.4,
			},
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: models.DashboardFeatures{
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
			stubCountry: models.RestCountryResponse{
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
			stubWeather: models.OpenMeteoResponse{},
			stubPM10:    0,
			stubPM25:    0,
			stubRates: map[string]float64{
				"EUR": 0.88,
				"USD": 1.15,
				"SEK": 10.4,
			},
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestDashboardHandler(t)

			clients.FetchCountryInfoFunc = func(ctx context.Context, isoCode string) (models.RestCountryResponse, error) {
				_ = ctx // test stub ignores ctx; OK because handler provides a context
				return tt.stubCountry, nil
			}

			clients.FetchWeatherInfoFunc = func(ctx context.Context, lat, lng float64) (models.OpenMeteoResponse, error) {
				_ = ctx
				return tt.stubWeather, nil
			}

			clients.FetchAirQualityInfoFunc = func(ctx context.Context, isoCode, cap string) (float64, float64, error) {
				_ = ctx
				return tt.stubPM10, tt.stubPM25, nil
			}

			clients.FetchExchangeRateFunc = func(ctx context.Context, targetCur []string, curr string) (map[string]float64, error) {
				_ = ctx
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

func TestDashboardHandlerFailure_TableDriven(t *testing.T) {
	origCountry := clients.FetchCountryInfoFunc
	defer func() {
		clients.FetchCountryInfoFunc = origCountry
	}()

	tests := []struct {
		name           string
		pathID         string
		seedDoc        map[string]any
		stubCountry    models.RestCountryResponse
		stubCountryErr error
		createBody     string
		wantStatus     int
		wantErrorMsg   string
	}{
		{
			name:         "missing registration ID",
			pathID:       "",
			createBody:   "",
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "missing registration id",
		},
		{
			name:         "registration not found",
			pathID:       "missing-id-001",
			createBody:   "",
			wantStatus:   http.StatusNotFound,
			wantErrorMsg: "registration not found",
		},
		{
			name: "registration missing isoCode",
			seedDoc: map[string]any{
				"country": "Norway",
				"isoCode": "",
				"features": map[string]any{
					"temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
					"targetCurrencies": []string{},
				},
				"lastChange": "old",
			},
			wantStatus:   http.StatusInternalServerError,
			wantErrorMsg: "registration missing isoCode",
		},
		{
			name: "country fetch failure",
			createBody: `{
				"country": "Norway",
				"isoCode": "NO",
				"features": {
					"temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
					"targetCurrencies": []
				}
			}`,
			stubCountryErr: fmt.Errorf("country fetch failure"),
			wantStatus:     http.StatusInternalServerError,
			wantErrorMsg:   "failed to fetch country information",
		},
		{
			name: "country coordinates unavailable",
			createBody: `{
				"country": "Norway",
				"isoCode": "NO",
				"features": {
					"temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      true,
					"population":       false,
					"area":             false,
					"targetCurrencies": []
				}
			}`,
			stubCountry: models.RestCountryResponse{
				ISOCode: "NO",
				Capital: []string{"Oslo"},
				Latlng:  []float64{59.91},
				Name: struct {
					Common string `json:"common"`
				}{
					Common: "Norway",
				},
			},
			stubCountryErr: nil,
			wantStatus:     http.StatusInternalServerError,
			wantErrorMsg:   "country coordinates unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := newTestDashboardHandler(t)

			clients.FetchCountryInfoFunc = func(ctx context.Context, isoCode string) (models.RestCountryResponse, error) {
				_ = ctx // test stub ignores ctx; OK because handler provides a context
				return tt.stubCountry, tt.stubCountryErr
			}

			var req *http.Request

			if tt.seedDoc != nil {
				id := "seeded-id-001"
				_, err := h.Client.Collection(utility.RegistrationsCollection).Doc(id).Set(context.Background(), tt.seedDoc)
				require.NoError(t, err)

				req = httptest.NewRequest(http.MethodGet, utility.DashboardPath+id, nil)
				req.SetPathValue("id", id)

			} else if tt.createBody != "" {
				created := createRegistrationThroughHandler(t, h, tt.createBody)
				req = httptest.NewRequest(http.MethodGet, utility.DashboardPath+created.ID, nil)
				req.SetPathValue("id", created.ID)

			} else {
				req = httptest.NewRequest(http.MethodGet, utility.DashboardPath+"dummy-id", nil)
				req.SetPathValue("id", tt.pathID)
			}

			rr := httptest.NewRecorder()
			h.DashboardHandler(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)

			var got map[string]string
			err := json.NewDecoder(rr.Body).Decode(&got)
			require.NoError(t, err)

			assert.Equal(t, tt.wantErrorMsg, got["error"])
		})
	}
}

func TestDashboardHandlerPartialFailure_TableDriven(t *testing.T) {
	origCountry := clients.FetchCountryInfoFunc
	origWeather := clients.FetchWeatherInfoFunc
	origAQ := clients.FetchAirQualityInfoFunc
	origRates := clients.FetchExchangeRateFunc
	defer func() {
		clients.FetchCountryInfoFunc = origCountry
		clients.FetchWeatherInfoFunc = origWeather
		clients.FetchAirQualityInfoFunc = origAQ
		clients.FetchExchangeRateFunc = origRates
	}()

	tests := []struct {
		name           string
		createBody     string
		stubCountry    models.RestCountryResponse
		stubWeather    models.OpenMeteoResponse
		stubWeatherErr error
		stubPM10       float64
		stubPM25       float64
		stubAQErr      error
		stubRates      map[string]float64
		stubRatesErr   error
		wantBody       models.DashboardResponse
	}{
		{
			name: "weather fetch failure but response is still 200",
			createBody: `{
				"country": "Norway",
				"isoCode": "NO",
				"features": {
						"temperature":      true,
						"precipitation":    true,
						"airQuality":       false,
						"capital":          true,
						"coordinates":      false,
						"population":       false,
						"area":             false,
						"targetCurrencies": []
				}
			}`,
			stubCountry: models.RestCountryResponse{
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
			stubWeather:    models.OpenMeteoResponse{},
			stubWeatherErr: fmt.Errorf("weather fetch failure"),
			stubPM10:       0,
			stubPM25:       0,
			stubAQErr:      nil,
			stubRates:      map[string]float64{},
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: models.DashboardFeatures{
					Capital: func() *string {
						s := "Oslo"
						return &s
					}(),
				},
			},
		},
		{
			name: "air quality fetch failure but response is still 200",
			createBody: `{
				"country": "Norway",
				"isoCode": "NO",
				"features": {
						"temperature":      false,
						"precipitation":    false,
						"airQuality":       true,
						"capital":          true,
						"coordinates":      false,
						"population":       false,
						"area":             false,
						"targetCurrencies": []
				}
			}`,
			stubCountry: models.RestCountryResponse{
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
			stubWeather:    models.OpenMeteoResponse{},
			stubWeatherErr: nil,
			stubPM10:       0,
			stubPM25:       0,
			stubAQErr:      fmt.Errorf("air quality fetch failure"),
			stubRates:      map[string]float64{},
			stubRatesErr:   nil,
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
				Features: models.DashboardFeatures{
					Capital: func() *string {
						s := "Oslo"
						return &s
					}(),
				},
			},
		},
		{
			name: "exchange rate fetch failure but response is still 200",
			createBody: `{
				"country": "Norway",
				"isoCode": "NO",
				"features": {
						"temperature":      false,
						"precipitation":    false,
						"airQuality":       false,
						"capital":          false,
						"coordinates":      false,
						"population":       false,
						"area":             false,
						"targetCurrencies": ["EUR"]
				}
			}`,
			stubCountry: models.RestCountryResponse{
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
			stubWeather:    models.OpenMeteoResponse{},
			stubWeatherErr: nil,
			stubPM10:       0,
			stubPM25:       0,
			stubAQErr:      nil,
			stubRates:      map[string]float64{},
			stubRatesErr:   fmt.Errorf("rate fetch failure"),
			wantBody: models.DashboardResponse{
				Country: "Norway",
				ISOCode: "NO",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestDashboardHandler(t)

			clients.FetchCountryInfoFunc = func(ctx context.Context, isoCode string) (models.RestCountryResponse, error) {
				_ = ctx // test stub ignores ctx; OK because handler provides a context
				return tt.stubCountry, nil
			}

			clients.FetchWeatherInfoFunc = func(ctx context.Context, lat, lng float64) (models.OpenMeteoResponse, error) {
				_ = ctx
				return tt.stubWeather, tt.stubWeatherErr
			}

			clients.FetchAirQualityInfoFunc = func(ctx context.Context, isoCode, cap string) (float64, float64, error) {
				_ = ctx
				return tt.stubPM10, tt.stubPM25, tt.stubAQErr
			}

			clients.FetchExchangeRateFunc = func(ctx context.Context, targetCur []string, curr string) (map[string]float64, error) {
				_ = ctx
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

func TestAirQualityLevelSuccess_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		pm25 float64
		want string
	}{
		{
			name: "good lower bound",
			pm25: 0,
			want: "Good",
		},
		{
			name: "good upper bound",
			pm25: 12.0,
			want: "Good",
		},
		{
			name: "moderate",
			pm25: 20.0,
			want: "Moderate",
		},
		{
			name: "unhealthy for sensitive groups",
			pm25: 40.0,
			want: "Unhealthy for Sensitive Groups",
		},
		{
			name: "unhealthy",
			pm25: 100.0,
			want: "Unhealthy",
		},
		{
			name: "very unhealthy",
			pm25: 200.0,
			want: "Very Unhealthy",
		},
		{
			name: "hazardous",
			pm25: 300.0,
			want: "Hazardous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := airQualityLevel(tt.pm25)
			assert.Equal(t, tt.want, got)
		})
	}
}
