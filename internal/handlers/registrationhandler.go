package handlers

import (
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"cloud.google.com/go/firestore"
)

// Handler holding shared dependencies and local registration counter.
type Handler struct {
	Client            *firestore.Client
	RegistrationCount atomic.Int64
}

// AddRegistration handles POST requests to the /registrations endpoint.
// It decodes, normalizes and validates the request body, stores a dashboard configuration in
// Firestore, increments the local registration counter and returns the generated
// registration ID and last-change timestamp.
func (h *Handler) AddRegistration(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(r.Body)

	log.Printf("Received %s request", r.Method)

	// Decode, normalize and validate
	regReq, ok := parseRegReq(w, r)
	if !ok {
		return
	}

	ctx := firestoreContext(r)
	lastChange := currentLastChange()

	// Store registration payload in Firestore and let Firestore generate document ID.
	ref, _, err := h.Client.Collection(utility.RegistrationsCollection).Add(ctx, map[string]any{
		"country":    regReq.Country,
		"isoCode":    regReq.IsoCode,
		"features":   regReq.Features,
		"lastChange": lastChange,
	})
	if err != nil {
		log.Printf("Failed to add registration: %v", err)
		http.Error(w, "failed to add registration", http.StatusInternalServerError)
		return
	}

	log.Printf("Registration created with ID: %s", ref.ID)
	// Increase the local registration count by one instance
	h.RegistrationCount.Add(1)

	// Define the key and value of the header map
	w.Header().Set("Content-Type", "application/json")
	// Respond with a status code of 201
	w.WriteHeader(http.StatusCreated)

	// Encode the response body as JSON
	encodeRegResp(w, ref, lastChange)

}

// ReplaceRegistration handles PUT requests to the /registrations/{id} endpoint.
// It decodes, normalizes and validates the request body, replaces the stored registration
// configuration for the given ID, updates the last-change timestamp, and returns
// an empty body.
func (h *Handler) ReplaceRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "invalid registration id", http.StatusBadRequest)
		return
	}

	// Decode, normalize and validate
	regReq, ok := parseRegReq(w, r)
	if !ok {
		return
	}

	ctx := firestoreContext(r)
	lastChange := currentLastChange()

	// Gets a stored registration for specific ID
	_, errGet := h.Client.Collection(utility.RegistrationsCollection).Doc(id).Get(ctx)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		http.Error(w, "failed getting registration", http.StatusNotFound)
		return
	}

	// Updates registration for the specific ID
	_, errSet := h.Client.Collection(utility.RegistrationsCollection).Doc(id).Set(ctx, map[string]any{
		"country":    regReq.Country,
		"isoCode":    regReq.IsoCode,
		"features":   regReq.Features,
		"lastChange": lastChange,
	})
	if errSet != nil {
		log.Printf("Failed to update registration: %v", errSet)
		http.Error(w, "failed updating registration", http.StatusInternalServerError)
		return
	}

	log.Printf("Replaced registration with ID: %s", id)
	// Respond with status code 200
	w.WriteHeader(http.StatusOK)
}

// decodeRegReq decodes the JSON request body into a registration request.
// Writes a HTTP error response if decoding fails.
func decodeRegReq(w http.ResponseWriter, r *http.Request, regReq *utility.RegistrationRequest) bool {
	if err := json.NewDecoder(r.Body).Decode(regReq); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return true
	}
	return false
}

// encodeRegResp encodes a registration response as JSON and logs an error if
// encoding fails.
func encodeRegResp(w http.ResponseWriter, ref *firestore.DocumentRef, lastChange string) {
	if err := json.NewEncoder(w).Encode(utility.RegistrationResponse{
		ID:         ref.ID,
		LastChange: lastChange,
	}); err != nil {
		log.Printf("Failed to encode registration response: %v", err)
	}
}

// currentLastChange keeps track of time and date of which a registration
// has been created or updated.
func currentLastChange() string {
	lastChange := time.Now().Format(time.RFC3339)
	return lastChange
}

// firestoreContext returns the request context used for Firestore operations.
func firestoreContext(r *http.Request) context.Context {
	return r.Context()
}

// validateRegReq validates registration fields and writes a HTTP error
// response if validation fails.
func validateRegReq(w http.ResponseWriter, regReq utility.RegistrationRequest) bool {
	if len(regReq.IsoCode) != 2 {
		http.Error(w, "iso-code must be 2-letter country code", http.StatusBadRequest)
		return true
	}

	for _, l := range regReq.IsoCode {
		if !unicode.IsLetter(l) {
			http.Error(w, "iso-code must contain only letters", http.StatusBadRequest)
			return true
		}
	}

	if regReq.Country == "" {
		http.Error(w, "missing country", http.StatusBadRequest)
		return true
	}

	return false
}

// normalizeFields normalizes registration request fields into consistent
// format before validation or storage.
func normalizeFields(regReq *utility.RegistrationRequest) {
	regReq.Country = strings.TrimSpace(regReq.Country)
	regReq.IsoCode = strings.ToUpper(strings.TrimSpace(regReq.IsoCode))

	for i, c := range regReq.Features.TargetCurrencies {
		regReq.Features.TargetCurrencies[i] = strings.ToUpper(strings.TrimSpace(c))
	}
}

// parseRegReq decodes, normalizes and validates a registration request.
// Returns the parsed request, false if any step fails.
func parseRegReq(w http.ResponseWriter, r *http.Request) (utility.RegistrationRequest, bool) {
	var regReq utility.RegistrationRequest

	if decodeRegReq(w, r, &regReq) {
		return utility.RegistrationRequest{}, false
	}

	normalizeFields(&regReq)

	if validateRegReq(w, regReq) {
		return utility.RegistrationRequest{}, false
	}
	return regReq, true
}
