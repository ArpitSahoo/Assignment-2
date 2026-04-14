package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"assignment-2/internal/handlers"
	"assignment-2/internal/utility"

	"github.com/stretchr/testify/assert"
)

func TestAPIKeyMiddleware_TableDrivenPositive(t *testing.T) {
	origValidate := handlers.ValidateAPIKeyImpl
	defer func() { handlers.ValidateAPIKeyImpl = origValidate }()

	// create a minimal handler instance to pass into middleware (middleware uses handlers.ValidateAPIKeyImpl)
	h := &handlers.Handler{}

	tests := []struct {
		name        string
		path        string
		method      string
		headerKey   string
		headerValue string
		validateFn  func(h *handlers.Handler, r *http.Request, rawKey string) (bool, error)
		wantStatus  int
	}{
		{
			name:       "allow status without key",
			path:       utility.StatusPath,
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "allow auth create without key",
			path:       utility.AuthPath,
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:        "valid key allows next handler",
			path:        utility.RegistrationPath,
			method:      http.MethodGet,
			headerKey:   utility.APIKeyHeader,
			headerValue: "good-key",
			validateFn: func(h *handlers.Handler, r *http.Request, rawKey string) (bool, error) {
				if rawKey == "good-key" {
					return true, nil
				}
				return false, nil
			},
			wantStatus: http.StatusOK,
		},
	}

	runApiMiddlewareTest(t, tests, h)
}

func TestAPIKeyMiddleware_TableDrivenNegative(t *testing.T) {
	origValidate := handlers.ValidateAPIKeyImpl
	defer func() { handlers.ValidateAPIKeyImpl = origValidate }()

	// create a minimal handler instance to pass into middleware (middleware uses handlers.ValidateAPIKeyImpl)
	h := &handlers.Handler{}

	tests := []struct {
		name        string
		path        string
		method      string
		headerKey   string
		headerValue string
		validateFn  func(h *handlers.Handler, r *http.Request, rawKey string) (bool, error)
		wantStatus  int
	}{
		{
			name:       "missing key returns 401",
			path:       utility.RegistrationPath,
			method:     http.MethodGet,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "invalid key returns 403",
			path:        utility.RegistrationPath,
			method:      http.MethodGet,
			headerKey:   utility.APIKeyHeader,
			headerValue: "bad-key",
			validateFn: func(h *handlers.Handler, r *http.Request, rawKey string) (bool, error) {
				return false, nil
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:        "validation error returns 500",
			path:        utility.RegistrationPath,
			method:      http.MethodGet,
			headerKey:   utility.APIKeyHeader,
			headerValue: "some-key",
			validateFn: func(h *handlers.Handler, r *http.Request, rawKey string) (bool, error) {
				return false, assert.AnError // use a non-nil error to force 500
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	runApiMiddlewareTest(t, tests, h)
}

func runApiMiddlewareTest(t *testing.T, tests []struct {
	name        string
	path        string
	method      string
	headerKey   string
	headerValue string
	validateFn  func(h *handlers.Handler, r *http.Request, rawKey string) (bool, error)
	wantStatus  int
}, h *handlers.Handler) {
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// set validate hook if provided; otherwise default to original behavior (should not be called)
			if tc.validateFn != nil {
				handlers.ValidateAPIKeyImpl = func(h *handlers.Handler, r *http.Request, rawKey string) (bool, error) {
					return tc.validateFn(h, r, rawKey)
				}
			} else {
				handlers.ValidateAPIKeyImpl = func(h *handlers.Handler, r *http.Request, rawKey string) (bool, error) {
					// default: return false to catch unexpected calls
					return false, nil
				}
			}

			mw := APIKeyMiddleware(h)
			// next handler writes OK so we can detect middleware pass-through
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("next"))
			})
			handler := mw(next)

			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBuffer(nil))
			if tc.headerKey != "" {
				req.Header.Set(tc.headerKey, tc.headerValue)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code, "body=%s", rr.Body.String())
		})
	}
}
