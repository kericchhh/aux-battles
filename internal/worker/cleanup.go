package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/media"
)

func (r *Runner) CleanupMedia(ctx context.Context) error {
	jobs, err := r.queries.ListSongJobMedia(ctx)
	if err != nil {
		return fmt.Errorf("list song jobs for media cleanup: %w", err)
	}
	songs, err := r.queries.ListSongOutputPaths(ctx)
	if err != nil {
		return fmt.Errorf("list song paths for media cleanup: %w", err)
	}

	now := time.Now()
	refs := media.References{ActiveAttempts: make(map[string]int32)}
	for _, job := range jobs {
		if job.State == "QUEUED" || job.State == "RUNNING" {
			refs.Inputs = append(refs.Inputs, job.InputPath)
		} else {
			r.removeInput(job.InputPath)
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

	removed, err := r.media.Cleanup(ctx, refs)
	for _, path := range removed.Inputs {
		r.logger.Info("removed orphaned song input", "path", path)
	}
	for _, path := range removed.Attempts {
		r.logger.Info("removed orphaned song attempt", "path", path)
	}
	return err
}
