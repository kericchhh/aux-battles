package server

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestSongListParams(t *testing.T) {
	params, err := songListParams(url.Values{})
	if err != nil || params.MaxSongs != defaultSongPage || params.Pattern.Valid || params.BeforeCreatedAt.Valid {
		t.Fatalf("defaults = %+v, %v", params, err)
	}

	params, err = songListParams(url.Values{"q": {"  50%_off\\  "}, "limit": {"100"}})
	if err != nil {
		t.Fatal(err)
	}
	if params.Pattern.String != `%50\%\_off\\%` || params.MaxSongs != 100 {
		t.Fatalf("escaped params = %+v", params)
	}

	var id pgtype.UUID
	if err := id.Scan("00000000-0000-4000-8000-000000000001"); err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, 11, 10, 12, 30, 0, 123456000, time.UTC)
	params, err = songListParams(url.Values{"cursor": {encodeSongCursor(createdAt, id)}})
	if err != nil || !params.BeforeCreatedAt.Time.Equal(createdAt) || params.BeforeID != id {
		t.Fatalf("cursor params = %+v, %v", params, err)
	}

	for name, values := range map[string]url.Values{
		"zero limit":      {"limit": {"0"}},
		"large limit":     {"limit": {"101"}},
		"text limit":      {"limit": {"ten"}},
		"long search":     {"q": {strings.Repeat("é", maxSongQuery+1)}},
		"invalid utf-8":   {"q": {"\xff"}},
		"nul in search":   {"q": {"a\x00b"}},
		"garbage cursor":  {"cursor": {"not base64!"}},
		"cursor no id":    {"cursor": {"MjAyNi0xMS0xMFQxMjozMDowMFo"}},
		"cursor bad uuid": {"cursor": {encodeSongCursor(createdAt, id)[:20]}},
	} {
		if _, err := songListParams(values); err == nil {
			t.Errorf("%s: accepted %v", name, values)
		}
	}
}

func TestRequireAdminChecksRole(t *testing.T) {
	db := newAuthTestDB()
	var id pgtype.UUID
	if err := id.Scan("00000000-0000-4000-8000-000000000001"); err != nil {
		t.Fatal(err)
	}
	token := "admin-token"
	sum := sha256.Sum256([]byte(token))
	db.sessions[hex.EncodeToString(sum[:])] = time.Now().Add(time.Hour)
	s := &server{queries: dbgen.New(db), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	handler := s.requireAdmin(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	cookie := &http.Cookie{Name: "aux_session", Value: token}

	for role, want := range map[string]int{"USER": http.StatusForbidden, "ADMIN": http.StatusNoContent} {
		db.user = &authTestUser{id: id, username: "alice", email: "alice@example.com", role: role}
		if response := authTestRequest(handler, http.MethodPost, "/songs", "", cookie); response.Code != want {
			t.Fatalf("%s: status = %d, want %d", role, response.Code, want)
		}
	}
	if response := authTestRequest(handler, http.MethodPost, "/songs", "", nil); response.Code != http.StatusUnauthorized {
		t.Fatalf("no session: status = %d, want 401", response.Code)
	}
}

func TestSongRoutesRequireAuthentication(t *testing.T) {
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/songs"},
		{http.MethodGet, "/songs/00000000-0000-4000-8000-000000000001"},
		{http.MethodPost, "/songs"},
		{http.MethodGet, "/songs/00000000-0000-4000-8000-000000000001/status"},
	} {
		response := authTestRequest(authTestHandler(newAuthTestDB(), false), tc.method, tc.path, "", nil)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d", tc.method, tc.path, response.Code)
		}
	}
}

func TestSongRoutesWithUserSession(t *testing.T) {
	db := newAuthTestDB()
	var id pgtype.UUID
	if err := id.Scan("00000000-0000-4000-8000-000000000001"); err != nil {
		t.Fatal(err)
	}
	db.user = &authTestUser{id: id, username: "alice", email: "alice@example.com", role: "USER"}
	token := "user-token"
	sum := sha256.Sum256([]byte(token))
	db.sessions[hex.EncodeToString(sum[:])] = time.Now().Add(time.Hour)
	handler := authTestHandler(db, false)
	cookie := &http.Cookie{Name: "aux_session", Value: token}

	for _, tc := range []struct {
		path string
		want int
	}{
		{"/songs/not-a-uuid", http.StatusBadRequest},
		{"/songs/00000000-0000-4000-8000-000000000001/status", http.StatusForbidden},
		{"/songs?q=%FF", http.StatusBadRequest},
		{"/songs?q=a%00b", http.StatusBadRequest},
	} {
		if response := authTestRequest(handler, http.MethodGet, tc.path, "", cookie); response.Code != tc.want {
			t.Errorf("GET %s = %d %s, want %d", tc.path, response.Code, response.Body.String(), tc.want)
		}
	}
}
