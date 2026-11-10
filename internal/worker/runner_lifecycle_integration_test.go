package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func workerTestInput(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "incoming", "song-test.mp3")
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("input"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func workerTestOutput(t *testing.T, root string, job Job) AudioFiles {
	t.Helper()
	dir := filepath.Join(root, "songs", job.SongID, "attempt-1")
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"full.mp3", "drums.mp3", "bass.mp3", "other.mp3", "vocals.mp3"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("audio"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return AudioFiles{Directory: dir, FullSongPath: filepath.Join(dir, "full.mp3"), DrumsPath: filepath.Join(dir, "drums.mp3"), BassPath: filepath.Join(dir, "bass.mp3"), MelodyPath: filepath.Join(dir, "other.mp3"), VocalsPath: filepath.Join(dir, "vocals.mp3")}
}

type completionTestDB struct {
	dbgen.DBTX
	apply bool
}

func (db completionTestDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if strings.HasPrefix(query, "-- name: CompleteSongJob") {
		var row pgx.Row
		if db.apply {
			row = db.DBTX.QueryRow(ctx, query, args...)
		}
		return completionTestRow{row: row}
	}
	return db.DBTX.QueryRow(ctx, query, args...)
}

type completionTestRow struct{ row pgx.Row }

func (r completionTestRow) Scan(dest ...any) error {
	if r.row != nil {
		if err := r.row.Scan(dest...); err != nil {
			return err
		}
	}
	return io.ErrUnexpectedEOF
}

func TestUncertainCompletionRetainsInputAndPublishedAudio(t *testing.T) {
	for _, apply := range []bool{false, true} {
		name := "no database update"
		if apply {
			name = "database update applied before lost reply"
		}
		t.Run(name, func(t *testing.T) {
			pool := testPool(t)
			root := t.TempDir()
			input := workerTestInput(t, root)
			id := insertQueuedJob(t, pool, input)
			var files AudioFiles
			processor := processorFunc(func(_ context.Context, job Job) (AudioFiles, error) {
				files = workerTestOutput(t, root, job)
				return files, nil
			})
			runner := New(pool, processor, root, slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultAttemptTimeout)
			runner.jobs.queries = dbgen.New(completionTestDB{DBTX: pool, apply: apply})
			outcome, err := runner.ProcessOne(context.Background())
			if outcome != AttemptUnrecorded || !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("completion=%s,%v", outcome, err)
			}
			for _, path := range []string{input, files.FullSongPath, files.DrumsPath, files.BassPath, files.MelodyPath, files.VocalsPath} {
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("uncertain completion removed %s: %v", path, err)
				}
			}
			var status, state string
			if err := pool.QueryRow(context.Background(), `SELECT s.status,j.state FROM songs s JOIN song_jobs j ON j.song_id=s.id WHERE s.id=$1::uuid`, id).Scan(&status, &state); err != nil {
				t.Fatal(err)
			}
			wantStatus, wantState := "PROCESSING", "RUNNING"
			if apply {
				wantStatus, wantState = "READY", "SUCCEEDED"
			}
			if status != wantStatus || state != wantState {
				t.Fatalf("database=%s/%s", status, state)
			}
			if got := testutil.ToFloat64(workerIterationOperational); got != 0 {
				t.Fatalf("unrecorded completion health=%v", got)
			}
		})
	}
}

func TestStaleCompletionDiscardsOnlyUnreferencedOutput(t *testing.T) {
	pool := testPool(t)
	root := t.TempDir()
	input := workerTestInput(t, root)
	id := insertQueuedJob(t, pool, input)
	var files AudioFiles
	processor := processorFunc(func(ctx context.Context, job Job) (AudioFiles, error) {
		files = workerTestOutput(t, root, job)
		_, err := pool.Exec(ctx, `UPDATE song_jobs SET attempt_count=attempt_count+1 WHERE song_id=$1::uuid`, id)
		return files, err
	})
	runner := New(pool, processor, root, slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultAttemptTimeout)
	outcome, err := runner.ProcessOne(context.Background())
	if outcome != AttemptUnrecorded || !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("stale completion=%s,%v", outcome, err)
	}
	if _, err := os.Stat(files.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale output remains: %v", err)
	}
	if _, err := os.Stat(input); err != nil {
		t.Fatalf("input for newer attempt removed: %v", err)
	}
}

func TestRenewalTimeoutCancelsProcessorAndKeepsJobRecoverable(t *testing.T) {
	pool := testPool(t)
	root := t.TempDir()
	input := workerTestInput(t, root)
	id := insertQueuedJob(t, pool, input)
	var lock pgx.Tx
	var files AudioFiles
	processor := processorFunc(func(ctx context.Context, job Job) (AudioFiles, error) {
		files = workerTestOutput(t, root, job)
		var err error
		lock, err = pool.Begin(ctx)
		if err != nil {
			return AudioFiles{}, err
		}
		var lockedID pgtype.UUID
		if err := lock.QueryRow(ctx, `SELECT song_id FROM song_jobs WHERE song_id=$1::uuid FOR UPDATE`, id).Scan(&lockedID); err != nil {
			return AudioFiles{}, err
		}
		<-ctx.Done()
		return files, nil
	})
	t.Cleanup(func() {
		if lock != nil {
			_ = lock.Rollback(context.Background())
		}
	})
	runner := New(pool, processor, root, slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Second)
	runner.renewInterval = 10 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	started := time.Now()
	outcome, err := runner.ProcessOne(ctx)
	if outcome != AttemptUnrecorded || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("renewal timeout=%s,%v", outcome, err)
	}
	if elapsed := time.Since(started); elapsed < renewQueryTimeout || elapsed > renewQueryTimeout+5*time.Second {
		t.Fatalf("renewal took %v, expected approximately %v", elapsed, renewQueryTimeout)
	}
	if err := lock.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(files.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("untrusted output retained: %v", err)
	}
	if _, err := os.Stat(input); err != nil {
		t.Fatalf("recoverable input lost: %v", err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM song_jobs WHERE song_id=$1::uuid`, id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "RUNNING" {
		t.Fatalf("lease failure incorrectly changed state=%s", state)
	}
	if got := testutil.ToFloat64(workerIterationOperational); got != 0 {
		t.Fatalf("renewal failure health=%v", got)
	}
}

func TestIdleWorkerDoesNotCountAnAttempt(t *testing.T) {
	pool := testPool(t)
	runner := New(pool, nil, t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultAttemptTimeout)
	before := attemptDurationCount(t)
	outcome, err := runner.ProcessOne(context.Background())
	if outcome != Idle || err != nil || attemptDurationCount(t) != before {
		t.Fatalf("idle result=%s,%v", outcome, err)
	}
}
