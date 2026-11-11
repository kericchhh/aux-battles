package battle

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func timedMatch(t *testing.T, rounds int32, hostAgo, guestAgo time.Duration) (*Lobby, context.Context, []pgtype.UUID, pgtype.UUID) {
	t.Helper()
	l, users, id, _ := activeMatch(t, rounds)
	ctx := t.Context()
	if _, err := l.pool.Exec(ctx, `UPDATE battles SET guess_seconds = 15 WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := l.pool.Exec(ctx, `UPDATE rounds SET host_stage_started_at = now() - make_interval(secs => $2),
		guest_stage_started_at = now() - make_interval(secs => $3) WHERE battle_id = $1 AND status = 'GUESSING'`,
		id, hostAgo.Seconds(), guestAgo.Seconds()); err != nil {
		t.Fatal(err)
	}
	return l, ctx, users, id
}

func roundFor(t *testing.T, l *Lobby, ctx context.Context, user, id pgtype.UUID) RoundView {
	t.Helper()
	view, err := l.Get(ctx, user, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Round == nil {
		t.Fatalf("no active round: %+v", view)
	}
	return *view.Round
}

func TestTimeoutCountsAsWrongGuess(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 17*time.Second, 0)
	before := roundFor(t, l, ctx, users[0], id)
	if before.StageEndsAt == nil {
		t.Fatal("timed round has no deadline")
	}
	expired, err := l.ExpireStages(ctx)
	if err != nil || expired != 1 {
		t.Fatalf("expired=%d err=%v", expired, err)
	}
	host := roundFor(t, l, ctx, users[0], id)
	if host.MyAttempts != 1 || host.MyStage != "BASS" || host.MyFinished || host.StageEndsAt == nil ||
		!host.StageEndsAt.Equal(before.StageEndsAt.Add(15*time.Second)) {
		t.Fatalf("host after timeout = %+v, before = %+v", host, before)
	}
	if guest := roundFor(t, l, ctx, users[1], id); guest.MyAttempts != 0 || guest.MyStage != "DRUM" {
		t.Fatalf("guest changed: %+v", guest)
	}
	var guess string
	var correct, timedOut bool
	var points int32
	if err := l.pool.QueryRow(ctx, `SELECT guess, correct, timed_out, points_awarded FROM guesses g
		JOIN rounds r ON r.id = g.round_id WHERE r.battle_id = $1 AND g.user_id = $2`, id, users[0]).Scan(&guess, &correct, &timedOut, &points); err != nil {
		t.Fatal(err)
	}
	if guess != "" || correct || !timedOut || points != 0 {
		t.Fatalf("recorded timeout guess=%q correct=%t timedOut=%t points=%d", guess, correct, timedOut, points)
	}
	if expired, err := l.ExpireStages(ctx); err != nil || expired != 0 {
		t.Fatalf("second sweep expired=%d err=%v", expired, err)
	}
}

func TestTimeoutsCatchUpMissedStems(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 40*time.Second, 0)
	if expired, err := l.ExpireStages(ctx); err != nil || expired != 2 {
		t.Fatalf("expired=%d err=%v", expired, err)
	}
	host := roundFor(t, l, ctx, users[0], id)
	if host.MyAttempts != 2 || host.MyStage != "MELODY" || host.StageEndsAt == nil {
		t.Fatalf("host after catch-up = %+v", host)
	}
	if remaining := time.Until(*host.StageEndsAt); remaining <= 0 || remaining > 6*time.Second {
		t.Fatalf("melody deadline in %v, want about 5s", remaining)
	}
}

func TestTimeoutsFinishRoundAfterOpponentGuessed(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 2, 0, 70*time.Second)
	if _, err := l.SubmitGuess(ctx, users[0], roundIDFor(t, l, ctx, users[0], id), "Guest selection", 1); err != nil {
		t.Fatal(err)
	}
	if expired, err := l.ExpireStages(ctx); err != nil || expired != 4 {
		t.Fatalf("expired=%d err=%v", expired, err)
	}
	view, err := l.Get(ctx, users[1], id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "ONGOING" || view.CurrentRound != 2 || view.Round == nil || view.Round.MyAttempts != 0 || view.Round.StageEndsAt == nil {
		t.Fatalf("view after timed out round = %+v", view)
	}
	if remaining := view.Round.StageEndsAt.Sub(view.ServerTime); remaining < 13*time.Second || remaining > 15*time.Second {
		t.Fatalf("next round deadline in %v, want a fresh 15s", remaining)
	}
	if len(view.CompletedRounds) != 1 || view.CompletedRounds[0].MyPoints != 0 || view.CompletedRounds[0].OpponentPoints != 100 {
		t.Fatalf("completed rounds = %+v", view.CompletedRounds)
	}
}

func TestTimeoutOnFinalRoundFinishesBattle(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 0, 70*time.Second)
	if _, err := l.SubmitGuess(ctx, users[0], roundIDFor(t, l, ctx, users[0], id), "Guest selection", 1); err != nil {
		t.Fatal(err)
	}
	if expired, err := l.ExpireStages(ctx); err != nil || expired != 4 {
		t.Fatalf("expired=%d err=%v", expired, err)
	}
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "FINISHED" || view.EndReason == nil || *view.EndReason != "COMPLETED" || view.Outcome == nil || *view.Outcome != "WIN" || view.MyScore != 100 {
		t.Fatalf("finished view = %+v", view)
	}
}

func TestIdleTimedRoundCancelsBattle(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 2, 70*time.Second, 70*time.Second)
	if expired, err := l.ExpireStages(ctx); err != nil || expired != 8 {
		t.Fatalf("expired=%d err=%v", expired, err)
	}
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "CANCELLED" || view.EndReason == nil || *view.EndReason != "TIMEOUT" || view.Outcome != nil || view.Round != nil {
		t.Fatalf("idle battle view = %+v", view)
	}
	for _, user := range users[:2] {
		if stats, err := l.Stats(ctx, user); err != nil || stats != (Stats{}) {
			t.Fatalf("stats after idle battle = %+v, %v", stats, err)
		}
	}
	if _, err := l.SubmitGuess(ctx, users[0], firstRoundID(t, l, ctx, id), "Guest selection", 1); !errors.Is(err, ErrTimeExpired) {
		t.Fatalf("guess after idle cancellation err = %v", err)
	}
}

func firstRoundID(t *testing.T, l *Lobby, ctx context.Context, battleID pgtype.UUID) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := l.pool.QueryRow(ctx, `SELECT id FROM rounds WHERE battle_id = $1 AND round_number = 1`, battleID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestGuessAfterSweptTimeoutReportsExpiredTime(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 17*time.Second, 70*time.Second)
	roundID := roundIDFor(t, l, ctx, users[0], id)
	if _, err := l.SubmitGuess(ctx, users[1], roundID, "wrong answer", 1); !errors.Is(err, ErrTimeExpired) {
		t.Fatalf("guest late guess err = %v", err)
	}
	if expired, err := l.ExpireStages(ctx); err != nil || expired != 1 {
		t.Fatalf("expired=%d err=%v", expired, err)
	}
	if _, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 1); !errors.Is(err, ErrTimeExpired) {
		t.Fatalf("host guess after sweep err = %v", err)
	}
	if _, err := l.SubmitGuess(ctx, users[1], roundID, "Host selection", 4); !errors.Is(err, ErrTimeExpired) {
		t.Fatalf("guest guess after round timed out err = %v", err)
	}
	if _, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 3); !errors.Is(err, ErrStaleAttempt) {
		t.Fatalf("future attempt err = %v", err)
	}
}

func TestBrokenBattleDoesNotBlockOtherTimeouts(t *testing.T) {
	l, ctx, users, broken := timedMatch(t, 1, 80*time.Second, 0)
	if _, err := l.pool.Exec(ctx, `UPDATE rounds SET host_stage = 'MELODY' WHERE battle_id = $1`, broken); err != nil {
		t.Fatal(err)
	}
	_, otherCtx, otherUsers, healthy := timedMatch(t, 1, 17*time.Second, 0)
	expired, err := l.ExpireStages(ctx)
	if err == nil || !strings.Contains(err.Error(), broken.String()) {
		t.Fatalf("sweep error = %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired %d stages, want the healthy battle's 1", expired)
	}
	if r := roundFor(t, l, otherCtx, otherUsers[0], healthy); r.MyAttempts != 1 {
		t.Fatalf("healthy battle attempts = %d", r.MyAttempts)
	}
	if r := roundFor(t, l, ctx, users[0], broken); r.MyAttempts != 0 {
		t.Fatalf("broken battle changed: %+v", r)
	}
}

func TestGuessWithinGraceIsNotSwept(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 15*time.Second+200*time.Millisecond, 0)
	roundID := roundIDFor(t, l, ctx, users[0], id)
	var wg sync.WaitGroup
	var result GuessResult
	var guessErr, sweepErr error
	var expired int
	wg.Go(func() { result, guessErr = l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 1) })
	wg.Go(func() { expired, sweepErr = l.ExpireStages(ctx) })
	wg.Wait()
	if guessErr != nil || sweepErr != nil || !result.Correct || expired != 0 {
		t.Fatalf("guess=%+v err=%v; sweep expired=%d err=%v", result, guessErr, expired, sweepErr)
	}
}

func TestLateGuessIsRejectedAfterRecordingTimeout(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 17*time.Second, 0)
	roundID := roundIDFor(t, l, ctx, users[0], id)
	if _, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 1); !errors.Is(err, ErrTimeExpired) {
		t.Fatalf("late guess err = %v", err)
	}
	if host := roundFor(t, l, ctx, users[0], id); host.MyAttempts != 1 || host.MyStage != "BASS" {
		t.Fatalf("host after late guess = %+v", host)
	}
	result, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 2)
	if err != nil || !result.Correct || result.PointsAwarded != 75 {
		t.Fatalf("bass guess = %+v, %v", result, err)
	}
}

func TestGuessWithinGraceIsAccepted(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 15*time.Second+300*time.Millisecond, 0)
	result, err := l.SubmitGuess(ctx, users[0], roundIDFor(t, l, ctx, users[0], id), "Guest selection", 1)
	if err != nil || !result.Correct || result.PointsAwarded != 100 {
		t.Fatalf("guess within grace = %+v, %v", result, err)
	}
}

func TestGuessResetsStemClock(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 10*time.Second, 0)
	if _, err := l.SubmitGuess(ctx, users[0], roundIDFor(t, l, ctx, users[0], id), "wrong answer", 1); err != nil {
		t.Fatal(err)
	}
	host := roundFor(t, l, ctx, users[0], id)
	if host.StageEndsAt == nil {
		t.Fatal("no deadline after wrong guess")
	}
	if remaining := time.Until(*host.StageEndsAt); remaining < 13*time.Second || remaining > 15*time.Second {
		t.Fatalf("bass deadline in %v, want a fresh 15s", remaining)
	}
}

func TestConcurrentStageSweepsExpireOnce(t *testing.T) {
	l, ctx, users, id := timedMatch(t, 1, 17*time.Second, 17*time.Second)
	var wg sync.WaitGroup
	totals := make([]int, 4)
	errs := make([]error, 4)
	for i := range totals {
		wg.Go(func() { totals[i], errs[i] = l.ExpireStages(ctx) })
	}
	wg.Wait()
	sum := 0
	for i, n := range totals {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		sum += n
	}
	if sum != 2 {
		t.Fatalf("concurrent sweeps expired %d stages, want 2", sum)
	}
	for _, user := range users[:2] {
		if r := roundFor(t, l, ctx, user, id); r.MyAttempts != 1 {
			t.Fatalf("attempts = %d, want 1", r.MyAttempts)
		}
	}
}

func TestUntimedBattleNeverExpires(t *testing.T) {
	l, users, id, _ := activeMatch(t, 1)
	ctx := t.Context()
	if _, err := l.pool.Exec(ctx, `UPDATE rounds SET host_stage_started_at = now() - interval '1 hour' WHERE battle_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if expired, err := l.ExpireStages(ctx); err != nil || expired != 0 {
		t.Fatalf("expired=%d err=%v", expired, err)
	}
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if view.GuessSeconds != nil || view.Round == nil || view.Round.StageEndsAt != nil {
		t.Fatalf("untimed view = %+v", view)
	}
}

func TestCreateStoresGuessTimer(t *testing.T) {
	l, ctx, u := lobbyFixture(t)
	id, err := l.Create(ctx, u[0], 1, 30)
	if err != nil {
		t.Fatal(err)
	}
	view, err := l.Get(ctx, u[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if view.GuessSeconds == nil || *view.GuessSeconds != 30 {
		t.Fatalf("guessSeconds = %v", view.GuessSeconds)
	}
}

func TestStatsCountOnlyFinishedBattles(t *testing.T) {
	l, users, id, _ := activeMatch(t, 1)
	ctx := t.Context()
	roundID := roundIDFor(t, l, ctx, users[0], id)
	if _, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 1); err != nil {
		t.Fatal(err)
	}
	for attempt := int32(1); attempt <= 4; attempt++ {
		if _, err := l.SubmitGuess(ctx, users[1], roundID, "wrong answer", attempt); err != nil {
			t.Fatal(err)
		}
	}
	cancelled, err := l.Create(ctx, users[0], 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Forfeit(ctx, users[0], cancelled); err != nil {
		t.Fatal(err)
	}
	host, err := l.Stats(ctx, users[0])
	if err != nil {
		t.Fatal(err)
	}
	if host != (Stats{Played: 1, Wins: 1, BestScore: 100, AverageScore: 100}) {
		t.Fatalf("host stats = %+v", host)
	}
	guest, err := l.Stats(ctx, users[1])
	if err != nil {
		t.Fatal(err)
	}
	if guest != (Stats{Played: 1, Losses: 1}) {
		t.Fatalf("guest stats = %+v", guest)
	}
	draw, drawUsers, drawID, _ := activeMatch(t, 1)
	drawRound := roundIDFor(t, draw, ctx, drawUsers[0], drawID)
	if _, err := draw.SubmitGuess(ctx, drawUsers[0], drawRound, "Guest selection", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := draw.SubmitGuess(ctx, drawUsers[1], drawRound, "Host selection", 1); err != nil {
		t.Fatal(err)
	}
	if stats, err := draw.Stats(ctx, drawUsers[1]); err != nil || stats != (Stats{Played: 1, Draws: 1, BestScore: 100, AverageScore: 100}) {
		t.Fatalf("draw stats = %+v, %v", stats, err)
	}
	if none, err := l.Stats(ctx, users[2]); err != nil || none != (Stats{}) {
		t.Fatalf("stats without battles = %+v, %v", none, err)
	}
}
