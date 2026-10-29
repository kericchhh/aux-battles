package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestSongUploadQueuesJobWithPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL upload test")
	}
	mediaRoot := t.TempDir()
	t.Setenv("MEDIA_ROOT", mediaRoot)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	handler := New(dbgen.New(pool), pool, slog.New(slog.NewTextHandler(io.Discard, nil)), false)

	username := fmt.Sprintf("upload_%d", time.Now().UnixNano())
	registerBody := fmt.Sprintf(`{"username":%q,"email":%q,"password":"correct horse battery staple"}`, username, username+"@example.com")
	register := authTestRequest(handler, http.MethodPost, "/users", registerBody, nil)
	if register.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", register.Code, register.Body.String())
	}
	defer func() { _, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = $1", username) }()
	login := authTestRequest(handler, http.MethodPost, "/users/login", fmt.Sprintf(`{"identifier":%q,"password":"correct horse battery staple"}`, username), nil)
	if login.Code != http.StatusOK || len(login.Result().Cookies()) != 1 {
		t.Fatalf("login = %d %s", login.Code, login.Body.String())
	}
	cookie := login.Result().Cookies()[0]

	request := multipartSongRequest(t, testMP3Bytes(t), "1")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("upload = %d %s", response.Code, response.Body.String())
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Status != "PROCESSING" {
		t.Fatalf("upload response = %+v", created)
	}
	defer func() { _, _ = pool.Exec(context.Background(), "DELETE FROM songs WHERE id = $1::uuid", created.ID) }()

	var songStatus, jobState, inputPath string
	var clipStart float64
	err = pool.QueryRow(ctx, `SELECT s.status, j.state, j.input_path, j.clip_start_seconds
		FROM songs s JOIN song_jobs j ON j.song_id = s.id WHERE s.id = $1::uuid`, created.ID).
		Scan(&songStatus, &jobState, &inputPath, &clipStart)
	if err != nil {
		t.Fatal(err)
	}
	if songStatus != "PROCESSING" || jobState != "QUEUED" || clipStart != 1 {
		t.Fatalf("stored song/job = %s/%s, clip start %v", songStatus, jobState, clipStart)
	}
	if _, err := os.Stat(inputPath); err != nil {
		t.Fatalf("queued input file is missing: %v", err)
	}
}
