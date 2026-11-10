package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/kericchhh/aux-battles/internal/media"
)

type AudioProcessor struct {
	media     media.Store
	demucsBin string
	logger    *slog.Logger
}

func NewAudioProcessor(mediaRoot, demucsBin string, logger *slog.Logger) *AudioProcessor {
	return &AudioProcessor{media: media.New(mediaRoot), demucsBin: demucsBin, logger: logger}
}

func (p *AudioProcessor) Process(ctx context.Context, job Job) (AudioFiles, error) {
	var empty AudioFiles
	realInput, err := p.media.InputPath(job.InputPath)
	if err != nil {
		return empty, err
	}

	attempt, err := p.media.PrepareAttempt(job.SongID, job.AttemptCount)
	if err != nil {
		return empty, err
	}
	defer attempt.Cleanup()
	logger := p.logger.With("song_id", job.SongID, "attempt", job.AttemptCount)

	start := time.Now()
	if err := runCommand(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-ss", strconv.FormatFloat(job.ClipStartSeconds, 'f', -1, 64),
		"-i", realInput, "-t", "12", "-vn", "-ac", "2", "-ar", "44100",
		"-codec:a", "libmp3lame", "-q:a", "2", attempt.ClipPath); err != nil {
		logAudioStage(logger, "extract_clip", start, err)
		return empty, fmt.Errorf("extract song clip: %w", err)
	}
	if err := attempt.CheckClip(); err != nil {
		logAudioStage(logger, "extract_clip", start, err)
		return empty, err
	}
	logAudioStage(logger, "extract_clip", start, nil)
	start = time.Now()
	if err := runCommand(ctx, p.demucsBin, "--name", "htdemucs", "--mp3", "--out", attempt.DemucsDir, attempt.ClipPath); err != nil {
		logAudioStage(logger, "run_demucs", start, err)
		return empty, fmt.Errorf("separate song: %w", err)
	}
	logAudioStage(logger, "run_demucs", start, nil)
	start = time.Now()
	files, err := attempt.Publish()
	logAudioStage(logger, "validate_and_publish", start, err)
	return files, err
}

func logAudioStage(logger *slog.Logger, stage string, start time.Time, err error) {
	outcome := "succeeded"
	if err != nil {
		outcome = "failed"
	}
	logger.Info("song audio stage finished", "stage", stage, "outcome", outcome,
		"duration_ms", float64(time.Since(start))/float64(time.Millisecond))
}
