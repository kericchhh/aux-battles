package battle

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
)

const MaxRounds = 7

type Lobby struct {
	pool  *pgxpool.Pool
	media media.Store
}

func New(pool *pgxpool.Pool, store media.Store) *Lobby { return &Lobby{pool: pool, media: store} }

func (l *Lobby) Create(ctx context.Context, userID pgtype.UUID, rounds int32) (pgtype.UUID, error) {
	return l.create(ctx, userID, rounds, newInviteCode)
}

func (l *Lobby) create(ctx context.Context, userID pgtype.UUID, rounds int32, generate func() (string, error)) (pgtype.UUID, error) {
	if rounds < 1 || rounds > MaxRounds {
		return pgtype.UUID{}, invalidInput("rounds must be between 1 and 7")
	}
	q := dbgen.New(l.pool)
	for range 5 {
		code, err := generate()
		if err != nil {
			return pgtype.UUID{}, fmt.Errorf("generate battle invite: %w", err)
		}
		id, err := q.CreateBattle(ctx, dbgen.CreateBattleParams{HostID: userID, InviteCode: code, RoundCount: rounds})
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return pgtype.UUID{}, fmt.Errorf("create battle: %w", err)
		}
		return id, nil
	}
	return pgtype.UUID{}, ErrInviteExhausted
}

func newInviteCode() (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var code [6]byte
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		code[i] = alphabet[n.Int64()]
	}
	return string(code[:]), nil
}

func (l *Lobby) Join(ctx context.Context, userID pgtype.UUID, code string) (pgtype.UUID, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 6 || strings.IndexFunc(code, func(r rune) bool { return !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') }) >= 0 {
		return pgtype.UUID{}, invalidInput("invite code must contain six letters or digits")
	}
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("begin battle join: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	id, err := q.GetBattleByInvite(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, ErrNotFound
	}
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("find battle invite: %w", err)
	}
	joined, err := q.JoinBattle(ctx, dbgen.JoinBattleParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, ErrUnavailable
	}
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("claim battle guest slot: %w", err)
	}
	count, err := q.CreateBattleRounds(ctx, dbgen.CreateBattleRoundsParams{BattleID: id, RoundCount: joined.RoundCount})
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("create battle rounds: %w", err)
	}
	if count != int64(joined.RoundCount) {
		return pgtype.UUID{}, fmt.Errorf("created %d battle rounds, expected %d", count, joined.RoundCount)
	}
	if err := tx.Commit(ctx); err != nil {
		return pgtype.UUID{}, fmt.Errorf("commit battle join: %w", err)
	}
	return id, nil
}

type View struct {
	ID                   string        `json:"id"`
	Status               string        `json:"status"`
	InviteCode           string        `json:"inviteCode"`
	CurrentRound         int32         `json:"currentRound"`
	Rounds               int32         `json:"rounds"`
	OpponentJoined       bool          `json:"opponentJoined"`
	MyLineupLocked       bool          `json:"myLineupLocked"`
	OpponentLineupLocked bool          `json:"opponentLineupLocked"`
	Outcome              *string       `json:"outcome"`
	MyScore              int32         `json:"myScore"`
	OpponentScore        int32         `json:"opponentScore"`
	Round                *RoundView    `json:"round"`
	PreviousRound        *RoundSummary `json:"previousRound"`
}

type RoundView struct {
	ID               string `json:"id"`
	Number           int32  `json:"number"`
	Status           string `json:"status"`
	MyStage          string `json:"myStage"`
	MyPoints         int32  `json:"myPoints"`
	MyFinished       bool   `json:"myFinished"`
	OpponentFinished bool   `json:"opponentFinished"`
	MyAttempts       int32  `json:"myAttempts"`
}

type RoundSummary struct {
	Number         int32 `json:"number"`
	MyPoints       int32 `json:"myPoints"`
	OpponentPoints int32 `json:"opponentPoints"`
}

func (l *Lobby) Get(ctx context.Context, userID, id pgtype.UUID) (View, error) {
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return View{}, fmt.Errorf("begin battle view: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	b, err := q.GetBattleForParticipant(ctx, dbgen.GetBattleForParticipantParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, ErrNotFound
	}
	if err != nil {
		return View{}, fmt.Errorf("read battle: %w", err)
	}
	rounds, err := q.ListBattleRounds(ctx, dbgen.ListBattleRoundsParams{BattleID: id, UserID: userID})
	if err != nil {
		return View{}, fmt.Errorf("read battle rounds: %w", err)
	}
	view := participantView(b, rounds, userID)
	if err := tx.Commit(ctx); err != nil {
		return View{}, fmt.Errorf("finish battle view: %w", err)
	}
	return view, nil
}

func participantView(b dbgen.Battle, rounds []dbgen.ListBattleRoundsRow, userID pgtype.UUID) View {
	isHost := b.HostID == userID
	v := View{ID: b.ID.String(), Status: b.Status, InviteCode: b.InviteCode, CurrentRound: b.CurrentRound, Rounds: b.RoundCount,
		OpponentJoined: b.GuestID.Valid, MyLineupLocked: len(rounds) == int(b.RoundCount), OpponentLineupLocked: len(rounds) == int(b.RoundCount)}
	for _, r := range rounds {
		mySong, opponentSong := r.HostSongID, r.GuestSongID
		myPoints, opponentPoints := r.HostPoints, r.GuestPoints
		myStage, myFinished, opponentFinished := r.HostStage, r.HostFinished, r.GuestFinished
		if !isHost {
			mySong, opponentSong = opponentSong, mySong
			myPoints, opponentPoints = opponentPoints, myPoints
			myStage, myFinished, opponentFinished = r.GuestStage, r.GuestFinished, r.HostFinished
		}
		v.MyLineupLocked = v.MyLineupLocked && mySong.Valid
		v.OpponentLineupLocked = v.OpponentLineupLocked && opponentSong.Valid
		v.MyScore += myPoints
		v.OpponentScore += opponentPoints
		if b.Status == "ONGOING" && r.RoundNumber == b.CurrentRound {
			v.Round = &RoundView{ID: r.ID.String(), Number: r.RoundNumber, Status: r.Status, MyStage: myStage, MyPoints: myPoints, MyFinished: myFinished, OpponentFinished: opponentFinished, MyAttempts: r.MyAttempts}
		}
		if r.Status == "FINISHED" {
			v.PreviousRound = &RoundSummary{Number: r.RoundNumber, MyPoints: myPoints, OpponentPoints: opponentPoints}
		}
	}
	if b.Status == "FINISHED" {
		result := "DRAW"
		if b.WinnerID.Valid {
			result = "LOSS"
			if b.WinnerID == userID {
				result = "WIN"
			}
		}
		v.Outcome = &result
	}
	return v
}
