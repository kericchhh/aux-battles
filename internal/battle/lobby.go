package battle

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

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
	Me                   Player        `json:"me"`
	Opponent             *Player       `json:"opponent"`
	OpponentJoined       bool          `json:"opponentJoined"`
	MyLineupLocked       bool          `json:"myLineupLocked"`
	OpponentLineupLocked bool          `json:"opponentLineupLocked"`
	Outcome              *string       `json:"outcome"`
	EndReason            *string       `json:"endReason"`
	MyScore              int32         `json:"myScore"`
	OpponentScore        int32         `json:"opponentScore"`
	Round                *RoundView    `json:"round"`
	CompletedRounds      []RoundResult `json:"completedRounds"`
}

type Player struct {
	Username string `json:"username"`
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

type RoundResult struct {
	Number         int32 `json:"number"`
	MyPoints       int32 `json:"myPoints"`
	OpponentPoints int32 `json:"opponentPoints"`
	MySong         Song  `json:"mySong"`
	OpponentSong   Song  `json:"opponentSong"`
}

type Song struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

func (l *Lobby) Get(ctx context.Context, userID, id pgtype.UUID) (View, error) {
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return View{}, fmt.Errorf("begin battle view: %w", err)
	}
	defer tx.Rollback(ctx)
	view, err := loadView(ctx, dbgen.New(tx), userID, id)
	if err != nil {
		return View{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return View{}, fmt.Errorf("finish battle view: %w", err)
	}
	return view, nil
}

func loadView(ctx context.Context, q *dbgen.Queries, userID, id pgtype.UUID) (View, error) {
	b, err := q.GetBattleForParticipant(ctx, dbgen.GetBattleForParticipantParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, ErrNotFound
	}
	if err != nil {
		return View{}, fmt.Errorf("read battle: %w", err)
	}
	players, err := q.GetBattlePlayers(ctx, id)
	if err != nil {
		return View{}, fmt.Errorf("read battle players: %w", err)
	}
	rounds, err := q.ListBattleRounds(ctx, dbgen.ListBattleRoundsParams{BattleID: id, UserID: userID})
	if err != nil {
		return View{}, fmt.Errorf("read battle rounds: %w", err)
	}
	songs, err := q.ListFinishedRoundSongs(ctx, id)
	if err != nil {
		return View{}, fmt.Errorf("read finished round songs: %w", err)
	}
	return participantView(b, players, rounds, songs, userID), nil
}

func participantView(b dbgen.Battle, players dbgen.GetBattlePlayersRow, rounds []dbgen.ListBattleRoundsRow, songs []dbgen.ListFinishedRoundSongsRow, userID pgtype.UUID) View {
	isHost := b.HostID == userID
	v := View{ID: b.ID.String(), Status: b.Status, InviteCode: b.InviteCode, CurrentRound: b.CurrentRound, Rounds: b.RoundCount,
		Me: Player{Username: players.HostUsername}, OpponentJoined: b.GuestID.Valid,
		MyLineupLocked: len(rounds) == int(b.RoundCount), OpponentLineupLocked: len(rounds) == int(b.RoundCount),
		CompletedRounds: make([]RoundResult, 0, len(songs))}
	if players.GuestUsername.Valid {
		v.Opponent = &Player{Username: players.GuestUsername.String}
	}
	if !isHost {
		v.Me = Player{Username: players.GuestUsername.String}
		v.Opponent = &Player{Username: players.HostUsername}
	}
	revealed := make(map[int32]dbgen.ListFinishedRoundSongsRow, len(songs))
	for _, s := range songs {
		revealed[s.RoundNumber] = s
	}
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
		if r.Status == "FINISHED" || b.Status != "ONGOING" {
			v.MyScore += myPoints
			v.OpponentScore += opponentPoints
		}
		if b.Status == "ONGOING" && r.RoundNumber == b.CurrentRound {
			v.Round = &RoundView{ID: r.ID.String(), Number: r.RoundNumber, Status: r.Status, MyStage: myStage, MyPoints: myPoints, MyFinished: myFinished, OpponentFinished: opponentFinished, MyAttempts: r.MyAttempts}
		}
		if s, ok := revealed[r.RoundNumber]; ok && r.Status == "FINISHED" {
			result := RoundResult{Number: r.RoundNumber, MyPoints: myPoints, OpponentPoints: opponentPoints,
				MySong:       Song{Title: s.HostSongTitle, Artist: s.HostSongArtist},
				OpponentSong: Song{Title: s.GuestSongTitle, Artist: s.GuestSongArtist}}
			if !isHost {
				result.MySong, result.OpponentSong = result.OpponentSong, result.MySong
			}
			v.CompletedRounds = append(v.CompletedRounds, result)
		}
	}
	v.Outcome = outcome(b.Status, b.WinnerID, userID)
	if b.EndReason.Valid {
		v.EndReason = &b.EndReason.String
	}
	return v
}

func outcome(status string, winnerID, userID pgtype.UUID) *string {
	if status != "FINISHED" {
		return nil
	}
	result := "DRAW"
	if winnerID.Valid {
		result = "LOSS"
		if winnerID == userID {
			result = "WIN"
		}
	}
	return &result
}

const maxListedBattles = 20

type Summary struct {
	ID             string    `json:"id"`
	Status         string    `json:"status"`
	Rounds         int32     `json:"rounds"`
	CurrentRound   int32     `json:"currentRound"`
	Opponent       *Player   `json:"opponent"`
	InviteCode     *string   `json:"inviteCode"`
	MyScore        int32     `json:"myScore"`
	OpponentScore  int32     `json:"opponentScore"`
	Outcome        *string   `json:"outcome"`
	EndReason      *string   `json:"endReason"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

func (l *Lobby) List(ctx context.Context, userID pgtype.UUID) ([]Summary, error) {
	rows, err := dbgen.New(l.pool).ListBattlesForUser(ctx, dbgen.ListBattlesForUserParams{UserID: userID, MaxBattles: maxListedBattles})
	if err != nil {
		return nil, fmt.Errorf("list battles: %w", err)
	}
	battles := make([]Summary, 0, len(rows))
	for _, r := range rows {
		s := Summary{ID: r.ID.String(), Status: r.Status, Rounds: r.RoundCount, CurrentRound: r.CurrentRound,
			MyScore: r.MyScore, OpponentScore: r.OpponentScore, Outcome: outcome(r.Status, r.WinnerID, userID),
			LastActivityAt: r.LastActivityAt.Time}
		if r.OpponentUsername.Valid {
			s.Opponent = &Player{Username: r.OpponentUsername.String}
		}
		if r.Status == "PENDING" && r.HostID == userID {
			s.InviteCode = &r.InviteCode
		}
		if r.EndReason.Valid {
			s.EndReason = &r.EndReason.String
		}
		battles = append(battles, s)
	}
	return battles, nil
}
