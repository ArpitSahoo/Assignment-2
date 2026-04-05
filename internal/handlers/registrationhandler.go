package handlers

import (
	"assignment-2/internal/utility"
	"context"
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

// addRegistrationDocImpl stores a registration document, returns a generated ID.
// Defined as variable to allow for replacement in tests.
var addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
	ref, _, err := client.Collection(utility.RegistrationsCollection).Add(ctx, reg)
	if err != nil {
		return "", err
	}
	return ref.ID, nil
}

// getRegistrationDoc retrieves a registration document registered to a specific
// registration ID. Defined as variable to allow for replacement in tests.
var getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
	_, err := client.Collection(utility.RegistrationsCollection).Doc(id).Get(ctx)
	return err
}

// setRegistrationDoc updates a registration document registered to a specific
// registration ID. Defined as variable to allow for replacement in tests.
var setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
	_, err := client.Collection(utility.RegistrationsCollection).Doc(id).Set(ctx, reg)
	return err
}

// deleteRegistrationDoc deletes a registration document registered to a specific
// registration ID. Defined as variable to allow for replacement in tests.
var deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
	_, err := client.Collection(utility.RegistrationsCollection).Doc(id).Delete(ctx)
	return err
}

// HandleRegReq routes incoming HTTP requests to the appropriate handler method
// based on the request method. Supports POST for adding registrations and GET
// for retrieving registrations. Responds with 405 Method Not Allowed for unsupported methods.
func (h *Handler) HandleRegReq(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.addRegistration(w, r)
	case http.MethodGet:
		h.handleAllGetRegistration(w, r)
	case http.MethodPut:
		h.replaceRegistration(w, r)
	case http.MethodDelete:
		h.deleteRegistration(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// addRegistration handles POST requests to the /registrations endpoint.
// It decodes, normalizes and validates the request body, stores a dashboard configuration in
// Firestore, increments the local registration counter and returns the generated
// registration ID and last-change timestamp.
func (h *Handler) addRegistration(w http.ResponseWriter, r *http.Request) {
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

	id, err := addRegistrationDocImpl(ctx, h.Client, map[string]any{
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

	log.Printf("Registration created with ID: %s", id)
	// Increase the local registration count by one instance
	h.RegistrationCount.Add(1)

	// Define the key and value of the header map
	w.Header().Set("Content-Type", "application/json")
	// Respond with a status code of 201
	w.WriteHeader(http.StatusCreated)

	// Encode the response body as JSON
	encodeRegResp(w, id, lastChange)

}

// handleAllGetRegistration handles all the GET requests to the /registrations endpoint.
// It checks if the request is a HEAD or GET request, and if an ID is provided in the path.
// If it's a HEAD request, it calls the handleHead method to return only headers.
// If it's a GET request without an ID, it retrieves all registrations. If an ID is provided,
// it validates the ID and retrieves the specific registration by ID.
func (h *Handler) handleAllGetRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)
	docID := strings.TrimSpace(r.PathValue("id"))
	w.Header().Set("Content-Type", "application/json")

	// HEAD → return headers only
	if r.Method == http.MethodHead {
		h.handleHead(w, r, docID)
		return
	}

	// GET logic
	if docID == "" {
		h.GetAllRegistrations(w, r)
		return
	}

	// If an ISO code is provided, validate it before querying Firestore
	if len(docID) != 20 {
		http.Error(w, "Invalid document ID, the document must be 20 characters", http.StatusBadRequest)
		return
	}

	h.GetRegistrationByID(w, r, docID)
}

// GetAllRegistrations retrieves all registration documents from Firestore and returns them as a
// JSON array in the response body. It iterates through all documents in the "registrations" collection, 4
// collects their data into a slice of maps, and encodes the result as JSON.
// If an error occurs during retrieval, it responds with a 500 Internal Server Error.
func (h *Handler) GetAllRegistrations(w http.ResponseWriter, r *http.Request) {
	ctx := firestoreContext(r)
	iter := h.Client.Collection(utility.RegistrationsCollection).Documents(ctx)
	defer iter.Stop()

	// Iterate through all documents and collect their data into a slice of maps
	var results []map[string]interface{}

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			http.Error(w, "Error retrieving data", http.StatusInternalServerError)
			return
		}
		results = append(results, doc.Data())
	}
	_ = json.NewEncoder(w).Encode(results)
}

// handleHead writes only HTTP headers for a HEAD request without returning a body.
// It performs the same validation and lookup logic as the GET handler to determine
// the appropriate status code, but intentionally omits writing any response body,
// as required by the HTTP HEAD method.
// This method mirrors the validation and lookup logic of the GET handler,
// but intentionally omits writing any response body, as required by the HTTP HEAD method.
func (h *Handler) handleHead(w http.ResponseWriter, r *http.Request, docID string) {
	docID = strings.TrimSpace(docID)

	// Checks if the docID is empty, if empty it will return a status code of 200.
	if strings.TrimSpace(docID) == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := firestoreContext(r)

	// Checks if the document with the provided docID exists in Firestore.
	//If it does not exist, it will return a status code of 404.
	_, err := h.Client.Collection(utility.RegistrationsCollection).Doc(docID).Get(ctx)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Printf("Document %s does not exist", docID)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetRegistrationByID handles a GET request that retrieves registration data
// for a specific country identified by its document ID. It goes through firebase
// query and check if there is a similar Documents with that ID and returns the
// matching documents as a JSON. if not it returns a status code of 404 not found.
func (h *Handler) GetRegistrationByID(w http.ResponseWriter, r *http.Request, docID string) {
	ctx := firestoreContext(r)

	// Query Firestore for documents where the "docID" field matches the provided id
	docResults, err := h.Client.Collection(utility.RegistrationsCollection).Doc(docID).Get(ctx)
	if err != nil {
		http.Error(w, "Registration not found", http.StatusNotFound)
		log.Printf("Registration with ID %s not found: %v", docID, err)
		return
	}

	// Encode the matching documents as JSON in the response body
	_ = json.NewEncoder(w).Encode(docResults.Data())
}

// replaceRegistration handles PUT requests to the /registrations/{id} endpoint.
// It decodes, normalizes and validates the request body, replaces the stored registration
// configuration for the given ID, updates the last-change timestamp, and returns
// an empty body.
func (h *Handler) replaceRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)
	id := strings.TrimSpace(r.PathValue("id"))
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
	errGet := getRegistrationDoc(ctx, h.Client, id)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		http.Error(w, "failed getting registration", http.StatusNotFound)
		return
	}

	// Updates registration for the specific ID
	errSet := setRegistrationDoc(ctx, h.Client, id, map[string]any{
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

// deleteRegistration handles DELETE requests to the /registrations/{id} endpoint.
// Validates the registration ID, ensures the registration exists, then deletes it
// from FireStore.
func (h *Handler) deleteRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "invalid registration id", http.StatusBadRequest)
		return
	}

	ctx := firestoreContext(r)

	// Retrieve the registration connected to the ID
	errGet := getRegistrationDoc(ctx, h.Client, id)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		http.Error(w, "registration not found", http.StatusNotFound)
		return
	}

	// Delete the registration connected to the ID
	errDel := deleteRegistrationDoc(ctx, h.Client, id)
	if errDel != nil {
		log.Printf("Failed to delete registration: %v", errDel)
		http.Error(w, "failed deleting registration", http.StatusInternalServerError)
		return
	}
	log.Printf("Deleted registration with ID: %s", id)
	w.WriteHeader(http.StatusNoContent)
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
func encodeRegResp(w http.ResponseWriter, id, lastChange string) {
	if err := json.NewEncoder(w).Encode(utility.RegistrationResponse{
		ID:         id,
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

	for _, m := range regReq.Country {
		if !unicode.IsLetter(m) {
			http.Error(w, "Country must contain only letters", http.StatusBadRequest)
			return true
		}
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
