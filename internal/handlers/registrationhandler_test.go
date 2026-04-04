package handlers

import (
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/go-jose/go-jose/v4/testutils/assert"
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

func TestAddRegistration(t *testing.T) {
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

func TestAddRegistrationFalseFields(t *testing.T) {
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

func TestAddRegistrationEmptyFeatures(t *testing.T) {
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

func TestAddRegistrationNormalization(t *testing.T) {
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

func TestAddRegistrationEmptyTargetCurrencies(t *testing.T) {
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

func TestAddRegistrationInvalidJson(t *testing.T) {
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
