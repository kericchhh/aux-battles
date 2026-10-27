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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestAuthFlowWithPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL auth test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	handler := New(dbgen.New(tx), slog.New(slog.NewTextHandler(io.Discard, nil)), false)
	username := fmt.Sprintf("auth_%d", time.Now().UnixNano())
	email := username + "@example.com"
	password := "correct horse battery staple"
	registerBody := fmt.Sprintf(`{"username":%q,"email":%q,"password":%q}`, username, email, password)
	register := authTestRequest(handler, http.MethodPost, "/users", registerBody, nil)
	if register.Code != http.StatusCreated {
		t.Fatalf("register = %d %q", register.Code, register.Body.String())
	}
	var storedHash string
	if err := tx.QueryRow(ctx, "SELECT password_hash FROM users WHERE username = $1", username).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if storedHash == password || !strings.HasPrefix(storedHash, "$argon2id$") {
		t.Fatal("registration did not store an Argon2id hash")
	}
	loginBody := fmt.Sprintf(`{"identifier":%q,"password":%q}`, strings.ToUpper(email), password)
	login := authTestRequest(handler, http.MethodPost, "/users/login", loginBody, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login = %d %q", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	sum := sha256.Sum256([]byte(cookie.Value))
	tokenHash := hex.EncodeToString(sum[:])
	var storedTokenHash string
	if err := tx.QueryRow(ctx, "SELECT token_hash FROM sessions WHERE token_hash = $1", tokenHash).Scan(&storedTokenHash); err != nil {
		t.Fatal(err)
	}
	if storedTokenHash != tokenHash || storedTokenHash == cookie.Value {
		t.Fatal("session token was not stored as its SHA-256 hash")
	}

	me := authTestRequest(handler, http.MethodGet, "/users/me", "", cookie)
	if me.Code != http.StatusOK {
		t.Fatalf("current user = %d %q", me.Code, me.Body.String())
	}
	var profile struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := json.Unmarshal(me.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Username != username || profile.Email != email || profile.Role != "USER" {
		t.Fatalf("current user = %+v", profile)
	}

	logout := authTestRequest(handler, http.MethodPost, "/users/logout", "", cookie)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %q", logout.Code, logout.Body.String())
	}
	var remaining int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE token_hash = $1", tokenHash).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatal("logout did not remove the database session")
	}
	if revoked := authTestRequest(handler, http.MethodGet, "/users/me", "", cookie); revoked.Code != http.StatusUnauthorized {
		t.Fatalf("revoked cookie = %d, want 401", revoked.Code)
	}

	// A PostgreSQL error aborts this test transaction, so check the conflict last.
	duplicate := authTestRequest(handler, http.MethodPost, "/users", registerBody, nil)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate registration = %d, want 409", duplicate.Code)
	}
}
