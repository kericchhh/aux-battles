package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kericchhh/aux-battles/internal/media"
)

const (
	maxSongFileBytes = 30 << 20
	maxSongBodyBytes = 31 << 20
	minClipSeconds   = 12

	maxSongFieldBytes = 4 << 10
	maxSongParts      = 100
)

var songFields = map[string]bool{"title": true, "artist": true, "genre": true, "clipStartSeconds": true}

var errInvalidMP3 = errors.New("upload a valid MP3 file")

func receiveSongUpload(w http.ResponseWriter, r *http.Request) (validatedSongUpload, int, error) {
	var empty validatedSongUpload
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		return empty, http.StatusUnsupportedMediaType, errors.New("expected multipart/form-data")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSongBodyBytes)
	parts, err := r.MultipartReader()
	if err != nil {
		return empty, http.StatusBadRequest, errors.New("invalid multipart body")
	}

	store := media.FromEnv()
	var path string
	keepFile := false
	defer func() {
		if path != "" && !keepFile {
			_ = store.RemoveInput(path)
		}
	}()
	fields := make(map[string]string, len(songFields))
	for count := 1; ; count++ {
		part, err := parts.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			status, err := uploadReadError(err)
			return empty, status, err
		}
		if count > maxSongParts {
			return empty, http.StatusBadRequest, errors.New("too many multipart parts")
		}
		name := part.FormName()
		switch {
		case name == "":
		case part.FileName() != "":
			if name != "song" || path != "" {
				return empty, http.StatusBadRequest, errors.New("one song file is required")
			}
			var status int
			path, status, err = saveSongPart(store, part)
			if err != nil {
				return empty, status, err
			}
		case songFields[name]:
			if _, ok := fields[name]; ok {
				return empty, http.StatusBadRequest, errors.New("invalid song fields")
			}
			value, err := io.ReadAll(io.LimitReader(part, maxSongFieldBytes+1))
			if err != nil {
				status, err := uploadReadError(err)
				return empty, status, err
			}
			if len(value) > maxSongFieldBytes {
				return empty, http.StatusBadRequest, errors.New("invalid song fields")
			}
			fields[name] = strings.TrimSpace(string(value))
		}
	}
	if path == "" {
		return empty, http.StatusBadRequest, errors.New("one song file is required")
	}

	title, artist, genre := fields["title"], fields["artist"], fields["genre"]
	clipText, clipOK := fields["clipStartSeconds"]
	if !clipOK || !validSongField(title, 255) || !validSongField(artist, 255) || !validSongField(genre, 50) {
		return empty, http.StatusBadRequest, errors.New("invalid song fields")
	}
	clipStart, err := strconv.ParseFloat(clipText, 64)
	if err != nil || math.IsNaN(clipStart) || math.IsInf(clipStart, 0) || clipStart < 0 {
		return empty, http.StatusBadRequest, errors.New("invalid clip start")
	}

	duration, err := probeMP3Duration(r.Context(), path)
	if errors.Is(err, errInvalidMP3) {
		return empty, http.StatusBadRequest, err
	}
	if err != nil {
		return empty, http.StatusInternalServerError, fmt.Errorf("inspect uploaded file: %w", err)
	}
	if duration-clipStart+0.05 < minClipSeconds {
		return empty, http.StatusBadRequest, errors.New("selected clip must contain 12 seconds of audio")
	}
	if duration > math.MaxInt32 {
		return empty, http.StatusBadRequest, errInvalidMP3
	}

	keepFile = true
	return validatedSongUpload{
		Title: title, Artist: artist, Genre: genre, Path: path,
		DurationSeconds:  int32(math.Max(1, math.Round(duration))),
		ClipStartSeconds: clipStart,
	}, 0, nil
}

func saveSongPart(store media.Store, part io.Reader) (string, int, error) {
	target, err := store.CreateInput()
	if err != nil {
		return "", http.StatusInternalServerError, err
	}
	path := target.Name()
	source := &readErrorRecorder{r: io.LimitReader(part, maxSongFileBytes+1)}
	written, err := io.Copy(target, source)
	if err == nil {
		err = target.Sync()
	}
	if closeErr := target.Close(); err == nil {
		err = closeErr
	}
	switch {
	case source.err != nil:
		status, readErr := uploadReadError(source.err)
		return path, status, readErr
	case err != nil:
		return path, http.StatusInternalServerError, fmt.Errorf("save uploaded file: %w", err)
	case written == 0:
		return path, http.StatusBadRequest, errors.New("song file is empty")
	case written > maxSongFileBytes:
		return path, http.StatusRequestEntityTooLarge, errors.New("song file is too large")
	}
	return path, 0, nil
}

type readErrorRecorder struct {
	r   io.Reader
	err error
}

func (r *readErrorRecorder) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		r.err = err
	}
	return n, err
}

func uploadReadError(err error) (int, error) {
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		return http.StatusRequestEntityTooLarge, errors.New("request body too large")
	}
	return http.StatusBadRequest, errors.New("invalid multipart body")
}

func validSongField(value string, maxRunes int) bool {
	if !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return false
	}
	count := utf8.RuneCountInString(value)
	return count > 0 && count <= maxRunes
}

func probeMP3Duration(ctx context.Context, path string) (float64, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(probeCtx, "ffprobe", "-v", "error",
		"-show_entries", "format=duration,format_name:stream=codec_type,codec_name",
		"-of", "json", path).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return 0, errInvalidMP3
		}
		return 0, err
	}
	var result struct {
		Format struct {
			Duration   string `json:"duration"`
			FormatName string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, errInvalidMP3
	}
	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil || math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0 {
		return 0, errInvalidMP3
	}
	validFormat := false
	for _, format := range strings.Split(result.Format.FormatName, ",") {
		if format == "mp3" {
			validFormat = true
			break
		}
	}
	validStream := false
	for _, stream := range result.Streams {
		if stream.CodecType == "audio" && stream.CodecName == "mp3" {
			validStream = true
			break
		}
	}
	if !validFormat || !validStream {
		return 0, errInvalidMP3
	}
	return duration, nil
}
