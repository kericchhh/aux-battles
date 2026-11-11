package battle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

const guessGrace = time.Second

func stageDeadline(b dbgen.Battle, s side) (time.Time, bool) {
	if !b.GuessSeconds.Valid || s.finished || !s.startedAt.Valid {
		return time.Time{}, false
	}
	return s.startedAt.Time.Add(time.Duration(b.GuessSeconds.Int32) * time.Second), true
}

func expireStages(ctx context.Context, q *dbgen.Queries, b dbgen.Battle, r dbgen.Round, isHost bool, now time.Time) (expired int, roundFinished bool, err error) {
	for {
		deadline, ok := stageDeadline(b, playerSide(r, isHost))
		if !ok || now.Before(deadline.Add(guessGrace)) {
			return expired, false, nil
		}
		userID := b.GuestID
		if isHost {
			userID = b.HostID
		}
		attempts, err := q.CountRoundGuesses(ctx, dbgen.CountRoundGuessesParams{RoundID: r.ID, UserID: userID})
		if err != nil {
			return expired, false, fmt.Errorf("count round guesses: %w", err)
		}
		if attempts >= maxAttempts {
			return expired, false, errors.New("round has inconsistent player progress")
		}
		result, updated, err := recordAttempt(ctx, q, b, r, isHost, attemptInput{number: attempts + 1, timedOut: true}, deadline)
		if err != nil {
			return expired, false, err
		}
		expired++
		if result.RoundFinished {
			return expired, true, nil
		}
		r = updated
	}
}

func (l *Lobby) ExpireStages(ctx context.Context) (int, error) {
	expired := 0
	var skip []pgtype.UUID
	var failures []error
	for {
		battleID, n, err := l.expireNextStage(ctx, skip)
		expired += n
		if !battleID.Valid {
			return expired, errors.Join(append(failures, err)...)
		}
		skip = append(skip, battleID)
		if err != nil {
			failures = append(failures, fmt.Errorf("battle %s: %w", battleID, err))
		}
	}
}

func (l *Lobby) expireNextStage(ctx context.Context, skip []pgtype.UUID) (pgtype.UUID, int, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return pgtype.UUID{}, 0, fmt.Errorf("begin stage expiry: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	row, err := q.LockNextExpiredStage(ctx, dbgen.LockNextExpiredStageParams{SkipIds: skip, GraceSeconds: guessGrace.Seconds()})
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, 0, nil
	}
	if err != nil {
		return pgtype.UUID{}, 0, fmt.Errorf("lock expired stage: %w", err)
	}
	id := row.Battle.ID
	expired, err := expireLockedRound(ctx, q, row)
	if err != nil {
		return id, 0, err
	}
	if expired == 0 {
		return id, 0, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return id, 0, fmt.Errorf("commit stage expiry: %w", err)
	}
	return id, expired, nil
}

func expireLockedRound(ctx context.Context, q *dbgen.Queries, row dbgen.LockNextExpiredStageRow) (int, error) {
	now, err := q.CurrentTime(ctx)
	if err != nil {
		return 0, fmt.Errorf("read database time: %w", err)
	}
	expired := 0
	for _, isHost := range []bool{true, false} {
		r, err := q.LockRoundForBattle(ctx, dbgen.LockRoundForBattleParams{ID: row.RoundID, BattleID: row.Battle.ID})
		if err != nil {
			return 0, fmt.Errorf("lock expired round: %w", err)
		}
		n, finished, err := expireStages(ctx, q, row.Battle, r, isHost, now.Time)
		if err != nil {
			return 0, err
		}
		expired += n
		if finished {
			break
		}
	}
	return expired, nil
}
