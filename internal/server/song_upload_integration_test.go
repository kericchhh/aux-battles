package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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

func TestPersistSongUploadRollsBackAndRemovesInputWhenEnqueueFails(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL upload test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	root := t.TempDir()
	t.Setenv("MEDIA_ROOT", root)
	input := filepath.Join(root, "incoming", "song-failed.mp3")
	if err := os.MkdirAll(filepath.Dir(input), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("staged input"), 0600); err != nil {
		t.Fatal(err)
	}
	title := fmt.Sprintf("failed upload %d", time.Now().UnixNano())
	s := &server{queries: dbgen.New(pool), pool: pool, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	_, err = s.persistSongUpload(ctx, validatedSongUpload{
		Title: title, Artist: "Test", Genre: "Test", DurationSeconds: 13,
		Path: input, ClipStartSeconds: -1,
	})
	if err == nil {
		t.Fatal("persistSongUpload() succeeded with an invalid clip start")
	}
	if _, err := os.Stat(input); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged input after failed enqueue: %v, want absence", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM songs WHERE title = $1`, title).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("songs with failed upload title = %d, want 0", count)
	}
}

type commitResultTx struct {
	pgx.Tx
	commitErr   error
	commitFirst bool
}

func (tx commitResultTx) Commit(ctx context.Context) error {
	if tx.commitFirst {
		if err := tx.Tx.Commit(ctx); err != nil {
			return err
		}
	}
	return tx.commitErr
}

func TestPersistSongUploadRetainsInputWhenCommitOutcomeIsUnknown(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL upload test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	root := t.TempDir()
	t.Setenv("MEDIA_ROOT", root)
	s := &server{queries: dbgen.New(pool), pool: pool, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	for _, tc := range []struct {
		name        string
		commitErr   error
		commitFirst bool
	}{
		{name: "uncertain commit", commitErr: errors.New("connection lost"), commitFirst: true},
		{name: "definite rollback", commitErr: pgx.ErrTxCommitRollback},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := filepath.Join(root, "incoming", "song-"+tc.name+".mp3")
			if err := os.MkdirAll(filepath.Dir(input), 0750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(input, []byte("staged input"), 0600); err != nil {
				t.Fatal(err)
			}
			title := fmt.Sprintf("commit outcome %s %d", tc.name, time.Now().UnixNano())
			t.Cleanup(func() {
				if _, err := pool.Exec(context.Background(), `DELETE FROM songs WHERE title = $1`, title); err != nil {
					t.Errorf("delete commit-outcome test song: %v", err)
				}
			})
			begin := func(ctx context.Context) (pgx.Tx, error) {
				tx, err := pool.Begin(ctx)
				if err != nil {
					return nil, err
				}
				return commitResultTx{Tx: tx, commitErr: tc.commitErr, commitFirst: tc.commitFirst}, nil
			}
			_, err := s.persistSongUploadWithBegin(ctx, validatedSongUpload{
				Title: title, Artist: "Test", Genre: "Test", DurationSeconds: 13,
				Path: input, ClipStartSeconds: 1,
			}, begin)
			if !errors.Is(err, tc.commitErr) {
				t.Fatalf("persistSongUploadWithBegin() error = %v, want %v", err, tc.commitErr)
			}
			_, statErr := os.Stat(input)
			if tc.commitFirst && statErr != nil {
				t.Fatalf("retained input: %v", statErr)
			}
			if !tc.commitFirst && !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("input after known rollback: %v, want absence", statErr)
			}
			var count int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM songs WHERE title = $1`, title).Scan(&count); err != nil {
				t.Fatal(err)
			}
			wantCount := 0
			if tc.commitFirst {
				wantCount = 1
			}
			if count != wantCount {
				t.Fatalf("songs after simulated commit error = %d, want %d", count, wantCount)
			}
		})
	}
}
