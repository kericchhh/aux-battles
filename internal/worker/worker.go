package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

const (
	maxAttempts           = 3
	pollInterval          = time.Second
	renewInterval         = time.Minute
	DefaultAttemptTimeout = 30 * time.Minute
)

type Job struct {
	SongID           string
	InputPath        string
	ClipStartSeconds float64
	AttemptCount     int32
}

type AudioFiles struct {
	Directory    string
	FullSongPath string
	DrumsPath    string
	BassPath     string
	MelodyPath   string
	VocalsPath   string
}

type Processor interface {
	Process(context.Context, Job) (AudioFiles, error)
}

type Runner struct {
	queries        *dbgen.Queries
	processor      Processor
	mediaRoot      string
	logger         *slog.Logger
	attemptTimeout time.Duration
}

func New(pool *pgxpool.Pool, processor Processor, mediaRoot string, logger *slog.Logger, attemptTimeout time.Duration) *Runner {
	if attemptTimeout <= 0 {
		attemptTimeout = DefaultAttemptTimeout
	}
	return &Runner{queries: dbgen.New(pool), processor: processor, mediaRoot: mediaRoot, logger: logger, attemptTimeout: attemptTimeout}
}

func (r *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	nextCleanup := time.Time{}
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if !time.Now().Before(nextCleanup) {
			if err := r.CleanupMedia(ctx); err != nil {
				r.logger.Error("media cleanup failed", "error", err)
			}
			nextCleanup = time.Now().Add(mediaCleanupInterval)
		}
		processed, err := r.ProcessOne(ctx)
		if err != nil {
			r.logger.Error("song worker iteration failed", "error", err)
		}
		if processed && err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Runner) ProcessOne(ctx context.Context) (bool, error) {
	expired, err := r.queries.FailExpiredSongJobs(ctx)
	if err != nil {
		return false, fmt.Errorf("fail expired jobs: %w", err)
	}
	for _, job := range expired {
		r.logger.Warn("song job exhausted after lease expiry", "song_id", job.SongID)
		r.removeInput(job.InputPath)
	}
	claimed, err := r.queries.ClaimSongJob(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim song job: %w", err)
	}
	job := Job{
		SongID: claimed.SongID, InputPath: claimed.InputPath,
		ClipStartSeconds: claimed.ClipStartSeconds, AttemptCount: claimed.AttemptCount,
	}
	r.logger.Info("processing song", "song_id", job.SongID, "attempt", job.AttemptCount)
	return true, r.process(ctx, job)
}

func (r *Runner) process(ctx context.Context, job Job) error {
	var songID pgtype.UUID
	if err := songID.Scan(job.SongID); err != nil {
		return fmt.Errorf("parse claimed song ID: %w", err)
	}
	workCtx, cancel := context.WithTimeout(ctx, r.attemptTimeout)
	defer cancel()
	renewResult := make(chan error, 1)
	done := make(chan struct{})
	go r.renewLease(workCtx, cancel, songID, job.AttemptCount, renewResult, done)
	files, processErr := r.processor.Process(workCtx, job)
	workErr := workCtx.Err()
	cancel()
	<-done
	select {
	case err := <-renewResult:
		r.discard(files.Directory)
		return fmt.Errorf("renew song lease: %w", err)
	default:
	}
	if err := ctx.Err(); err != nil {
		r.discard(files.Directory)
		return err
	}
	if errors.Is(workErr, context.DeadlineExceeded) {
		r.discard(files.Directory)
		return r.recordFailure(ctx, job, songID, fmt.Errorf("song processing timed out: %w", workErr))
	}
	if processErr != nil {
		return r.recordFailure(ctx, job, songID, processErr)
	}

	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer finishCancel()
	_, err := r.queries.CompleteSongJob(finishCtx, dbgen.CompleteSongJobParams{
		SongID: songID, AttemptCount: job.AttemptCount,
		FullSongPath: files.FullSongPath, DrumsPath: files.DrumsPath,
		BassPath: files.BassPath, MelodyPath: files.MelodyPath, VocalsPath: files.VocalsPath,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.discard(files.Directory)
		}
		return fmt.Errorf("complete song job: %w", err)
	}
	r.removeInput(job.InputPath)
	r.logger.Info("song ready", "song_id", job.SongID)
	return nil
}

func (r *Runner) renewLease(ctx context.Context, cancel context.CancelFunc, id pgtype.UUID, attempt int32, result chan<- error, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := r.queries.RenewSongJob(ctx, dbgen.RenewSongJobParams{SongID: id, AttemptCount: attempt})
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				result <- err
				cancel()
				return
			}
		}
	}
}

func (r *Runner) recordFailure(parent context.Context, job Job, id pgtype.UUID, cause error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancel()
	if job.AttemptCount < maxAttempts {
		_, err := r.queries.RetrySongJob(ctx, dbgen.RetrySongJobParams{
			SongID: id, AttemptCount: job.AttemptCount,
			DelaySeconds: 15 * job.AttemptCount,
		})
		if err != nil {
			return fmt.Errorf("retry song job after %v: %w", cause, err)
		}
		r.logger.Warn("song processing will retry", "song_id", job.SongID,
			"attempt", job.AttemptCount, "error", cause)
		return cause
	}
	_, err := r.queries.FailSongJob(ctx, dbgen.FailSongJobParams{
		SongID: id, AttemptCount: job.AttemptCount,
		ProcessingError: "Song processing failed after three attempts.",
	})
	if err != nil {
		return fmt.Errorf("fail song job after %v: %w", cause, err)
	}
	r.removeInput(job.InputPath)
	r.logger.Error("song processing failed", "song_id", job.SongID, "error", cause)
	return cause
}

func (r *Runner) removeInput(path string) {
	if !pathWithin(filepath.Join(r.mediaRoot, "incoming"), path) {
		r.logger.Warn("refusing to remove input outside media directory", "path", path)
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		r.logger.Warn("remove song input", "path", path, "error", err)
	}
}

func (r *Runner) discard(directory string) {
	if directory == "" {
		return
	}
	if !pathWithin(filepath.Join(r.mediaRoot, "songs"), directory) {
		r.logger.Warn("refusing to discard output outside media directory", "path", directory)
		return
	}
	if err := os.RemoveAll(directory); err != nil {
		r.logger.Warn("discard unpublished song output", "path", directory, "error", err)
	}
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != "." && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
