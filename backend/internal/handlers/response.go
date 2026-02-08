package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// RespondJSON writes a JSON response with the given status code
func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("ERROR: Failed to encode JSON response: %v", err)
	}
}

// RespondError writes a JSON error response with the given status code
func RespondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"code":    code,
		"message": message,
	}); err != nil {
		log.Printf("ERROR: Failed to encode JSON error response: %v", err)
	}
}
