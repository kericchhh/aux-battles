package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/battle"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestRoundGameplayHTTP(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL round HTTP tests")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	root := t.TempDir()
	t.Setenv("MEDIA_ROOT", root)
	handler := New(dbgen.New(pool), pool, slog.New(slog.NewTextHandler(io.Discard, nil)), false)
	cookies := make([]*http.Cookie, 3)
	for i := range cookies {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO users(username,email,password_hash) VALUES(gen_random_uuid()::text,gen_random_uuid()::text,'test') RETURNING id::text`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM battles WHERE host_id=$1::uuid OR guest_id=$1::uuid`, id); err != nil {
				t.Error(err)
			}
			if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1::uuid`, id); err != nil {
				t.Error(err)
			}
		})
		sum := sha256.Sum256([]byte(id))
		if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2::uuid,now()+interval '1 hour')`, hex.EncodeToString(sum[:]), id); err != nil {
			t.Fatal(err)
		}
		cookies[i] = &http.Cookie{Name: "aux_session", Value: id}
	}
	created := authTestRequest(handler, http.MethodPost, "/battles", `{"rounds":1}`, cookies[0])
	var createdBody struct {
		ID string `json:"id"`
	}
	if created.Code != 201 {
		t.Fatalf("create=%d %s", created.Code, created.Body.String())
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	battlePath := "/battles/" + createdBody.ID
	response := authTestRequest(handler, http.MethodGet, battlePath, "", cookies[0])
	var view battle.View
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	joined := authTestRequest(handler, http.MethodPost, "/battles/join", fmt.Sprintf(`{"inviteCode":%q}`, view.InviteCode), cookies[1])
	if joined.Code != 200 {
		t.Fatalf("join=%d %s", joined.Code, joined.Body.String())
	}
	for _, cookie := range cookies[:2] {
		songID := readyLineupTrack(t, pool, root)
		response := authTestRequest(handler, http.MethodPut, battlePath+"/lineup", fmt.Sprintf(`{"songIds":[%q]}`, songID), cookie)
		if response.Code != 200 {
			t.Fatalf("lineup=%d %s", response.Code, response.Body.String())
		}
	}
	response = authTestRequest(handler, http.MethodGet, battlePath, "", cookies[0])
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Round == nil {
		t.Fatal("no active round")
	}
	path := "/rounds/" + view.Round.ID
	for _, tc := range []struct {
		method, path, body string
		cookie             *http.Cookie
		status             int
	}{
		{"GET", path + "/audio", "", nil, 401},
		{"POST", path + "/guess", `{}`, nil, 401},
		{"GET", "/rounds/invalid/audio", "", cookies[0], 400},
		{"POST", "/rounds/invalid/guess", `{}`, cookies[0], 400},
		{"GET", path + "/audio", "", cookies[2], 404},
		{"POST", path + "/guess", `{"guess":"Cleared fixture","expectedAttempt":1}`, cookies[2], 404},
		{"GET", "/rounds/00000000-0000-4000-8000-000000000001/audio", "", cookies[0], 404},
		{"POST", path + "/guess", `{"guess":"!!!","expectedAttempt":1}`, cookies[0], 400},
		{"POST", path + "/guess", `{"guess":"title","expectedAttempt":null}`, cookies[0], 400},
		{"POST", path + "/guess", `{"guess":"title","expectedAttempt":1.5}`, cookies[0], 400},
		{"POST", path + "/guess", `{"guess":"title","expectedAttempt":1,"userId":"spoofed"}`, cookies[0], 400},
	} {
		got := authTestRequest(handler, tc.method, tc.path, tc.body, tc.cookie)
		if got.Code != tc.status {
			t.Fatalf("%s %s=%d %s want %d", tc.method, tc.path, got.Code, got.Body.String(), tc.status)
		}
	}
	request := httptest.NewRequest("GET", path+"/audio", nil)
	request.AddCookie(cookies[0])
	request.Header.Set("Range", "bytes=0-4")
	audio := httptest.NewRecorder()
	handler.ServeHTTP(audio, request)
	if audio.Code != 206 || audio.Body.String() != "audio" || audio.Header().Get("Content-Type") != "audio/mpeg" || audio.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("audio=%d %s headers=%v", audio.Code, audio.Body.String(), audio.Header())
	}
	for attempt, guess := range []string{"wrong", "wrong", "Cleared fixture"} {
		body := fmt.Sprintf(`{"guess":%q,"expectedAttempt":%d}`, guess, attempt+1)
		response = authTestRequest(handler, "POST", path+"/guess", body, cookies[0])
		if response.Code != 200 {
			t.Fatalf("guess=%d %s", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "Cleared fixture") || strings.Contains(response.Body.String(), root) {
			t.Fatal("guess response leaked answer or media path")
		}
	}
	var scored battle.GuessResult
	if err := json.Unmarshal(response.Body.Bytes(), &scored); err != nil {
		t.Fatal(err)
	}
	if scored.PointsAwarded != 50 || scored.BasePoints != 50 || scored.StreakBonus != 0 || !scored.Correct {
		t.Fatalf("melody score=%+v", scored)
	}
	response = authTestRequest(handler, "POST", path+"/guess", `{"guess":"Cleared fixture","expectedAttempt":1}`, cookies[1])
	if response.Code != 200 {
		t.Fatalf("last guess=%d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &scored); err != nil {
		t.Fatal(err)
	}
	if scored.BattleStatus != "FINISHED" || !scored.RoundFinished {
		t.Fatalf("last guess=%+v", scored)
	}
	response = authTestRequest(handler, "GET", path+"/audio", "", cookies[0])
	if response.Code != 409 {
		t.Fatalf("finished audio=%d %s", response.Code, response.Body.String())
	}
}
