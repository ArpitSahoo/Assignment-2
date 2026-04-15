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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidateAPIKeyImpl is a public function that serves as a delegator to the private validateAPIKey
// method of the Handler struct. It allows external packages to call the validation logic without exposing
// the internal implementation details of the Handler struct.
// This design promotes encapsulation while still providing necessary functionality to other parts of the application,
// such as middleware that needs to validate API keys for incoming requests.
var ValidateAPIKeyImpl = func(h *Handler, r *http.Request, rawKey string) (bool, error) {
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
		}
	}(r.Body)

	var req utility.AuthenticationRequest // struct to decode the incoming JSON request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { // decode the JSON request body into the struct
		// if there is an error during decoding (e.g., invalid JSON), return a 400 Bad Request error
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return // if decoding fails, return a 400 Bad Request error
	}

	req.Name = strings.TrimSpace(req.Name) // trim whitespace from the name and email fields
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" || req.Email == "" { // if name or email is empty after trimming, return a 400 Bad Request error
		http.Error(w, "name and email are required", http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil { // validate the email format using the net/mail package
		// if invalid, return a 400 Bad Request error
		http.Error(w, "invalid email. Please use a valid email", http.StatusBadRequest)
		return
	}

	rawKey, err := generateAPIKey() // generate a new API key using the generateAPIKey function
	// if it fails, return a 500 Internal Server Error
	if err != nil {
		http.Error(w, "failed to generate API key", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()   // a timestamp variable for the current time in UTC, used for the CreatedAt field in the APIKeyDoc
	doc := utility.APIKeyDoc{ // create a new APIKeyDoc struct to represent the API key document to be stored in Firestore
		Name:      req.Name,
		Email:     req.Email,
		Hash:      hashAPIKey(rawKey), // hash the raw API to store only the hash in Firestore for security reasons
		CreatedAt: now,
	}

	ctx := firestoreContext(r)
	_, err = h.Client.Collection(utility.APIKeysCollection).Doc(doc.Hash).Set(ctx, doc) // store the API key document in the Firestore
	if err != nil {                                                                     // if it fails, return a 500 Internal Server Error
		http.Error(w, "failed to store API key", http.StatusInternalServerError)
		return
	}

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusCreated)                             // create header and status code for the response
	_ = json.NewEncoder(w).Encode(utility.AuthenticationResponse{ // encode the response body as JSON
		Key:       rawKey,                       // return the raw API key
		CreatedAt: now.Format("20060102 15:04"), // return timestamp
	})
}

// revokeAPIKey revokes an API key by deleting its corresponding document in Firestore.
// It retrieves the API key from the URL path, hashes it, and deletes the matching document.
// If the API key is not provided, it returns a 400 Bad Request error.
// If the deletion fails due to a Firestore error, it returns a 500 Internal Server Error.
// This operation is idempotent: if the API key does not exist, the handler still returns
// a 204 No Content response.
// On successful execution, it returns a 204 No Content status.
func (h *Handler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	rawKey := strings.TrimSpace(r.PathValue("key")) // extract the raw API key from the URL path and trim whitespace
	if rawKey == "" {                               // if the raw key is empty, return a 404 Not Found error
		http.Error(w, "API key not provided, please add the API key in the url", http.StatusBadRequest)
		return
	}

	hash := hashAPIKey(rawKey) // hash the raw API key to find the corresponding document in Firestore
	ctx := firestoreContext(r)
	ref := h.Client.Collection(utility.APIKeysCollection).Doc(hash) // get the document reference for the API key in Firestore

	if _, err := ref.Delete(ctx); err != nil { // Delete file in firebase, if it fails, return a 500 Internal Server Error
		http.Error(w, "failed to revoke API key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
	return true, nil // if the document is found and decoded successfully, return true (valid API key)
}

//TODO move the code down below to their own package

// generateAPIKey creates a new random API key string with a specific prefix. It generates 8 random bytes,
// encodes them as a hexadecimal string, and concatenates it with the prefix "sk-envdash-" to form the complete API key.
// If there is an error during random byte generation, it returns an empty string and the error.
func generateAPIKey() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return utility.BaseAPIKeyStarter + hex.EncodeToString(b), nil
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
