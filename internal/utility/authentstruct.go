package utility

import "time"

type AuthenticationRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AuthenticationResponse struct {
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
}

type APIKeyDoc struct {
	Name      string    `firestore:"name"`
	Email     string    `firestore:"email"`
	Hash      string    `firestore:"hash"`
	CreatedAt time.Time `firestore:"createdAt"`
}
