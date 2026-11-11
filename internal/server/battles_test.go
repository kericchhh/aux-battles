package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestBattleRoutesRequireAuthentication(t *testing.T) {
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/battles", ""},
		{http.MethodPost, "/battles", `{}`},
		{http.MethodPost, "/battles/join", `{"inviteCode":"ABC123"}`},
		{http.MethodGet, "/battles/00000000-0000-4000-8000-000000000001", ""},
		{http.MethodPut, "/battles/00000000-0000-4000-8000-000000000001/lineup", `{"songIds":[]}`},
	} {
		response := authTestRequest(authTestHandler(newAuthTestDB(), false), tc.method, tc.path, tc.body, nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s = %d", tc.method, tc.path, response.Code)
		}
	}
}

func TestBattleHandlerUsesMiddlewareIdentity(t *testing.T) {
	db := newAuthTestDB()
	var id pgtype.UUID
	if err := id.Scan("00000000-0000-4000-8000-000000000001"); err != nil {
		t.Fatal(err)
	}
	db.user = &authTestUser{id: id, username: "host", email: "host@example.com", role: "USER"}
	token := "middleware-token"
	sum := sha256.Sum256([]byte(token))
	db.sessions[hex.EncodeToString(sum[:])] = time.Now().Add(time.Hour)
	s := &server{queries: dbgen.New(db), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	handler := s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		db.getSessionErr = errors.New("a second session lookup must not occur")
		got, ok := s.battleActor(w, r)
		if !ok || got != id {
			t.Errorf("middleware identity=%v, %t", got, ok)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	response := authTestRequest(handler, http.MethodGet, "/battles", "", &http.Cookie{Name: "aux_session", Value: token})
	if response.Code != http.StatusNoContent {
		t.Fatalf("identity response=%d %s", response.Code, response.Body.String())
	}
}

func TestListBattlesResponseShape(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL battle list test")
	}
	pool, err := pgxpool.New(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	db, cookie := catalogUser(t, pool)
	handler := New(db, pool, slog.New(slog.NewTextHandler(io.Discard, nil)), false)

	response := authTestRequest(handler, http.MethodGet, "/battles", "", cookie)
	if response.Code != http.StatusOK || response.Body.String() != "[]" {
		t.Fatalf("empty list = %d %q", response.Code, response.Body.String())
	}

	response = authTestRequest(handler, http.MethodPost, "/battles", `{"rounds":3}`, cookie)
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

	response = authTestRequest(handler, http.MethodGet, "/battles", "", cookie)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("list = %d %q", response.Code, response.Header().Get("Content-Type"))
	}
	var list []map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %s", response.Body.String())
	}
	want := map[string]string{
		"id": `"` + created.ID + `"`, "status": `"PENDING"`, "rounds": "3", "currentRound": "1",
		"opponent": "null", "myScore": "0", "opponentScore": "0", "outcome": "null", "endReason": "null",
	}
	for key, value := range want {
		if string(list[0][key]) != value {
			t.Errorf("%s = %s, want %s", key, list[0][key], value)
		}
	}
	var code string
	if err := json.Unmarshal(list[0]["inviteCode"], &code); err != nil || len(code) != 6 {
		t.Errorf("inviteCode = %s", list[0]["inviteCode"])
	}
	if _, ok := list[0]["lastActivityAt"]; !ok || len(list[0]) != len(want)+2 {
		t.Errorf("fields = %s", response.Body.String())
	}
}
