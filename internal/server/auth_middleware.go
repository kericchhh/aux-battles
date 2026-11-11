package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/kericchhh/aux-battles/internal/dbgen"
)

type authenticatedUserKey struct{}

func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		user, err := s.currentUser(r)
		if errors.Is(err, errUnauthenticated) {
			writeJSONError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		if err != nil {
			s.loggerFor(r.Context()).Error("authenticate request failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "could not authenticate request")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), authenticatedUserKey{}, user)))
	}
}

func (s *server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(authenticatedUserKey{}).(dbgen.GetValidSessionRow)
		if !ok || user.Role != "ADMIN" {
			writeJSONError(w, http.StatusForbidden, "admin access required")
			return
		}
		next(w, r)
	})
}
