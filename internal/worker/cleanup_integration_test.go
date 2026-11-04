package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func writeOldMedia(t *testing.T, path string, directory bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if directory {
		if err := os.MkdirAll(path, 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "leftover.mp3"), []byte("audio"), 0600); err != nil {
			t.Fatal(err)
		}
	} else if err := os.WriteFile(path, []byte("audio"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
}

func requirePath(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Stat(path)
	if want && err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
	if !want && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s to be absent: %v", path, err)
	}
}

func TestCleanupMediaReapsCrashArtifactsAndPreservesLiveFiles(t *testing.T) {
	pool := testPool(t)
	root := t.TempDir()
	queries := dbgen.New(pool)

	activeInput := filepath.Join(root, "incoming", "active.mp3")
	writeOldMedia(t, activeInput, false)
	activeID := insertQueuedJob(t, pool, activeInput)
	claimed, err := queries.ClaimSongJob(context.Background(), maxAttempts)
	if err != nil || claimed.SongID != activeID {
		t.Fatalf("claim active job = %+v, %v", claimed, err)
	}
	activeWork := filepath.Join(root, "work", activeID, "attempt-1")
	activeOutput := filepath.Join(root, "songs", activeID, "attempt-1")
	writeOldMedia(t, activeWork, true)
	writeOldMedia(t, activeOutput, true)

	readyInput := filepath.Join(root, "incoming", "ready.mp3")
	writeOldMedia(t, readyInput, false)
	readyID := insertQueuedJob(t, pool, readyInput)
	readyOutput := filepath.Join(root, "songs", readyID, "attempt-1")
	writeOldMedia(t, readyOutput, true)
	_, err = pool.Exec(context.Background(), `UPDATE song_jobs SET state = 'SUCCEEDED',
		attempt_count = 1 WHERE song_id = $1::uuid`, readyID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `UPDATE songs SET status = 'READY',
		full_song_path = $2, drums_path = $3, bass_path = $4, melody_path = $5, vocals_path = $6
		WHERE id = $1::uuid`, readyID, filepath.Join(readyOutput, "full.mp3"),
		filepath.Join(readyOutput, "drums.mp3"), filepath.Join(readyOutput, "bass.mp3"),
		filepath.Join(readyOutput, "other.mp3"), filepath.Join(readyOutput, "vocals.mp3"))
	if err != nil {
		t.Fatal(err)
	}
	staleOutput := filepath.Join(root, "songs", readyID, "attempt-0")
	writeOldMedia(t, staleOutput, true)

	expiredInput := filepath.Join(root, "incoming", "expired.mp3")
	writeOldMedia(t, expiredInput, false)
	expiredID := insertQueuedJob(t, pool, expiredInput)
	_, err = pool.Exec(context.Background(), `UPDATE song_jobs SET state = 'RUNNING',
		attempt_count = 1, lease_expires_at = now() - interval '1 hour'
		WHERE song_id = $1::uuid`, expiredID)
	if err != nil {
		t.Fatal(err)
	}
	expiredWork := filepath.Join(root, "work", expiredID, "attempt-1")
	expiredOutput := filepath.Join(root, "songs", expiredID, "attempt-1")
	writeOldMedia(t, expiredWork, true)
	writeOldMedia(t, expiredOutput, true)

	orphanInput := filepath.Join(root, "incoming", "song-orphan.mp3")
	writeOldMedia(t, orphanInput, false)
	recentOutput := filepath.Join(root, "songs", "orphan", "attempt-1")
	if err := os.MkdirAll(recentOutput, 0750); err != nil {
		t.Fatal(err)
	}

	runner := New(pool, nil, root, slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Minute)
	if err := runner.CleanupMedia(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{activeInput, activeWork, activeOutput, readyOutput, expiredInput, recentOutput} {
		requirePath(t, path, true)
	}
	for _, path := range []string{readyInput, staleOutput, expiredWork, expiredOutput, orphanInput} {
		requirePath(t, path, false)
	}
}
