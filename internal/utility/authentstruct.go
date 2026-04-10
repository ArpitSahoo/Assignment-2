package utility

type AuthenticationRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AuthenticationResponse struct {
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
}
