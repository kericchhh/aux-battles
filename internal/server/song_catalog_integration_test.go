package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestCatalogOnlyShowsReadySongs(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL catalog test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var readyID, pendingID string
	err = pool.QueryRow(ctx, `INSERT INTO songs (title, artist, genre, duration_seconds, status,
		full_song_path, drums_path, bass_path, melody_path, vocals_path)
		VALUES ('Ready catalog test', 'Test', 'Test', 12, 'READY', 'full', 'drums', 'bass', 'other', 'vocals')
		RETURNING id::text`).Scan(&readyID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM songs WHERE id = $1::uuid`, readyID) })
	err = pool.QueryRow(ctx, `INSERT INTO songs (title, artist, genre, duration_seconds)
		VALUES ('Pending catalog test', 'Test', 'Test', 12) RETURNING id::text`).Scan(&pendingID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM songs WHERE id = $1::uuid`, pendingID) })

	queries := dbgen.New(pool)
	listed, err := queries.ListSongs(ctx, dbgen.ListSongsParams{MaxSongs: 1000})
	if err != nil {
		t.Fatal(err)
	}
	var foundReady, foundPending bool
	for _, song := range listed {
		foundReady = foundReady || song.ID.String() == readyID
		foundPending = foundPending || song.ID.String() == pendingID
	}
	if !foundReady || foundPending {
		t.Fatalf("catalog contains ready=%t pending=%t, want true/false", foundReady, foundPending)
	}
	var id pgtype.UUID
	if err := id.Scan(pendingID); err != nil {
		t.Fatal(err)
	}
	_, err = queries.GetSong(ctx, id)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("lookup of pending song = %v, want not found", err)
	}
}

func TestCatalogSearchAndPaging(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL catalog test")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	tag := fmt.Sprintf("paging%d", time.Now().UnixNano())
	titles := []string{tag + " one", tag + " two", tag + " three", tag + " 100% wild_card"}
	for i, title := range titles {
		var id string
		err := pool.QueryRow(ctx, `INSERT INTO songs (title, artist, genre, duration_seconds, status,
			full_song_path, drums_path, bass_path, melody_path, vocals_path, created_at)
			VALUES ($1, 'Pager', 'Test', 12, 'READY', 'f', 'd', 'b', 'm', 'v', now() - make_interval(secs => $2))
			RETURNING id::text`, title, float64(i)).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM songs WHERE id = $1::uuid`, id) })
	}

	db, cookie := catalogUser(t, pool)
	handler := New(db, pool, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), false)
	var got []string
	cursor := ""
	for pages := 0; ; pages++ {
		path := "/songs?limit=3&q=" + tag
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		response := authTestRequest(handler, http.MethodGet, path, "", cookie)
		if response.Code != http.StatusOK {
			t.Fatalf("page %d = %d %s", pages, response.Code, response.Body.String())
		}
		var page struct {
			Songs []struct {
				Title string `json:"title"`
			} `json:"songs"`
			NextCursor *string `json:"nextCursor"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		for _, song := range page.Songs {
			got = append(got, song.Title)
		}
		if page.NextCursor == nil {
			break
		}
		if pages > 2 {
			t.Fatal("paging did not terminate")
		}
		cursor = *page.NextCursor
	}
	if strings.Join(got, "|") != strings.Join(titles, "|") {
		t.Fatalf("paged titles = %v, want %v", got, titles)
	}

	for query, want := range map[string]int{"100%": 1, "wild_card": 1, "_": 1, "PAGER": 4, "nothing-matches-" + tag: 0} {
		response := authTestRequest(handler, http.MethodGet, "/songs?limit=100&q="+neturl.QueryEscape(query), "", cookie)
		var page struct {
			Songs []struct {
				Title string `json:"title"`
			} `json:"songs"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		matched := 0
		for _, song := range page.Songs {
			if strings.HasPrefix(song.Title, tag) {
				matched++
			}
		}
		if matched != want {
			t.Errorf("q=%q matched %d of ours, want %d", query, matched, want)
		}
	}
}

func catalogUser(t *testing.T, pool *pgxpool.Pool) (*dbgen.Queries, *http.Cookie) {
	t.Helper()
	ctx := t.Context()
	var id pgtype.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO users(username,email,password_hash)
		VALUES(gen_random_uuid()::text,gen_random_uuid()::text,'test') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id) })
	token := id.String()
	sum := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '1 hour')`, hex.EncodeToString(sum[:]), id); err != nil {
		t.Fatal(err)
	}
	return dbgen.New(pool), &http.Cookie{Name: "aux_session", Value: token}
}
