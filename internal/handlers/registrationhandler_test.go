package handlers

import (
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
)

func TestHandleAllGetRegistrations(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations", nil)
	w := httptest.NewRecorder()

	handler.handleAllGetRegistration(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %d want %d", w.Code, http.StatusOK)
	}

}

/*
func TestHandleAllGetRegistrationsBadRequestTooShort(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations/S", nil)
	req.SetPathValue("id", "S")
	w := httptest.NewRecorder()

	handler.handleAllGetRegistration(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %v, got %v", http.StatusBadRequest, w.Code)
	}
}


func TestHandleAllGetRegistrationsBadRequestTooLong(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations/SEE", nil)
	req.SetPathValue("id", "SEE")
	w := httptest.NewRecorder()
	handler.handleAllGetRegistration(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %v, got %v", http.StatusBadRequest, w.Code)
	}
}
*/

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

func TestDeleteRegistration(t *testing.T) {
	originGet := getRegistrationDoc
	originDelete := deleteRegistrationDoc

	var gotID string
	var deleteCalled bool

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		return nil
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotID = id
		deleteCalled = true
		return nil
	}

	defer func() {
		getRegistrationDoc = originGet
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
	assert.True(t, deleteCalled)
	assert.Equal(t, "mock-id-001", gotID)
}

func TestDeleteRegistrationGetFailure(t *testing.T) {
	originGet := getRegistrationDoc
	originDelete := deleteRegistrationDoc

	var gotID string
	var deleteCalled bool

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotID = id
		return errors.New("could not find registration")
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		deleteCalled = true
		return nil
	}

	defer func() {
		getRegistrationDoc = originGet
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
	originGet := getRegistrationDoc
	originDelete := deleteRegistrationDoc

	var gotID string
	var gotDeleteID string
	var deleteCalled bool

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotID = id
		return nil
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotDeleteID = id
		deleteCalled = true
		return errors.New("failed to delete registration")
	}

	defer func() {
		getRegistrationDoc = originGet
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
	originGet := getRegistrationDoc
	originDelete := deleteRegistrationDoc

	var gotID string
	var gotDeleteID string
	var deleteCalled bool

	getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotID = id
		return nil
	}

	deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
		gotDeleteID = id
		deleteCalled = true
		return errors.New("failed to delete registration")
	}

	defer func() {
		getRegistrationDoc = originGet
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
