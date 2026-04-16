package handlers

import (
	"assignment-2/internal/utility"
	"encoding/json"
	"net/http"
)

// writeJSONError writes a JSON-encoded error response with the given HTTP status code
// and message to the response writer.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}
