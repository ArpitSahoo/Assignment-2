package handlers

import (
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerLifecycleWebhooks(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	var received utility.WebhookInvocationPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := client.Collection(utility.WebhooksCollection).Add(context.Background(), map[string]any{
		"url":     server.URL,
		"country": "NO",
		"event":   "INVOKE",
	})
	require.NoError(t, err)

	h := &Handler{Client: client}
	h.triggerLifecycleWebhooks(context.Background(), "INVOKE", "NO")

	assert.Equal(t, "INVOKE", received.Event)
	assert.Equal(t, "NO", received.Country)
}

func TestTriggerThresholdWebhooks(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	var received utility.WebhookInvocationPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := client.Collection(utility.WebhooksCollection).Add(context.Background(), map[string]any{
		"url":     server.URL,
		"country": "NO",
		"event":   "THRESHOLD",
		"threshold": map[string]any{
			"field":    "temperature",
			"operator": ">",
			"value":    -100.0,
		},
	})
	require.NoError(t, err)

	temp := 5.0
	dashboard := utility.DashboardResponse{
		Features: utility.DashboardFeatures{
			Temperature: &temp,
		},
	}

	h := &Handler{Client: client}
	h.triggerThresholdWebhooks(context.Background(), "NO", dashboard)

	assert.Equal(t, "THRESHOLD", received.Event)
	assert.Equal(t, "NO", received.Country)
	assert.Equal(t, 5.0, received.Details.MeasuredValue)
}

func TestTriggerLifecycleWebhooksWrongEvent(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	fired := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fired = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := client.Collection(utility.WebhooksCollection).Add(context.Background(), map[string]any{
		"url":     server.URL,
		"country": "NO",
		"event":   "REGISTER",
	})
	require.NoError(t, err)

	h := &Handler{Client: client}
	h.triggerLifecycleWebhooks(context.Background(), "INVOKE", "NO")

	assert.False(t, fired, "webhook should not fire for wrong event")
}

func TestTriggerLifecycleWebhooksWrongCountry(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	fired := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fired = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := client.Collection(utility.WebhooksCollection).Add(context.Background(), map[string]any{
		"url":     server.URL,
		"country": "SE",
		"event":   "INVOKE",
	})
	require.NoError(t, err)

	h := &Handler{Client: client}
	h.triggerLifecycleWebhooks(context.Background(), "INVOKE", "NO")

	assert.False(t, fired, "webhook should not fire for wrong country")
}

func TestTriggerLifecycleWebhooksEmptyCountryFiresForAll(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	var received utility.WebhookInvocationPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := client.Collection(utility.WebhooksCollection).Add(context.Background(), map[string]any{
		"url":     server.URL,
		"country": "",
		"event":   "INVOKE",
	})
	require.NoError(t, err)

	h := &Handler{Client: client}
	h.triggerLifecycleWebhooks(context.Background(), "INVOKE", "NO")

	assert.Equal(t, "INVOKE", received.Event)
	assert.Equal(t, "NO", received.Country)
}

func TestTriggerThresholdWebhooksNegative(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	fired := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fired = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := client.Collection(utility.WebhooksCollection).Add(context.Background(), map[string]any{
		"url":     server.URL,
		"country": "NO",
		"event":   "THRESHOLD",
		"threshold": map[string]any{
			"field":    "temperature",
			"operator": ">",
			"value":    100.0,
		},
	})
	require.NoError(t, err)

	temp := 5.0
	dashboard := utility.DashboardResponse{
		Features: utility.DashboardFeatures{
			Temperature: &temp,
		},
	}

	h := &Handler{Client: client}
	h.triggerThresholdWebhooks(context.Background(), "NO", dashboard)

	assert.False(t, fired, "webhook should not have fired")
}

func TestTriggerThresholdWebhooksCompound(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	var received utility.WebhookInvocationPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, _, err := client.Collection(utility.WebhooksCollection).Add(context.Background(), map[string]any{
		"url":     server.URL,
		"country": "NO",
		"event":   "THRESHOLD",
		"threshold": map[string]any{
			"field":         "temperature",
			"operator":      ">",
			"value":         -100.0,
			"upperOperator": "<",
			"upperValue":    100.0,
		},
	})
	require.NoError(t, err)

	temp := 5.0
	dashboard := utility.DashboardResponse{
		Features: utility.DashboardFeatures{
			Temperature: &temp,
		},
	}

	h := &Handler{Client: client}
	h.triggerThresholdWebhooks(context.Background(), "NO", dashboard)

	assert.Equal(t, "THRESHOLD", received.Event)
	assert.Equal(t, 5.0, received.Details.MeasuredValue)
}

func TestCheckThreshold(t *testing.T) {
	tests := []struct {
		operator string
		measured float64
		value    float64
		expected bool
	}{
		{">", 10.0, 5.0, true},
		{">", 3.0, 5.0, false},
		{"<", 3.0, 5.0, true},
		{">=", 5.0, 5.0, true},
		{"<=", 5.0, 5.0, true},
		{"=", 5.0, 5.0, true},
		{"=", 5.1, 5.0, false},
		{"invalid", 5.0, 5.0, false},
	}

	for _, tt := range tests {
		result := checkThreshold(tt.operator, tt.measured, tt.value)
		if result != tt.expected {
			t.Errorf("checkThreshold(%s, %v, %v) = %v, want %v",
				tt.operator, tt.measured, tt.value, result, tt.expected)
		}
	}
}

func TestGetMeasuredValuePositive(t *testing.T) {
	temp := 31.29
	perc := 5.00
	dashboard := utility.DashboardResponse{
		Features: utility.DashboardFeatures{
			Temperature:   &temp,
			Precipitation: &perc,
			AirQuality: &utility.AirQuality{
				PM25: 225.67,
				PM10: 89.0,
			},
		},
	}

	tests := []struct {
		field    string
		expected float64
		ok       bool
	}{
		{"temperature", 31.29, true},
		{"pm25", 225.67, true},
		{"pm10", 89.0, true},
		{"precipitation", 5.00, true},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		val, ok := getMeasuredValue(tt.field, dashboard)
		if ok != tt.ok || val != tt.expected {
			t.Errorf("getMeasuredValue(%s) = (%v, %v), want (%v, %v)",
				tt.field, val, ok, tt.expected, tt.ok)
		}
	}
}

func TestGetMeasuredValueNegative(t *testing.T) {
	dashboard := utility.DashboardResponse{
		Features: utility.DashboardFeatures{
			AirQuality: &utility.AirQuality{
				PM25: 225.67,
				PM10: 89.0,
			},
		},
	}

	tests := []struct {
		field    string
		expected float64
		ok       bool
	}{
		{"temperature", 0, false},
		{"precipitation", 0, false},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		val, ok := getMeasuredValue(tt.field, dashboard)
		if ok != tt.ok || val != tt.expected {
			t.Errorf("getMeasuredValue(%s) = (%v, %v), want (%v, %v)",
				tt.field, val, ok, tt.expected, tt.ok)
		}
	}
}

func TestMatchesCountry(t *testing.T) {
	tests := []struct {
		webhookCountry string
		country        string
		expected       bool
	}{
		{"NO", "NO", false},
		{"NO", "IN", true},
		{"", "NO", false},
	}

	for _, tt := range tests {
		webhook := utility.RegisterWebhook{Country: tt.webhookCountry}
		result := matchesCountry(webhook, tt.country)
		if result != tt.expected {
			t.Errorf("matchesCountry(%s, %s) = %v, want %v",
				tt.webhookCountry, tt.country, result, tt.expected)
		}
	}
}
