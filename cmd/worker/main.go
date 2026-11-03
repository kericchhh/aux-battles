package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, logger); err != nil {
		logger.Error("song worker stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	mediaRoot := os.Getenv("MEDIA_ROOT")
	if mediaRoot == "" {
		mediaRoot = "storage"
	}
	mediaRoot, err := filepath.Abs(mediaRoot)
	if err != nil {
		return fmt.Errorf("resolve media directory: %w", err)
	}
	demucsBin := os.Getenv("DEMUCS_BIN")
	if demucsBin == "" {
		demucsBin = "demucs"
	}
	if _, err := exec.LookPath(demucsBin); err != nil {
		return fmt.Errorf("find Demucs executable: %w", err)
	}
	attemptTimeout := worker.DefaultAttemptTimeout
	if raw := os.Getenv("WORKER_ATTEMPT_TIMEOUT"); raw != "" {
		attemptTimeout, err = time.ParseDuration(raw)
		if err != nil || attemptTimeout <= 0 {
			return fmt.Errorf("WORKER_ATTEMPT_TIMEOUT must be a positive duration")
		}
	}

	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(connectCtx, url)
	if err != nil {
		return fmt.Errorf("configure PostgreSQL pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(connectCtx); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	logger.Info("song worker started", "media_root", mediaRoot)
	processor := worker.NewAudioProcessor(mediaRoot, demucsBin)
	return worker.New(pool, processor, mediaRoot, logger, attemptTimeout).Run(ctx)
}
