package handlers

import (
	"assignment-2/internal/utility"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Handler holding shared dependencies and local registration counter.
type Handler struct {
	Client            *firestore.Client
	RegistrationCount atomic.Int64
}

// HandleMessage routes incoming HTTP requests to the appropriate handler method
// based on the request method. Supports POST for adding registrations and GET
// for retrieving registrations. Responds with 405 Method Not Allowed for unsupported methods.
func (h *Handler) HandleMessage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.AddRegistration(w, r)
	case http.MethodGet:
		h.RetrieveAllRegistrations(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AddRegistration handles POST requests to the /registrations endpoint.
// Validates and normalizes the request body, stores a dashboard configuration in
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

	var regReq utility.RegistrationRequest
	// Decode the original request
	if decodeRegReq(w, r, &regReq) {
		return
	}
	// Apply normalization on the original request
	normalizeFields(&regReq)

	if validateRegReq(w, regReq) {
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
	encodeRegResp(w, ref, lastChange)

}

// RetrieveAllRegistrations handles the GET request to the /registrations endpoint.
// If an ISO code is provided in the URL path, it retrieves the registration(s) matching that ISO code.0
// If no ISO code is provided, it retrieves all registrations.
// The results are returned as a JSON array. If no matching documents are found,
// it responds with a 404 Not Found status.
func (h *Handler) RetrieveAllRegistrations(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)

	isoCode := strings.ToUpper(strings.TrimSpace(r.PathValue("id")))

	ctx := r.Context()

	w.Header().Set("Content-Type", "application/json")

	// GET ALL if no ISO code is given
	if len(isoCode) == 0 {
		iter := h.Client.Collection(utility.RegistrationsCollection).Documents(ctx)
		defer iter.Stop()

		var results []map[string]interface{}

		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				log.Printf("Error iterating documents: %v", err)
				http.Error(w, "Error retrieving data", http.StatusInternalServerError)
				return
			}
			results = append(results, doc.Data())
		}

		_ = json.NewEncoder(w).Encode(results)
		return
	} else if len(isoCode) != 2 {
		http.Error(w, "Invalid iso code", http.StatusBadRequest)
		log.Printf("Invalid iso code: %s", isoCode)
		return
	}

	// GET BY ISO CODE
	iter := h.Client.Collection(utility.RegistrationsCollection).
		Where("isoCode", "==", isoCode).
		Documents(ctx)
	defer iter.Stop()

	var results []map[string]interface{}

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			log.Printf("Error retrieving document with isoCode %s: %v", isoCode, err)
			http.Error(w, "Error retrieving data", http.StatusInternalServerError)
			return
		}
		results = append(results, doc.Data())
	}

	if len(results) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		log.Printf("Document not found: %s", isoCode)
		return
	}

	_ = json.NewEncoder(w).Encode(results)
}

// decodeRegReq decodes the JSON request body into a registration request.
// Writes a HTTP error response if decoding fails.
func decodeRegReq(w http.ResponseWriter, r *http.Request, regReq *utility.RegistrationRequest) bool {
	if err := json.NewDecoder(r.Body).Decode(regReq); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
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

// validateRegReq validates registration fields and writes a HTTP error
// response if validation fails.
func validateRegReq(w http.ResponseWriter, regReq utility.RegistrationRequest) bool {
	if len(regReq.IsoCode) != 2 {
		http.Error(w, "ISO-code must be 2-letter country code", http.StatusBadRequest)
		return true
	}

	for _, l := range regReq.IsoCode {
		if !unicode.IsLetter(l) {
			http.Error(w, "ISO-code must contain only letters", http.StatusBadRequest)
			return true
		}
	}

	if regReq.Country == "" {
		http.Error(w, "Missing country", http.StatusBadRequest)
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
