package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
)

const maxAttempts int32 = 3

func expiredProcessingError() string {
	return fmt.Sprintf("Song processing stopped after %d attempts.", maxAttempts)
}

type jobTransitions struct {
	queries *dbgen.Queries
}

func (j jobTransitions) readyQueueStats(ctx context.Context) (int64, float64, error) {
	stats, err := j.queries.GetReadyQueueStats(ctx, maxAttempts)
	return stats.ReadyJobs, stats.OldestWaitSeconds, err
}

func (j jobTransitions) failExpired(ctx context.Context) ([]dbgen.FailExpiredSongJobsRow, error) {
	return j.queries.FailExpiredSongJobs(ctx, dbgen.FailExpiredSongJobsParams{
		MaxAttempts:     maxAttempts,
		ProcessingError: expiredProcessingError(),
	})
}

func (j jobTransitions) mediaSnapshot(ctx context.Context) (media.References, []string, error) {
	var refs media.References
	jobs, err := j.queries.ListSongJobMedia(ctx)
	if err != nil {
		return refs, nil, fmt.Errorf("list song jobs for media cleanup: %w", err)
	}
	songs, err := j.queries.ListSongOutputPaths(ctx)
	if err != nil {
		return refs, nil, fmt.Errorf("list song paths for media cleanup: %w", err)
	}

	now := time.Now()
	refs.ActiveAttempts = make(map[string]int32)
	var terminalInputs []string
	for _, job := range jobs {
		if job.State == "QUEUED" || job.State == "RUNNING" {
			refs.Inputs = append(refs.Inputs, job.InputPath)
		} else {
			terminalInputs = append(terminalInputs, job.InputPath)
		}
		if job.State == "RUNNING" && job.LeaseExpiresAt.Valid && job.LeaseExpiresAt.Time.After(now) {
			refs.ActiveAttempts[job.SongID] = job.AttemptCount
		}
	}
	for _, song := range songs {
		for _, path := range []pgtype.Text{
			song.FullSongPath, song.DrumsPath, song.BassPath,
			song.MelodyPath, song.VocalsPath,
		} {
			if path.Valid {
				refs.Outputs = append(refs.Outputs, path.String)
			}
		}
	}
	return refs, terminalInputs, nil
}

func (j jobTransitions) claim(ctx context.Context) (Job, error) {
	claimed, err := j.queries.ClaimSongJob(ctx, maxAttempts)
	if err != nil {
		return Job{}, err
	}
	return Job{
		SongID: claimed.SongID, InputPath: claimed.InputPath,
		ClipStartSeconds: claimed.ClipStartSeconds, AttemptCount: claimed.AttemptCount,
	}, nil
}

func (j jobTransitions) complete(ctx context.Context, id pgtype.UUID, attempt int32, files AudioFiles) error {
	_, err := j.queries.CompleteSongJob(ctx, dbgen.CompleteSongJobParams{
		SongID: id, AttemptCount: attempt,
		FullSongPath: files.FullSongPath, DrumsPath: files.DrumsPath,
		BassPath: files.BassPath, MelodyPath: files.MelodyPath, VocalsPath: files.VocalsPath,
	})
	return err
}

func (j jobTransitions) renew(ctx context.Context, id pgtype.UUID, attempt int32) error {
	_, err := j.queries.RenewSongJob(ctx, dbgen.RenewSongJobParams{SongID: id, AttemptCount: attempt})
	return err
}

func (j jobTransitions) fail(ctx context.Context, id pgtype.UUID, attempt int32, cause error) (Outcome, error) {
	if attempt < maxAttempts {
		_, err := j.queries.RetrySongJob(ctx, dbgen.RetrySongJobParams{
			SongID: id, AttemptCount: attempt, DelaySeconds: 15 * attempt,
		})
		if err != nil {
			return AttemptUnrecorded, fmt.Errorf("retry song job after %v: %w", cause, err)
		}
		return RetryScheduled, nil
	}
	_, err := j.queries.FailSongJob(ctx, dbgen.FailSongJobParams{
		SongID: id, AttemptCount: attempt,
		ProcessingError: fmt.Sprintf("Song processing failed after %d attempts.", maxAttempts),
	})
	if err != nil {
		return AttemptUnrecorded, fmt.Errorf("fail song job after %v: %w", cause, err)
	}
	return PermanentlyFailed, nil
}
