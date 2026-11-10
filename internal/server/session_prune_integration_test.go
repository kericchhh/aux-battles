package server

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestPruneSessionsDeletesOnlyExpiredSessions(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL session pruning test")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	username := fmt.Sprintf("prune_%d", time.Now().UnixNano())
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO users (username, email, password_hash) VALUES ($1, $1 || '@example.com', 'unused') RETURNING id`, username).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Error(err)
		}
	})
	_, expired, err := newSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	_, valid, err := newSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES
		($1, $3, now() - interval '1 minute'), ($2, $3, now() + interval '1 hour')`, expired, valid, userID); err != nil {
		t.Fatal(err)
	}

	logs := &syncBuffer{}
	pruneCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		PruneSessions(pruneCtx, dbgen.New(pool), slog.New(slog.NewTextHandler(logs, nil)), 10*time.Millisecond)
	}()

	sessionExists := func(hash string) bool {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sessions WHERE token_hash = $1)`, hash).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		return exists
	}
	deadline := time.Now().Add(5 * time.Second)
	for sessionExists(expired) {
		if time.Now().After(deadline) {
			t.Fatal("expired session was not deleted")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("PruneSessions did not stop after cancellation")
	}
	if !sessionExists(valid) {
		t.Fatal("valid session was deleted")
	}
	if !strings.Contains(logs.String(), "deleted expired sessions") {
		t.Fatalf("missing deletion log: %s", logs.String())
	}
}
