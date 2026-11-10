package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/observability"
	"github.com/kericchhh/aux-battles/internal/server"
	"golang.org/x/sync/errgroup"
)

const (
	shutdownTimeout      = 8 * time.Second
	sessionPruneInterval = time.Hour
)

func main() {
	logger, err := observability.NewLogger(os.Stdout, "api", os.Getenv("LOG_LEVEL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := run(logger); err != nil {
		logger.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is required")
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

	queries := dbgen.New(pool)
	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           server.New(queries, pool, logger, os.Getenv("APP_ENV") == "production"),
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		logger.Info("API listening", slog.String("address", httpServer.Addr))
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve API: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		server.PruneSessions(groupCtx, queries, logger, sessionPruneInterval)
		return nil
	})
	group.Go(func() error {
		<-groupCtx.Done()
		stop()
		logger.Info("API shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		err := httpServer.Shutdown(shutdownCtx)
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Info("API shutdown timed out; closing remaining connections")
			return httpServer.Close()
		}
		if err != nil {
			return errors.Join(fmt.Errorf("shut down API: %w", err), httpServer.Close())
		}
		return nil
	})
	return group.Wait()
}
