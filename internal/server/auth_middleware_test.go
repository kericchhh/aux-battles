package server

import (
	"errors"
	"net/http"
	"testing"
)

func TestSongStatusRejectsRequestsWithoutAValidSession(t *testing.T) {
	const path = "/songs/00000000-0000-4000-8000-000000000001/status"
	for _, tc := range []struct {
		name   string
		cookie *http.Cookie
	}{
		{name: "missing cookie"},
		{name: "unknown session", cookie: &http.Cookie{Name: "aux_session", Value: "unknown-token"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := authTestRequest(authTestHandler(newAuthTestDB(), false), http.MethodGet, path, "", tc.cookie)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401; body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestSongStatusAuthLookupFailureReturnsServerError(t *testing.T) {
	db := newAuthTestDB()
	db.getSessionErr = errors.New("database unavailable")
	cookie := &http.Cookie{Name: "aux_session", Value: "some-token"}
	response := authTestRequest(authTestHandler(db, false), http.MethodGet, "/songs/00000000-0000-4000-8000-000000000001/status", "", cookie)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", response.Code, response.Body.String())
	}
}
