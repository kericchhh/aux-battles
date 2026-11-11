package live

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const hubTestApplication = "aux_battles_live_hub_test"

func hubFixture(t *testing.T) (*Hub, *pgxpool.Pool, pgtype.UUID, context.CancelFunc, <-chan struct{}) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL live update tests")
	}
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["application_name"] = hubTestApplication
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var host, battle pgtype.UUID
	if err := pool.QueryRow(t.Context(), `INSERT INTO users (username,email,password_hash)
		VALUES (gen_random_uuid()::text,gen_random_uuid()::text,'test') RETURNING id`).Scan(&host); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(t.Context(), `INSERT INTO battles (host_id, invite_code)
		VALUES ($1, upper(substr(md5(random()::text), 1, 6))) RETURNING id`, host).Scan(&battle); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM battles WHERE id=$1`, battle)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, host)
	})
	hub := NewHub(pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	changes, unsubscribe := hub.Subscribe(battle.String())
	t.Cleanup(unsubscribe)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	expectSignal(t, changes, "initial connection")
	return hub, pool, battle, cancel, changes
}

func expectSignal(t *testing.T, changes <-chan struct{}, what string) {
	t.Helper()
	select {
	case _, open := <-changes:
		if !open {
			t.Fatalf("%s: subscription closed", what)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: no signal", what)
	}
}

func expectQuiet(t *testing.T, changes <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-changes:
		t.Fatalf("%s: unexpected signal", what)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestHubDeliversCommittedBattleChanges(t *testing.T) {
	hub, pool, battle, _, changes := hubFixture(t)
	ctx := t.Context()
	other, unsubscribe := hub.Subscribe("00000000-0000-4000-8000-000000000000")
	defer unsubscribe()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE battles SET last_activity_at = now() WHERE id = $1`, battle); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE battles SET last_activity_at = now() WHERE id = $1`, battle); err != nil {
		t.Fatal(err)
	}
	expectQuiet(t, changes, "before commit")
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	expectSignal(t, changes, "battle update")
	expectQuiet(t, changes, "duplicate notifications in one transaction")
	expectQuiet(t, other, "another battle's subscriber")

	if _, err := pool.Exec(ctx, `INSERT INTO rounds (battle_id, round_number) VALUES ($1, 1)`, battle); err != nil {
		t.Fatal(err)
	}
	expectSignal(t, changes, "round insert")
	if _, err := pool.Exec(ctx, `UPDATE rounds SET host_points = 0 WHERE battle_id = $1`, battle); err != nil {
		t.Fatal(err)
	}
	expectSignal(t, changes, "round update")
}

func TestHubSignalsEveryoneAfterReconnecting(t *testing.T) {
	_, pool, battle, _, changes := hubFixture(t)
	ctx := t.Context()
	var terminated int
	if err := pool.QueryRow(ctx, `SELECT count(pg_terminate_backend(pid))::integer FROM pg_stat_activity
		WHERE datname = current_database() AND application_name = $1 AND query = 'LISTEN battle_events'`,
		hubTestApplication).Scan(&terminated); err != nil {
		t.Fatal(err)
	}
	if terminated == 0 {
		t.Fatal("found no listener connection to terminate")
	}
	expectSignal(t, changes, "reconnect")
	if _, err := pool.Exec(ctx, `UPDATE battles SET last_activity_at = now() WHERE id = $1`, battle); err != nil {
		t.Fatal(err)
	}
	expectSignal(t, changes, "update after reconnect")
}

func TestHubClosesSubscriptionsWhenStopped(t *testing.T) {
	hub, _, battle, cancel, changes := hubFixture(t)
	cancel()
	select {
	case _, open := <-changes:
		if open {
			if _, open = <-changes; open {
				t.Fatal("subscription stayed open after stop")
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("subscription was not closed")
	}
	late, unsubscribe := hub.Subscribe(battle.String())
	defer unsubscribe()
	if _, open := <-late; open {
		t.Fatal("subscription after stop is open")
	}
}
