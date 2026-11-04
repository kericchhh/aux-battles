package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestSongAudioRequiresAuthentication(t *testing.T) {
	handler := New(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), false)
	request := httptest.NewRequest(http.MethodGet, "/songs/00000000-0000-4000-8000-000000000001/audio/drums", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated audio request = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSongAudioStreamsReadyMedia(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL audio test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	root := t.TempDir()
	t.Setenv("MEDIA_ROOT", root)
	handler := New(dbgen.New(pool), pool, slog.New(slog.NewTextHandler(io.Discard, nil)), false)
	username := fmt.Sprintf("audio_%d", time.Now().UnixNano())
	register := authTestRequest(handler, http.MethodPost, "/users",
		fmt.Sprintf(`{"username":%q,"email":%q,"password":"correct horse battery staple"}`,
			username, username+"@example.com"), nil)
	if register.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", register.Code, register.Body.String())
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = $1", username) })
	login := authTestRequest(handler, http.MethodPost, "/users/login",
		fmt.Sprintf(`{"identifier":%q,"password":"correct horse battery staple"}`, username), nil)
	if login.Code != http.StatusOK || len(login.Result().Cookies()) != 1 {
		t.Fatalf("login = %d %s", login.Code, login.Body.String())
	}

	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO songs (title, artist, genre, duration_seconds)
		VALUES ('Audio test', 'Test', 'Test', 12) RETURNING id::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1::uuid", id) })
	dir := filepath.Join(root, "songs", id, "attempt-1")
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	parts := []struct {
		name string
		file string
		body string
	}{
		{"full", "full.mp3", "full audio bytes"},
		{"drums", "drums.mp3", "drums audio bytes"},
		{"bass", "bass.mp3", "bass audio bytes"},
		{"melody", "other.mp3", "melody audio bytes"},
		{"vocals", "vocals.mp3", "vocals audio bytes"},
	}
	for _, part := range parts {
		if err := os.WriteFile(filepath.Join(dir, part.file), []byte(part.body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err = pool.Exec(ctx, `UPDATE songs SET status = 'READY', full_song_path = $2,
		drums_path = $3, bass_path = $4, melody_path = $5, vocals_path = $6
		WHERE id = $1::uuid`, id, filepath.Join(dir, "full.mp3"), filepath.Join(dir, "drums.mp3"),
		filepath.Join(dir, "bass.mp3"), filepath.Join(dir, "other.mp3"), filepath.Join(dir, "vocals.mp3"))
	if err != nil {
		t.Fatal(err)
	}

	requestAudio := func(songID, part, byteRange string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/songs/"+songID+"/audio/"+part, nil)
		request.AddCookie(login.Result().Cookies()[0])
		if byteRange != "" {
			request.Header.Set("Range", byteRange)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	for _, part := range parts {
		t.Run(part.name, func(t *testing.T) {
			response := requestAudio(id, part.name, "")
			if response.Code != http.StatusOK || response.Body.String() != part.body {
				t.Fatalf("audio response = %d %q, want 200 %q", response.Code, response.Body.String(), part.body)
			}
			if got := response.Header().Get("Content-Type"); got != "audio/mpeg" {
				t.Fatalf("content type = %q, want audio/mpeg", got)
			}
		})
	}
	t.Run("byte range", func(t *testing.T) {
		response := requestAudio(id, "drums", "bytes=0-4")
		if response.Code != http.StatusPartialContent || response.Body.String() != "drums" {
			t.Fatalf("range response = %d %q, want 206 drums", response.Code, response.Body.String())
		}
		if got := response.Header().Get("Content-Range"); got != "bytes 0-4/17" {
			t.Fatalf("content range = %q, want bytes 0-4/17", got)
		}
	})
	t.Run("unknown part", func(t *testing.T) {
		if response := requestAudio(id, "guitar", ""); response.Code != http.StatusNotFound {
			t.Fatalf("unknown part = %d, want 404", response.Code)
		}
	})
	t.Run("invalid ID", func(t *testing.T) {
		if response := requestAudio("not-a-uuid", "drums", ""); response.Code != http.StatusBadRequest {
			t.Fatalf("invalid ID = %d, want 400", response.Code)
		}
	})
	t.Run("processing song", func(t *testing.T) {
		var pendingID string
		if err := pool.QueryRow(ctx, `INSERT INTO songs (title, artist, genre, duration_seconds)
			VALUES ('Pending audio test', 'Test', 'Test', 12) RETURNING id::text`).Scan(&pendingID); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1::uuid", pendingID) })
		if response := requestAudio(pendingID, "drums", ""); response.Code != http.StatusNotFound {
			t.Fatalf("processing song = %d, want 404", response.Code)
		}
	})
	t.Run("missing file", func(t *testing.T) {
		if err := os.Remove(filepath.Join(dir, "full.mp3")); err != nil {
			t.Fatal(err)
		}
		if response := requestAudio(id, "full", ""); response.Code != http.StatusInternalServerError {
			t.Fatalf("missing file = %d, want 500", response.Code)
		}
	})
	t.Run("path outside media directory", func(t *testing.T) {
		outside := filepath.Join(root, "outside.mp3")
		if err := os.WriteFile(outside, []byte("private bytes"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE songs SET bass_path = $2 WHERE id = $1::uuid`, id, outside); err != nil {
			t.Fatal(err)
		}
		response := requestAudio(id, "bass", "")
		if response.Code != http.StatusInternalServerError || response.Body.String() == "private bytes" {
			t.Fatalf("outside path = %d %q, want 500 without file", response.Code, response.Body.String())
		}
	})
	t.Run("symlink outside media directory", func(t *testing.T) {
		outside := filepath.Join(root, "outside.mp3")
		inside := filepath.Join(dir, "vocals.mp3")
		if err := os.Remove(inside); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, inside); err != nil {
			t.Fatal(err)
		}
		response := requestAudio(id, "vocals", "")
		if response.Code != http.StatusInternalServerError || response.Body.String() == "private bytes" {
			t.Fatalf("symlink = %d %q, want 500 without file", response.Code, response.Body.String())
		}
	})
}
