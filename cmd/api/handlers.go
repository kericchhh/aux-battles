package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type songResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok\n"))
}

func (s *server) listSongs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListSongs(r.Context())
	if err != nil {
		s.logger.Error("list songs failed", slog.String("error", err.Error()))
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	songs := make([]songResponse, 0, len(rows))
	for _, row := range rows {
		songs = append(songs, songResponse{
			ID: row.ID, Title: row.Title, Artist: row.Artist,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(songs); err != nil {
		s.logger.Error("write songs response failed", slog.String("error", err.Error()))
	}
}

func (s *server) getSong(w http.ResponseWriter, r *http.Request) {
	idText := r.PathValue("id")
	if idText == "" {
		http.Error(w, "song ID is required", http.StatusBadRequest)
		return
	}
	var id pgtype.UUID
	if err := id.Scan(idText); err != nil {
		http.Error(w, "invalid song ID", http.StatusBadRequest)
		return
	}
	song, err := s.queries.GetSong(r.Context(),id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.logger.Error("get songs failed", slog.String("error", err.Error()))
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(songResponse{
		ID: song.ID, Title: song.Title, Artist: song.Artist,
	})
}
