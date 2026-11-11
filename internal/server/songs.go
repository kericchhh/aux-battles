package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

type songResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

const (
	defaultSongPage = 50
	maxSongPage     = 100
	maxSongQuery    = 100
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

type songPage struct {
	Songs      []songResponse `json:"songs"`
	NextCursor *string        `json:"nextCursor"`
}

func (s *server) listSongs(w http.ResponseWriter, r *http.Request) {
	params, err := songListParams(r.URL.Query())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := params.MaxSongs
	params.MaxSongs++
	rows, err := s.queries.ListSongs(r.Context(), params)
	if err != nil {
		s.loggerFor(r.Context()).Error("list songs failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database error")
		return
	}

	page := songPage{Songs: make([]songResponse, 0, min(len(rows), int(limit)))}
	for i, row := range rows {
		if i == int(limit) {
			cursor := encodeSongCursor(rows[i-1].CreatedAt.Time, rows[i-1].ID)
			page.NextCursor = &cursor
			break
		}
		page.Songs = append(page.Songs, songResponse{ID: row.ID.String(), Title: row.Title, Artist: row.Artist})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(page); err != nil {
		s.loggerFor(r.Context()).Error("write songs response failed", "error", err)
	}
}

func songListParams(values url.Values) (dbgen.ListSongsParams, error) {
	params := dbgen.ListSongsParams{MaxSongs: defaultSongPage}
	if q := strings.TrimSpace(values.Get("q")); q != "" {
		if !utf8.ValidString(q) || strings.ContainsRune(q, 0) {
			return params, errors.New("search contains invalid characters")
		}
		if utf8.RuneCountInString(q) > maxSongQuery {
			return params, fmt.Errorf("search must be at most %d characters", maxSongQuery)
		}
		params.Pattern = pgtype.Text{String: "%" + likeEscaper.Replace(q) + "%", Valid: true}
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxSongPage {
			return params, fmt.Errorf("limit must be between 1 and %d", maxSongPage)
		}
		params.MaxSongs = int32(limit)
	}
	if raw := values.Get("cursor"); raw != "" {
		createdAt, id, err := decodeSongCursor(raw)
		if err != nil {
			return params, errors.New("invalid cursor")
		}
		params.BeforeCreatedAt = pgtype.Timestamptz{Time: createdAt, Valid: true}
		params.BeforeID = id
	}
	return params, nil
}

func encodeSongCursor(createdAt time.Time, id pgtype.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(createdAt.UTC().Format(time.RFC3339Nano) + " " + id.String()))
}

func decodeSongCursor(raw string) (time.Time, pgtype.UUID, error) {
	var id pgtype.UUID
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, id, err
	}
	stamp, rawID, ok := strings.Cut(string(decoded), " ")
	if !ok {
		return time.Time{}, id, errors.New("malformed cursor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil {
		return time.Time{}, id, err
	}
	if err := id.Scan(rawID); err != nil {
		return time.Time{}, id, err
	}
	return createdAt, id, nil
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
