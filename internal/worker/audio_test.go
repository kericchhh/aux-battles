package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func testAudioInput(t *testing.T, root string) string {
	t.Helper()
	input := filepath.Join(root, "incoming", "test.mp3")
	if err := os.MkdirAll(filepath.Dir(input), 0750); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=13.2",
		"-q:a", "6", "-y", input)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("make test MP3: %v: %s", err, output)
	}
	return input
}

func testDemucsCommand(t *testing.T, root string, omitVocals bool) string {
	t.Helper()
	path := filepath.Join(root, "fake-demucs")
	script := "#!/bin/sh\nset -eu\n" +
		"test \"$1\" = '--name'\ntest \"$2\" = 'htdemucs'\n" +
		"test \"$3\" = '--mp3'\ntest \"$4\" = '--out'\n" +
		"mkdir -p \"$5/htdemucs/clip\"\n" +
		"for stem in drums bass other; do cp \"$6\" \"$5/htdemucs/clip/$stem.mp3\"; done\n"
	if !omitVocals {
		script += "cp \"$6\" \"$5/htdemucs/clip/vocals.mp3\"\n"
	}
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAudioProcessorPublishesClipAndStems(t *testing.T) {
	root := t.TempDir()
	input := testAudioInput(t, root)
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	processor := NewAudioProcessor(root, testDemucsCommand(t, root, false), logger)
	files, err := processor.Process(context.Background(), Job{
		SongID: "00000000-0000-4000-8000-000000000001", InputPath: input,
		ClipStartSeconds: 1, AttemptCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{files.FullSongPath, files.DrumsPath, files.BassPath, files.MelodyPath, files.VocalsPath} {
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			t.Fatalf("published audio %q: %v", path, err)
		}
	}
	output, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", files.FullSongPath).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(output) < 2 || string(output[:2]) != "12" {
		t.Fatalf("full clip duration = %q, want about 12 seconds", output)
	}
	decoder := json.NewDecoder(&logOutput)
	for _, stage := range []string{"extract_clip", "run_demucs", "validate_and_publish"} {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("read %s stage log: %v", stage, err)
		}
		if entry["stage"] != stage || entry["outcome"] != "succeeded" ||
			entry["song_id"] != "00000000-0000-4000-8000-000000000001" || entry["attempt"] != float64(1) {
			t.Fatalf("stage log = %v", entry)
		}
		if duration, ok := entry["duration_ms"].(float64); !ok || duration < 0 {
			t.Fatalf("invalid stage duration: %v", entry["duration_ms"])
		}
	}
}

func TestAudioProcessorRejectsMissingStemWithoutPublishing(t *testing.T) {
	root := t.TempDir()
	input := testAudioInput(t, root)
	var logOutput bytes.Buffer
	processor := NewAudioProcessor(root, testDemucsCommand(t, root, true), slog.New(slog.NewJSONHandler(&logOutput, nil)))
	job := Job{SongID: "00000000-0000-4000-8000-000000000002", InputPath: input, AttemptCount: 1}
	_, err := processor.Process(context.Background(), job)
	if err == nil {
		t.Fatal("processing without vocals succeeded")
	}
	if _, err := os.Stat(filepath.Join(root, "songs", job.SongID, "attempt-1")); !os.IsNotExist(err) {
		t.Fatalf("incomplete output was published: %v", err)
	}
	decoder := json.NewDecoder(&logOutput)
	var entry map[string]any
	for {
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("missing validation stage log: %v", err)
		}
		if entry["stage"] == "validate_and_publish" {
			break
		}
	}
	if entry["outcome"] != "failed" {
		t.Fatalf("missing stem stage log = %v", entry)
	}
}

func TestAudioProcessorRejectsInputOutsideIncomingDirectory(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside.mp3")
	if err := os.WriteFile(outside, []byte("audio"), 0600); err != nil {
		t.Fatal(err)
	}
	processor := NewAudioProcessor(root, testDemucsCommand(t, root, false), slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := processor.Process(context.Background(), Job{
		SongID: "00000000-0000-4000-8000-000000000003", InputPath: outside, AttemptCount: 1,
	})
	if err == nil {
		t.Fatal("input outside incoming directory was accepted")
	}
}
