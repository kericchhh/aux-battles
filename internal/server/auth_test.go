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
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

type authTestUser struct {
	id           pgtype.UUID
	username     string
	email        string
	passwordHash string
	role         string
}

type authTestDB struct {
	user             *authTestUser
	sessions         map[string]time.Time
	getSessionErr    error
	createSessionErr error
	deleteSessionErr error
}

func newAuthTestDB() *authTestDB {
	return &authTestDB{sessions: make(map[string]time.Time)}
}

func (db *authTestDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	switch {
	case strings.Contains(query, "INSERT INTO users"):
		if db.user != nil {
			return authTestRow{err: &pgconn.PgError{Code: "23505"}}
		}
		var id pgtype.UUID
		_ = id.Scan("00000000-0000-4000-8000-000000000001")
		db.user = &authTestUser{id: id, username: args[0].(string), email: args[1].(string), passwordHash: args[2].(string), role: "USER"}
		return authTestRow{values: []any{id, db.user.username, db.user.email, db.user.role, pgtype.Timestamptz{Time: time.Now(), Valid: true}}}
	case strings.Contains(query, "SELECT id, username, email, password_hash, role"):
		identifier := args[0].(string)
		if db.user == nil || (identifier != db.user.username && identifier != db.user.email) {
			return authTestRow{err: pgx.ErrNoRows}
		}
		return authTestRow{values: []any{db.user.id, db.user.username, db.user.email, db.user.passwordHash, db.user.role}}
	case strings.Contains(query, "FROM sessions AS s"):
		if db.getSessionErr != nil {
			return authTestRow{err: db.getSessionErr}
		}
		expiresAt, ok := db.sessions[args[0].(string)]
		if !ok || !expiresAt.After(time.Now()) {
			return authTestRow{err: pgx.ErrNoRows}
		}
		return authTestRow{values: []any{db.user.id, db.user.username, db.user.email, db.user.role, pgtype.Timestamptz{Time: expiresAt, Valid: true}}}
	default:
		panic("unexpected auth query")
	}
}

func (db *authTestDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	switch {
	case strings.Contains(query, "INSERT INTO sessions"):
		if db.createSessionErr != nil {
			return pgconn.CommandTag{}, db.createSessionErr
		}
		db.sessions[args[0].(string)] = args[2].(pgtype.Timestamptz).Time
	case strings.Contains(query, "DELETE FROM sessions"):
		if db.deleteSessionErr != nil {
			return pgconn.CommandTag{}, db.deleteSessionErr
		}
		delete(db.sessions, args[0].(string))
	default:
		panic("unexpected auth exec")
	}
	return pgconn.CommandTag{}, nil
}

func (*authTestDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected auth query returning rows")
}

type authTestRow struct {
	values []any
	err    error
}

func (row authTestRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	for i, value := range row.values {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func authTestHandler(db *authTestDB, secure bool) http.Handler {
	return New(dbgen.New(db), slog.New(slog.NewTextHandler(io.Discard, nil)), secure)
}

func authTestRequest(handler http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestRegisterNormalizesAndHashesPassword(t *testing.T) {
	db := newAuthTestDB()
	handler := authTestHandler(db, false)
	response := authTestRequest(handler, http.MethodPost, "/users", `{"username":" Alice_1 ","email":" Alice@Example.com ","password":"correct horse battery staple"}`, nil)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", response.Code, response.Body.String())
	}
	if db.user.username != "alice_1" || db.user.email != "alice@example.com" {
		t.Fatalf("stored identity = %q %q", db.user.username, db.user.email)
	}
	if db.user.passwordHash == "correct horse battery staple" {
		t.Fatal("password stored in plaintext")
	}
	match, err := argon2id.ComparePasswordAndHash("correct horse battery staple", db.user.passwordHash)
	if err != nil || !match {
		t.Fatalf("stored password hash cannot verify password: match=%v err=%v", match, err)
	}
	if strings.Contains(response.Body.String(), "password") || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("registration response exposes password or is cacheable: %s", response.Body.String())
	}

	duplicate := authTestRequest(handler, http.MethodPost, "/users", `{"username":"alice_1","email":"other@example.com","password":"correct horse battery staple"}`, nil)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", duplicate.Code)
	}
}

func TestLoginSessionAndLogout(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "local", true: "secure"}[secure], func(t *testing.T) {
			db := newAuthTestDB()
			hash, err := argon2id.CreateHash("correct horse battery staple", argon2id.DefaultParams)
			if err != nil {
				t.Fatal(err)
			}
			var id pgtype.UUID
			_ = id.Scan("00000000-0000-4000-8000-000000000001")
			db.user = &authTestUser{id: id, username: "alice", email: "alice@example.com", passwordHash: hash, role: "USER"}
			handler := authTestHandler(db, secure)

			wrong := authTestRequest(handler, http.MethodPost, "/users/login", `{"identifier":"alice","password":"wrong password"}`, nil)
			missing := authTestRequest(handler, http.MethodPost, "/users/login", `{"identifier":"nobody@example.com","password":"wrong password"}`, nil)
			if wrong.Code != http.StatusUnauthorized || missing.Code != http.StatusUnauthorized || wrong.Body.String() != missing.Body.String() {
				t.Fatalf("invalid login responses differ: wrong=%d %q missing=%d %q", wrong.Code, wrong.Body.String(), missing.Code, missing.Body.String())
			}
			if len(wrong.Result().Cookies()) != 0 || len(missing.Result().Cookies()) != 0 {
				t.Fatal("invalid login issued a cookie")
			}
			if wrong.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("login failure is cacheable")
			}

			login := authTestRequest(handler, http.MethodPost, "/users/login", `{"identifier":" ALICE@EXAMPLE.COM ","password":"correct horse battery staple"}`, nil)
			if login.Code != http.StatusOK {
				t.Fatalf("login status = %d, body = %s", login.Code, login.Body.String())
			}
			cookies := login.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("login cookies = %d, want 1", len(cookies))
			}
			cookie := cookies[0]
			wantName := "aux_session"
			if secure {
				wantName = "__Host-aux_session"
			}
			if cookie.Name != wantName || cookie.Path != "/" || !cookie.HttpOnly || cookie.Secure != secure || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge <= 0 {
				t.Fatalf("login cookie attributes = %+v", cookie)
			}
			sum := sha256.Sum256([]byte(cookie.Value))
			tokenHash := hex.EncodeToString(sum[:])
			if _, ok := db.sessions[tokenHash]; !ok {
				t.Fatal("session token hash was not stored")
			}
			if _, ok := db.sessions[cookie.Value]; ok {
				t.Fatal("raw session token was stored")
			}

			me := authTestRequest(handler, http.MethodGet, "/users/me", "", cookie)
			if me.Code != http.StatusOK || me.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("current user status = %d, body = %s", me.Code, me.Body.String())
			}
			var profile map[string]any
			if err := json.Unmarshal(me.Body.Bytes(), &profile); err != nil {
				t.Fatal(err)
			}
			if profile["username"] != "alice" || profile["role"] != "USER" || profile["password_hash"] != nil {
				t.Fatalf("current user response = %v", profile)
			}

			logout := authTestRequest(handler, http.MethodPost, "/users/logout", "", cookie)
			if logout.Code != http.StatusNoContent || logout.Body.Len() != 0 {
				t.Fatalf("logout response = %d %q", logout.Code, logout.Body.String())
			}
			if _, ok := db.sessions[tokenHash]; ok {
				t.Fatal("logout left session valid")
			}
			cleared := logout.Result().Cookies()
			if len(cleared) != 1 || cleared[0].Name != wantName || cleared[0].Path != "/" || cleared[0].MaxAge >= 0 {
				t.Fatalf("logout did not clear matching cookie: %+v", cleared)
			}
			if after := authTestRequest(handler, http.MethodGet, "/users/me", "", cookie); after.Code != http.StatusUnauthorized {
				t.Fatalf("revoked session status = %d, want 401", after.Code)
			}
		})
	}
}

func TestExpiredSessionIsRejectedAndCanBeLoggedOut(t *testing.T) {
	db := newAuthTestDB()
	var id pgtype.UUID
	_ = id.Scan("00000000-0000-4000-8000-000000000001")
	db.user = &authTestUser{id: id, username: "alice", email: "alice@example.com", role: "USER"}
	handler := authTestHandler(db, false)
	cookie := &http.Cookie{Name: "aux_session", Value: "expired-token"}
	sum := sha256.Sum256([]byte(cookie.Value))
	hash := hex.EncodeToString(sum[:])
	db.sessions[hash] = time.Now().Add(-time.Minute)
	if response := authTestRequest(handler, http.MethodGet, "/users/me", "", cookie); response.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status = %d, want 401", response.Code)
	}
	if response := authTestRequest(handler, http.MethodPost, "/users/logout", "", cookie); response.Code != http.StatusNoContent {
		t.Fatalf("expired session logout status = %d, want 204", response.Code)
	}
	if _, ok := db.sessions[hash]; ok {
		t.Fatal("expired session still stored after logout")
	}
}

func TestAuthDatabaseFailuresDoNotLookLikeInvalidCredentials(t *testing.T) {
	db := newAuthTestDB()
	var id pgtype.UUID
	_ = id.Scan("00000000-0000-4000-8000-000000000001")
	db.user = &authTestUser{id: id, username: "alice", email: "alice@example.com", role: "USER"}
	handler := authTestHandler(db, false)
	cookie := &http.Cookie{Name: "aux_session", Value: "some-token"}
	db.getSessionErr = errors.New("database unavailable")
	if response := authTestRequest(handler, http.MethodGet, "/users/me", "", cookie); response.Code != http.StatusInternalServerError {
		t.Fatalf("session lookup error status = %d, want 500", response.Code)
	}
	db.deleteSessionErr = errors.New("database unavailable")
	logout := authTestRequest(handler, http.MethodPost, "/users/logout", "", cookie)
	if logout.Code != http.StatusInternalServerError || len(logout.Result().Cookies()) != 0 {
		t.Fatalf("logout database error response = %d, cookies = %+v", logout.Code, logout.Result().Cookies())
	}
}

func TestRegisterRejectsOversizedPasswordClearly(t *testing.T) {
	handler := authTestHandler(newAuthTestDB(), false)
	body := `{"username":"alice","email":"alice@example.com","password":"` + strings.Repeat("x", 1025) + `"}`
	response := authTestRequest(handler, http.MethodPost, "/users", body, nil)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "too long") {
		t.Fatalf("oversized password response = %d %q", response.Code, response.Body.String())
	}
}

func TestRegisterRejectsInvalidRequestsBeforeWritingUser(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"invalid username", `{"username":"a!","email":"alice@example.com","password":"long-enough"}`, http.StatusBadRequest},
		{"invalid email", `{"username":"alice","email":"not-an-email","password":"long-enough"}`, http.StatusBadRequest},
		{"short password", `{"username":"alice","email":"alice@example.com","password":"short"}`, http.StatusBadRequest},
		{"unknown field", `{"username":"alice","email":"alice@example.com","password":"long-enough","role":"ADMIN"}`, http.StatusBadRequest},
		{"too large body", `{"username":"alice","email":"alice@example.com","password":"` + strings.Repeat("x", 65<<10) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newAuthTestDB()
			response := authTestRequest(authTestHandler(db, false), http.MethodPost, "/users", tt.body, nil)
			if response.Code != tt.wantStatus || db.user != nil {
				t.Fatalf("registration response = %d %q, user = %+v", response.Code, response.Body.String(), db.user)
			}
		})
	}
}

func TestLoginDoesNotIssueCookieWhenSessionInsertFails(t *testing.T) {
	db := newAuthTestDB()
	hash, err := argon2id.CreateHash("correct horse battery staple", argon2id.DefaultParams)
	if err != nil {
		t.Fatal(err)
	}
	var id pgtype.UUID
	_ = id.Scan("00000000-0000-4000-8000-000000000001")
	db.user = &authTestUser{id: id, username: "alice", email: "alice@example.com", passwordHash: hash, role: "USER"}
	db.createSessionErr = errors.New("database unavailable")
	response := authTestRequest(authTestHandler(db, false), http.MethodPost, "/users/login", `{"identifier":"alice","password":"correct horse battery staple"}`, nil)
	if response.Code != http.StatusInternalServerError || len(response.Result().Cookies()) != 0 {
		t.Fatalf("failed session insert response = %d, cookies = %+v", response.Code, response.Result().Cookies())
	}
}

func TestMissingSessionIsUnauthorizedAndLogoutIsIdempotent(t *testing.T) {
	handler := authTestHandler(newAuthTestDB(), false)
	if response := authTestRequest(handler, http.MethodGet, "/users/me", "", nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("missing session status = %d, want 401", response.Code)
	}
	logout := authTestRequest(handler, http.MethodPost, "/users/logout", "", nil)
	if logout.Code != http.StatusNoContent || len(logout.Result().Cookies()) != 1 || logout.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout without session = %d, cookies = %+v", logout.Code, logout.Result().Cookies())
	}
}
