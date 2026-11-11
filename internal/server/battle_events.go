package server

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	streamKeepAlive    = 25 * time.Second
	streamWriteTimeout = 10 * time.Second
	maxStreamDuration  = 30 * time.Minute
)

func (s *server) battleEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	var id pgtype.UUID
	if err := id.Scan(r.PathValue("battleId")); err != nil || !id.Valid {
		writeJSONError(w, http.StatusBadRequest, "invalid battle ID")
		return
	}
	if s.events == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "live updates are unavailable")
		return
	}
	if _, err := s.battles.Get(r.Context(), userID, id); err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	changes, unsubscribe := s.events.Subscribe(id.String())
	defer unsubscribe()

	controller := http.NewResponseController(w)
	if err := controller.SetReadDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		s.loggerFor(r.Context()).Error("clear battle stream read deadline", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not open live updates")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	send := func(message string) bool {
		if err := controller.SetWriteDeadline(time.Now().Add(streamWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return false
		}
		if _, err := io.WriteString(w, message); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	if !send("retry: 3000\nevent: changed\ndata: {}\n\n") {
		return
	}
	keepAlive := time.NewTicker(streamKeepAlive)
	defer keepAlive.Stop()
	expire := time.NewTimer(maxStreamDuration)
	defer expire.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-expire.C:
			return
		case _, open := <-changes:
			if !open || !send("event: changed\ndata: {}\n\n") {
				return
			}
		case <-keepAlive.C:
			if !send(": keep-alive\n\n") {
				return
			}
		}
	}
}
