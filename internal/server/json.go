package server

import (
	"encoding/json/v2"
	"errors"
	"mime"
	"net/http"
)

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSONError(w, http.StatusUnsupportedMediaType, "expected application/json")
		return false
	}

	body := http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.UnmarshalRead(body, dst, json.RejectUnknownMembers(true)); err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
		}
		return false
	}
	return true
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, struct {
		Message string `json:"message"`
	}{Message: message})
}
