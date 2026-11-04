package worker

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kericchhh/aux-battles/internal/media"
)

type AudioProcessor struct {
	media     media.Store
	demucsBin string
}

func NewAudioProcessor(mediaRoot, demucsBin string) *AudioProcessor {
	return &AudioProcessor{media: media.New(mediaRoot), demucsBin: demucsBin}
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

	if err := runCommand(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-ss", strconv.FormatFloat(job.ClipStartSeconds, 'f', -1, 64),
		"-i", realInput, "-t", "12", "-vn", "-ac", "2", "-ar", "44100",
		"-codec:a", "libmp3lame", "-q:a", "2", attempt.ClipPath); err != nil {
		return empty, fmt.Errorf("extract song clip: %w", err)
	}
	if err := attempt.CheckClip(); err != nil {
		return empty, err
	}
	if err := runCommand(ctx, p.demucsBin, "--name", "htdemucs", "--mp3", "--out", attempt.DemucsDir, attempt.ClipPath); err != nil {
		return empty, fmt.Errorf("separate song: %w", err)
	}
	return attempt.Publish()
}

func runCommand(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	detail := strings.TrimSpace(string(output))
	if len(detail) > 2000 {
		detail = detail[len(detail)-2000:]
	}
	if detail == "" {
		return fmt.Errorf("%s: %w", filepath.Base(name), err)
	}
	return fmt.Errorf("%s: %w: %s", filepath.Base(name), err, detail)
}
