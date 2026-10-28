package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestIPLimiterRejectsOnlyTheExhaustedClient(t *testing.T) {
	limiter := newIPLimiter(rate.Every(time.Hour), 1)
	calls := 0
	handler := limiter.wrap(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	})

	request := func(remoteAddr string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/limited", nil)
		r.RemoteAddr = remoteAddr
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	if got := request("203.0.113.10:1000").Code; got != http.StatusNoContent {
		t.Fatalf("first request = %d, want 204", got)
	}
	rejected := request("203.0.113.10:2000")
	if rejected.Code != http.StatusTooManyRequests {
		t.Fatalf("same IP on another port = %d, want 429", rejected.Code)
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rejected.Body.Bytes(), &body); err != nil || body.Message != "too many requests" {
		t.Fatalf("rate limit response = %q, error = %v", rejected.Body.String(), err)
	}
	if got := request("203.0.113.11:3000").Code; got != http.StatusNoContent {
		t.Fatalf("other IP request = %d, want 204", got)
	}
	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2", calls)
	}
}

func TestAuthRouteLimitersDoNotThrottleOtherRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(nil, logger, false)
	request := func(method, path string) int {
		r := httptest.NewRequest(method, path, strings.NewReader("{}"))
		r.RemoteAddr = "203.0.113.10:1000"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}
	for i := 0; i < 3; i++ {
		if got := request(http.MethodPost, "/users"); got != http.StatusUnsupportedMediaType {
			t.Fatalf("registration request %d = %d, want 415", i+1, got)
		}
	}
	if got := request(http.MethodPost, "/users"); got != http.StatusTooManyRequests {
		t.Fatalf("exhausted registration route = %d, want 429", got)
	}
	if got := request(http.MethodPost, "/users/login"); got != http.StatusUnsupportedMediaType {
		t.Fatalf("login after registration limit = %d, want 415", got)
	}
	if got := request(http.MethodGet, "/healthz"); got != http.StatusOK {
		t.Fatalf("health after registration limit = %d, want 200", got)
	}
}
