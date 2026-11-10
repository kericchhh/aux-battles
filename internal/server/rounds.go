package server

import (
	"encoding/json/v2"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *server) submitGuess(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	var roundID pgtype.UUID
	if err := roundID.Scan(r.PathValue("roundId")); err != nil || !roundID.Valid {
		writeJSONError(w, http.StatusBadRequest, "invalid round ID")
		return
	}
	var input struct {
		Guess           string `json:"guess"`
		ExpectedAttempt int32  `json:"expectedAttempt"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	result, err := s.battles.SubmitGuess(r.Context(), userID, roundID, input.Guess, input.ExpectedAttempt)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, result); err != nil {
		s.loggerFor(r.Context()).Error("write guess response", "error", err)
	}
}

func (s *server) getRoundAudio(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	var roundID pgtype.UUID
	if err := roundID.Scan(r.PathValue("roundId")); err != nil || !roundID.Valid {
		writeJSONError(w, http.StatusBadRequest, "invalid round ID")
		return
	}
	file, info, err := s.battles.OpenRoundAudio(r.Context(), userID, roundID)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	defer file.Close()
	s.serveAudio(w, r, "round.mp3", "private, no-store", file, info)
}
