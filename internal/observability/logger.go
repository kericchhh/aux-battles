package observability

import (
	"fmt"
	"io"
	"log/slog"
)

func NewLogger(output io.Writer, service, rawLevel string) (*slog.Logger, error) {
	level := slog.LevelInfo
	if rawLevel != "" {
		if err := level.UnmarshalText([]byte(rawLevel)); err != nil {
			return nil, fmt.Errorf("invalid LOG_LEVEL: %w", err)
		}
	}
	logger := slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level}))
	return logger.With("service", service), nil
}
