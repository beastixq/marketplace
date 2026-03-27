package handler

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Add("Content-type", "application/json")
	body, _ := json.Marshal(payload)
	w.WriteHeader(status)
	w.Write(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-type", "application/json")
	body, _ := json.Marshal(map[string]string{"error": message})
	w.WriteHeader(status)
	w.Write(body)
}

