package main

import (
	"encoding/json"
	"net/http"
)

func (s *APIServer) writeJSON(w http.ResponseWriter, payload any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

func (s *APIServer) writeJSONError(w http.ResponseWriter, message string, status int) {
	type Envelope struct {
		Message string `json:"message"`
	}
	if err := s.writeJSON(w, Envelope{Message: message}, status); err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
	}
}
