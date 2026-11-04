package worker

import (
	"context"
)

func (r *Runner) CleanupMedia(ctx context.Context) error {
	refs, terminalInputs, err := r.jobs.mediaSnapshot(ctx)
	if err != nil {
		return err
	}
	for _, path := range terminalInputs {
		r.removeInput(path)
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
