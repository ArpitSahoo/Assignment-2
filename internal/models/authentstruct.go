package models

import "time"

// AuthenticationRequest and Response structs for handling API key creation and revocation requests,
// as well as the Firestore document structure for storing API key information securely.
type AuthenticationRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AuthenticationResponse struct represents the response sent back to the client after successfully creating an API key.
type AuthenticationResponse struct {
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
}

// APIKeyDoc struct represents the authentication information stored in Firestore,
// including the name, email, hashed API key, and creation timestamp.
type APIKeyDoc struct {
	Name      string    `firestore:"name"`
	Email     string    `firestore:"email"`
	Hash      string    `firestore:"hash"`
	CreatedAt time.Time `firestore:"createdAt"`
}
