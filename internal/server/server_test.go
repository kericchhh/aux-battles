package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

func TestHealthHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	New(nil, nil, logger, false).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok\n" {
		t.Fatalf("health response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestRequestMetricsCountNormalRequestsButNotScrapes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(nil, nil, logger, false)
	beforeRequests := testutil.ToFloat64(apiRequests.WithLabelValues("GET /healthz", "200"))
	beforeScrapes := testutil.ToFloat64(apiRequests.WithLabelValues("GET /metrics", "200"))
	durationCount := func() uint64 {
		t.Helper()
		metric := &dto.Metric{}
		observer := apiRequestDuration.WithLabelValues("GET /healthz")
		if err := observer.(prometheus.Metric).Write(metric); err != nil {
			t.Fatal(err)
		}
		return metric.GetHistogram().GetSampleCount()
	}
	beforeDurations := durationCount()

	for _, path := range []string{"/healthz", "/metrics"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s returned %d, want 200", path, response.Code)
		}
	}
	if got := testutil.ToFloat64(apiRequests.WithLabelValues("GET /healthz", "200")); got != beforeRequests+1 {
		t.Fatalf("health request count = %v, want %v", got, beforeRequests+1)
	}
	if got := durationCount(); got != beforeDurations+1 {
		t.Fatalf("health duration count = %v, want %v", got, beforeDurations+1)
	}
	if got := testutil.ToFloat64(apiRequests.WithLabelValues("GET /metrics", "200")); got != beforeScrapes {
		t.Fatalf("scrape request count = %v, want %v", got, beforeScrapes)
	}
}

func TestSongUploadRequiresAuthentication(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(nil, nil, logger, false)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/songs", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("upload without a session = %d, want 401", response.Code)
	}
}

func TestRequestLogsCorrelateErrorsWithoutLoggingRawPath(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /songs/{id}", func(w http.ResponseWriter, r *http.Request) {
		(&server{logger: logger}).loggerFor(r.Context()).Error("lookup failed")
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	handler := logRequests(logger, mux)
	request := httptest.NewRequest(http.MethodGet, "/songs/private-id?token=secret", nil)
	request.Header.Set("X-Request-ID", "client-supplied")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requestID := response.Header().Get("X-Request-ID")
	if len(requestID) != 32 || requestID == "client-supplied" {
		t.Fatalf("response request ID = %q", requestID)
	}
	if _, err := hex.DecodeString(requestID); err != nil {
		t.Fatalf("request ID is not hex: %v", err)
	}
	if strings.Contains(output.String(), "private-id") || strings.Contains(output.String(), "secret") {
		t.Fatalf("raw path or query leaked into log: %s", output.String())
	}

	decoder := json.NewDecoder(&output)
	var errorEntry, requestEntry map[string]any
	if err := decoder.Decode(&errorEntry); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&requestEntry); err != nil {
		t.Fatal(err)
	}
	if errorEntry["request_id"] != requestID || requestEntry["request_id"] != requestID {
		t.Fatalf("request ID missing from logs: error=%v request=%v", errorEntry, requestEntry)
	}
	if requestEntry["method"] != "GET" || requestEntry["route"] != "GET /songs/{id}" ||
		requestEntry["status"] != float64(http.StatusServiceUnavailable) {
		t.Fatalf("request fields missing: %v", requestEntry)
	}
	if duration, ok := requestEntry["duration_ms"].(float64); !ok || duration < 0 {
		t.Fatalf("invalid request duration: %v", requestEntry["duration_ms"])
	}
}

func TestRequestLogRecordsImplicitAndUnmatchedStatuses(t *testing.T) {
	for _, tt := range []struct {
		name       string
		path       string
		wantStatus int
		wantRoute  string
	}{
		{"implicit success", "/healthz", http.StatusOK, "GET /healthz"},
		{"unmatched route", "/missing", http.StatusNotFound, "unmatched"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			mux := http.NewServeMux()
			mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("ok"))
			})
			response := httptest.NewRecorder()
			logRequests(logger, mux).ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))

			var entry map[string]any
			if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
				t.Fatal(err)
			}
			if entry["status"] != float64(tt.wantStatus) || entry["route"] != tt.wantRoute {
				t.Fatalf("request log = %v", entry)
			}
		})
	}
}

func TestAuthenticationErrorLogSharesRequestID(t *testing.T) {
	db := newAuthTestDB()
	db.getSessionErr = errors.New("database unavailable")
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := New(dbgen.New(db), nil, logger, false)
	request := httptest.NewRequest(http.MethodGet, "/songs/00000000-0000-4000-8000-000000000001/status", nil)
	request.AddCookie(&http.Cookie{Name: "aux_session", Value: "some-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}

	requestID := response.Header().Get("X-Request-ID")
	decoder := json.NewDecoder(&output)
	var errorEntry, requestEntry map[string]any
	if err := decoder.Decode(&errorEntry); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&requestEntry); err != nil {
		t.Fatal(err)
	}
	if errorEntry["msg"] != "authenticate request failed" || errorEntry["request_id"] != requestID ||
		requestEntry["request_id"] != requestID || requestEntry["status"] != float64(http.StatusInternalServerError) {
		t.Fatalf("uncorrelated error and completion logs: error=%v completion=%v", errorEntry, requestEntry)
	}
}

func TestPanickingHandlerLogsRequestIDWithoutPanicValue(t *testing.T) {
	before := testutil.ToFloat64(apiRequests.WithLabelValues("GET /healthz", "500"))
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(http.ResponseWriter, *http.Request) {
		panic("unexpected failure")
	})
	handler := logRequests(logger, mux)
	server := httptest.NewUnstartedServer(handler)
	server.Config.ErrorLog = slog.NewLogLogger(logger.Handler(), slog.LevelError)
	server.Start()
	response, err := server.Client().Get(server.URL + "/healthz")
	if err == nil {
		response.Body.Close()
	}
	server.Close()
	if err == nil {
		t.Fatal("panicking handler returned a normal HTTP response")
	}
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	requestID, ok := entry["request_id"].(string)
	panicSource, hasSource := entry["panic_source"].(string)
	if entry["msg"] != "request handler panicked" || !ok || len(requestID) != 32 ||
		entry["panic_type"] != "string" || entry["method"] != "GET" || entry["route"] != "GET /healthz" ||
		!hasSource || !strings.HasPrefix(panicSource, "server_test.go:") {
		t.Fatalf("panic log = %v", entry)
	}
	if strings.Contains(output.String(), "unexpected failure") {
		t.Fatalf("panic value leaked into log: %s", output.String())
	}
	if got := testutil.ToFloat64(apiRequests.WithLabelValues("GET /healthz", "500")); got != before+1 {
		t.Fatalf("panic request count = %v, want %v", got, before+1)
	}
}

func TestRequestLoggingPreservesAudioByteRanges(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := logRequests(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "clip.mp3", time.Time{}, strings.NewReader("abcdef"))
	}))
	request := httptest.NewRequest(http.MethodGet, "/audio", nil)
	request.Header.Set("Range", "bytes=1-3")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent || response.Body.String() != "bcd" {
		t.Fatalf("range response = %d %q", response.Code, response.Body.String())
	}
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["status"] != float64(http.StatusPartialContent) {
		t.Fatalf("range status log = %v", entry)
	}
}

func TestClientErrorsUseJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(nil, nil, logger, false)
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantText   string
	}{
		{"register content type", http.MethodPost, "/users", `{}`, http.StatusUnsupportedMediaType, "expected application/json"},
		{"catalog without session", http.MethodGet, "/songs/not-a-uuid", "", http.StatusUnauthorized, "not authenticated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			handler.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var response struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Message != tt.wantText {
				t.Fatalf("message = %q, want %q", response.Message, tt.wantText)
			}
		})
	}
}

func TestLoginRejectsMissingCredentials(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(nil, nil, logger, false)
	request := httptest.NewRequest(http.MethodPost, "/users/login", strings.NewReader(`{"identifier":"","password":""}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var response struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Message != "identifier and password are required" {
		t.Fatalf("message = %q", response.Message)
	}
}
