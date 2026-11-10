package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type songResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

func (s *server) listSongs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListSongs(r.Context())
	if err != nil {
		s.loggerFor(r.Context()).Error("list songs failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database error")
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
		s.loggerFor(r.Context()).Error("write songs response failed", "error", err)
	}
}

func (s *server) getSong(w http.ResponseWriter, r *http.Request) {
	id, err := checkValidID(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	song, err := s.queries.GetSong(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "song not found")
		return
	}
	if err != nil {
		s.loggerFor(r.Context()).Error("get songs failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(songResponse{
		ID: song.ID, Title: song.Title, Artist: song.Artist,
	})
}

type songStatusResponse struct {
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	Artist          string      `json:"artist"`
	Status          string      `json:"status"`
	ProcessingError pgtype.Text `json:"processingError"`
	WorkerAvailable *bool       `json:"workerAvailable"`
}

func (s *server) getSongStatus(w http.ResponseWriter, r *http.Request) {
	id, err := checkValidID(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	song, err := s.queries.GetSongStatus(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "song not found")
		return
	}
	if err != nil {
		s.loggerFor(r.Context()).Error("get song status failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(songStatusResponse{
		ID: song.ID, Title: song.Title, Artist: song.Artist, Status: song.Status, ProcessingError: song.ProcessingError,
	})
}

func checkValidID(raw string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if raw == "" {
		return id, errors.New("song ID is required")
	}
	if err := id.Scan(raw); err != nil {
		return id, errors.New("invalid song ID")
	}
	return id, nil
}

type validatedSongUpload struct {
	Title, Artist, Genre string
	Path                 string
	DurationSeconds      int32
	ClipStartSeconds     float64
}

const transferDeadline = 5 * time.Minute

func extendDeadlines(w http.ResponseWriter, d time.Duration) error {
	deadline := time.Now().Add(d)
	controller := http.NewResponseController(w)
	if err := controller.SetReadDeadline(deadline); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return err
	}
	if err := controller.SetWriteDeadline(deadline); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return err
	}
	return nil
}

func (s *server) uploadSong(w http.ResponseWriter, r *http.Request) {
	if err := extendDeadlines(w, transferDeadline); err != nil {
		s.loggerFor(r.Context()).Error("extend upload deadlines", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not upload song")
		return
	}
	in, errorStatus, err := receiveSongUpload(w, r)
	if err != nil {
		if errorStatus == http.StatusInternalServerError {
			s.loggerFor(r.Context()).Error("receive song upload failed", "error", err)
			writeJSONError(w, errorStatus, "could not upload song")
			return
		}
		writeJSONError(w, errorStatus, err.Error())
		return
	}
	song, err := s.persistSongUpload(r.Context(), in)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not upload song")
		return
	}
	s.loggerFor(r.Context()).Info("song queued", "song_id", song.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"id": song.ID, "status": song.Status,
	}); err != nil {
		s.loggerFor(r.Context()).Error("write song upload response", "error", err)
	}
}
