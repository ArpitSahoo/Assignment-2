package handlers

import (
	"assignment-2/internal/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleStatus_MethodNotAllowed(t *testing.T) {
	h := &Handler{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/status", nil)

	h.HandleStatus(w, r)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCheckAPIStatusPositive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	got := checkAPIStatus(srv.URL, "1")
	assert.Equal(t, 200, got)
}

func TestCheckAPIStatusNegative(t *testing.T) {
	got := checkAPIStatus("", "")
	assert.Equal(t, 500, got)
}

func TestCheckAPIStatus_SetsAPIKeyHeader(t *testing.T) {
	var key bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = r.Header.Get("X-Api-Key") != ""
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checkAPIStatus(srv.URL, "key")

	assert.True(t, key)
}

func TestHandleStatus_FirestoreReachable(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	h := &Handler{Client: client}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/status", nil)

	h.HandleStatus(w, r)

	var body models.StatusResponse
	err := json.NewDecoder(w.Result().Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, body.NotificationDB)
}

func TestHandleStatus_WebhookCount(t *testing.T) {
	client := newTestFirestoreClient(t)
	clearFirestoreEmulator(t)

	h := &Handler{Client: client}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/status", nil)

	h.HandleStatus(w, r)

	var body models.StatusResponse
	err := json.NewDecoder(w.Result().Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, 0, body.Webhooks)
}

func TestHandleStatus_WebhookCount_Error(t *testing.T) {
	client := newTestFirestoreClient(t)
	err := client.Close()
	if err != nil {
		return
	}

	h := &Handler{Client: client}

	count := h.getWebhookCount()

	assert.Equal(t, -1, count)
}
