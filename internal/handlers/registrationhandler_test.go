package handlers

import (
	"assignment-2/internal/utility"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRegistrationHandler(t *testing.T) *Handler {
	t.Helper()

	clearFirestoreEmulator(t)

	return &Handler{
		Client: newTestFirestoreClient(t),
	}
}

func createRegistrationThroughHandler(t *testing.T, h *Handler, body string) utility.RegistrationResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, utility.RegistrationPath, bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.addRegistration(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())

	var resp utility.RegistrationResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp), "failed to decode registration response")
	assert.NotEmpty(t, resp.ID, "expected registration ID in response")

	return resp
}

func TestBuildReplaceCurrencyUpdateNil(t *testing.T) {
	got := buildReplaceCurrencyUpdate(nil)
	assert.Nil(t, got)
}

func TestBuildAddCurrencyUpdateNil(t *testing.T) {
	got := buildAddCurrencyUpdate(nil)
	assert.Nil(t, got)
}

func TestBuildAddCurrencyUpdateEmptySlice(t *testing.T) {
	var currencies []string
	got := buildAddCurrencyUpdate(&currencies)
	assert.Nil(t, got)
}

func TestBuildRemoveCurrencyUpdateNil(t *testing.T) {
	got := buildRemoveCurrencyUpdate(nil)
	assert.Nil(t, got)
}

func TestBuildRemoveCurrencyUpdateEmptySlice(t *testing.T) {
	var currencies []string
	got := buildRemoveCurrencyUpdate(&currencies)
	assert.Nil(t, got)
}

func TestBuildCurrencyPatchUpdateNilFeatures(t *testing.T) {
	got := buildCurrencyPatchUpdate(nil)
	assert.Nil(t, got)
}

func TestBuildCurrencyPatchUpdateEmptyFeatures(t *testing.T) {
	got := buildCurrencyPatchUpdate(&utility.RegistrationPatchFeatures{})
	assert.Nil(t, got)
}

func TestAddRegistrationWithPartialUpdateRegistrationSuccess_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		createBody string
		patchBody  string
		wantStatus int
		wantDoc    map[string]any
	}{
		{
			name: "add registration and update registration by replacing target currencies",
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
			patchBody: `{
            	"features": {
                	"targetCurrencies": ["AUD"]
				}
            }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"AUD"},
				},
			},
		},
		{
			name: "add registration and update registration by setting booleans to false",
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
			patchBody: `{
            	"features": {
                    "temperature": false,
                    "precipitation": false,
                    "airQuality": false,
                    "capital": false,
                    "coordinates": false,
                    "population": false,
                    "area": false
				}
            }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
			},
		},
		{
			name: "add registration and update registration by replacing country, with normalization check",
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
			patchBody: `{
                "country": " New Zealand ",
                "isoCode": "Nz ",
            	"features": {
				}
            }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "New Zealand",
				"isoCode": "NZ",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRegistrationHandler(t)
			ctx := context.Background()

			created := createRegistrationThroughHandler(t, h, tt.createBody)

			req := httptest.NewRequest(http.MethodPatch, utility.RegistrationPathID+created.ID, bytes.NewBufferString(tt.patchBody))
			req.SetPathValue("id", created.ID)
			req.Header.Set(utility.ContentType, utility.ApplicationJSON)

			rr := httptest.NewRecorder()
			h.partialUpdateRegistration(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code, rr.Body.String())
			assert.Equal(t, "", rr.Body.String())

			doc, err := h.Client.Collection(utility.RegistrationsCollection).Doc(created.ID).Get(ctx)
			require.NoError(t, err)

			gotDoc := doc.Data()

			assert.Equal(t, tt.wantDoc["country"], gotDoc["country"])
			assert.Equal(t, tt.wantDoc["isoCode"], gotDoc["isoCode"])
			assert.NotEmpty(t, gotDoc["features"])

			gotFeatures, ok := gotDoc["features"].(map[string]any)
			require.True(t, ok)

			wantFeatures := tt.wantDoc["features"].(map[string]any)
			assert.Equal(t, wantFeatures["temperature"], gotFeatures["temperature"])
			assert.Equal(t, wantFeatures["precipitation"], gotFeatures["precipitation"])
			assert.Equal(t, wantFeatures["airQuality"], gotFeatures["airQuality"])
			assert.Equal(t, wantFeatures["capital"], gotFeatures["capital"])
			assert.Equal(t, wantFeatures["coordinates"], gotFeatures["coordinates"])
			assert.Equal(t, wantFeatures["population"], gotFeatures["population"])
			assert.Equal(t, wantFeatures["area"], gotFeatures["area"])

			assert.ElementsMatch(t, wantFeatures["targetCurrencies"], gotFeatures["targetCurrencies"])
		})
	}
}

func TestPartialUpdateRegistrationSuccess_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		initialDocID string
		initialDoc   map[string]any
		body         string
		wantStatus   int
		wantDoc      map[string]any
	}{
		{
			name:         "replace all target currencies",
			initialDocID: "reg-001",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
                "features": {
                    "targetCurrencies": ["AUD"] 
			    }
		    }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"AUD"},
				},
			},
		},
		{
			name:         "replace empty target currencies list with normalization check",
			initialDocID: "reg-011",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{},
				},
				"lastChange": "old",
			},
			body: `{
                "features": {
                    "targetCurrencies": [" aUd "] 
			    }
		    }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"AUD"},
				},
			},
		},
		{
			name:         "replace all target currencies and set booleans to false",
			initialDocID: "reg-021",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
                "features": {
                    "temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
                    "targetCurrencies": ["AUD"] 
			    }
		    }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
					"targetCurrencies": []string{"AUD"},
				},
			},
		},
		{
			name:         "add an additional target currency",
			initialDocID: "reg-002",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
				"features": {
					"addTargetCurrencies": ["AUD"]
				}
			}`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK", "AUD"},
				},
			},
		},
		{
			name:         "add target currency into an empty starting array",
			initialDocID: "reg-022",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{},
				},
				"lastChange": "old",
			},
			body: `{
				"features": {
					"addTargetCurrencies": ["EUR"]
				}
			}`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR"},
				},
			},
		},
		{
			name:         "add additional target currency and add ones that already exists with normalization check",
			initialDocID: "reg-012",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
				"features": {
					"addTargetCurrencies": ["EuR ", "uSD ", " SEk ", "aud"]
				}
			}`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK", "AUD"},
				},
			},
		},
		{
			name:         "add an additional target currency and set booleans to false",
			initialDocID: "reg-032",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
				"features": {
					"temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
					"addTargetCurrencies": ["AUD"]
				}
			}`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
					"targetCurrencies": []string{"EUR", "USD", "SEK", "AUD"},
				},
			},
		},
		{
			name:         "remove one target currency",
			initialDocID: "reg-003",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
                "features": {
                    "removeTargetCurrencies": ["EUR"]
				}
			}`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"USD", "SEK"},
				},
			},
		},
		{
			name:         "remove all target currencies",
			initialDocID: "reg-013",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
                "features": {
                    "removeTargetCurrencies": ["EUR", "USD", "SEK"]
                }
		   }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{},
				},
			},
		},
		{
			name:         "remove all target currencies with normalization check",
			initialDocID: "reg-014",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
                "features": {
                    "removeTargetCurrencies": [" EuR", " usd ", "SEk "]
                }
		   }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{},
				},
			},
		},
		{
			name:         "remove one target currency and set booleans to false",
			initialDocID: "reg-015",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
                "features": {
                    "temperature":      false,
					"precipitation":    false,
                    "airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
                    "removeTargetCurrencies": ["EUR"]
				}
			}`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      false,
					"precipitation":    false,
					"airQuality":       false,
					"capital":          false,
					"coordinates":      false,
					"population":       false,
					"area":             false,
					"targetCurrencies": []string{"USD", "SEK"},
				},
			},
		},
		{
			name:         "replace country with normalization check",
			initialDocID: "reg-006",
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			body: `{
                "country": " New Zealand ",
                "isoCode": "nZ ",
				"features": {}
		    }`,
			wantStatus: http.StatusOK,
			wantDoc: map[string]any{
				"country": "New Zealand",
				"isoCode": "NZ",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRegistrationHandler(t)
			ctx := context.Background()

			_, err := h.Client.Collection(utility.RegistrationsCollection).Doc(tt.initialDocID).Set(ctx, tt.initialDoc)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPatch, utility.RegistrationPathID+tt.initialDocID, bytes.NewBufferString(tt.body))
			req.SetPathValue("id", tt.initialDocID)
			req.Header.Set(utility.ContentType, utility.ApplicationJSON)

			rr := httptest.NewRecorder()

			h.partialUpdateRegistration(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code, rr.Body.String())
			assert.Equal(t, "", rr.Body.String())

			doc, errGet := h.Client.Collection(utility.RegistrationsCollection).Doc(tt.initialDocID).Get(ctx)
			require.NoError(t, errGet)

			gotDoc := doc.Data()

			assert.Equal(t, tt.wantDoc["country"], gotDoc["country"])
			assert.Equal(t, tt.wantDoc["isoCode"], gotDoc["isoCode"])
			assert.NotEmpty(t, gotDoc["lastChange"])

			gotFeatures, ok := gotDoc["features"].(map[string]any)
			require.True(t, ok)

			wantFeatures := tt.wantDoc["features"].(map[string]any)
			assert.Equal(t, wantFeatures["temperature"], gotFeatures["temperature"])
			assert.Equal(t, wantFeatures["precipitation"], gotFeatures["precipitation"])
			assert.Equal(t, wantFeatures["airQuality"], gotFeatures["airQuality"])
			assert.Equal(t, wantFeatures["capital"], gotFeatures["capital"])
			assert.Equal(t, wantFeatures["coordinates"], gotFeatures["coordinates"])
			assert.Equal(t, wantFeatures["population"], gotFeatures["population"])
			assert.Equal(t, wantFeatures["area"], gotFeatures["area"])
			assert.ElementsMatch(t, wantFeatures["targetCurrencies"], gotFeatures["targetCurrencies"])
		})
	}
}

func TestPartialUpdateRegistrationInvalidJSON_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		initialDocID     string
		seedDocument     bool
		initialDoc       map[string]any
		pathID           string
		body             string
		wantStatus       int
		wantBodyContains string
		wantUnchanged    bool
	}{
		{
			name:         "invalid json payload; missing closing brace",
			initialDocID: "reg-neg-001",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID:           "reg-neg-001",
			body:             `{"features": {`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; missing comma between fields",
			initialDocID: "reg-neg-011",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-011",
			body: `{
								"country": "Norway"
								"isoCode": "NO"
								"features": {
									"temperature":      true
									"precipitation":    true
									"airQuality":       true
									"capital":          true
									"coordinates":      true
									"population":       true
									"area":             true
									"targetCurrencies": ["EUR", "USD", "SEK"]
								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; trailing comma",
			initialDocID: "reg-neg-021",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-021",
			body: `{
								"country": "Norway",
								"isoCode": "NO",
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; unquoted object key",
			initialDocID: "reg-neg-031",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-031",
			body: `{
								country: "Norway"
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; single quotes instead of double quotes",
			initialDocID: "reg-neg-041",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-041",
			body: `{
								'country': 'Norway'
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; features as array instead of object",
			initialDocID: "reg-neg-051",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-051",
			body: `{
								"features": [{"temperature": true}]
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; country as array instead of string",
			initialDocID: "reg-neg-061",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-061",
			body: `{
								"country": ["Norway"]
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; boolean field as string",
			initialDocID: "reg-neg-091",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-091",
			body: `{
								"features": {
								"temperature": "true"
                                }
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; targetCurrencies as string instead of array",
			initialDocID: "reg-neg-101",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-101",
			body: `{
								"features": {
								"targetCurrencies": "AUD"
                                }
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
		{
			name:         "invalid json payload; targetCurrencies wrong type",
			initialDocID: "reg-neg-111",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg-111",
			body: `{
								"features": {
								"targetCurrencies": "1!3"
                                }
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid json payload",
			wantUnchanged:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRegistrationHandler(t)
			ctx := context.Background()

			if tt.seedDocument {
				_, err := h.Client.Collection(utility.RegistrationsCollection).Doc(tt.initialDocID).Set(ctx, tt.initialDoc)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPatch, utility.RegistrationPathID+tt.pathID, bytes.NewBufferString(tt.body))
			req.SetPathValue("id", tt.pathID)
			req.Header.Set(utility.ContentType, utility.ApplicationJSON)

			rr := httptest.NewRecorder()
			h.partialUpdateRegistration(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.wantBodyContains)

			if tt.wantUnchanged {
				doc, err := h.Client.Collection(utility.RegistrationsCollection).Doc(tt.initialDocID).Get(ctx)
				require.NoError(t, err)

				gotDoc := doc.Data()
				assert.Equal(t, tt.initialDoc["country"], gotDoc["country"])
				assert.Equal(t, tt.initialDoc["isoCode"], gotDoc["isoCode"])
			}
		})
	}
}

func TestPartialUpdateRegistrationValidationFailure_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		initialDocID     string
		seedDocument     bool
		initialDoc       map[string]any
		pathID           string
		body             string
		wantStatus       int
		wantBodyContains string
		wantUnchanged    bool
	}{
		{
			name:         "isoCode as symbols",
			initialDocID: "reg-neg2-001",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-001",
			body: `{
								"country": "Norway",
								"isoCode": "!@"
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "iso-code must contain only letters",
			wantUnchanged:    true,
		},
		{
			name:         "country as symbols",
			initialDocID: "reg-neg2-011",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-011",
			body: `{
								"country": "$@!!"
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "country must contain only letters",
			wantUnchanged:    true,
		},
		{
			name:         "isoCode too many characters",
			initialDocID: "reg-neg2-021",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-021",
			body: `{
								"country": "Norway",
                                "isoCode": "NOR"
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "iso-code must be 2-letter country code",
			wantUnchanged:    true,
		},
		{
			name:         "country as numbers",
			initialDocID: "reg-neg2-031",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-031",
			body: `{
								"country": "1234"
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "country must contain only letters",
			wantUnchanged:    true,
		},
		{
			name:         "isoCode as numbers",
			initialDocID: "reg-neg2-041",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-041",
			body: `{
								"country": "Norway",
								"isoCode": "12"	
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "iso-code must contain only letters",
			wantUnchanged:    true,
		},
		{
			name:         "country blank",
			initialDocID: "reg-neg2-051",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-051",
			body: `{
								"country": ""
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "country cannot be blank",
			wantUnchanged:    true,
		},
		{
			name:         "isoCode blank",
			initialDocID: "reg-neg2-061",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-061",
			body: `{
								"country": "Norway",
								"isoCode": ""	
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "iso-code must be 2-letter country code",
			wantUnchanged:    true,
		},
		{
			name:         "currencies as symbols addTargetCurrencies",
			initialDocID: "reg-neg2-071",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-071",
			body: `{
								"features": {
                                	"addTargetCurrencies": ["!!!"] 
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "add target currency must contain only letters",
			wantUnchanged:    true,
		},
		{
			name:         "currencies as numbers addTargetCurrencies",
			initialDocID: "reg-neg2-081",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-081",
			body: `{
								"features": {
                                	"addTargetCurrencies": ["123"] 
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "add target currency must contain only letters",
			wantUnchanged:    true,
		},
		{
			name:         "currency over symbol limit addTargetCurrencies",
			initialDocID: "reg-neg2-091",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-091",
			body: `{
								"features": {
                                	"addTargetCurrencies": ["EURO"] 
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "add target currency must be 3-letter ISO-code",
			wantUnchanged:    true,
		},
		{
			name:         "currency over symbol limit targetCurrencies",
			initialDocID: "reg-neg2-092",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-092",
			body: `{
								"features": {
                                	"targetCurrencies": ["EURO"] 
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "target currency must be 3-letter ISO-code",
			wantUnchanged:    true,
		},
		{
			name:         "currency over symbol limit removeTargetCurrencies",
			initialDocID: "reg-neg2-093",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-093",
			body: `{
								"features": {
                                	"removeTargetCurrencies": ["EURO"] 
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "remove target currency must be 3-letter ISO-code",
			wantUnchanged:    true,
		},
		{
			name:         "currency blank addTargetCurrencies",
			initialDocID: "reg-neg2-101",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-101",
			body: `{
								"features": {
                                	"addTargetCurrencies": [""] 
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "add target currency must be 3-letter ISO-code",
			wantUnchanged:    true,
		},
		{
			name:         "mixing PATCH modes addTargetCurrencies and removeTargetCurrencies",
			initialDocID: "reg-neg2-201",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-201",
			body: `{
								"features": {
                                	"addTargetCurrencies": ["AUD"],
                                    "removeTargetCurrencies": ["EUR"]
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "target currencies cannot be added and removed in the same request",
			wantUnchanged:    true,
		},
		{
			name:         "mixing PATCH modes targetCurrencies and removeTargetCurrencies",
			initialDocID: "reg-neg2-301",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-301",
			body: `{
								"features": {
                                	"targetCurrencies": ["AUD"],
                                    "removeTargetCurrencies": ["EUR"]
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "target currencies cannot be replaced and added/removed in the same request",
			wantUnchanged:    true,
		},
		{
			name:         "mixing PATCH modes targetCurrencies and addTargetCurrencies",
			initialDocID: "reg-neg2-401",
			seedDocument: true,
			initialDoc: map[string]any{
				"country": "Norway",
				"isoCode": "NO",
				"features": map[string]any{
					"temperature":      true,
					"precipitation":    true,
					"airQuality":       true,
					"capital":          true,
					"coordinates":      true,
					"population":       true,
					"area":             true,
					"targetCurrencies": []string{"EUR", "USD", "SEK"},
				},
				"lastChange": "old",
			},
			pathID: "reg-neg2-401",
			body: `{
								"features": {
                                	"targetCurrencies": ["AUD"],
                                    "addTargetCurrencies": ["RUB"]
   								}
			}`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "target currencies cannot be replaced and added/removed in the same request",
			wantUnchanged:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRegistrationHandler(t)
			ctx := context.Background()

			if tt.seedDocument {
				_, err := h.Client.Collection(utility.RegistrationsCollection).Doc(tt.initialDocID).Set(ctx, tt.initialDoc)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPatch, utility.RegistrationPathID+tt.pathID, bytes.NewBufferString(tt.body))
			req.SetPathValue("id", tt.pathID)
			req.Header.Set(utility.ContentType, utility.ApplicationJSON)

			rr := httptest.NewRecorder()
			h.partialUpdateRegistration(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.wantBodyContains)

			if tt.wantUnchanged {
				doc, err := h.Client.Collection(utility.RegistrationsCollection).Doc(tt.initialDocID).Get(ctx)
				require.NoError(t, err)

				gotDoc := doc.Data()
				assert.Equal(t, tt.initialDoc["country"], gotDoc["country"])
				assert.Equal(t, tt.initialDoc["isoCode"], gotDoc["isoCode"])
			}
		})
	}

}

func TestPartialUpdateRegistrationHandlerFailure_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		pathID           string
		seedDoc          bool
		initialDocID     string
		initialDoc       map[string]any
		body             string
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:         "blank id",
			pathID:       "",
			seedDoc:      false,
			initialDocID: "",
			initialDoc:   nil,
			body: `{
                        "country": "Norway"
            }`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid registration id",
		},
		{
			name:         "whitespace id",
			pathID:       " ",
			seedDoc:      false,
			initialDocID: "",
			initialDoc:   nil,
			body: `{
                        "country": "Norway"
            }`,
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid registration id",
		},
		{
			name:         "registration not found",
			pathID:       "reg-neg3-001",
			seedDoc:      false,
			initialDocID: "",
			initialDoc:   nil,
			body: `{
					"country": "Norway"
			}`,
			wantStatus:       http.StatusNotFound,
			wantBodyContains: "failed getting registration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRegistrationHandler(t)
			ctx := context.Background()

			requestPathID := tt.pathID
			if strings.TrimSpace(requestPathID) == "" {
				requestPathID = "dummy-id"
			}

			if tt.seedDoc {
				_, err := h.Client.Collection(utility.RegistrationsCollection).Doc(tt.initialDocID).Set(ctx, tt.initialDoc)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPatch, utility.RegistrationPathID+requestPathID, bytes.NewBufferString(tt.body))
			req.SetPathValue("id", tt.pathID)
			req.Header.Set(utility.ContentType, utility.ApplicationJSON)

			rr := httptest.NewRecorder()
			h.partialUpdateRegistration(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.wantBodyContains)
		})
	}
}

func TestHandleAllGetRegistrations(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations", nil)
	w := httptest.NewRecorder()

	handler.handleAllGetRegistration(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %d want %d", w.Code, http.StatusOK)
	}

}

func TestHandleAllGetRegistrationInvalidDocIDLength(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations/{id}", nil)
	req.SetPathValue("id", "short-id") // len != 20
	w := httptest.NewRecorder()

	h.handleAllGetRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
}

/*
func TestHandleAllGetRegistrationGetNoIDCallsGetAll(t *testing.T) {
	origin := listRegistrationDocs
	listRegistrationDocs = func(ctx context.Context, client *firestore.Client) ([]map[string]interface{}, error) {
		return []map[string]interface{}{
			{"country": "Norway", "isoCode": "NO"},
		}, nil
	}
	t.Cleanup(func() { listRegistrationDocs = origin })

	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations", nil)
	w := httptest.NewRecorder()

	h.handleAllGetRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(t)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var got []map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&got)
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "Norway", got[0]["country"])
}

*/

func TestHandleRegReqGetInvalidDocIDLength(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations/{id}", nil)
	req.SetPathValue("id", "123") // len != 20
	w := httptest.NewRecorder()

	h.HandleRegReq(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleAllGetRegistrationHeadNoIDReturnsStatusOK(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations", nil)
	w := httptest.NewRecorder()

	h.handleAllGetRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestHandleAllGetRegistrationHeadWhitespaceIDStatusOK(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations/{id}", nil)
	req.SetPathValue("id", "   ") // trimmed => empty
	w := httptest.NewRecorder()

	h.handleAllGetRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestGetAllRegistrations(t *testing.T) {
	clearFirestoreEmulator(t)

	ctx := context.Background()

	client, err := firestore.NewClient(ctx, "test-project")
	require.NoError(t, err)
	defer func(client *firestore.Client) {
		err := client.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(client)

	_, _, err = client.Collection(utility.RegistrationsCollection).Add(ctx, map[string]any{
		"country": "Norway",
		"isoCode": "NO",
	})
	require.NoError(t, err)

	h := &Handler{Client: client}

	req := httptest.NewRequest(http.MethodGet, "/registrations", nil)
	w := httptest.NewRecorder()

	h.GetAllRegistrations(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetRegistrationByID_Success(t *testing.T) {
	clearFirestoreEmulator(t)

	ctx := context.Background()

	client, err := firestore.NewClient(ctx, "test-project")
	require.NoError(t, err)
	defer client.Close()

	// Seed document
	docRef, _, err := client.Collection(utility.RegistrationsCollection).Add(ctx, map[string]any{
		"country": "Norway",
		"isoCode": "NO",
		"features": map[string]any{
			"temperature": true,
			"airQuality":  true,
		},
		"lastChange": "2026-01-01T00:00:00Z",
	})
	require.NoError(t, err)

	h := &Handler{Client: client}

	req := httptest.NewRequest(
		http.MethodGet,
		"/registrations/"+docRef.ID,
		nil,
	)

	w := httptest.NewRecorder()

	h.GetRegistrationByID(w, req, docRef.ID)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got utility.StoredRegistration
	err = json.NewDecoder(resp.Body).Decode(&got)

	assert.NoError(t, err)
	assert.Equal(t, "Norway", got.Country)
	assert.Equal(t, "NO", got.IsoCode)
	assert.Equal(t, docRef.ID, got.ID)
}

func TestGetRegistrationByID_NotFound(t *testing.T) {
	clearFirestoreEmulator(t)

	ctx := context.Background()

	client, err := firestore.NewClient(ctx, "test-project")
	require.NoError(t, err)
	defer client.Close()

	h := &Handler{Client: client}

	req := httptest.NewRequest(
		http.MethodGet,
		"/registrations/nonexistent",
		nil,
	)

	w := httptest.NewRecorder()

	h.GetRegistrationByID(w, req, "nonexistent")

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandleHeadDocIDNotFoundReturns404(t *testing.T) {
	origin := getRegistrationDoc
	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		return errors.New("not found")
	}
	t.Cleanup(func() { getRegistrationDoc = origin })

	h := &Handler{}
	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations/{id}", nil)
	w := httptest.NewRecorder()

	// Call directly to target handleHead branch with non-empty ID.
	h.handleHead(w, req, "12345678901234567890")

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(t)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandleHeadDocIDExistsReturns200(t *testing.T) {
	origin := getRegistrationDoc
	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		return nil
	}
	t.Cleanup(func() { getRegistrationDoc = origin })

	h := &Handler{}
	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations/{id}", nil)
	w := httptest.NewRecorder()

	h.handleHead(w, req, "12345678901234567890")

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal()
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAddRegistrationSuccess(t *testing.T) {
	origin := addRegistrationDocImpl
	addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
		return "mock-id-001", nil
	}

	defer func() {
		addRegistrationDocImpl = origin
	}()

	h := &Handler{}

	body := `{
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
	}`
	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, before+1, after)

	var got utility.RegistrationResponse
	err := json.NewDecoder(resp.Body).Decode(&got)

	assert.NoError(t, err)
	assert.Equal(t, "mock-id-001", got.ID)
}

func TestAddRegistrationSpaceSuccess(t *testing.T) {
	origin := addRegistrationDocImpl
	addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
		return "mock-id-001", nil
	}

	defer func() {
		addRegistrationDocImpl = origin
	}()

	h := &Handler{}

	// Ensure it can handle country names with space in them
	body := `{
   		"country": "New Zealand",
   		"isoCode": "NZ",
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
	}`
	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, before+1, after)

	var got utility.RegistrationResponse
	err := json.NewDecoder(resp.Body).Decode(&got)

	assert.NoError(t, err)
	assert.Equal(t, "mock-id-001", got.ID)
}

func TestAddRegistrationFalseFieldsSuccess(t *testing.T) {
	origin := addRegistrationDocImpl
	addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
		return "mock-id-001", nil
	}

	defer func() {
		addRegistrationDocImpl = origin
	}()

	h := &Handler{}
	// Ensure that a registration feature fields can be false
	body := `{
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
	}`
	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, before+1, after)

	var got utility.RegistrationResponse
	err := json.NewDecoder(resp.Body).Decode(&got)

	assert.NoError(t, err)
	assert.Equal(t, "mock-id-001", got.ID)
}

func TestAddRegistrationEmptyFeaturesSuccess(t *testing.T) {
	origin := addRegistrationDocImpl
	addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
		return "mock-id-001", nil
	}

	defer func() {
		addRegistrationDocImpl = origin
	}()

	h := &Handler{}
	// Don't include feature fields
	body := `{
		"country": "Norway",
		"isoCode": "NO",
		"features": {}
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, before+1, after)

	var got utility.RegistrationResponse
	err := json.NewDecoder(resp.Body).Decode(&got)

	assert.NoError(t, err)
	assert.Equal(t, "mock-id-001", got.ID)
}

func TestAddRegistrationNormalizationSuccess(t *testing.T) {
	origin := addRegistrationDocImpl
	addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
		return "mock-id-001", nil
	}

	defer func() {
		addRegistrationDocImpl = origin
	}()

	h := &Handler{}
	// Ensure that normalization works as intended
	body := `{
   		"country": " Norway ",
   		"isoCode": "nO ",
   		"features": {
      		"temperature": false,
      		"precipitation": false,
      		"airQuality": false,
      		"capital": false,
      		"coordinates": false,
      		"population": false,
      		"area": false,
      		"targetCurrencies": ["eur", " UsD", "SEk  "]
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, before+1, after)

	var got utility.RegistrationResponse
	err := json.NewDecoder(resp.Body).Decode(&got)

	assert.NoError(t, err)
	assert.Equal(t, "mock-id-001", got.ID)
}

func TestAddRegistrationEmptyTargetCurrenciesSuccess(t *testing.T) {
	origin := addRegistrationDocImpl
	addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
		return "mock-id-001", nil
	}

	defer func() {
		addRegistrationDocImpl = origin
	}()

	h := &Handler{}

	body := `{
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
      		"targetCurrencies": []
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, before+1, after)

	var got utility.RegistrationResponse
	err := json.NewDecoder(resp.Body).Decode(&got)

	assert.NoError(t, err)
	assert.Equal(t, "mock-id-001", got.ID)
}

func TestAddRegistrationInvalidIso1(t *testing.T) {
	h := &Handler{}
	// set ISO-code to N0 instead of NO
	body := `{
   		"country": "Norway",
   		"isoCode": "N0",
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
	}`

	var req = httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidIso2(t *testing.T) {
	h := &Handler{}
	// ISO-code blank
	body := `{
   		"country": "Norway",
   		"isoCode": "",
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
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidIso3(t *testing.T) {
	h := &Handler{}
	// Too many letters ISO-code
	body := `{
   		"country": "Norway",
   		"isoCode": "NOR",
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
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidIso4(t *testing.T) {
	h := &Handler{}
	// Numbers instead of letters ISO-code
	body := `{
   		"country": "Norway",
   		"isoCode": "12",
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
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidIso5(t *testing.T) {
	h := &Handler{}
	// Symbols instead of letters
	body := `{
   		"country": "Norway",
   		"isoCode": "@$",
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
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidCountry1(t *testing.T) {
	h := &Handler{}
	// Blank country
	body := `{
   		"country": "",
   		"isoCode": "No",
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
	}`

	var req = httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidCountry2(t *testing.T) {
	h := &Handler{}
	// Country containing numbers
	body := `{
   		"country": "123456",
   		"isoCode": "No",
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
	}`

	var req = httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidCountry3(t *testing.T) {
	h := &Handler{}
	// Country containing symbols
	body := `{
   		"country": "#¤%&!()=?@£$€",
   		"isoCode": "No",
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
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidCurrency1(t *testing.T) {
	h := &Handler{}
	// Change EUR to EU
	body := `{
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
      		"targetCurrencies": ["EU", "USD", "SEK"]
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidCurrency2(t *testing.T) {
	h := &Handler{}
	// Have one currency, but EU instead of EUR
	body := `{
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
      		"targetCurrencies": ["EU"]
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidCurrency3(t *testing.T) {
	h := &Handler{}
	// Use symbols as currency
	body := `{
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
      		"targetCurrencies": ["!!!"]
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidCurrency4(t *testing.T) {
	h := &Handler{}
	// Invalid currency length
	body := `{
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
      		"targetCurrencies": ["EURO"]
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidJson1(t *testing.T) {
	h := &Handler{}
	// removed brackets for features
	body := `{
   		"country": "Norway",
   		"isoCode": "NOR",
   		"features": 
      		"temperature": true,
      		"precipitation": true,
      		"airQuality": true,
      		"capital": true,
      		"coordinates": true,
      		"population": true,
      		"area": true,
      		"targetCurrencies": ["EUR", "USD", "SEK"]
		
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidJson2(t *testing.T) {
	h := &Handler{}
	// removed last bracket for features
	body := `{
   		"country": "Norway",
   		"isoCode": "NOR",
   		"features": {
      		"temperature": true,
      		"precipitation": true,
      		"airQuality": true,
      		"capital": true,
      		"coordinates": true,
      		"population": true,
      		"area": true,
      		"targetCurrencies": ["EUR", "USD", "SEK"]
		
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationInvalidJson3(t *testing.T) {
	h := &Handler{}
	// use array instead for object
	body := `{
   		"country": "Norway",
   		"isoCode": "NOR",
   		"features": [
      		"temperature": true,
      		"precipitation": true,
      		"airQuality": true,
      		"capital": true,
      		"coordinates": true,
      		"population": true,
      		"area": true,
      		"targetCurrencies": ["EUR", "USD", "SEK"]
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()
	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestAddRegistrationServerError(t *testing.T) {
	origin := addRegistrationDocImpl
	addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
		return "", errors.New("failed to add registration")
	}

	defer func() {
		addRegistrationDocImpl = origin
	}()

	h := &Handler{}

	body := `{
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
	}`
	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	before := h.RegistrationCount.Load()

	h.addRegistration(w, req)

	after := h.RegistrationCount.Load()

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, before, after)
}

func TestReplaceRegistrationSuccess(t *testing.T) {
	originGet := getRegistrationDoc
	originSet := setRegistrationDoc

	var gotID string
	var gotReg map[string]any

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		return nil
	}

	setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
		gotID = id
		gotReg = reg
		return nil
	}

	defer func() {
		getRegistrationDoc = originGet
		setRegistrationDoc = originSet
	}()

	h := &Handler{}

	body := `{
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
	}`

	req := httptest.NewRequest(http.MethodPut, "/envdash/v1/registrations/{id}", strings.NewReader(body))
	req.SetPathValue("id", "mock-id-001")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.replaceRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "mock-id-001", gotID)
	assert.Equal(t, "Norway", gotReg["country"])
	assert.Equal(t, "NO", gotReg["isoCode"])
	assert.NotEmpty(t, gotReg["lastChange"])

	features, ok := gotReg["features"].(utility.RegistrationFeatures)
	assert.True(t, ok)

	assert.True(t, features.Temperature)
	assert.True(t, features.Precipitation)
	assert.True(t, features.AirQuality)
	assert.True(t, features.Capital)
	assert.True(t, features.Coordinates)
	assert.True(t, features.Population)
	assert.True(t, features.Area)
	assert.Equal(t, []string{"EUR", "USD", "SEK"}, features.TargetCurrencies)
}

func TestReplaceRegistrationNormalizationSuccess(t *testing.T) {
	originGet := getRegistrationDoc
	originSet := setRegistrationDoc

	var gotID string
	var gotReg map[string]any

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		return nil
	}

	setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
		gotID = id
		gotReg = reg
		return nil
	}

	defer func() {
		getRegistrationDoc = originGet
		setRegistrationDoc = originSet
	}()

	h := &Handler{}

	body := `{
   		"country": " Norway",
   		"isoCode": "No ",
   		"features": {
      		"temperature": true,
      		"precipitation": true,
      		"airQuality": true,
      		"capital": true,
      		"coordinates": true,
      		"population": true,
      		"area": true,
      		"targetCurrencies": [" EUR", "UsD", "SEk "]
		}
	}`

	req := httptest.NewRequest(http.MethodPut, "/envdash/v1/registrations/{id}", strings.NewReader(body))
	req.SetPathValue("id", "mock-id-001")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.replaceRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "mock-id-001", gotID)
	assert.Equal(t, "Norway", gotReg["country"])
	assert.Equal(t, "NO", gotReg["isoCode"])
	assert.NotEmpty(t, gotReg["lastChange"])

	features, ok := gotReg["features"].(utility.RegistrationFeatures)
	assert.True(t, ok)
	assert.Equal(t, []string{"EUR", "USD", "SEK"}, features.TargetCurrencies)
}

func TestReplaceRegistrationEmptyFeaturesSuccess(t *testing.T) {
	originGet := getRegistrationDoc
	originSet := setRegistrationDoc

	var gotID string
	var gotReg map[string]any

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		return nil
	}

	setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
		gotID = id
		gotReg = reg
		return nil
	}

	defer func() {
		getRegistrationDoc = originGet
		setRegistrationDoc = originSet
	}()

	h := &Handler{}

	body := `{
   		"country": " Norway",
   		"isoCode": "No ",
   		"features": {}
	}`

	req := httptest.NewRequest(http.MethodPut, "/envdash/v1/registrations/{id}", strings.NewReader(body))
	req.SetPathValue("id", "mock-id-001")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.replaceRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "mock-id-001", gotID)
	assert.Equal(t, "Norway", gotReg["country"])
	assert.Equal(t, "NO", gotReg["isoCode"])
	assert.NotEmpty(t, gotReg["lastChange"])
	features, ok := gotReg["features"].(utility.RegistrationFeatures)
	assert.True(t, ok)
	// Features should be stored as zero-value struct
	assert.False(t, features.Temperature)
	assert.False(t, features.Precipitation)
	assert.False(t, features.AirQuality)
	assert.False(t, features.Capital)
	assert.False(t, features.Coordinates)
	assert.False(t, features.Population)
	assert.False(t, features.Area)
	assert.Empty(t, features.TargetCurrencies)
}

func TestReplaceRegistrationInvalidJson(t *testing.T) {
	originGet := getRegistrationDoc
	originSet := setRegistrationDoc

	var getCalled bool
	var setCalled bool

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		getCalled = true
		return nil
	}

	setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
		setCalled = true
		return nil
	}

	defer func() {
		getRegistrationDoc = originGet
		setRegistrationDoc = originSet
	}()

	h := &Handler{}
	// Removed last object boundary
	body := `{
   		"country": " Norway",
   		"isoCode": "No ",
   		"features": {
	}`

	req := httptest.NewRequest(http.MethodPut, "/envdash/v1/registrations/{id}", strings.NewReader(body))
	req.SetPathValue("id", "mock-id-001")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.replaceRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.False(t, getCalled)
	assert.False(t, setCalled)
}

func TestReplaceRegistrationNotFound(t *testing.T) {
	originGet := getRegistrationDoc
	originSet := setRegistrationDoc

	var setCalled bool

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		return errors.New("not found")
	}

	setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
		setCalled = true
		return nil
	}

	defer func() {
		getRegistrationDoc = originGet
		setRegistrationDoc = originSet
	}()

	h := &Handler{}

	body := `{
   		"country": " Norway",
   		"isoCode": "No ",
   		"features": {}
	}`

	req := httptest.NewRequest(http.MethodPut, "/envdash/v1/registrations/{id}", strings.NewReader(body))
	req.SetPathValue("id", "mock-id-001")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.replaceRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.False(t, setCalled)
}

func TestReplaceRegistrationUpdateFailure(t *testing.T) {
	originGet := getRegistrationDoc
	originSet := setRegistrationDoc

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		return nil
	}

	setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
		return errors.New("update failed")
	}

	defer func() {
		getRegistrationDoc = originGet
		setRegistrationDoc = originSet
	}()

	h := &Handler{}

	body := `{
   		"country": " Norway",
   		"isoCode": "No ",
   		"features": {}
	}`

	req := httptest.NewRequest(http.MethodPut, "/envdash/v1/registrations/{id}", strings.NewReader(body))
	req.SetPathValue("id", "mock-id-001")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.replaceRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestReplaceRegistrationInvalidId(t *testing.T) {
	originGet := getRegistrationDoc
	originSet := setRegistrationDoc

	var gotID string
	var setCalled bool

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotID = id
		return nil
	}

	setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
		setCalled = true
		gotID = id
		return nil
	}

	defer func() {
		getRegistrationDoc = originGet
		setRegistrationDoc = originSet
	}()

	h := &Handler{}

	body := `{
   		"country": " Norway",
   		"isoCode": "No ",
   		"features": {}
	}`

	req := httptest.NewRequest(http.MethodPut, "/envdash/v1/registrations/{id}", strings.NewReader(body))
	req.SetPathValue("id", "")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.replaceRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "", gotID)
	assert.False(t, setCalled)
}

func TestDeleteRegistrationSuccess(t *testing.T) {
	originGet := getRegistrationByIDDocImpl
	originDelete := deleteRegistrationDoc

	var gotID string
	var gotDeleteID string
	var deleteCalled bool

	getRegistrationByIDDocImpl = func(ctx context.Context, client *firestore.Client, id string) (map[string]any, error) {
		gotID = id
		return map[string]any{
			"isoCode": "NO",
		}, nil
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotDeleteID = id
		deleteCalled = true
		return nil
	}

	defer func() {
		getRegistrationByIDDocImpl = originGet
		deleteRegistrationDoc = originDelete
	}()

	h := &Handler{}

	req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/registrations/{id}", nil)
	req.SetPathValue("id", "mock-id-001")
	w := httptest.NewRecorder()

	h.deleteRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "mock-id-001", gotID)
	assert.Equal(t, "mock-id-001", gotDeleteID)
	assert.True(t, deleteCalled)
}

func TestDeleteRegistrationGetFailure(t *testing.T) {
	originGet := getRegistrationByIDDocImpl
	originDelete := deleteRegistrationDoc

	var gotID string
	var deleteCalled bool

	getRegistrationByIDDocImpl = func(ctx context.Context, client *firestore.Client, id string) (map[string]any, error) {
		gotID = id
		return nil, errors.New("could not find registration")
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		deleteCalled = true
		return nil
	}

	defer func() {
		getRegistrationByIDDocImpl = originGet
		deleteRegistrationDoc = originDelete
	}()

	h := &Handler{}
	req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/registrations/{id}", nil)
	req.SetPathValue("id", "mock-id-001")
	w := httptest.NewRecorder()

	h.deleteRegistration(w, req)

	resp := w.Result()

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "mock-id-001", gotID)
	assert.False(t, deleteCalled)
}

func TestDeleteRegistrationInvalidId(t *testing.T) {
	originGet := getRegistrationByIDDocImpl
	originDelete := deleteRegistrationDoc

	var gotID string
	var gotDeleteID string
	var deleteCalled bool

	getRegistrationByIDDocImpl = func(ctx context.Context, client *firestore.Client, id string) (map[string]any, error) {
		gotID = id
		return map[string]any{
			"isoCode": "NO",
		}, nil
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotDeleteID = id
		deleteCalled = true
		return nil
	}

	defer func() {
		getRegistrationByIDDocImpl = originGet
		deleteRegistrationDoc = originDelete
	}()

	h := &Handler{}
	req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/registrations/{id}", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	h.deleteRegistration(w, req)

	resp := w.Result()

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "", gotID)
	assert.Equal(t, "", gotDeleteID)
	assert.False(t, deleteCalled)
}

func TestDeleteRegistrationDeleteFailure(t *testing.T) {
	originGet := getRegistrationByIDDocImpl
	originDelete := deleteRegistrationDoc

	var gotID string
	var gotDeleteID string
	var deleteCalled bool

	getRegistrationByIDDocImpl = func(ctx context.Context, client *firestore.Client, id string) (map[string]any, error) {
		gotID = id
		return map[string]any{
			"isoCode": "NO",
		}, nil
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotDeleteID = id
		deleteCalled = true
		return errors.New("failed to delete registration")
	}

	defer func() {
		getRegistrationByIDDocImpl = originGet
		deleteRegistrationDoc = originDelete
	}()

	h := &Handler{}
	req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/registrations/{id}", nil)
	req.SetPathValue("id", "mock-id-001")
	w := httptest.NewRecorder()

	h.deleteRegistration(w, req)

	resp := w.Result()

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "mock-id-001", gotID)
	assert.Equal(t, "mock-id-001", gotDeleteID)
	assert.True(t, deleteCalled)
}

func TestDeleteRegistrationInvalidIsoCodeType(t *testing.T) {
	originGet := getRegistrationByIDDocImpl
	originDelete := deleteRegistrationDoc

	var deleteCalled bool

	getRegistrationByIDDocImpl = func(ctx context.Context, client *firestore.Client, id string) (map[string]any, error) {
		return map[string]any{
			"isoCode": 123,
		}, nil
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		deleteCalled = true
		return nil
	}

	defer func() {
		getRegistrationByIDDocImpl = originGet
		deleteRegistrationDoc = originDelete
	}()

	h := &Handler{}

	req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/registrations/{id}", nil)
	req.SetPathValue("id", "mock-id-001")
	w := httptest.NewRecorder()

	h.deleteRegistration(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.False(t, deleteCalled)
}
