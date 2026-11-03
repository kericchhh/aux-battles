package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type AudioProcessor struct {
	mediaRoot string
	demucsBin string
}

func NewAudioProcessor(mediaRoot, demucsBin string) *AudioProcessor {
	return &AudioProcessor{mediaRoot: mediaRoot, demucsBin: demucsBin}
}

func (p *AudioProcessor) Process(ctx context.Context, job Job) (AudioFiles, error) {
	var empty AudioFiles
	incoming := filepath.Join(p.mediaRoot, "incoming")
	if !pathWithin(incoming, job.InputPath) {
		return empty, fmt.Errorf("song input is outside incoming media directory")
	}
	realIncoming, err := filepath.EvalSymlinks(incoming)
	if err != nil {
		return empty, fmt.Errorf("resolve incoming directory: %w", err)
	}
	realInput, err := filepath.EvalSymlinks(job.InputPath)
	if err != nil {
		return empty, fmt.Errorf("resolve song input: %w", err)
	}
	if !pathWithin(realIncoming, realInput) {
		return empty, fmt.Errorf("song input resolves outside incoming media directory")
	}
	info, err := os.Stat(realInput)
	if err != nil {
		return empty, fmt.Errorf("stat song input: %w", err)
	}
	if !info.Mode().IsRegular() {
		return empty, fmt.Errorf("song input is not a regular file")
	}

	attempt := fmt.Sprintf("attempt-%d", job.AttemptCount)
	workDir := filepath.Join(p.mediaRoot, "work", job.SongID, attempt)
	resultDir := filepath.Join(workDir, "result")
	demucsDir := filepath.Join(workDir, "demucs")
	finalDir := filepath.Join(p.mediaRoot, "songs", job.SongID, attempt)
	if err := os.MkdirAll(resultDir, 0750); err != nil {
		return empty, fmt.Errorf("create song work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	clipPath := filepath.Join(workDir, "clip.mp3")
	if err := runCommand(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-ss", strconv.FormatFloat(job.ClipStartSeconds, 'f', -1, 64),
		"-i", realInput, "-t", "12", "-vn", "-ac", "2", "-ar", "44100",
		"-codec:a", "libmp3lame", "-q:a", "2", clipPath); err != nil {
		return empty, fmt.Errorf("extract song clip: %w", err)
	}
	if err := requireAudioFile(clipPath); err != nil {
		return empty, err
	}
	if err := runCommand(ctx, p.demucsBin, "--name", "htdemucs", "--mp3", "--out", demucsDir, clipPath); err != nil {
		return empty, fmt.Errorf("separate song: %w", err)
	}
	demucsOutput := filepath.Join(demucsDir, "htdemucs", "clip")
	for _, name := range []string{"drums.mp3", "bass.mp3", "other.mp3", "vocals.mp3"} {
		if err := requireAudioFile(filepath.Join(demucsOutput, name)); err != nil {
			return empty, err
		}
	}
	if err := os.Rename(clipPath, filepath.Join(resultDir, "full.mp3")); err != nil {
		return empty, fmt.Errorf("save full clip: %w", err)
	}
	for _, name := range []string{"drums.mp3", "bass.mp3", "other.mp3", "vocals.mp3"} {
		if err := os.Rename(filepath.Join(demucsOutput, name), filepath.Join(resultDir, name)); err != nil {
			return empty, fmt.Errorf("save %s stem: %w", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(finalDir), 0750); err != nil {
		return empty, fmt.Errorf("create song output directory: %w", err)
	}
	if _, err := os.Lstat(finalDir); err == nil {
		return empty, fmt.Errorf("song attempt output already exists")
	} else if !os.IsNotExist(err) {
		return empty, fmt.Errorf("check song output directory: %w", err)
	}
	if err := os.Rename(resultDir, finalDir); err != nil {
		return empty, fmt.Errorf("publish song output: %w", err)
	}
	return AudioFiles{
		Directory:    finalDir,
		FullSongPath: filepath.Join(finalDir, "full.mp3"),
		DrumsPath:    filepath.Join(finalDir, "drums.mp3"),
		BassPath:     filepath.Join(finalDir, "bass.mp3"),
		MelodyPath:   filepath.Join(finalDir, "other.mp3"),
		VocalsPath:   filepath.Join(finalDir, "vocals.mp3"),
	}, nil
}

func requireAudioFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect audio output %s: %w", filepath.Base(path), err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("audio output %s is empty or not a file", filepath.Base(path))
	}
	return nil
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
