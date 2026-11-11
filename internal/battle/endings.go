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

const (
	lobbyTimeout = 15 * time.Minute
	gameTimeout  = 5 * time.Minute
)

func (l *Lobby) Forfeit(ctx context.Context, userID, battleID pgtype.UUID) (View, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return View{}, fmt.Errorf("begin forfeit: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	b, err := q.LockBattleForParticipant(ctx, dbgen.LockBattleForParticipantParams{ID: battleID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, ErrNotFound
	}
	if err != nil {
		return View{}, fmt.Errorf("lock battle for forfeit: %w", err)
	}
	end := dbgen.EndBattleParams{ID: b.ID, EndReason: "FORFEIT"}
	switch b.Status {
	case "PENDING":
		end.Status = "CANCELLED"
	case "SELECTING", "ONGOING":
		end.Status, end.WinnerID = "FINISHED", b.GuestID
		if userID == b.GuestID {
			end.WinnerID = b.HostID
		}
	default:
		return View{}, ErrBattleEnded
	}
	if err := endBattle(ctx, q, end); err != nil {
		return View{}, err
	}
	view, err := loadView(ctx, q, userID, battleID)
	if err != nil {
		return View{}, fmt.Errorf("read forfeited battle: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return View{}, fmt.Errorf("commit forfeit: %w", err)
	}
	return view, nil
}

func (l *Lobby) ExpireInactive(ctx context.Context) (int, error) {
	ended := 0
	for {
		done, err := l.expireNextInactive(ctx)
		if err != nil || done {
			return ended, err
		}
		ended++
	}
}

func (l *Lobby) expireNextInactive(ctx context.Context) (bool, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin battle expiry: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	b, err := q.LockNextInactiveBattle(ctx, dbgen.LockNextInactiveBattleParams{
		LobbyTimeoutSeconds: lobbyTimeout.Seconds(),
		GameTimeoutSeconds:  gameTimeout.Seconds(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock inactive battle: %w", err)
	}
	rounds, err := q.ListBattleRounds(ctx, dbgen.ListBattleRoundsParams{BattleID: b.ID, UserID: b.HostID})
	if err != nil {
		return false, fmt.Errorf("read inactive battle rounds: %w", err)
	}
	end := timeoutEnding(b, rounds)
	if err := endBattle(ctx, q, end); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit battle expiry: %w", err)
	}
	return false, nil
}

func timeoutEnding(b dbgen.Battle, rounds []dbgen.ListBattleRoundsRow) dbgen.EndBattleParams {
	end := dbgen.EndBattleParams{ID: b.ID, Status: "CANCELLED", EndReason: "TIMEOUT"}
	var hostReady, guestReady bool
	switch b.Status {
	case "SELECTING":
		hostReady, guestReady = true, true
		for _, r := range rounds {
			hostReady = hostReady && r.HostSongID.Valid
			guestReady = guestReady && r.GuestSongID.Valid
		}
	case "ONGOING":
		for _, r := range rounds {
			if r.RoundNumber == b.CurrentRound {
				hostReady, guestReady = r.HostFinished, r.GuestFinished
			}
		}
	}
	switch {
	case hostReady && !guestReady:
		end.Status, end.WinnerID = "FINISHED", b.HostID
	case guestReady && !hostReady:
		end.Status, end.WinnerID = "FINISHED", b.GuestID
	}
	return end
}

func endBattle(ctx context.Context, q *dbgen.Queries, end dbgen.EndBattleParams) error {
	count, err := q.EndBattle(ctx, end)
	if err != nil {
		return fmt.Errorf("end battle: %w", err)
	}
	if count != 1 {
		return errors.New("battle changed while ending")
	}
	return nil
}
