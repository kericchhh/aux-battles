package server

import (
	"encoding/json/v2"
	"errors"
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
		Rounds int32 `json:"rounds"`
	}{Rounds: 5}
	if !readJSON(w, r, &input) {
		return
	}
	id, err := s.battles.Create(r.Context(), userID, input.Rounds)
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
