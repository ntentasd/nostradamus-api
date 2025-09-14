// Package utils
package utils

import (
	"encoding/json"
	"net/http"
)

func ReplyJSON(w http.ResponseWriter, status int, body map[string]any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}
