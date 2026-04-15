package handlers

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuthHandler(t *testing.T) *Handler {
	t.Helper()

	clearFirestoreEmulator(t)

	return &Handler{
		Client: newTestFirestoreClient(t),
	}
}

func createAPIKeyHelper(t *testing.T, h *Handler, name, email string) (models.AuthenticationResponse, int) {
	t.Helper()
	body := fmt.Sprintf(`{"name":"%s","email":"%s"}`, name, email)
	req := httptest.NewRequest(http.MethodPost, utility.AuthPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.HandleAuthenticationReq(rr, req)

	var resp models.AuthenticationResponse
	// attempt to decode if body is JSON; ignore decode error for negative cases
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	return resp, rr.Code
}

func TestRevokeAPIKey_EmptyKey_ReturnsBadRequest(t *testing.T) {
	h := &Handler{} // no firestore client required because the handler returns early

	req := httptest.NewRequest(http.MethodDelete, utility.AuthPath+"{key}", nil)
	req.SetPathValue("key", "") // explicitly empty
	rr := httptest.NewRecorder()

	// call the public delegator (same as your other tests)
	h.HandleAuthenticationReq(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400; got %d; body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "API key not provided") {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestCreateAPIKey_BadRequests_TableDriven(t *testing.T) {
	// These exercises the validation branches that return before using Firestore.
	h := &Handler{} // Client may be nil because validation fails first

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantSubstr string
	}{
		{
			name:       "NoEmail",
			body:       `{"name":"test-client","email":""}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "name and email are required",
		},
		{
			name:       "NoName",
			body:       `{"name":"","email":"user@example.com"}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "name and email are required",
		},
		{
			name:       "BadEmailFormat",
			body:       `{"name":"test-client","email":"not-an-email"}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "invalid email",
		},
		{
			name:       "NoEmailDomain",
			body:       `{"name":"test-client","email":"user@"}`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "invalid email",
		},
		{
			name:       "badBody",
			body:       `"name":"test-client","email":"user@"`,
			wantStatus: http.StatusBadRequest,
			wantSubstr: "invalid request body",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, utility.AuthPath, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.HandleAuthenticationReq(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status: got %d want %d; body=%q", rr.Code, tc.wantStatus, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), tc.wantSubstr) {
				t.Fatalf("response body does not contain expected substring: got %q want contains %q", rr.Body.String(), tc.wantSubstr)
			}
		})
	}
}

func TestRevokeAPIKey_EmptyOrWhitespaceKey_Table(t *testing.T) {
	h := &Handler{}

	cases := []struct {
		name string
		key  string
	}{
		{"Empty", ""},
		{"Whitespace", "   "},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, utility.AuthPath+"{key}", nil)
			// If you want to simulate "not set", simply don't call SetPathValue.
			if tc.name != "MissingSet" {
				req.SetPathValue("key", tc.key)
			}
			rr := httptest.NewRecorder()

			h.HandleAuthenticationReq(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("case %s: expected 400, got %d; body=%q", tc.name, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestCreateAndValidateAPIKey_PositiveTest(t *testing.T) {
	h := newTestAuthHandler(t)

	resp, code := createAPIKeyHelper(t, h, "test-client", "test@example.com")
	require.Equal(t, http.StatusCreated, code, "create response body should be JSON with key")
	require.NotEmpty(t, resp.Key)

	// validate by calling the exported hook ValidateAPIKeyImpl (delegates to h.validateAPIKey)
	validateReq := httptest.NewRequest(http.MethodGet, "/", nil)
	valid, err := ValidateAPIKeyImpl(h, validateReq, resp.Key)
	require.NoError(t, err)
	assert.True(t, valid, "created key should validate")
}

func TestValidateWithWrongKey_ReturnsFalse(t *testing.T) {
	h := newTestAuthHandler(t)

	// create a valid key to ensure emulator is ready
	resp, code := createAPIKeyHelper(t, h, "test-client", "test@example.com")
	require.Equal(t, http.StatusCreated, code)
	require.NotEmpty(t, resp.Key)

	validateReq := httptest.NewRequest(http.MethodGet, "/", nil)
	valid, err := ValidateAPIKeyImpl(h, validateReq, resp.Key+"-wrong")
	require.NoError(t, err)
	assert.False(t, valid, "wrong key should not validate")
}

func TestRevokeAPIKeyHandlerSucceeds(t *testing.T) {
	h := newTestAuthHandler(t)
	resp, code := createAPIKeyHelper(t, h, "to-revoke-handler", "revoke-handler@example.com")
	require.Equal(t, http.StatusCreated, code)
	require.NotEmpty(t, resp.Key)

	delReq := httptest.NewRequest(http.MethodDelete, utility.AuthPath+"{key}", nil)
	delReq.SetPathValue("key", resp.Key)
	delRR := httptest.NewRecorder()
	h.HandleAuthenticationReq(delRR, delReq)
	assert.Equal(t, http.StatusNoContent, delRR.Code)
	assert.Equal(t, 0, delRR.Body.Len())
}

func TestRevokedKeyIsActuallyRemoved(t *testing.T) {
	h := newTestAuthHandler(t)
	resp, code := createAPIKeyHelper(t, h, "to-revoke-handler", "revoke-handler@example.com")
	require.Equal(t, http.StatusCreated, code)

	// revoke
	delReq := httptest.NewRequest(http.MethodDelete, utility.AuthPath+"{key}", nil)
	delReq.SetPathValue("key", resp.Key)
	delRR := httptest.NewRecorder()
	h.HandleAuthenticationReq(delRR, delReq)
	require.Equal(t, http.StatusNoContent, delRR.Code)

	// negative assertion
	validateReq := httptest.NewRequest(http.MethodGet, "/", nil)
	valid, err := ValidateAPIKeyImpl(h, validateReq, resp.Key)
	require.NoError(t, err)
	assert.False(t, valid)
}
