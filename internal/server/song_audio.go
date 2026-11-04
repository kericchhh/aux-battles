package server

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/media"
)

func (s *server) getSongAudio(w http.ResponseWriter, r *http.Request) {
	id, err := checkValidID(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	song, err := s.queries.GetReadySongAudio(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "song not found")
		return
	}
	if err != nil {
		s.loggerFor(r.Context()).Error("get song audio", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "audio unavailable")
		return
	}

	part := r.PathValue("part")
	var path pgtype.Text
	switch part {
	case "full":
		path = song.FullSongPath
	case "drums":
		path = song.DrumsPath
	case "bass":
		path = song.BassPath
	case "melody":
		path = song.MelodyPath
	case "vocals":
		path = song.VocalsPath
	default:
		writeJSONError(w, http.StatusNotFound, "audio not found")
		return
	}
	if !path.Valid {
		s.loggerFor(r.Context()).Error("ready song has no audio path", "song_id", id.String(), "part", part)
		writeJSONError(w, http.StatusInternalServerError, "audio unavailable")
		return
	}

	file, info, err := media.FromEnv().OpenAudio(path.String)
	if err != nil {
		s.loggerFor(r.Context()).Error("open song audio", "song_id", id.String(), "part", part, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "audio unavailable")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "private")
	http.ServeContent(w, r, part+".mp3", info.ModTime(), file)
}
