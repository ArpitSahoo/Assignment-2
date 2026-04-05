package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
*/

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
