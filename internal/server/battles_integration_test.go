package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/battle"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestBattleLobbyHTTPWithPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL battle HTTP tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cookies := make([]*http.Cookie, 3)
	for i := range cookies {
		var id pgtype.UUID
		if err := pool.QueryRow(ctx, `INSERT INTO users(username,email,password_hash)
   VALUES(gen_random_uuid()::text,gen_random_uuid()::text,'test') RETURNING id`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM battles WHERE host_id=$1 OR guest_id=$1`, id); err != nil {
				t.Error(err)
			}
			if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id); err != nil {
				t.Error(err)
			}
		})
		token := id.String()
		sum := sha256.Sum256([]byte(token))
		if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '1 hour')`, hex.EncodeToString(sum[:]), id); err != nil {
			t.Fatal(err)
		}
		cookies[i] = &http.Cookie{Name: "aux_session", Value: token}
	}
	root := t.TempDir()
	t.Setenv("MEDIA_ROOT", root)
	handler := New(dbgen.New(pool), pool, slog.New(slog.NewTextHandler(io.Discard, nil)), false)
	for _, body := range []string{`{"rounds":0}`, `{"rounds":8}`, `{"rounds":-1}`, `{"rounds":1.5}`, `{"rounds":null}`, `{"rounds":"7"}`, `{"rounds":7,"hostId":"spoofed"}`} {
		response := authTestRequest(handler, http.MethodPost, "/battles", body, cookies[0])
		if response.Code != http.StatusBadRequest {
			t.Fatalf("create %s = %d %s", body, response.Code, response.Body.String())
		}
	}
	created := authTestRequest(handler, http.MethodPost, "/battles", `{}`, cookies[0])
	if created.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", created.Code, created.Body.String())
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	path := "/battles/" + result.ID
	get := authTestRequest(handler, http.MethodGet, path, "", cookies[0])
	if get.Code != http.StatusOK {
		t.Fatalf("get=%d %s", get.Code, get.Body.String())
	}
	if get.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("battle view may be cached")
	}
	var view battle.View
	if err := json.Unmarshal(get.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Status != "PENDING" || view.Rounds != 5 {
		t.Fatalf("default lobby=%+v", view)
	}
	if response := authTestRequest(handler, http.MethodGet, path, "", cookies[2]); response.Code != http.StatusNotFound {
		t.Fatalf("outsider=%d", response.Code)
	}
	for _, tc := range []struct {
		cookie *http.Cookie
		body   string
		status int
	}{
		{cookies[0], `{"inviteCode":"` + view.InviteCode + `"}`, http.StatusConflict},
		{cookies[1], `{"inviteCode":"bad"}`, http.StatusBadRequest},
		{cookies[1], `{"inviteCode":"  ` + view.InviteCode + `  "}`, http.StatusOK},
		{cookies[2], `{"inviteCode":"` + view.InviteCode + `"}`, http.StatusConflict},
	} {
		response := authTestRequest(handler, http.MethodPost, "/battles/join", tc.body, tc.cookie)
		if response.Code != tc.status {
			t.Fatalf("join=%d %s, want %d", response.Code, response.Body.String(), tc.status)
		}
	}
	for _, cookie := range cookies[:2] {
		response := authTestRequest(handler, http.MethodGet, path, "", cookie)
		if response.Code != http.StatusOK {
			t.Fatalf("participant=%d", response.Code)
		}
		if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
			t.Fatal(err)
		}
		if view.Status != "SELECTING" || !view.OpponentJoined {
			t.Fatalf("joined view=%+v", view)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"hostSongId", "guestSongId", "hostId", "guestId"} {
			if _, ok := fields[field]; ok {
				t.Fatalf("view exposes %s", field)
			}
		}
	}
	if response := authTestRequest(handler, http.MethodGet, "/battles/not-a-uuid", "", cookies[0]); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid ID=%d", response.Code)
	}
	if response := authTestRequest(handler, http.MethodGet, "/battles/00000000-0000-4000-8000-000000000000", "", cookies[0]); response.Code != http.StatusNotFound {
		t.Fatalf("missing battle=%d", response.Code)
	}
	lineupPath := path + "/lineup"
	for _, body := range []string{`{}`, `{"songIds":null}`, `{"songIds":[]}`, `{"songIds":["not-a-uuid"]}`, `{"songIds":["00000000-0000-4000-8000-000000000000"]}`, `{"songIds":[],"userId":"spoofed"}`} {
		response := authTestRequest(handler, http.MethodPut, lineupPath, body, cookies[0])
		if response.Code != http.StatusBadRequest {
			t.Fatalf("lineup %s = %d %s", body, response.Code, response.Body.String())
		}
	}
	ids := make([]string, 5)
	for i := range ids {
		ids[i] = readyLineupTrack(t, pool, root)
	}
	data, err := json.Marshal(struct {
		SongIDs []string `json:"songIds"`
	}{ids})
	if err != nil {
		t.Fatal(err)
	}
	if response := authTestRequest(handler, http.MethodPut, lineupPath, string(data), cookies[2]); response.Code != http.StatusNotFound {
		t.Fatalf("outsider lineup=%d", response.Code)
	}
	if response := authTestRequest(handler, http.MethodPut, "/battles/not-a-uuid/lineup", string(data), cookies[0]); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid lineup battle ID=%d", response.Code)
	}
	audioPath := filepath.Join(root, "songs", ids[0], "attempt-1", "full.mp3")
	if err := os.Remove(audioPath); err != nil {
		t.Fatal(err)
	}
	missingAudio := authTestRequest(handler, http.MethodPut, lineupPath, string(data), cookies[0])
	if missingAudio.Code != http.StatusConflict || strings.Contains(missingAudio.Body.String(), root) || strings.Contains(missingAudio.Body.String(), ids[0]) {
		t.Fatalf("missing audio response=%d %s", missingAudio.Code, missingAudio.Body.String())
	}
	if err := os.WriteFile(audioPath, []byte("audio fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for i, cookie := range cookies[:2] {
		response := authTestRequest(handler, http.MethodPut, lineupPath, string(data), cookie)
		if response.Code != http.StatusOK {
			t.Fatalf("lineup=%d %s", response.Code, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("lineup response may be cached")
		}
		var result battle.LineupResult
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := "WAITING_ON_OPPONENT"
		if i == 1 {
			want = "BATTLE_STARTED"
		}
		if result.Status != want || result.BattleID != view.ID {
			t.Fatalf("lineup result=%+v", result)
		}
		if repeated := authTestRequest(handler, http.MethodPut, lineupPath, string(data), cookie); repeated.Code != http.StatusConflict {
			t.Fatalf("repeat lineup=%d", repeated.Code)
		}
	}
	response := authTestRequest(handler, http.MethodGet, path, "", cookies[0])
	if response.Code != http.StatusOK {
		t.Fatalf("started battle=%d", response.Code)
	}
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Status != "ONGOING" || view.Round == nil || view.Round.Status != "GUESSING" || !view.MyLineupLocked || !view.OpponentLineupLocked {
		t.Fatalf("started battle=%+v", view)
	}
	for _, id := range ids {
		if strings.Contains(response.Body.String(), id) {
			t.Fatal("battle view exposes a selected song ID")
		}
	}
}
