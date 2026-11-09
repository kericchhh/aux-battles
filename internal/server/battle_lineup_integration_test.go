package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func readyLineupTrack(t *testing.T, pool *pgxpool.Pool, root string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO songs(title,artist,genre,duration_seconds) VALUES('Cleared fixture','Test','Test',12) RETURNING id::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM battles WHERE id IN (SELECT battle_id FROM rounds WHERE host_song_id=$1::uuid OR guest_song_id=$1::uuid)`, id); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM songs WHERE id=$1::uuid`, id); err != nil {
			t.Error(err)
		}
	})
	dir := filepath.Join(root, "songs", id, "attempt-1")
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, 5)
	for _, name := range []string{"full.mp3", "drums.mp3", "bass.mp3", "other.mp3", "vocals.mp3"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("audio fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE songs SET status='READY',full_song_path=$2,drums_path=$3,bass_path=$4,melody_path=$5,vocals_path=$6 WHERE id=$1::uuid`, id, paths[0], paths[1], paths[2], paths[3], paths[4]); err != nil {
		t.Fatal(err)
	}
	return id
}
