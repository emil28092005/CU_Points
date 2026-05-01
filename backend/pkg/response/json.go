// Package response provides helpers for writing consistent JSON API responses.
// Every handler must use these helpers — never call json.Encode directly.
package response

import (
	"encoding/json"
	"net/http"
)

type successBody struct {
	Data interface{} `json:"data"`
}

type errorBody struct {
	Error string `json:"error"`
}

// JSON writes a successful JSON response with the given HTTP status code and data payload.
// The payload is wrapped in {"data": ...} to match the API envelope convention.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(successBody{Data: data}) //nolint:errcheck
}

// Error writes a JSON error response with the given HTTP status code and human-readable message.
// The message is wrapped in {"error": ...}.
func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorBody{Error: message}) //nolint:errcheck
}
