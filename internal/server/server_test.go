package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	New(nil, logger, false).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok\n" {
		t.Fatalf("health response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestRequestLogIncludesMethodAndPath(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := logRequests(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["method"] != "GET" || entry["path"] != "/healthz" {
		t.Fatalf("request fields missing: %v", entry)
	}
}

func TestClientErrorsUseJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(nil, logger, false)
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantText   string
	}{
		{"register content type", http.MethodPost, "/users", `{}`, http.StatusUnsupportedMediaType, "expected application/json"},
		{"invalid song ID", http.MethodGet, "/songs/not-a-uuid", "", http.StatusBadRequest, "invalid song ID"},
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
	handler := New(nil, logger, false)
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
