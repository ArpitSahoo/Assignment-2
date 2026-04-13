package handlers

import (
	"assignment-2/internal/utility"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	cloudfirestore "cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestFirestoreClient creates a new Firestore client for testing, configured to connect to the Firestore emulator
// if the FIRESTORE_EMULATOR_HOST environment variable is set. If the variable is not set, the test will be skipped.
func newTestFirestoreClient(t *testing.T) *cloudfirestore.Client {
	t.Helper()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "test-project"
	}

	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set; skipping Firestore integration test")
	}

	client, err := cloudfirestore.NewClient(context.Background(), projectID)
	require.NoError(t, err, "failed to create firestore client")

	t.Cleanup(func() {
		_ = client.Close()
	})

	return client
}

// clearFirestoreEmulator sends a DELETE request to the Firestore emulator's REST API to clear all documents in the default database.
func clearFirestoreEmulator(t *testing.T) {
	t.Helper()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "test-project"
	}

	host := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if host == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set; skipping Firestore integration test")
	}

	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("http://%s/emulator/v1/projects/%s/databases/(default)/documents", host, projectID),
		nil,
	)
	require.NoError(t, err, "failed to create emulator cleanup request")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to clear firestore emulator")
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Logf("Error closing response body: %v", err)
			return
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected cleanup status")
}

func newTestHandler(t *testing.T) *HandlerWebhooks {
	t.Helper()

	clearFirestoreEmulator(t)

	return &HandlerWebhooks{
		Client: newTestFirestoreClient(t),
	}
}

func createWebhookThroughHandler(t *testing.T, h *HandlerWebhooks, body string) utility.WebhookResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, utility.NotificationPath, bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.WebhookHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())

	var resp utility.WebhookResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp), "failed to decode webhook response")
	assert.NotEmpty(t, resp.ID, "expected webhook ID in response")

	return resp
}

func TestWebhookHandler_RegisterWebhook_Integration(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, utility.NotificationPath, bytes.NewBufferString(`{
		"url":"https://webhook.site/1",
		"country":"NO",
		"event":"REGISTER"
	}`))
	rr := httptest.NewRecorder()

	h.WebhookHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())

	var resp utility.WebhookResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp), "failed to decode response")
	assert.NotEmpty(t, resp.ID, "expected webhook ID to be set")
}

func TestValidateWebhook_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		webhook    utility.RegisterWebhook
		wantFailed bool
		wantStatus int
	}{
		{
			name: "missing url",
			webhook: utility.RegisterWebhook{
				Event: "REGISTER",
			},
			wantFailed: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid event",
			webhook: utility.RegisterWebhook{
				Url:   "https://https://webhook.site/1",
				Event: "WRONG",
			},
			wantFailed: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "threshold provided for non-threshold event",
			webhook: utility.RegisterWebhook{
				Url:   "https://https://webhook.site/1",
				Event: "REGISTER",
				Threshold: &utility.Threshold{
					Field:    "pm25",
					Operator: ">",
					Value:    10,
				},
			},
			wantFailed: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "threshold event missing threshold block",
			webhook: utility.RegisterWebhook{
				Url:   "https://https://webhook.site/1",
				Event: "THRESHOLD",
			},
			wantFailed: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid threshold field",
			webhook: utility.RegisterWebhook{
				Url:   "https://https://webhook.site/1",
				Event: "THRESHOLD",
				Threshold: &utility.Threshold{
					Field:    "nok",
					Operator: ">",
					Value:    10,
				},
			},
			wantFailed: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid threshold operator",
			webhook: utility.RegisterWebhook{
				Url:   "https://https://webhook.site/1",
				Event: "THRESHOLD",
				Threshold: &utility.Threshold{
					Field:    "pm25",
					Operator: "=",
					Value:    10,
				},
			},
			wantFailed: true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "valid register webhook",
			webhook: utility.RegisterWebhook{
				Url:     "https://https://webhook.site/1",
				Country: "NO",
				Event:   "REGISTER",
			},
			wantFailed: false,
			wantStatus: http.StatusOK,
		},
		{
			name: "valid threshold webhook",
			webhook: utility.RegisterWebhook{
				Url:     "https://https://webhook.site/1",
				Country: "NO",
				Event:   "THRESHOLD",
				Threshold: &utility.Threshold{
					Field:    "pm25",
					Operator: ">",
					Value:    20,
				},
			},
			wantFailed: false,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			gotFailed := validateWebhook(rr, tt.webhook)

			assert.Equal(t, tt.wantFailed, gotFailed)

			if tt.wantFailed {
				assert.Equal(t, tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestWebhookHandler_RegisterWebhook_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, utility.NotificationPath, bytes.NewBufferString(`{"url":`))
	rr := httptest.NewRecorder()

	h.WebhookHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, "body=%s", rr.Body.String())
}

func TestWebhookHandler_GetAllWebhooks_Integration(t *testing.T) {
	h := newTestHandler(t)

	createWebhookThroughHandler(t, h, `{
		"url":"https://webhook.site/1",
		"country":"NO",
		"event":"REGISTER"
	}`)

	createWebhookThroughHandler(t, h, `{
		"url":"https://webhook.site/2",
		"country":"SE",
		"event":"CHANGE"
	}`)

	req := httptest.NewRequest(http.MethodGet, utility.NotificationPath, nil)
	rr := httptest.NewRecorder()

	h.WebhookHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())

	var got []utility.RegisterWebhook
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got), "failed to decode response")
	assert.Len(t, got, 2)
}

func TestWebhookIDHandler_GetWebhookByID_Integration(t *testing.T) {
	h := newTestHandler(t)

	created := createWebhookThroughHandler(t, h, `{
		"url":"https://webhook.site/1",
		"country":"NO",
		"event":"REGISTER"
	}`)

	req := httptest.NewRequest(http.MethodGet, utility.NotificationPath+"/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()

	h.WebhookIDHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())

	var got utility.RegisterWebhook
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got), "failed to decode response")

	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "https://webhook.site/1", got.Url)
}

func TestWebhookIDHandler_GetWebhookByID_NotFound(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, utility.NotificationPath+"/missing-id", nil)
	req.SetPathValue("id", "missing-id")
	rr := httptest.NewRecorder()

	h.WebhookIDHandler(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code, "body=%s", rr.Body.String())
}

func TestWebhookIDHandler_DeleteWebhook_Integration(t *testing.T) {
	h := newTestHandler(t)

	created := createWebhookThroughHandler(t, h, `{
		"url":"https://webhook.site/1",
		"country":"NO",
		"event":"REGISTER"
	}`)

	delReq := httptest.NewRequest(http.MethodDelete, utility.NotificationPath+"/"+created.ID, nil)
	delReq.SetPathValue("id", created.ID)
	delRR := httptest.NewRecorder()

	h.WebhookIDHandler(delRR, delReq)

	assert.Equal(t, http.StatusNoContent, delRR.Code, "body=%s", delRR.Body.String())

	getReq := httptest.NewRequest(http.MethodGet, utility.NotificationPath+"/"+created.ID, nil)
	getReq.SetPathValue("id", created.ID)
	getRR := httptest.NewRecorder()

	h.WebhookIDHandler(getRR, getReq)

	assert.Equal(t, http.StatusNotFound, getRR.Code)
}

func TestWebhookIDHandler_DeleteWebhook_NotFound(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, utility.NotificationPath+"/no-id", nil)
	req.SetPathValue("id", "no-id")
	rr := httptest.NewRecorder()

	h.WebhookIDHandler(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code, "body=%s", rr.Body.String())
}

func TestWebhookHandler_MethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPut, utility.NotificationPath, nil)
	rr := httptest.NewRecorder()

	h.WebhookHandler(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestWebhookIDHandler_MethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, utility.NotificationPath+"/nok", nil)
	req.SetPathValue("id", "nok")
	rr := httptest.NewRecorder()

	h.WebhookIDHandler(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}
