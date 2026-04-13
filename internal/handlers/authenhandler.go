package handlers

import (
	"assignment-2/internal/utility"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var validateAPIKeyImpl = func(h *Handler, r *http.Request, rawKey string) (bool, error) {
	return h.validateAPIKey(r, rawKey)
}

// HandleAuthenticationReq handles both creating and revoking API keys based on the HTTP method.
// POST creates a new API key, while DELETE revokes an existing key.
func (h *Handler) HandleAuthenticationReq(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createAPIKey(w, r)
	case http.MethodDelete:
		h.revokeAPIKey(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createAPIKey generates a new API key, stores its hash in Firestore, and returns the raw key to the client.
// It validates the request body for required fields and proper email format before proceeding.
// The raw key is only returned once and is not stored in plaintext anywhere for security reasons.
func (h *Handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
			http.Error(w, "Error closing body", http.StatusInternalServerError)
		}
	}(r.Body)

	var req utility.AuthenticationRequest // struct to decode the incoming JSON request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { // decode the JSON request body into the struct
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return // if decoding fails, return a 400 Bad Request error
	}

	req.Name = strings.TrimSpace(req.Name) // trim whitespace from the name and email fields
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" || req.Email == "" { // if name or email is empty after trimming, return a 400 Bad Request error
		http.Error(w, "name and email are required", http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil { // validate the email format using the net/mail package; if invalid, return a 400 Bad Request error
		http.Error(w, "invalid email. Please use a valid email", http.StatusBadRequest)
		return
	}

	rawKey, err := generateAPIKey() // generate a new API key using the generateAPIKey function; if it fails, return a 500 Internal Server Error
	if err != nil {
		http.Error(w, "failed to generate api key", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()   // a timestamp variable for the current time in UTC, used for the CreatedAt field in the APIKeyDoc
	doc := utility.APIKeyDoc{ // create a new APIKeyDoc struct to represent the API key document to be stored in Firestore
		Name:      req.Name,
		Email:     req.Email,
		Hash:      hashAPIKey(rawKey), // hash the raw API to store only the hash in Firestore for security reasons
		CreatedAt: now,
		Revoked:   false,
	}

	ctx := firestoreContext(r)
	_, err = h.Client.Collection(utility.APIKeysCollection).Doc(doc.Hash).Set(ctx, doc) // store the API key document in the Firestore
	if err != nil {                                                                     // if it fails, return a 500 Internal Server Error
		http.Error(w, "failed to store api key", http.StatusInternalServerError)
		return
	}

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusCreated)                             // create header and status code for the response
	_ = json.NewEncoder(w).Encode(utility.AuthenticationResponse{ // encode the response body as JSON
		Key:       rawKey,                       // return the raw API key
		CreatedAt: now.Format("20060102 15:04"), // return timestamp
	})
}

// revokeAPIKey revokes an existing API key by setting its "revoked" status to true in Firestore.
// It retrieves the API key from the URL path, validates its existence, and updates the document accordingly.
// If the key is not found or already revoked, it returns a 404 Not Found error.
// On successful revocation, it returns a 204 No Content status.
func (h *Handler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	rawKey := strings.TrimSpace(r.PathValue("key")) // extract the raw API key from the URL path and trim whitespace
	if rawKey == "" {                               // if the raw key is empty, return a 404 Not Found error
		http.Error(w, "api key not provided", http.StatusNotFound)
		return
	}

	hash := hashAPIKey(rawKey) // hash the raw API key to find the corresponding document in Firestore
	ctx := firestoreContext(r)
	ref := h.Client.Collection(utility.APIKeysCollection).Doc(hash) // get the document reference for the API key in Firestore
	doc, err := ref.Get(ctx)
	if err != nil { // if there is an error retrieving the document, check if it's a Not Found error; if so, return a 404 Not Found error
		if isNotFound(err) { // check if it's a Not Found error; if so, return a 404 Not Found error
			http.Error(w, "api key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to revoke api key", http.StatusInternalServerError) // otherwise return a 500 Internal Server Error
		return
	}

	var existing utility.APIKeyDoc // create a variable to hold the existing API key document data
	// decode the Firestore document data into the existing variable;
	if err := doc.DataTo(&existing); err != nil { //if it fails, return a 500 Internal Server Error
		http.Error(w, "failed to revoke api key", http.StatusInternalServerError)
		return
	}

	if existing.Revoked { // if the API key is already revoked, return a 404 Not Found error
		http.Error(w, "api key not found", http.StatusNotFound)
		return
	}

	_, err = ref.Set(ctx, map[string]any{
		"revoked":   true,             // if the API key is valid and not revoked,
		"revokedAt": time.Now().UTC(), //update the document to set "revoked" to true and record the revocation time
	}, firestore.MergeAll)

	if err != nil { // if there is an error updating the document, return a 500 Internal Server Error
		http.Error(w, "failed to revoke api key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// APIKeyMiddleware is an HTTP middleware that checks for the presence and validity of an API key in incoming requests.
// It allows unauthenticated access to the status and authentication endpoints, while protecting all other endpoints.
// If the API key is missing, invalid, or revoked, it returns appropriate HTTP error responses.
func (h *Handler) APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // return a new http.HandlerFunc that wraps the next handler
		if r.URL.Path == utility.StatusPath { // allow unauthenticated access to the status endpoint
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == utility.AuthPath && r.Method == http.MethodPost {
			//if the request is a POST to the authentication endpoint,
			//allow it without an API key (since it's for creating new keys)
			next.ServeHTTP(w, r)
			return
		}

		apiKey := strings.TrimSpace(r.Header.Get(utility.APIKeyHeader))
		if apiKey == "" { // if the API key is missing from the request header, return a 401 Unauthorized error
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		valid, err := validateAPIKeyImpl(h, r, apiKey) // validate the API key using the validateAPIKeyImpl function
		if err != nil {                                // if there is an error during validation, return a 500 Internal Server Error
			http.Error(w, "failed to validate api key", http.StatusInternalServerError)
			return
		}
		if !valid { // if the API key is invalid or revoked, return a 403 Forbidden error
			http.Error(w, "invalid or revoked api key", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r) // if the API key is valid, proceed to the next handler in the chain
	})
}

// validateAPIKey checks if the provided raw API key is valid and not revoked by looking up its hash in Firestore.
// It retrieves the corresponding document based on the hashed key and checks the "revoked" status.
// If the document is not found, it returns false without an error. If there is any other error during
// retrieval or decoding, it returns the error.
func (h *Handler) validateAPIKey(r *http.Request, rawKey string) (bool, error) {
	hash := hashAPIKey(rawKey) // hash the raw API key to find the corresponding document in Firestore and then find it
	doc, err := h.Client.Collection(utility.APIKeysCollection).Doc(hash).Get(firestoreContext(r))
	if err != nil { // if there is an error retrieving the document, check if it's a Not Found error
		if isNotFound(err) {
			return false, nil // if so, return false without an error
		}
		return false, err // otherwise return the error
	}

	var data utility.APIKeyDoc                // create a variable to hold the API key document data
	if err := doc.DataTo(&data); err != nil { // decode the Firestore document data into the variable
		return false, err // if it fails, return false with the error
	}
	return !data.Revoked, nil // return true if the API key is valid (not revoked), otherwise return false
}

// generateAPIKey creates a new random API key string with a specific prefix. It generates 8 random bytes,
// encodes them as a hexadecimal string, and concatenates it with the prefix "sk-envdash-" to form the complete API key.
// If there is an error during random byte generation, it returns an empty string and the error.
func generateAPIKey() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-envdash-" + hex.EncodeToString(b), nil
}

// hashAPIKey takes a raw API key string and returns its SHA-256 hash encoded as a hexadecimal string.
// This function is used to store only the hash of the API key in Firestore for security reasons,
// ensuring that the raw key is never stored in plaintext.
func hashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

// isNotFound checks if the given error is a Firestore "Not Found" error by examining its gRPC status code.
func isNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}
