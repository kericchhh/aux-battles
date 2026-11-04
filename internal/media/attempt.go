package media

import (
	"fmt"
	"os"
	"path/filepath"
)

type Attempt struct {
	ClipPath  string
	DemucsDir string
	workDir   string
	resultDir string
	finalDir  string
}

type Published struct {
	Directory    string
	FullSongPath string
	DrumsPath    string
	BassPath     string
	MelodyPath   string
	VocalsPath   string
}

func (s Store) PrepareAttempt(songID string, number int32) (Attempt, error) {
	workRoot, err := s.dir("work")
	if err != nil {
		return Attempt{}, fmt.Errorf("resolve song work directory: %w", err)
	}
	songsRoot, err := s.dir("songs")
	if err != nil {
		return Attempt{}, fmt.Errorf("resolve song output directory: %w", err)
	}
	name := fmt.Sprintf("attempt-%d", number)
	workDir := filepath.Join(workRoot, songID, name)
	resultDir := filepath.Join(workDir, "result")
	if err := os.MkdirAll(resultDir, 0750); err != nil {
		return Attempt{}, fmt.Errorf("create song work directory: %w", err)
	}
	return Attempt{
		ClipPath:  filepath.Join(workDir, "clip.mp3"),
		DemucsDir: filepath.Join(workDir, "demucs"),
		workDir:   workDir,
		resultDir: resultDir,
		finalDir:  filepath.Join(songsRoot, songID, name),
	}, nil
}

func (a Attempt) CheckClip() error {
	return requireAudioFile(a.ClipPath)
}

func (a Attempt) Cleanup() {
	_ = os.RemoveAll(a.workDir)
}

func (a Attempt) Publish() (Published, error) {
	demucsOutput := filepath.Join(a.DemucsDir, "htdemucs", "clip")
	for _, name := range []string{"drums.mp3", "bass.mp3", "other.mp3", "vocals.mp3"} {
		if err := requireAudioFile(filepath.Join(demucsOutput, name)); err != nil {
			return Published{}, err
		}
	}
	if err := os.Rename(a.ClipPath, filepath.Join(a.resultDir, "full.mp3")); err != nil {
		return Published{}, fmt.Errorf("save full clip: %w", err)
	}
	for _, name := range []string{"drums.mp3", "bass.mp3", "other.mp3", "vocals.mp3"} {
		if err := os.Rename(filepath.Join(demucsOutput, name), filepath.Join(a.resultDir, name)); err != nil {
			return Published{}, fmt.Errorf("save %s stem: %w", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(a.finalDir), 0750); err != nil {
		return Published{}, fmt.Errorf("create song output directory: %w", err)
	}
	if _, err := os.Lstat(a.finalDir); err == nil {
		return Published{}, fmt.Errorf("song attempt output already exists")
	} else if !os.IsNotExist(err) {
		return Published{}, fmt.Errorf("check song output directory: %w", err)
	}
	if err := os.Rename(a.resultDir, a.finalDir); err != nil {
		return Published{}, fmt.Errorf("publish song output: %w", err)
	}
	return Published{
		Directory:    a.finalDir,
		FullSongPath: filepath.Join(a.finalDir, "full.mp3"),
		DrumsPath:    filepath.Join(a.finalDir, "drums.mp3"),
		BassPath:     filepath.Join(a.finalDir, "bass.mp3"),
		MelodyPath:   filepath.Join(a.finalDir, "other.mp3"),
		VocalsPath:   filepath.Join(a.finalDir, "vocals.mp3"),
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
