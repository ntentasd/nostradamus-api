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

func ReplyBadRequest(w http.ResponseWriter, err string) error {
	return ReplyJSON(w, http.StatusBadRequest, Body{
		"error": err,
	})
}

func ReplyInternalServerError(w http.ResponseWriter, err string) error {
	return ReplyJSON(w, http.StatusInternalServerError, Body{
		"error": err,
	})
}

func ReplyNotFound(w http.ResponseWriter, err string) error {
	return ReplyJSON(w, http.StatusNotFound, Body{
		"error": err,
	})
}

func ReplyUnavailable(w http.ResponseWriter, err string) error {
	return ReplyJSON(w, http.StatusServiceUnavailable, Body{
		"error": err,
	})
}
