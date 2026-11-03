package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type processorFunc func(context.Context, Job) (AudioFiles, error)

func (fn processorFunc) Process(ctx context.Context, job Job) (AudioFiles, error) {
	return fn(ctx, job)
}

func TestRunnerCompletesSongAndRemovesInput(t *testing.T) {
	pool := testPool(t)
	root := t.TempDir()
	input := filepath.Join(root, "incoming", "upload.mp3")
	if err := os.MkdirAll(filepath.Dir(input), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("test input"), 0600); err != nil {
		t.Fatal(err)
	}
	id := insertQueuedJob(t, pool, input)
	processor := processorFunc(func(_ context.Context, job Job) (AudioFiles, error) {
		if job.SongID != id || job.AttemptCount != 1 || job.ClipStartSeconds != 1 {
			return AudioFiles{}, fmt.Errorf("wrong job: %+v", job)
		}
		dir := filepath.Join(root, "songs", id, "attempt-1")
		if err := os.MkdirAll(dir, 0750); err != nil {
			return AudioFiles{}, err
		}
		for _, name := range []string{"full.mp3", "drums.mp3", "bass.mp3", "other.mp3", "vocals.mp3"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("audio"), 0600); err != nil {
				return AudioFiles{}, err
			}
		}
		return AudioFiles{Directory: dir, FullSongPath: filepath.Join(dir, "full.mp3"),
			DrumsPath: filepath.Join(dir, "drums.mp3"), BassPath: filepath.Join(dir, "bass.mp3"),
			MelodyPath: filepath.Join(dir, "other.mp3"), VocalsPath: filepath.Join(dir, "vocals.mp3")}, nil
	})
	runner := New(pool, processor, root, slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Minute)
	processed, err := runner.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("process one = %t, %v; want true, nil", processed, err)
	}
	var status, state, fullPath string
	if err := pool.QueryRow(context.Background(), `SELECT s.status, j.state, s.full_song_path
		FROM songs s JOIN song_jobs j ON s.id = j.song_id WHERE s.id = $1::uuid`, id).
		Scan(&status, &state, &fullPath); err != nil {
		t.Fatal(err)
	}
	if status != "READY" || state != "SUCCEEDED" {
		t.Fatalf("song/job = %s/%s, want READY/SUCCEEDED", status, state)
	}
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("published full clip is missing: %v", err)
	}
	if _, err := os.Stat(input); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("input remains after success: %v", err)
	}
}

func TestRunnerRetriesThenMarksSongFailed(t *testing.T) {
	pool := testPool(t)
	root := t.TempDir()
	input := filepath.Join(root, "incoming", "upload.mp3")
	if err := os.MkdirAll(filepath.Dir(input), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("test input"), 0600); err != nil {
		t.Fatal(err)
	}
	id := insertQueuedJob(t, pool, input)
	processor := processorFunc(func(context.Context, Job) (AudioFiles, error) {
		return AudioFiles{}, errors.New("private demucs details")
	})
	runner := New(pool, processor, root, slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Minute)
	for attempt := 1; attempt <= 3; attempt++ {
		processed, err := runner.ProcessOne(context.Background())
		if !processed || err == nil {
			t.Fatalf("attempt %d = %t, %v; want processed error", attempt, processed, err)
		}
		if attempt < 3 {
			_, err := pool.Exec(context.Background(), `UPDATE song_jobs SET available_at = now()
				WHERE song_id = $1::uuid`, id)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	var status, state, publicError string
	if err := pool.QueryRow(context.Background(), `SELECT s.status, j.state, s.processing_error
		FROM songs s JOIN song_jobs j ON s.id = j.song_id WHERE s.id = $1::uuid`, id).
		Scan(&status, &state, &publicError); err != nil {
		t.Fatal(err)
	}
	if status != "FAILED" || state != "FAILED" || publicError == "" || publicError == "private demucs details" {
		t.Fatalf("final state = %s/%s %q, want FAILED/FAILED and safe error", status, state, publicError)
	}
	if _, err := os.Stat(input); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("input remains after final failure: %v", err)
	}
}

func TestRunnerTimesOutStalledProcessorAndRequeues(t *testing.T) {
	pool := testPool(t)
	id := insertQueuedJob(t, pool, "/tmp/timeout-test.mp3")
	processor := processorFunc(func(ctx context.Context, _ Job) (AudioFiles, error) {
		<-ctx.Done()
		return AudioFiles{}, nil
	})
	runner := New(pool, processor, t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Millisecond)
	processed, err := runner.ProcessOne(context.Background())
	if !processed || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timed out attempt = %t, %v; want processed deadline error", processed, err)
	}
	var songStatus, jobState string
	var attempts int32
	if err := pool.QueryRow(context.Background(), `SELECT s.status, j.state, j.attempt_count
		FROM songs s JOIN song_jobs j ON s.id = j.song_id WHERE s.id = $1::uuid`, id).
		Scan(&songStatus, &jobState, &attempts); err != nil {
		t.Fatal(err)
	}
	if songStatus != "PROCESSING" || jobState != "QUEUED" || attempts != 1 {
		t.Fatalf("timed out job = %s/%s attempt %d, want PROCESSING/QUEUED attempt 1", songStatus, jobState, attempts)
	}
}
