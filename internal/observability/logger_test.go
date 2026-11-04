package observability

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewLoggerDefaultsToInfoAndIncludesService(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger(&output, "api", "")
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("hidden")
	logger.Info("ready")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["service"] != "api" || entry["level"] != "INFO" || entry["msg"] != "ready" {
		t.Fatalf("unexpected log entry: %v", entry)
	}
}

func TestNewLoggerAcceptsConfiguredLevel(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger(&output, "worker", "DEBUG")
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("processing")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["service"] != "worker" || entry["level"] != "DEBUG" {
		t.Fatalf("unexpected log entry: %v", entry)
	}
}

func TestNewLoggerRejectsInvalidLevel(t *testing.T) {
	if _, err := NewLogger(&bytes.Buffer{}, "api", "verbose"); err == nil {
		t.Fatal("invalid log level was accepted")
	}
}
