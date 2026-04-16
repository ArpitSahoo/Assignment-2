package utility

import (
	"encoding/json"
	"net/http"
)

// WriteJSONError writes a JSON-encoded error response with the given HTTP status code
// and message to the response writer.
func WriteJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}
