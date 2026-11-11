package battle

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
)

func lobbyFixture(t *testing.T) (*Lobby, context.Context, []pgtype.UUID) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL battle tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	users := make([]pgtype.UUID, 3)
	for i := range users {
		err := pool.QueryRow(ctx, `INSERT INTO users (username,email,password_hash)
   VALUES (gen_random_uuid()::text,gen_random_uuid()::text,'test') RETURNING id`).Scan(&users[i])
		if err != nil {
			t.Fatal(err)
		}
		user := users[i]
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM battles WHERE host_id=$1 OR guest_id=$1`, user); err != nil {
				t.Error(err)
			}
			if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, user); err != nil {
				t.Error(err)
			}
		})
	}
	return New(pool, media.FromEnv()), ctx, users
}

func TestLobbyConcurrentJoinAndParticipantView(t *testing.T) {
	l, ctx, u := lobbyFixture(t)
	id, err := l.Create(ctx, u[0], 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := l.Get(ctx, u[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Status != "PENDING" || initial.Rounds != 7 || initial.OpponentJoined || initial.MyLineupLocked || initial.Round != nil || len(initial.InviteCode) != 6 {
		t.Fatalf("initial view = %+v", initial)
	}
	if _, err := l.Get(ctx, u[1], id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider view error = %v", err)
	}
	if _, err := l.Join(ctx, u[0], initial.InviteCode); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("self join error = %v", err)
	}
	start := make(chan struct{})
	type result struct {
		user pgtype.UUID
		err  error
	}
	results := make(chan result, 2)
	for _, user := range u[1:] {
		go func() {
			<-start
			_, err := l.Join(ctx, user, "  "+strings.ToLower(initial.InviteCode)+"  ")
			results <- result{user, err}
		}()
	}
	close(start)
	var winner pgtype.UUID
	successes, conflicts := 0, 0
	for range 2 {
		r := <-results
		if r.err == nil {
			successes++
			winner = r.user
		} else if errors.Is(r.err, ErrUnavailable) {
			conflicts++
		} else {
			t.Fatal(r.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("joins: %d success, %d conflict", successes, conflicts)
	}
	var count int
	if err := l.pool.QueryRow(ctx, `SELECT count(*) FROM rounds WHERE battle_id=$1`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 7 {
		t.Fatalf("created rounds=%d, want 7", count)
	}
	for _, user := range []pgtype.UUID{u[0], winner} {
		view, err := l.Get(ctx, user, id)
		if err != nil {
			t.Fatal(err)
		}
		if view.Status != "SELECTING" || !view.OpponentJoined || view.MyLineupLocked || view.OpponentLineupLocked || view.MyScore != 0 || view.OpponentScore != 0 || view.Outcome != nil || view.Round != nil || len(view.CompletedRounds) != 0 {
			t.Fatalf("joined view=%+v", view)
		}
	}
	if _, err := l.Join(ctx, winner, initial.InviteCode); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("repeat join error=%v", err)
	}
}

func TestLobbyJoinRollsBackWhenRoundCreationFails(t *testing.T) {
	l, ctx, u := lobbyFixture(t)
	id, err := l.Create(ctx, u[0], 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	view, err := l.Get(ctx, u[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.pool.Exec(ctx, `INSERT INTO rounds (battle_id,round_number) VALUES ($1,1)`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Join(ctx, u[1], view.InviteCode); err == nil {
		t.Fatal("join succeeded despite duplicate round")
	}
	view, err = l.Get(ctx, u[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "PENDING" || view.OpponentJoined {
		t.Fatalf("join was not rolled back: %+v", view)
	}
	var count int
	if err := l.pool.QueryRow(ctx, `SELECT count(*) FROM rounds WHERE battle_id=$1`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("round count after rollback=%d", count)
	}
}

func TestLobbyRoundLimitAndInviteCollisions(t *testing.T) {
	l, ctx, u := lobbyFixture(t)
	for _, rounds := range []int32{-1, 0, 8, 10} {
		if _, err := l.Create(ctx, u[0], rounds, 0); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("rounds=%d: %v", rounds, err)
		}
	}
	id, err := l.Create(ctx, u[0], 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	view, err := l.Get(ctx, u[0], id)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	_, err = l.create(ctx, u[0], 5, 0, func() (string, error) { calls++; return view.InviteCode, nil })
	if !errors.Is(err, ErrInviteExhausted) || calls != 5 {
		t.Fatalf("collision retry: calls=%d err=%v", calls, err)
	}
	_, err = dbgen.New(l.pool).CreateBattle(ctx, dbgen.CreateBattleParams{HostID: u[0], InviteCode: "ABCDEF", RoundCount: 8})
	var constraintErr *pgconn.PgError
	if !errors.As(err, &constraintErr) || constraintErr.Code != "23514" || constraintErr.ConstraintName != "battles_round_count_max_seven" {
		t.Fatalf("eight-round insert = %v, want seven-round constraint violation", err)
	}
}
