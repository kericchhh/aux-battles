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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxSongFileBytes = 30 << 20
	maxSongBodyBytes = 31 << 20
	minClipSeconds   = 12
)

var errInvalidMP3 = errors.New("upload a valid MP3 file")

func receiveSongUpload(w http.ResponseWriter, r *http.Request) (validatedSongUpload, int, error) {
	var empty validatedSongUpload
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		return empty, http.StatusUnsupportedMediaType, errors.New("expected multipart/form-data")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSongBodyBytes)
	parseErr := r.ParseMultipartForm(1 << 20)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if parseErr != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(parseErr, &sizeErr) {
			return empty, http.StatusRequestEntityTooLarge, errors.New("request body too large")
		}
		return empty, http.StatusBadRequest, errors.New("invalid multipart body")
	}

	files := r.MultipartForm.File["song"]
	if len(files) != 1 || len(r.MultipartForm.File) != 1 {
		return empty, http.StatusBadRequest, errors.New("one song file is required")
	}
	if files[0].Size == 0 {
		return empty, http.StatusBadRequest, errors.New("song file is empty")
	}
	if files[0].Size > maxSongFileBytes {
		return empty, http.StatusRequestEntityTooLarge, errors.New("song file is too large")
	}

	field := func(name string) (string, bool) {
		values := r.MultipartForm.Value[name]
		if len(values) != 1 {
			return "", false
		}
		return strings.TrimSpace(values[0]), true
	}
	title, titleOK := field("title")
	artist, artistOK := field("artist")
	genre, genreOK := field("genre")
	clipText, clipOK := field("clipStartSeconds")
	if !titleOK || !artistOK || !genreOK || !clipOK || !validSongField(title, 255) || !validSongField(artist, 255) || !validSongField(genre, 50) {
		return empty, http.StatusBadRequest, errors.New("invalid song fields")
	}
	clipStart, err := strconv.ParseFloat(clipText, 64)
	if err != nil || math.IsNaN(clipStart) || math.IsInf(clipStart, 0) || clipStart < 0 {
		return empty, http.StatusBadRequest, errors.New("invalid clip start")
	}

	mediaRoot := os.Getenv("MEDIA_ROOT")
	if mediaRoot == "" {
		mediaRoot = "storage"
	}
	incomingDir, err := filepath.Abs(filepath.Join(mediaRoot, "incoming"))
	if err != nil {
		return empty, http.StatusInternalServerError, fmt.Errorf("resolve media directory: %w", err)
	}
	if err := os.MkdirAll(incomingDir, 0750); err != nil {
		return empty, http.StatusInternalServerError, fmt.Errorf("create media directory: %w", err)
	}

	source, err := files[0].Open()
	if err != nil {
		return empty, http.StatusInternalServerError, fmt.Errorf("open uploaded file: %w", err)
	}
	defer source.Close()
	target, err := os.CreateTemp(incomingDir, "song-*.mp3")
	if err != nil {
		return empty, http.StatusInternalServerError, fmt.Errorf("save uploaded file: %w", err)
	}
	path := target.Name()
	keepFile := false
	defer func() {
		if !keepFile {
			_ = os.Remove(path)
		}
	}()

	written, copyErr := io.Copy(target, io.LimitReader(source, maxSongFileBytes+1))
	if copyErr != nil {
		_ = target.Close()
		return empty, http.StatusInternalServerError, fmt.Errorf("copy uploaded file: %w", copyErr)
	}
	if written > maxSongFileBytes {
		_ = target.Close()
		return empty, http.StatusRequestEntityTooLarge, errors.New("song file is too large")
	}
	if err := target.Sync(); err != nil {
		_ = target.Close()
		return empty, http.StatusInternalServerError, fmt.Errorf("sync uploaded file: %w", err)
	}
	if err := target.Close(); err != nil {
		return empty, http.StatusInternalServerError, fmt.Errorf("close uploaded file: %w", err)
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
