package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

type songResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

func (s *server) listSongs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListSongs(r.Context())
	if err != nil {
		s.logger.Error("list songs failed", slog.String("error", err.Error()))
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
		s.logger.Error("write songs response failed", slog.String("error", err.Error()))
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
		s.logger.Error("get songs failed", slog.String("error", err.Error()))
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
		s.logger.Error("get song status failed", slog.String("error", err.Error()))
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

func (s *server) uploadSong(w http.ResponseWriter, r *http.Request) {
	in, errorStatus, err := receiveSongUpload(w, r)
	if err != nil {
		if errorStatus == http.StatusInternalServerError {
			s.logger.Error("receive song upload failed", "error", err)
			writeJSONError(w, errorStatus, "could not upload song")
			return
		}
		writeJSONError(w, errorStatus, err.Error())
		return
	}
	keepFile := false
	defer func() {
		if !keepFile {
			if err := os.Remove(in.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				s.logger.Warn("remove failed song upload", "path", in.Path, "error", err)
			}
		}
	}()
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Error("begin song upload transaction", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not upload song")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)
	song, err := q.CreateSong(ctx, dbgen.CreateSongParams{
		Title:           in.Title,
		Artist:          in.Artist,
		Genre:           in.Genre,
		DurationSeconds: in.DurationSeconds,
	})
	if err != nil {
		s.logger.Error("create song", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not upload song")
		return
	}

	var songID pgtype.UUID
	if err := songID.Scan(song.ID); err != nil {
		s.logger.Error("parse created song ID", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not upload song")
		return
	}
	if err := q.EnqueueSongJob(ctx, dbgen.EnqueueSongJobParams{
		SongID:           songID,
		InputPath:        in.Path,
		ClipStartSeconds: in.ClipStartSeconds,
	}); err != nil {
		s.logger.Error("enqueue song", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not upload song")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		if !errors.Is(err, pgx.ErrTxCommitRollback) {
			keepFile = true
			s.logger.Warn("song upload commit outcome unknown; retaining file", "song_id", song.ID, "path", in.Path)
		}
		s.logger.Error("commit song upload", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not upload song")
		return
	}
	keepFile = true

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"id": song.ID, "status": song.Status,
	}); err != nil {
		s.logger.Error("write song upload response", "error", err)
	}
}
