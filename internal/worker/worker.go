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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	pollInterval          = time.Second
	maxPollBackoff        = 30 * time.Second
	queueSampleInterval   = 15 * time.Second
	queueSampleTimeout    = 5 * time.Second
	renewInterval         = time.Minute
	renewQueryTimeout     = 5 * time.Second
	DefaultAttemptTimeout = 30 * time.Minute
)

var (
	workerAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aux_battles_worker_attempts_total",
			Help: "Song processing attempts by outcome",
		}, []string{"outcome"},
	)
	workerAttemptDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "aux_battles_worker_attempt_duration_seconds",
			Help:    "Time spent processing an attempt",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12),
		},
	)
	workerQueueSampleSuccess = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "aux_battles_worker_queue_sample_success",
		Help: "Whether the last worker queue sample succeeded.",
	})
	workerIterationOperational = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "aux_battles_worker_iteration_operational",
		Help: "Whether the last worker iteration reached a recorded job state or found no job.",
	})
	workerReadyJobs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "aux_battles_worker_ready_jobs",
		Help: "Queued or expired running song jobs eligible for processing.",
	})
	workerOldestReadyJobWait = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "aux_battles_worker_oldest_ready_job_wait_seconds",
		Help: "Seconds since the oldest eligible queued job became available or running job lease expired.",
	})
)

type Job struct {
	SongID           string
	InputPath        string
	ClipStartSeconds float64
	AttemptCount     int32
}

// Outcome describes how a worker iteration ended.
type Outcome string

const (
	Idle              Outcome = "idle"
	Completed         Outcome = "completed"
	RetryScheduled    Outcome = "retry_scheduled"
	PermanentlyFailed Outcome = "permanently_failed"
	PollFailed        Outcome = "poll_failed"
	AttemptUnrecorded Outcome = "attempt_unrecorded"
)

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
	renewInterval  time.Duration
}

func New(pool *pgxpool.Pool, processor Processor, mediaRoot string, logger *slog.Logger, attemptTimeout time.Duration) *Runner {
	if attemptTimeout <= 0 {
		attemptTimeout = DefaultAttemptTimeout
	}
	queries := dbgen.New(pool)
	return &Runner{
		jobs:           jobTransitions{queries: queries},
		processor:      processor,
		media:          media.New(mediaRoot),
		logger:         logger,
		attemptTimeout: attemptTimeout,
		renewInterval:  renewInterval,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	backoff := pollInterval
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
		outcome, err := r.ProcessOne(ctx)
		if ctx.Err() != nil {
			return nil
		}
		wait := pollInterval
		if outcome == PollFailed {
			wait = backoff
			r.logger.Error("song worker iteration failed", "error", err, "retry_in", wait.String())
			backoff = min(backoff*2, maxPollBackoff)
		} else {
			backoff = pollInterval
		}
		if outcome == Completed {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}
	}
}

func (r *Runner) MonitorQueue(ctx context.Context) error {
	ticker := time.NewTicker(queueSampleInterval)
	defer ticker.Stop()
	hadFailure := false
	for {
		if ctx.Err() != nil {
			return nil
		}
		sampleCtx, cancel := context.WithTimeout(ctx, queueSampleTimeout)
		err := r.sampleQueue(sampleCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if !hadFailure {
				r.logger.Error("worker queue sample failed", "error", err)
				hadFailure = true
			}
		} else {
			if hadFailure {
				r.logger.Info("worker queue sampling recovered")
				hadFailure = false
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Runner) sampleQueue(ctx context.Context) error {
	readyJobs, oldestWait, err := r.jobs.readyQueueStats(ctx)
	if err != nil {
		workerQueueSampleSuccess.Set(0)
		workerReadyJobs.Set(0)
		workerOldestReadyJobWait.Set(0)
		return err
	}
	workerQueueSampleSuccess.Set(1)
	workerReadyJobs.Set(float64(readyJobs))
	workerOldestReadyJobWait.Set(oldestWait)
	return nil
}

func (r *Runner) ProcessOne(ctx context.Context) (Outcome, error) {
	expired, err := r.jobs.failExpired(ctx)
	if err != nil {
		workerIterationOperational.Set(0)
		return PollFailed, fmt.Errorf("fail expired jobs: %w", err)
	}
	for _, job := range expired {
		logger := r.logger.With("song_id", job.SongID)
		logger.Error("song job exhausted after lease expiry")
		r.removeInputWithLogger(logger, job.InputPath)
	}
	job, err := r.jobs.claim(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		workerIterationOperational.Set(1)
		return Idle, nil
	}
	if err != nil {
		workerIterationOperational.Set(0)
		return PollFailed, fmt.Errorf("claim song job: %w", err)
	}
	workerIterationOperational.Set(1)
	logger := r.jobLogger(job)
	logger.Info("processing song")
	start := time.Now()
	result, err := r.processAttempt(ctx, job)
	outcome := "failed"
	if result == Completed {
		outcome = "succeeded"
	}
	if err != nil {
		workerIterationOperational.Set(0)
		logger.Error("song attempt failed", "error", err)
	}
	duration := time.Since(start)
	workerAttempts.WithLabelValues(outcome).Inc()
	workerAttemptDuration.Observe(duration.Seconds())
	logger.Info("song attempt finished", "outcome", outcome,
		"duration_ms", float64(duration)/float64(time.Millisecond))
	return result, err
}

func (r *Runner) jobLogger(job Job) *slog.Logger {
	return r.logger.With("song_id", job.SongID, "attempt", job.AttemptCount)
}

func (r *Runner) processAttempt(ctx context.Context, job Job) (Outcome, error) {
	logger := r.jobLogger(job)
	var songID pgtype.UUID
	if err := songID.Scan(job.SongID); err != nil {
		return AttemptUnrecorded, fmt.Errorf("parse claimed song ID: %w", err)
	}
	workCtx, cancel := context.WithTimeout(ctx, r.attemptTimeout)
	defer cancel()
	leaseResult := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(r.renewInterval)
		defer ticker.Stop()
		err := maintainLease(workCtx, ticker.C, func(renewCtx context.Context) error {
			return r.jobs.renew(renewCtx, songID, job.AttemptCount)
		})
		leaseResult <- err
		if err != nil {
			cancel()
		}
	}()
	files, processErr := r.processor.Process(workCtx, job)
	workErr := workCtx.Err()
	cancel()
	if err := <-leaseResult; err != nil {
		r.discard(logger, files.Directory)
		return AttemptUnrecorded, fmt.Errorf("renew song lease: %w", err)
	}
	if err := ctx.Err(); err != nil {
		r.discard(logger, files.Directory)
		return AttemptUnrecorded, err
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
		return AttemptUnrecorded, fmt.Errorf("complete song job: %w", err)
	}
	r.removeInputWithLogger(logger, job.InputPath)
	logger.Info("song ready")
	return Completed, nil
}

func maintainLease(ctx context.Context, ticks <-chan time.Time, renew func(context.Context) error) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticks:
			renewCtx, cancel := context.WithTimeout(ctx, renewQueryTimeout)
			err := renew(renewCtx)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

func (r *Runner) recordFailure(parent context.Context, job Job, id pgtype.UUID, cause error) (Outcome, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancel()
	result, err := r.jobs.fail(ctx, id, job.AttemptCount, cause)
	if err != nil {
		return AttemptUnrecorded, err
	}
	if result == RetryScheduled {
		r.jobLogger(job).Info("song processing will retry", "error", cause)
		return result, nil
	}
	logger := r.jobLogger(job)
	r.removeInputWithLogger(logger, job.InputPath)
	logger.Error("song processing failed", "error", cause)
	return result, nil
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
