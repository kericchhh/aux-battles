package server

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestCatalogOnlyShowsReadySongs(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL catalog test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var readyID, pendingID string
	err = pool.QueryRow(ctx, `INSERT INTO songs (title, artist, genre, duration_seconds, status,
		full_song_path, drums_path, bass_path, melody_path, vocals_path)
		VALUES ('Ready catalog test', 'Test', 'Test', 12, 'READY', 'full', 'drums', 'bass', 'other', 'vocals')
		RETURNING id::text`).Scan(&readyID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM songs WHERE id = $1::uuid`, readyID) })
	err = pool.QueryRow(ctx, `INSERT INTO songs (title, artist, genre, duration_seconds)
		VALUES ('Pending catalog test', 'Test', 'Test', 12) RETURNING id::text`).Scan(&pendingID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM songs WHERE id = $1::uuid`, pendingID) })

	queries := dbgen.New(pool)
	listed, err := queries.ListSongs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var foundReady, foundPending bool
	for _, song := range listed {
		foundReady = foundReady || song.ID == readyID
		foundPending = foundPending || song.ID == pendingID
	}
	if !foundReady || foundPending {
		t.Fatalf("catalog contains ready=%t pending=%t, want true/false", foundReady, foundPending)
	}
	var id pgtype.UUID
	if err := id.Scan(pendingID); err != nil {
		t.Fatal(err)
	}
	_, err = queries.GetSong(ctx, id)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("lookup of pending song = %v, want not found", err)
	}
}
