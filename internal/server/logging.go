package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	apiRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aux_battles_http_requests_total",
			Help: "API requests by route and response status.",
		}, []string{"route", "status"},
	)
	apiRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aux_battles_http_request_duration_seconds",
			Help:    "API request duration by route.",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 13),
		}, []string{"route"},
	)
)

type requestLoggerKey struct{}

func (s *server) loggerFor(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(requestLoggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return s.logger
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) ReadFrom(reader io.Reader) (int64, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if writer, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return writer.ReadFrom(reader)
	}
	return io.Copy(w.ResponseWriter, reader)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func panicSource() string {
	_, loggingFile, _, _ := runtime.Caller(0)
	var pcs [16]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.File != "" && frame.File != loggingFile && !strings.HasPrefix(frame.Function, "runtime.") {
			return fmt.Sprintf("%s:%d", filepath.Base(frame.File), frame.Line)
		}
		if !more {
			return "unknown"
		}
	}
}

func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			logger.Error("generate request ID", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		requestID := hex.EncodeToString(random[:])
		requestLogger := logger.With("request_id", requestID)
		w.Header().Set("X-Request-ID", requestID)
		response := &statusWriter{ResponseWriter: w}
		r = r.WithContext(context.WithValue(r.Context(), requestLoggerKey{}, requestLogger))
		start := time.Now()
		defer func() {
			recovered := recover()
			status := response.status
			if status == 0 {
				status = http.StatusOK
				if recovered != nil {
					status = http.StatusInternalServerError
				}
			}
			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}
			duration := time.Since(start)
			if route != "GET /metrics" {
				apiRequests.WithLabelValues(route, strconv.Itoa(status)).Inc()
				apiRequestDuration.WithLabelValues(route).Observe(duration.Seconds())
			}
			if recovered != nil {
				requestLogger.Error("request handler panicked", "method", r.Method, "route", route,
					"panic_type", fmt.Sprintf("%T", recovered), "panic_source", panicSource())
				panic(http.ErrAbortHandler)
			}
			requestLogger.Info("served request",
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", status),
				slog.Float64("duration_ms", float64(duration)/float64(time.Millisecond)),
			)
		}()
		next.ServeHTTP(response, r)
	})
}
