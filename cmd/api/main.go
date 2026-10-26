package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("API stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return fmt.Errorf("configure PostgreSQL pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           server.New(dbgen.New(pool), logger, os.Getenv("APP_ENV") == "production"),
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Info("API listening", slog.String("address", httpServer.Addr))
	return httpServer.ListenAndServe()
}
