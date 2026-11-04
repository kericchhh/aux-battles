package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const CleanupInterval = time.Hour

type References struct {
	Inputs         []string
	ActiveAttempts map[string]int32
	Outputs        []string
}

type Removed struct {
	Inputs   []string
	Attempts []string
}

func (s Store) Cleanup(ctx context.Context, refs References) (Removed, error) {
	var removed Removed
	incoming, err := s.dir("incoming")
	if err != nil {
		return removed, err
	}
	work, err := s.dir("work")
	if err != nil {
		return removed, err
	}
	songs, err := s.dir("songs")
	if err != nil {
		return removed, err
	}

	needed := make(map[string]bool, len(refs.Inputs))
	for _, path := range refs.Inputs {
		needed[filepath.Clean(path)] = true
	}
	referenced := make(map[string]bool)
	for _, path := range refs.Outputs {
		if within(songs, path) {
			referenced[filepath.Clean(filepath.Dir(path))] = true
		}
	}

	cutoff := time.Now().Add(-CleanupInterval)
	if err := cleanupIncoming(ctx, incoming, needed, cutoff, &removed); err != nil {
		return removed, err
	}
	if err := cleanupAttempts(ctx, work, refs.ActiveAttempts, nil, cutoff, &removed); err != nil {
		return removed, err
	}
	if err := cleanupAttempts(ctx, songs, refs.ActiveAttempts, referenced, cutoff, &removed); err != nil {
		return removed, err
	}
	return removed, nil
}

func cleanupIncoming(ctx context.Context, root string, needed map[string]bool, cutoff time.Time, removed *Removed) error {
	entries, err := readDirectory(root)
	if err != nil {
		return fmt.Errorf("read incoming media: %w", err)
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !strings.HasPrefix(entry.Name(), "song-") || filepath.Ext(entry.Name()) != ".mp3" {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if needed[path] {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect incoming media: %w", err)
		}
		if !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove orphaned input: %w", err)
		}
		removed.Inputs = append(removed.Inputs, path)
	}
	return nil
}

func cleanupAttempts(ctx context.Context, root string, active map[string]int32, referenced map[string]bool, cutoff time.Time, removed *Removed) error {
	songDirs, err := readDirectory(root)
	if err != nil {
		return fmt.Errorf("read song media directory: %w", err)
	}
	for _, songDir := range songDirs {
		if !songDir.IsDir() {
			continue
		}
		songPath := filepath.Join(root, songDir.Name())
		attemptDirs, err := readDirectory(songPath)
		if err != nil {
			return fmt.Errorf("read song attempts: %w", err)
		}
		for _, attemptDir := range attemptDirs {
			if err := ctx.Err(); err != nil {
				return err
			}
			if !attemptDir.IsDir() || !strings.HasPrefix(attemptDir.Name(), "attempt-") {
				continue
			}
			attempt, err := strconv.ParseInt(strings.TrimPrefix(attemptDir.Name(), "attempt-"), 10, 32)
			if err != nil || attempt < 0 {
				continue
			}
			path := filepath.Join(songPath, attemptDir.Name())
			if (active[songDir.Name()] > 0 && active[songDir.Name()] == int32(attempt)) || referenced[path] {
				continue
			}
			info, err := attemptDir.Info()
			if err != nil {
				return fmt.Errorf("inspect song attempt: %w", err)
			}
			if !info.ModTime().Before(cutoff) {
				continue
			}
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("remove orphaned song attempt: %w", err)
			}
			removed.Attempts = append(removed.Attempts, path)
		}
	}
	return nil
}

func readDirectory(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return entries, err
}
