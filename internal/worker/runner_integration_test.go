package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kericchhh/aux-battles/internal/media"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

type processorFunc func(context.Context, Job) (AudioFiles, error)

func (fn processorFunc) Process(ctx context.Context, job Job) (AudioFiles, error) {
	return fn(ctx, job)
}

func attemptDurationCount(t *testing.T) uint64 {
	t.Helper()
	metric := &dto.Metric{}
	if err := workerAttemptDuration.Write(metric); err != nil {
		t.Fatal(err)
	}
	return metric.GetHistogram().GetSampleCount()
}

func TestJobCleanupErrorsIncludeJobFields(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.mp3")
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	runner := &Runner{media: media.New(root), logger: logger}
	job := Job{SongID: "00000000-0000-4000-8000-000000000001", AttemptCount: 2}
	jobLogger := runner.jobLogger(job)
	runner.removeInputWithLogger(jobLogger, outside)
	runner.discard(jobLogger, outside)

	decoder := json.NewDecoder(&output)
	for _, message := range []string{
		"refusing to remove input outside media directory",
		"refusing to discard output outside media directory",
	} {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		if entry["msg"] != message || entry["song_id"] != job.SongID || entry["attempt"] != float64(job.AttemptCount) {
			t.Fatalf("cleanup log = %v", entry)
		}
	}
}

func TestRunnerCompletesSongAndRemovesInput(t *testing.T) {
	beforeSuccess := testutil.ToFloat64(workerAttempts.WithLabelValues("succeeded"))
	beforeDuration := attemptDurationCount(t)
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
	var logOutput bytes.Buffer
	runner := New(pool, processor, root, slog.New(slog.NewJSONHandler(&logOutput, nil)), 30*time.Minute)
	outcome, err := runner.ProcessOne(context.Background())
	if err != nil || outcome != Completed {
		t.Fatalf("process one = %s, %v; want completed, nil", outcome, err)
	}
	if got := testutil.ToFloat64(workerIterationOperational); got != 1 {
		t.Fatalf("worker iteration operational = %v, want 1", got)
	}
	if got := testutil.ToFloat64(workerAttempts.WithLabelValues("succeeded")); got != beforeSuccess+1 {
		t.Fatalf("successful attempt count = %v, want %v", got, beforeSuccess+1)
	}
	if got := attemptDurationCount(t); got != beforeDuration+1 {
		t.Fatalf("attempt duration count = %v, want %v", got, beforeDuration+1)
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
	decoder := json.NewDecoder(&logOutput)
	found := false
	for {
		var entry map[string]any
		err := decoder.Decode(&entry)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if entry["msg"] != "song attempt finished" {
			continue
		}
		found = true
		if entry["song_id"] != id || entry["attempt"] != float64(1) || entry["outcome"] != "succeeded" {
			t.Fatalf("attempt log = %v", entry)
		}
		if duration, ok := entry["duration_ms"].(float64); !ok || duration < 0 {
			t.Fatalf("invalid attempt duration: %v", entry["duration_ms"])
		}
	}
	if !found {
		t.Fatalf("missing song attempt log: %s", logOutput.String())
	}
}

func TestRunnerRetriesThenMarksSongFailed(t *testing.T) {
	beforeFailures := testutil.ToFloat64(workerAttempts.WithLabelValues("failed"))
	beforeDuration := attemptDurationCount(t)
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
		outcome, err := runner.ProcessOne(context.Background())
		want := RetryScheduled
		if attempt == 3 {
			want = PermanentlyFailed
		}
		if outcome != want || err != nil {
			t.Fatalf("attempt %d = %s, %v; want %s, nil", attempt, outcome, err, want)
		}
		if attempt < 3 {
			var delaySeconds float64
			if err := pool.QueryRow(context.Background(), `SELECT EXTRACT(EPOCH FROM available_at - updated_at)::double precision
				FROM song_jobs WHERE song_id = $1::uuid`, id).Scan(&delaySeconds); err != nil {
				t.Fatal(err)
			}
			if want := float64(15 * attempt); delaySeconds != want {
				t.Fatalf("retry delay after attempt %d = %v seconds, want %v", attempt, delaySeconds, want)
			}
			_, err := pool.Exec(context.Background(), `UPDATE song_jobs SET available_at = now()
				WHERE song_id = $1::uuid`, id)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := testutil.ToFloat64(workerIterationOperational); got != 1 {
		t.Fatalf("recorded failure left worker iteration operational = %v, want 1", got)
	}
	if got := testutil.ToFloat64(workerAttempts.WithLabelValues("failed")); got != beforeFailures+3 {
		t.Fatalf("failed attempt count = %v, want %v", got, beforeFailures+3)
	}
	if got := attemptDurationCount(t); got != beforeDuration+3 {
		t.Fatalf("attempt duration count = %v, want %v", got, beforeDuration+3)
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
	outcome, err := runner.ProcessOne(context.Background())
	if outcome != RetryScheduled || err != nil {
		t.Fatalf("timed out attempt = %s, %v; want retry scheduled, nil", outcome, err)
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

func TestQueueSampleReportsJobsAndDatabaseFailure(t *testing.T) {
	pool := testPool(t)
	insertQueuedJob(t, pool, "/tmp/queue-metrics-test.mp3")
	runner := New(pool, nil, t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultAttemptTimeout)
	if err := runner.sampleQueue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := testutil.ToFloat64(workerQueueSampleSuccess); got != 1 {
		t.Fatalf("queue sample success = %v, want 1", got)
	}
	if got := testutil.ToFloat64(workerReadyJobs); got != 1 {
		t.Fatalf("ready jobs = %v, want 1", got)
	}
	if got := testutil.ToFloat64(workerOldestReadyJobWait); got < 0 {
		t.Fatalf("oldest wait = %v, want nonnegative", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runner.sampleQueue(ctx); err == nil {
		t.Fatal("canceled queue query unexpectedly succeeded")
	}
	if got := testutil.ToFloat64(workerQueueSampleSuccess); got != 0 {
		t.Fatalf("queue sample success after query failure = %v, want 0", got)
	}
	if got := testutil.ToFloat64(workerReadyJobs); got != 0 {
		t.Fatalf("ready jobs after query failure = %v, want 0", got)
	}
	outcome, err := runner.ProcessOne(ctx)
	if outcome != PollFailed || err == nil {
		t.Fatalf("canceled worker iteration = %s, %v; want poll failure", outcome, err)
	}
	if got := testutil.ToFloat64(workerIterationOperational); got != 0 {
		t.Fatalf("worker iteration operational after query failure = %v, want 0", got)
	}
}
