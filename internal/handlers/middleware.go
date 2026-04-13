package handlers

import (
	"assignment-2/internal/utility"
	"net/http"
	"strings"
)

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
			http.Error(w, "missing API key, please enter an API key.", http.StatusUnauthorized)
			return
		}

		valid, err := validateAPIKeyImpl(h, r, apiKey) // validate the API key using the validateAPIKeyImpl function
		if err != nil {                                // if there is an error during validation, return a 500 Internal Server Error
			http.Error(w, "failed to validate API key", http.StatusInternalServerError)
			return
		}
		if !valid { // if the API key is invalid or revoked, return a 403 Forbidden error
			http.Error(w, "invalid or revoked API key", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r) // if the API key is valid, proceed to the next handler in the chain
	})
}

// TODO find out if middleware should be in this folder or in a separate one, and if it should be in the same
// file as the auth handler or not. I think it should be in a separate file, beacuse it is a separate concern from the auth handler,
// and it could be in the handlers folder because it is related to handling HTTP requests.
