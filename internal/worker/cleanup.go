package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const mediaCleanupInterval = time.Hour

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
	activeAttempts := make(map[string]int32)
	neededInputs := make(map[string]bool)
	for _, job := range jobs {
		if job.State == "QUEUED" || job.State == "RUNNING" {
			neededInputs[filepath.Clean(job.InputPath)] = true
		} else {
			r.removeInput(job.InputPath)
		}
		if job.State == "RUNNING" && job.LeaseExpiresAt.Valid && job.LeaseExpiresAt.Time.After(now) {
			activeAttempts[job.SongID] = job.AttemptCount
		}
	}

	songsRoot := filepath.Join(r.mediaRoot, "songs")
	referencedOutputs := make(map[string]bool)
	for _, song := range songs {
		for _, path := range []pgtype.Text{
			song.FullSongPath, song.DrumsPath, song.BassPath,
			song.MelodyPath, song.VocalsPath,
		} {
			if path.Valid && pathWithin(songsRoot, path.String) {
				referencedOutputs[filepath.Clean(filepath.Dir(path.String))] = true
			}
		}
	}

	cutoff := now.Add(-mediaCleanupInterval)
	if err := r.cleanupIncoming(ctx, neededInputs, cutoff); err != nil {
		return err
	}
	if err := r.cleanupAttempts(ctx, filepath.Join(r.mediaRoot, "work"), activeAttempts, nil, cutoff); err != nil {
		return err
	}
	return r.cleanupAttempts(ctx, songsRoot, activeAttempts, referencedOutputs, cutoff)
}

func (r *Runner) cleanupIncoming(ctx context.Context, needed map[string]bool, cutoff time.Time) error {
	root := filepath.Join(r.mediaRoot, "incoming")
	entries, err := readMediaDirectory(root)
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
		r.logger.Info("removed orphaned song input", "path", path)
	}
	return nil
}

func (r *Runner) cleanupAttempts(ctx context.Context, root string, active map[string]int32, referenced map[string]bool, cutoff time.Time) error {
	songDirs, err := readMediaDirectory(root)
	if err != nil {
		return fmt.Errorf("read song media directory: %w", err)
	}
	for _, songDir := range songDirs {
		if !songDir.IsDir() {
			continue
		}
		songPath := filepath.Join(root, songDir.Name())
		attemptDirs, err := readMediaDirectory(songPath)
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
			r.logger.Info("removed orphaned song attempt", "path", path)
		}
	}
	return nil
}

func readMediaDirectory(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return entries, err
}
