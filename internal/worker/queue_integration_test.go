package worker

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_WORKER_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_WORKER_DATABASE_URL to run PostgreSQL worker tests")
	}
	if url == os.Getenv("TEST_DATABASE_URL") {
		t.Fatal("TEST_WORKER_DATABASE_URL must use a different database from TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func insertQueuedJob(t *testing.T, pool *pgxpool.Pool, inputPath string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `INSERT INTO songs (title, artist, genre, duration_seconds)
		VALUES ('Worker test', 'Test', 'Test', 13) RETURNING id::text`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM songs WHERE id = $1::uuid`, id); err != nil {
			t.Errorf("delete test song: %v", err)
		}
	})
	_, err = pool.Exec(context.Background(), `INSERT INTO song_jobs (song_id, input_path, clip_start_seconds)
		VALUES ($1::uuid, $2, 1)`, id, inputPath)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestClaimSongJobIsExclusiveAndFencesStaleAttempt(t *testing.T) {
	pool := testPool(t)
	id := insertQueuedJob(t, pool, "/tmp/worker-test.mp3")
	queries := dbgen.New(pool)

	type result struct {
		id      string
		attempt int32
		err     error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			job, err := queries.ClaimSongJob(context.Background())
			results <- result{id: job.SongID, attempt: job.AttemptCount, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	claimed, empty := 0, 0
	for result := range results {
		switch {
		case result.err == nil:
			claimed++
			if result.id != id || result.attempt != 1 {
				t.Fatalf("first claim = %+v, want song %s attempt 1", result, id)
			}
		case errors.Is(result.err, pgx.ErrNoRows):
			empty++
		default:
			t.Fatal(result.err)
		}
	}
	if claimed != 1 || empty != 1 {
		t.Fatalf("claims = %d, empty = %d; want 1 each", claimed, empty)
	}

	_, err := pool.Exec(context.Background(), `UPDATE song_jobs SET lease_expires_at = now() - interval '1 second'
		WHERE song_id = $1::uuid`, id)
	if err != nil {
		t.Fatal(err)
	}
	reclaimed, err := queries.ClaimSongJob(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed.SongID != id || reclaimed.AttemptCount != 2 {
		t.Fatalf("reclaimed job = %+v, want song %s attempt 2", reclaimed, id)
	}
	var songID pgtype.UUID
	if err := songID.Scan(id); err != nil {
		t.Fatal(err)
	}
	_, err = queries.CompleteSongJob(context.Background(), dbgen.CompleteSongJobParams{
		SongID: songID, AttemptCount: 1, FullSongPath: "full", DrumsPath: "drums",
		BassPath: "bass", MelodyPath: "other", VocalsPath: "vocals",
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("stale attempt completion = %v, want no rows", err)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM song_jobs WHERE song_id = $1::uuid`, id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "RUNNING" {
		t.Fatalf("job state after stale completion = %s, want RUNNING", state)
	}
}

func TestSongJobRetriesThenFails(t *testing.T) {
	pool := testPool(t)
	id := insertQueuedJob(t, pool, "/tmp/retry-test.mp3")
	queries := dbgen.New(pool)
	var songID pgtype.UUID
	if err := songID.Scan(id); err != nil {
		t.Fatal(err)
	}
	for attempt := int32(1); attempt <= 3; attempt++ {
		job, err := queries.ClaimSongJob(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if job.SongID != id || job.AttemptCount != attempt {
			t.Fatalf("claim = %+v, want song %s attempt %d", job, id, attempt)
		}
		if _, err := queries.RenewSongJob(context.Background(), dbgen.RenewSongJobParams{
			SongID: songID, AttemptCount: attempt,
		}); err != nil {
			t.Fatal(err)
		}
		if attempt < 3 {
			if _, err := queries.RetrySongJob(context.Background(), dbgen.RetrySongJobParams{
				SongID: songID, AttemptCount: attempt, DelaySeconds: 0,
			}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if _, err := queries.FailSongJob(context.Background(), dbgen.FailSongJobParams{
			SongID: songID, AttemptCount: attempt, ProcessingError: "Song processing failed after three attempts.",
		}); err != nil {
			t.Fatal(err)
		}
	}
	var songStatus, jobState, publicError string
	err := pool.QueryRow(context.Background(), `SELECT s.status, j.state, s.processing_error
		FROM songs s JOIN song_jobs j ON s.id = j.song_id WHERE s.id = $1::uuid`, id).
		Scan(&songStatus, &jobState, &publicError)
	if err != nil {
		t.Fatal(err)
	}
	if songStatus != "FAILED" || jobState != "FAILED" || publicError == "" {
		t.Fatalf("final status = %s/%s %q, want FAILED/FAILED and an error", songStatus, jobState, publicError)
	}
}

func TestExpiredThirdAttemptFailsWithoutReclaim(t *testing.T) {
	pool := testPool(t)
	id := insertQueuedJob(t, pool, "/tmp/expired-test.mp3")
	queries := dbgen.New(pool)
	for attempt := int32(1); attempt <= 3; attempt++ {
		job, err := queries.ClaimSongJob(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if job.SongID != id || job.AttemptCount != attempt {
			t.Fatalf("claim = %+v, want attempt %d", job, attempt)
		}
		_, err = pool.Exec(context.Background(), `UPDATE song_jobs SET lease_expires_at = now() - interval '1 second'
			WHERE song_id = $1::uuid`, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	expired, err := queries.FailExpiredSongJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 1 || expired[0].SongID != id {
		t.Fatalf("expired jobs = %+v, want song %s", expired, id)
	}
	var songStatus, jobState string
	if err := pool.QueryRow(context.Background(), `SELECT s.status, j.state FROM songs s
		JOIN song_jobs j ON s.id = j.song_id WHERE s.id = $1::uuid`, id).
		Scan(&songStatus, &jobState); err != nil {
		t.Fatal(err)
	}
	if songStatus != "FAILED" || jobState != "FAILED" {
		t.Fatalf("expired final attempt = %s/%s, want FAILED/FAILED", songStatus, jobState)
	}
}
