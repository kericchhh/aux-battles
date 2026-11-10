package battle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func backdateBattle(t *testing.T, l *Lobby, ctx context.Context, id pgtype.UUID, age time.Duration) {
	t.Helper()
	if _, err := l.pool.Exec(ctx, `UPDATE battles SET last_activity_at = now() - make_interval(secs => $2) WHERE id = $1`, id, age.Seconds()); err != nil {
		t.Fatal(err)
	}
}

type battleEnding struct {
	status string
	reason pgtype.Text
	winner pgtype.UUID
}

func readEnding(t *testing.T, l *Lobby, ctx context.Context, id pgtype.UUID) battleEnding {
	t.Helper()
	var e battleEnding
	if err := l.pool.QueryRow(ctx, `SELECT status, end_reason, winner_id FROM battles WHERE id = $1`, id).Scan(&e.status, &e.reason, &e.winner); err != nil {
		t.Fatal(err)
	}
	return e
}

func recentlyActive(t *testing.T, l *Lobby, ctx context.Context, id pgtype.UUID) bool {
	t.Helper()
	var recent bool
	if err := l.pool.QueryRow(ctx, `SELECT now() - last_activity_at < interval '1 minute' FROM battles WHERE id = $1`, id).Scan(&recent); err != nil {
		t.Fatal(err)
	}
	return recent
}

func roundIDFor(t *testing.T, l *Lobby, ctx context.Context, user, id pgtype.UUID) pgtype.UUID {
	t.Helper()
	view, err := l.Get(ctx, user, id)
	if err != nil {
		t.Fatal(err)
	}
	var roundID pgtype.UUID
	if view.Round == nil {
		t.Fatalf("no active round: %+v", view)
	}
	if err := roundID.Scan(view.Round.ID); err != nil {
		t.Fatal(err)
	}
	return roundID
}

func TestForfeitInEachBattleState(t *testing.T) {
	l, ctx, users := lobbyFixture(t)
	pending, err := l.Create(ctx, users[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Forfeit(ctx, users[1], pending); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider forfeit error=%v", err)
	}
	view, err := l.Forfeit(ctx, users[0], pending)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "CANCELLED" || view.Outcome != nil || view.EndReason == nil || *view.EndReason != "FORFEIT" {
		t.Fatalf("cancelled lobby=%+v", view)
	}
	if _, err := l.Forfeit(ctx, users[0], pending); !errors.Is(err, ErrBattleEnded) {
		t.Fatalf("second forfeit error=%v", err)
	}
	if _, err := l.Join(ctx, users[1], view.InviteCode); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("join cancelled lobby error=%v", err)
	}

	l, ctx, users, selecting, root := lineupFixture(t, 1)
	view, err = l.Forfeit(ctx, users[1], selecting)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "FINISHED" || view.Outcome == nil || *view.Outcome != "LOSS" || *view.EndReason != "FORFEIT" {
		t.Fatalf("forfeiting guest view=%+v", view)
	}
	if e := readEnding(t, l, ctx, selecting); e.winner != users[0] {
		t.Fatalf("selecting forfeit ending=%+v", e)
	}
	if _, err := l.SubmitLineup(ctx, users[0], selecting, []pgtype.UUID{lineupSong(t, l, ctx, root, true)}); !errors.Is(err, ErrLineupConflict) {
		t.Fatalf("lineup after forfeit error=%v", err)
	}

	l, users, ongoing, _ := activeMatch(t, 1)
	ctx = t.Context()
	roundID := roundIDFor(t, l, ctx, users[0], ongoing)
	if _, err := l.Forfeit(ctx, users[0], ongoing); err != nil {
		t.Fatal(err)
	}
	guestView, err := l.Get(ctx, users[1], ongoing)
	if err != nil {
		t.Fatal(err)
	}
	if guestView.Outcome == nil || *guestView.Outcome != "WIN" || guestView.Round != nil {
		t.Fatalf("winning guest view=%+v", guestView)
	}
	if _, err := l.SubmitGuess(ctx, users[1], roundID, "Host selection", 1); !errors.Is(err, ErrRoundUnavailable) {
		t.Fatalf("guess after forfeit error=%v", err)
	}
	if _, _, err := l.OpenRoundAudio(ctx, users[1], roundID); !errors.Is(err, ErrRoundUnavailable) {
		t.Fatalf("audio after forfeit error=%v", err)
	}
}

func TestExpireInactiveEndsOnlyStaleBattles(t *testing.T) {
	l, ctx, users := lobbyFixture(t)
	stalePending, err := l.Create(ctx, users[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	backdateBattle(t, l, ctx, stalePending, lobbyTimeout+time.Minute)
	freshPending, err := l.Create(ctx, users[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	backdateBattle(t, l, ctx, freshPending, lobbyTimeout-5*time.Minute)

	hostLineup, _, selUsers, hostLineupID, root := lineupFixture(t, 1)
	if _, err := hostLineup.SubmitLineup(ctx, selUsers[0], hostLineupID, []pgtype.UUID{lineupSong(t, hostLineup, ctx, root, true)}); err != nil {
		t.Fatal(err)
	}
	backdateBattle(t, l, ctx, hostLineupID, gameTimeout+time.Minute)
	_, _, _, noLineupID, _ := lineupFixture(t, 1)
	backdateBattle(t, l, ctx, noLineupID, gameTimeout+time.Minute)

	guestDone, guestDoneUsers, guestDoneID, _ := activeMatch(t, 1)
	if _, err := guestDone.SubmitGuess(ctx, guestDoneUsers[1], roundIDFor(t, guestDone, ctx, guestDoneUsers[1], guestDoneID), "Host selection", 1); err != nil {
		t.Fatal(err)
	}
	backdateBattle(t, l, ctx, guestDoneID, gameTimeout+time.Minute)
	_, _, nobodyDoneID, _ := activeMatch(t, 1)
	backdateBattle(t, l, ctx, nobodyDoneID, gameTimeout+time.Minute)
	_, _, freshGameID, _ := activeMatch(t, 1)
	backdateBattle(t, l, ctx, freshGameID, gameTimeout-time.Minute)

	ended, err := l.ExpireInactive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ended < 5 {
		t.Fatalf("ended %d battles, want at least 5", ended)
	}
	timeout := pgtype.Text{String: "TIMEOUT", Valid: true}
	for _, tc := range []struct {
		name string
		id   pgtype.UUID
		want battleEnding
	}{
		{"stale lobby", stalePending, battleEnding{"CANCELLED", timeout, pgtype.UUID{}}},
		{"fresh lobby", freshPending, battleEnding{"PENDING", pgtype.Text{}, pgtype.UUID{}}},
		{"one lineup", hostLineupID, battleEnding{"FINISHED", timeout, selUsers[0]}},
		{"no lineups", noLineupID, battleEnding{"CANCELLED", timeout, pgtype.UUID{}}},
		{"guest finished round", guestDoneID, battleEnding{"FINISHED", timeout, guestDoneUsers[1]}},
		{"nobody finished round", nobodyDoneID, battleEnding{"CANCELLED", timeout, pgtype.UUID{}}},
		{"fresh game", freshGameID, battleEnding{"ONGOING", pgtype.Text{}, pgtype.UUID{}}},
	} {
		if got := readEnding(t, l, ctx, tc.id); got != tc.want {
			t.Errorf("%s ending=%+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestMovesRecordActivityAndCompletionReason(t *testing.T) {
	l, ctx, users := lobbyFixture(t)
	id, err := l.Create(ctx, users[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	backdateBattle(t, l, ctx, id, 10*time.Minute)
	if _, err := l.Join(ctx, users[1], view.InviteCode); err != nil {
		t.Fatal(err)
	}
	if !recentlyActive(t, l, ctx, id) {
		t.Fatal("join did not record activity")
	}

	l, ctx, users, id, root := lineupFixture(t, 1)
	backdateBattle(t, l, ctx, id, 10*time.Minute)
	if _, err := l.SubmitLineup(ctx, users[0], id, []pgtype.UUID{lineupSong(t, l, ctx, root, true)}); err != nil {
		t.Fatal(err)
	}
	if !recentlyActive(t, l, ctx, id) {
		t.Fatal("lineup did not record activity")
	}

	l, users, id, _ = activeMatch(t, 1)
	ctx = t.Context()
	roundID := roundIDFor(t, l, ctx, users[0], id)
	backdateBattle(t, l, ctx, id, 10*time.Minute)
	if _, err := l.SubmitGuess(ctx, users[0], roundID, "wrong title", 1); err != nil {
		t.Fatal(err)
	}
	if !recentlyActive(t, l, ctx, id) {
		t.Fatal("guess did not record activity")
	}
	for i, user := range users[:2] {
		answer := "Guest selection"
		attempt := int32(2)
		if i == 1 {
			answer, attempt = "Host selection", 1
		}
		if _, err := l.SubmitGuess(ctx, user, roundID, answer, attempt); err != nil {
			t.Fatal(err)
		}
	}
	if e := readEnding(t, l, ctx, id); e.status != "FINISHED" || e.reason.String != "COMPLETED" {
		t.Fatalf("completed battle ending=%+v", e)
	}
}
