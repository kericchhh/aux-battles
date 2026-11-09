package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestBattleRoutesRequireAuthentication(t *testing.T) {
	for _, tc := range []struct{ method, path, body string }{
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
