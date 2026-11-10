package battle

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestEndingsMigrationBackfillsAndRejectsUnsafeRollback(t *testing.T) {
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
	schema := pgx.Identifier{"endings_" + suffix}.Sanitize()
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+schema+"; SET LOCAL search_path TO "+schema); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"00001_initial_schema", "00002_song_jobs", "00003_battle_round_limit", "00004_round_scoring"} {
		migration, err := os.ReadFile(fmt.Sprintf("../../db/migrations/%s.sql", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, strings.Split(string(migration), "-- +goose Down")[0]); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO users(username,email,password_hash) VALUES('host','host@test','test');
        INSERT INTO battles(host_id,invite_code,status,round_count) SELECT id,'DONE01','FINISHED',1 FROM users;
        INSERT INTO battles(host_id,invite_code) SELECT id,'OPEN01' FROM users;`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../db/migrations/00005_battle_endings.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(migration), "-- +goose Down")
	if _, err := tx.Exec(ctx, parts[0]); err != nil {
		t.Fatal(err)
	}
	reasons := func() map[string]string {
		rows, err := tx.Query(ctx, `SELECT invite_code, coalesce(end_reason, '') FROM battles`)
		if err != nil {
			t.Fatal(err)
		}
		got, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) ([2]string, error) {
			var pair [2]string
			return pair, row.Scan(&pair[0], &pair[1])
		})
		if err != nil {
			t.Fatal(err)
		}
		m := make(map[string]string, len(got))
		for _, pair := range got {
			m[pair[0]] = pair[1]
		}
		return m
	}
	if got := reasons(); got["DONE01"] != "COMPLETED" || got["OPEN01"] != "" {
		t.Fatalf("backfilled reasons=%v", got)
	}
	for _, invalid := range []string{
		`UPDATE battles SET status='CANCELLED' WHERE invite_code='OPEN01'`,
		`UPDATE battles SET end_reason='FORFEIT' WHERE invite_code='OPEN01'`,
		`UPDATE battles SET end_reason='FORFEIT' WHERE invite_code='DONE01'`,
	} {
		if _, err := tx.Exec(ctx, `SAVEPOINT invalid_state`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, invalid); err == nil {
			t.Fatalf("accepted inconsistent ending: %s", invalid)
		}
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT invalid_state`); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE battles SET status='CANCELLED', end_reason='TIMEOUT' WHERE invite_code='OPEN01'`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SAVEPOINT rollback_check`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, parts[1]); err == nil {
		t.Fatal("rollback accepted a cancelled battle")
	}
	if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT rollback_check`); err != nil {
		t.Fatal(err)
	}
	if got := reasons(); got["OPEN01"] != "TIMEOUT" {
		t.Fatalf("refused rollback changed data: %v", got)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM battles WHERE invite_code='OPEN01'`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, parts[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, parts[0]); err != nil {
		t.Fatal(err)
	}
	if got := reasons(); got["DONE01"] != "COMPLETED" || len(got) != 1 {
		t.Fatalf("round-trip reasons=%v", got)
	}
}
