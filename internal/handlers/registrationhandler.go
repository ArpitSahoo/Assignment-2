package handlers

import (
	"assignment-2/internal/utility"
	"encoding/json"
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

// addRegistration handles POST requests to the /registration endpoint.
// Validates and normalizes the request body, stores a dashboard configuration in
// Firestore, increments the local registration counter and returns the generated
// registration ID and last-change timestamp.
// TODO: consider creating a validation function
func (h *Handler) addRegistration(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	log.Printf("Received %s request", r.Method)

	var regReq utility.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	// Apply normalization on the original request
	normalizeFields(&regReq)

	if len(regReq.IsoCode) != 2 {
		http.Error(w, "ISO-code must be 2-letter country code", http.StatusBadRequest)
		return
	}

	for _, l := range regReq.IsoCode {
		if !unicode.IsLetter(l) {
			http.Error(w, "ISO-code must contain only letters", http.StatusBadRequest)
			return
		}
	}

	if regReq.Country == "" {
		http.Error(w, "Missing country", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	lastChange := time.Now().Format(time.RFC3339)

	// Store registration payload in Firestore and let Firestore generate document ID.
	ref, _, err := h.Client.Collection(utility.RegistrationsCollection).Add(ctx, map[string]any{
		"country":    regReq.Country,
		"isoCode":    regReq.IsoCode,
		"features":   regReq.Features,
		"lastChange": lastChange,
	})
	if err != nil {
		log.Printf("Failed to add registration: %v", err)
		http.Error(w, "Failed to add registration", http.StatusInternalServerError)
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
	_ = json.NewEncoder(w).Encode(utility.RegistrationResponse{
		ID:         ref.ID,
		LastChange: lastChange,
	})

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
