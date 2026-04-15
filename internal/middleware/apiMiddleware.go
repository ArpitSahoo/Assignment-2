package middleware

import (
	"assignment-2/internal/handlers"
	"assignment-2/internal/utility"
	"net/http"
	"strings"
)

// APIKeyMiddleware returns an http middleware function that validates API keys
// using the provided handlers.Handler instance (it will call handlers.ValidateAPIKeyImpl).
// This keeps the middleware in a separate package while avoiding invalid method receivers.
func APIKeyMiddleware(h *handlers.Handler) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// allow public status endpoint
			if r.URL.Path == utility.StatusPath {
				next.ServeHTTP(w, r)
				return
			}
			// allow creating API keys without an existing key
			if r.URL.Path == utility.AuthPath && r.Method == http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := strings.TrimSpace(r.Header.Get(utility.APIKeyHeader))
			if apiKey == "" {
				http.Error(w, "missing API key, please enter an API key.", http.StatusUnauthorized)
				return
			}

			// call the exported adapter in handlers to validate the key
			valid, err := handlers.ValidateAPIKeyImpl(h, r, apiKey)
			if err != nil {
				http.Error(w, "failed to validate API key", http.StatusInternalServerError)
				return
			}
			if !valid {
				http.Error(w, "invalid or revoked API key in headers", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TODO find out if middleware should be in this folder or in a separate one, and if it should be in the same
// file as the auth handler or not. I think it should be in a separate file, because it is a separate concern from the auth handler,
// and it could be in the handlers folder because it is related to handling HTTP requests.
