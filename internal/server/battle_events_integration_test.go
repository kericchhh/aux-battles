package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/live"
)

func TestBattleEventsStreamChanges(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL live update test")
	}
	pool, err := pgxpool.New(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := live.NewHub(pool, logger)
	hubCtx, stopHub := context.WithCancel(context.Background())
	connected, unsubscribe := hub.Subscribe("connection-probe")
	defer unsubscribe()
	hubDone := make(chan struct{})
	go func() {
		hub.Run(hubCtx)
		close(hubDone)
	}()
	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		t.Fatal("event listener did not connect")
	}
	db, host := catalogUser(t, pool)
	_, stranger := catalogUser(t, pool)
	api := httptest.NewUnstartedServer(New(db, pool, hub, logger, false))
	api.Config.ReadTimeout = 500 * time.Millisecond
	api.Config.WriteTimeout = 500 * time.Millisecond
	api.Start()
	t.Cleanup(api.Close)

	response := authTestRequest(api.Config.Handler, http.MethodPost, "/battles", `{"rounds":1,"guessSeconds":15}`, host)
	if response.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", response.Code, response.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM battles WHERE id=$1::uuid`, created.ID) })
	view := authTestRequest(api.Config.Handler, http.MethodGet, "/battles/"+created.ID, "", host)
	if !strings.Contains(view.Body.String(), `"guessSeconds":15`) {
		t.Fatalf("view = %s", view.Body.String())
	}

	if status := streamStatus(t, api.URL+"/battles/"+created.ID+"/events", stranger); status != http.StatusNotFound {
		t.Fatalf("stranger stream = %d", status)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, api.URL+"/battles/"+created.ID+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(host)
	stream, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Body.Close()
	if stream.StatusCode != http.StatusOK || stream.Header.Get("Content-Type") != "text/event-stream" || stream.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("stream = %d %v", stream.StatusCode, stream.Header)
	}
	events := make(chan string, 8)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(stream.Body)
		for scanner.Scan() {
			if line, ok := strings.CutPrefix(scanner.Text(), "event: "); ok {
				events <- line
			}
		}
	}()
	nextEvent := func(what string) {
		t.Helper()
		select {
		case event, open := <-events:
			if !open || event != "changed" {
				t.Fatalf("%s: event %q open=%t", what, event, open)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: no event", what)
		}
	}
	nextEvent("initial event")
	time.Sleep(time.Second)
	if forfeit := authTestRequest(api.Config.Handler, http.MethodPost, "/battles/"+created.ID+"/forfeit", "", host); forfeit.Code != http.StatusOK {
		t.Fatalf("forfeit = %d %s", forfeit.Code, forfeit.Body.String())
	}
	nextEvent("forfeit")

	stopHub()
	<-hubDone
	select {
	case _, open := <-events:
		if open {
			t.Fatal("stream sent an event instead of closing")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stream stayed open after the hub stopped")
	}
}

func streamStatus(t *testing.T, url string, cookie *http.Cookie) int {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	return response.StatusCode
}

func TestStatsResponseShape(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL stats test")
	}
	pool, err := pgxpool.New(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	db, cookie := catalogUser(t, pool)
	handler := New(db, pool, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), false)
	response := authTestRequest(handler, http.MethodGet, "/users/me/stats", "", cookie)
	want := `{"played":0,"wins":0,"losses":0,"draws":0,"bestScore":0,"averageScore":0}`
	if response.Code != http.StatusOK || response.Body.String() != want || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("stats = %d %s", response.Code, response.Body.String())
	}
}
