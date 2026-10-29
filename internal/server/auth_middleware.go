package server

import (
	"errors"
	"log/slog"
	"net/http"
)

func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		_, err := s.currentUser(r)
		if errors.Is(err, errUnauthenticated) {
			writeJSONError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		if err != nil {
			s.logger.Error("authenticate request failed", slog.String("error", err.Error()))
			writeJSONError(w, http.StatusInternalServerError, "could not authenticate request")
			return
		}
		next(w, r)
	}
}
