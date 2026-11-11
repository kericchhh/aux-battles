package server

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/battle"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func (s *server) battleActor(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	user, ok := r.Context().Value(authenticatedUserKey{}).(dbgen.GetValidSessionRow)
	if !ok || !user.ID.Valid {
		s.loggerFor(r.Context()).Error("battle handler has no authenticated identity")
		writeJSONError(w, http.StatusInternalServerError, "could not authenticate request")
		return pgtype.UUID{}, false
	}
	return user.ID, true
}

func (s *server) createBattle(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	input := struct {
		Rounds       int32  `json:"rounds"`
		GuessSeconds *int32 `json:"guessSeconds"`
	}{Rounds: 5}
	if !readJSON(w, r, &input) {
		return
	}
	var guessSeconds int32
	if input.GuessSeconds != nil {
		guessSeconds = *input.GuessSeconds
		if guessSeconds == 0 {
			writeJSONError(w, http.StatusBadRequest, "guessSeconds must be 15, 30 or null")
			return
		}
	}
	id, err := s.battles.Create(r.Context(), userID, input.Rounds, guessSeconds)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.MarshalWrite(w, struct {
		ID string `json:"id"`
	}{ID: id.String()}); err != nil {
		s.loggerFor(r.Context()).Error("write created battle response", "error", err)
	}
}

func (s *server) joinBattle(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	var input struct {
		InviteCode string `json:"inviteCode"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	id, err := s.battles.Join(r.Context(), userID, input.InviteCode)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, struct {
		ID string `json:"id"`
	}{ID: id.String()}); err != nil {
		s.loggerFor(r.Context()).Error("write joined battle response", "error", err)
	}
}

func (s *server) listBattles(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	battles, err := s.battles.List(r.Context(), userID)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, battles); err != nil {
		s.loggerFor(r.Context()).Error("write battle list", "error", err)
	}
}

func (s *server) getBattle(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	var id pgtype.UUID
	if err := id.Scan(r.PathValue("battleId")); err != nil || !id.Valid {
		writeJSONError(w, http.StatusBadRequest, "invalid battle ID")
		return
	}
	view, err := s.battles.Get(r.Context(), userID, id)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, view); err != nil {
		s.loggerFor(r.Context()).Error("write battle view", "error", err)
	}
}

func (s *server) forfeitBattle(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	var id pgtype.UUID
	if err := id.Scan(r.PathValue("battleId")); err != nil || !id.Valid {
		writeJSONError(w, http.StatusBadRequest, "invalid battle ID")
		return
	}
	view, err := s.battles.Forfeit(r.Context(), userID, id)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, view); err != nil {
		s.loggerFor(r.Context()).Error("write forfeited battle view", "error", err)
	}
}

func (s *server) getStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	stats, err := s.battles.Stats(r.Context(), userID)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.MarshalWrite(w, stats); err != nil {
		s.loggerFor(r.Context()).Error("write player stats", "error", err)
	}
}

func ExpireStages(ctx context.Context, battles *battle.Lobby, logger *slog.Logger) {
	expired, err := battles.ExpireStages(ctx)
	switch {
	case ctx.Err() != nil:
	case err != nil:
		logger.Error("expire timed out stages", "expired", expired, "error", err)
	case expired > 0:
		logger.Info("expired timed out stages", "count", expired)
	}
}

func ExpireBattles(ctx context.Context, battles *battle.Lobby, logger *slog.Logger) {
	ended, err := battles.ExpireInactive(ctx)
	switch {
	case ctx.Err() != nil:
	case err != nil:
		logger.Error("expire inactive battles", "ended", ended, "error", err)
	case ended > 0:
		logger.Info("expired inactive battles", "count", ended)
	}
}

var battleErrorStatuses = map[battle.ErrorKind]int{
	battle.InvalidInput: http.StatusBadRequest,
	battle.NotFound:     http.StatusNotFound,
	battle.Conflict:     http.StatusConflict,
	battle.Unavailable:  http.StatusServiceUnavailable,
}

func (s *server) writeBattleError(w http.ResponseWriter, r *http.Request, err error) {
	var failure *battle.Error
	if errors.As(err, &failure) {
		if status, ok := battleErrorStatuses[failure.Kind]; ok {
			if errors.Unwrap(failure) != nil {
				s.loggerFor(r.Context()).Error("battle operation failed", "error", err)
			}
			writeJSONError(w, status, failure.Message)
			return
		}
	}
	s.loggerFor(r.Context()).Error("battle operation failed", "error", err)
	writeJSONError(w, http.StatusInternalServerError, "could not complete battle operation")
}
