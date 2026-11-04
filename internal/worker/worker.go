package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
)

const (
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

type recordedJobFailure struct {
	cause error
}

func (e *recordedJobFailure) Error() string { return e.cause.Error() }
func (e *recordedJobFailure) Unwrap() error { return e.cause }

type AudioFiles = media.Published

type Processor interface {
	Process(context.Context, Job) (AudioFiles, error)
}

type Runner struct {
	jobs           jobTransitions
	processor      Processor
	media          media.Store
	logger         *slog.Logger
	attemptTimeout time.Duration
}

func New(pool *pgxpool.Pool, processor Processor, mediaRoot string, logger *slog.Logger, attemptTimeout time.Duration) *Runner {
	if attemptTimeout <= 0 {
		attemptTimeout = DefaultAttemptTimeout
	}
	queries := dbgen.New(pool)
	return &Runner{jobs: jobTransitions{queries: queries}, processor: processor, media: media.New(mediaRoot), logger: logger, attemptTimeout: attemptTimeout}
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
			nextCleanup = time.Now().Add(media.CleanupInterval)
		}
		processed, err := r.ProcessOne(ctx)
		if err != nil && !processed {
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
	expired, err := r.jobs.failExpired(ctx)
	if err != nil {
		return false, fmt.Errorf("fail expired jobs: %w", err)
	}
	for _, job := range expired {
		logger := r.logger.With("song_id", job.SongID)
		logger.Error("song job exhausted after lease expiry")
		r.removeInputWithLogger(logger, job.InputPath)
	}
	job, err := r.jobs.claim(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim song job: %w", err)
	}
	logger := r.jobLogger(job)
	logger.Info("processing song")
	start := time.Now()
	err = r.process(ctx, job)
	outcome := "succeeded"
	if err != nil {
		outcome = "failed"
		var recorded *recordedJobFailure
		if !errors.As(err, &recorded) {
			logger.Error("song attempt failed", "error", err)
		}
	}
	logger.Info("song attempt finished", "outcome", outcome,
		"duration_ms", float64(time.Since(start))/float64(time.Millisecond))
	return true, err
}

func (r *Runner) jobLogger(job Job) *slog.Logger {
	return r.logger.With("song_id", job.SongID, "attempt", job.AttemptCount)
}

func (r *Runner) process(ctx context.Context, job Job) error {
	logger := r.jobLogger(job)
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
		r.discard(logger, files.Directory)
		return fmt.Errorf("renew song lease: %w", err)
	default:
	}
	if err := ctx.Err(); err != nil {
		r.discard(logger, files.Directory)
		return err
	}
	if errors.Is(workErr, context.DeadlineExceeded) {
		r.discard(logger, files.Directory)
		return r.recordFailure(ctx, job, songID, fmt.Errorf("song processing timed out: %w", workErr))
	}
	if processErr != nil {
		return r.recordFailure(ctx, job, songID, processErr)
	}

	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer finishCancel()
	err := r.jobs.complete(finishCtx, songID, job.AttemptCount, files)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.discard(logger, files.Directory)
		}
		return fmt.Errorf("complete song job: %w", err)
	}
	r.removeInputWithLogger(logger, job.InputPath)
	logger.Info("song ready")
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
			err := r.jobs.renew(ctx, id, attempt)
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
	terminal, err := r.jobs.fail(ctx, id, job.AttemptCount, cause)
	if err != nil {
		return err
	}
	if !terminal {
		r.jobLogger(job).Info("song processing will retry", "error", cause)
		return &recordedJobFailure{cause: cause}
	}
	logger := r.jobLogger(job)
	r.removeInputWithLogger(logger, job.InputPath)
	logger.Error("song processing failed", "error", cause)
	return &recordedJobFailure{cause: cause}
}

func (r *Runner) removeInput(path string) {
	r.removeInputWithLogger(r.logger, path)
}

func (r *Runner) removeInputWithLogger(logger *slog.Logger, path string) {
	err := r.media.RemoveInput(path)
	if errors.Is(err, media.ErrOutsideMedia) {
		logger.Error("refusing to remove input outside media directory", "path", path)
		return
	}
	if err != nil {
		logger.Error("remove song input", "path", path, "error", err)
	}
}

func (r *Runner) discard(logger *slog.Logger, directory string) {
	err := r.media.DiscardOutput(directory)
	if errors.Is(err, media.ErrOutsideMedia) {
		logger.Error("refusing to discard output outside media directory", "path", directory)
		return
	}
	if err != nil {
		logger.Error("discard unpublished song output", "path", directory, "error", err)
	}
}
