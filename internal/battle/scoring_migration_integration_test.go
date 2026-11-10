package battle

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestScoringMigrationPreservesScoresAndRejectsUnsafeRollback(t *testing.T) {
	l, ctx, _ := lobbyFixture(t)
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var suffix string
	if err := tx.QueryRow(ctx, `SELECT replace(gen_random_uuid()::text,'-','')`).Scan(&suffix); err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{"scoring_" + suffix}.Sanitize()
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+schema); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile("../../db/migrations/00001_initial_schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, strings.Split(string(original), "-- +goose Down")[0]); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO users(username,email,password_hash) VALUES('host','host@test','test');
        INSERT INTO battles(host_id,invite_code) SELECT id,'ABC123' FROM users;
        INSERT INTO rounds(battle_id,round_number,host_points) SELECT id,1,100 FROM battles;
        INSERT INTO guesses(round_id,user_id,attempt,guess,correct,points_awarded) SELECT r.id,u.id,1,'answer',true,100 FROM rounds r CROSS JOIN users u;`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../db/migrations/00004_round_scoring.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(migration), "-- +goose Down")
	if _, err := tx.Exec(ctx, parts[0]); err != nil {
		t.Fatal(err)
	}
	var points int32
	if err := tx.QueryRow(ctx, `SELECT points_awarded FROM guesses`).Scan(&points); err != nil || points != 100 {
		t.Fatalf("old score=%d err=%v", points, err)
	}
	for _, score := range []int32{110, 30} {
		if _, err := tx.Exec(ctx, `UPDATE guesses SET points_awarded=$1`, score); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE rounds SET host_points=$1`, score); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `SAVEPOINT rollback_check`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, parts[1]); err == nil {
			t.Fatalf("rollback accepted incompatible score %d", score)
		}
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT rollback_check`); err != nil {
			t.Fatal(err)
		}
		if err := tx.QueryRow(ctx, `SELECT points_awarded FROM guesses`).Scan(&points); err != nil || points != score {
			t.Fatalf("new score lost=%d want=%d err=%v", points, score, err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE guesses SET points_awarded=100; UPDATE rounds SET host_points=100;`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, parts[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, parts[0]); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT points_awarded FROM guesses`).Scan(&points); err != nil || points != 100 {
		t.Fatalf("round-trip score=%d err=%v", points, err)
	}
}
