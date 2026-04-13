package handlers

import (
	"assignment-2/internal/utility"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyMiddleware_AllowsStatusWithoutKey(t *testing.T) {
	old := validateAPIKeyImpl
	validateAPIKeyImpl = func(h *Handler, r *http.Request, rawKey string) (bool, error) {
		t.Fatal("validator should not be called for status path")
		return false, nil
	}
	defer func() { validateAPIKeyImpl = old }()

	h := &Handler{}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, utility.StatusPath, nil)
	rr := httptest.NewRecorder()

	h.APIKeyMiddleware(next).ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestAPIKeyMiddleware_AllowsPostAuthWithoutKey(t *testing.T) {
	old := validateAPIKeyImpl
	validateAPIKeyImpl = func(h *Handler, r *http.Request, rawKey string) (bool, error) {
		t.Fatal("validator should not be called for POST /auth/")
		return false, nil
	}
	defer func() { validateAPIKeyImpl = old }()

	h := &Handler{}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, utility.AuthPath, nil)
	rr := httptest.NewRecorder()

	h.APIKeyMiddleware(next).ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestAPIKeyMiddleware_MissingKeyReturns401(t *testing.T) {
	old := validateAPIKeyImpl
	validateAPIKeyImpl = func(h *Handler, r *http.Request, rawKey string) (bool, error) {
		t.Fatal("validator should not be called without key")
		return false, nil
	}
	defer func() { validateAPIKeyImpl = old }()

	h := &Handler{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/dashboards/7f3a91bc04e2d158", nil)
	rr := httptest.NewRecorder()

	h.APIKeyMiddleware(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAPIKeyMiddleware_InvalidKeyReturns403(t *testing.T) {
	old := validateAPIKeyImpl
	validateAPIKeyImpl = func(h *Handler, r *http.Request, rawKey string) (bool, error) {
		if rawKey != "sk-envdash-test" {
			t.Fatalf("expected raw key to be passed through, got %q", rawKey)
		}
		return false, nil
	}
	defer func() { validateAPIKeyImpl = old }()

	h := &Handler{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/dashboards/7f3a91bc04e2d158", nil)
	req.Header.Set(utility.APIKeyHeader, "sk-envdash-test")
	rr := httptest.NewRecorder()

	h.APIKeyMiddleware(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestAPIKeyMiddleware_ValidKeyCallsNext(t *testing.T) {
	old := validateAPIKeyImpl
	validateAPIKeyImpl = func(h *Handler, r *http.Request, rawKey string) (bool, error) {
		return true, nil
	}
	defer func() { validateAPIKeyImpl = old }()

	h := &Handler{}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/dashboards/7f3a91bc04e2d158", nil)
	req.Header.Set(utility.APIKeyHeader, "sk-envdash-valid")
	rr := httptest.NewRecorder()

	h.APIKeyMiddleware(next).ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}
