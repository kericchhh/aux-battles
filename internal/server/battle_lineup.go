package server

import (
	"encoding/json/v2"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *server) submitLineup(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.battleActor(w, r)
	if !ok {
		return
	}
	var battleID pgtype.UUID
	if err := battleID.Scan(r.PathValue("battleId")); err != nil || !battleID.Valid {
		writeJSONError(w, http.StatusBadRequest, "invalid battle ID")
		return
	}
	var input struct {
		SongIDs []string `json:"songIds"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	ids := make([]pgtype.UUID, len(input.SongIDs))
	for i, raw := range input.SongIDs {
		if err := ids[i].Scan(raw); err != nil || !ids[i].Valid {
			writeJSONError(w, http.StatusBadRequest, "invalid song ID")
			return
		}
	}
	result, err := s.battles.SubmitLineup(r.Context(), userID, battleID, ids)
	if err != nil {
		s.writeBattleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, result); err != nil {
		s.loggerFor(r.Context()).Error("write lineup response", "error", err)
	}
}
