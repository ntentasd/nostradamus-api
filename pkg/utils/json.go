// Package utils
package utils

import (
	"encoding/json"
	"net/http"
)

type Body map[string]any

func ReplyJSON(w http.ResponseWriter, status int, body map[string]any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}

func ReplyMethodNotAllowed(w http.ResponseWriter) error {
	return ReplyJSON(w, http.StatusMethodNotAllowed, Body{
		"error": "Method Not Allowed",
	})
}
