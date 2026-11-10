package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtendDeadlinesAllowsSlowBodies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		extend bool
		want   bool
	}{
		{"server timeouts", false, false},
		{"extended deadlines", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.extend {
					if err := extendDeadlines(w, 5*time.Second); err != nil {
						t.Error(err)
					}
				}
				if _, err := io.Copy(io.Discard, r.Body); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			srv := httptest.NewUnstartedServer(logRequests(slog.New(slog.NewTextHandler(io.Discard, nil)), handler))
			srv.Config.ReadTimeout = 100 * time.Millisecond
			srv.Config.WriteTimeout = 100 * time.Millisecond
			srv.Start()
			defer srv.Close()

			body, writer := io.Pipe()
			go func() {
				for range 5 {
					time.Sleep(50 * time.Millisecond)
					if _, err := writer.Write([]byte("chunk")); err != nil {
						return
					}
				}
				writer.Close()
			}()
			resp, err := srv.Client().Post(srv.URL, "application/octet-stream", body)
			got := err == nil && resp.StatusCode == http.StatusNoContent
			if err == nil {
				resp.Body.Close()
			}
			if got != tc.want {
				t.Fatalf("slow body succeeded=%t, want %t (err=%v)", got, tc.want, err)
			}
		})
	}
}

func TestServeAudioOutlastsWriteTimeout(t *testing.T) {
	const size = 128 << 20
	path := filepath.Join(t.TempDir(), "large.mp3")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, size); err != nil {
		t.Fatal(err)
	}
	app := &server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	for _, tc := range []struct {
		name   string
		extend bool
	}{{"server timeout", false}, {"extended deadline", true}} {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				file, err := os.Open(path)
				if err != nil {
					t.Error(err)
					return
				}
				defer file.Close()
				info, err := file.Stat()
				if err != nil {
					t.Error(err)
					return
				}
				if tc.extend {
					app.serveAudio(w, r, "large.mp3", "private", file, info)
					return
				}
				http.ServeContent(w, r, "large.mp3", info.ModTime(), file)
			})
			srv := httptest.NewUnstartedServer(handler)
			srv.Config.WriteTimeout = 100 * time.Millisecond
			srv.Start()
			defer srv.Close()

			resp, err := srv.Client().Get(srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			time.Sleep(300 * time.Millisecond)
			read, err := io.Copy(io.Discard, resp.Body)
			if got := read == size; got != tc.extend {
				t.Fatalf("read %d of %d bytes (err=%v); complete=%t, want %t", read, size, err, got, tc.extend)
			}
			if tc.extend && err != nil {
				t.Fatalf("extended download failed: %v", err)
			}
		})
	}
}
