package battle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

var roundStages = [...]struct {
	name   string
	points int32
}{{"DRUM", 100}, {"BASS", 75}, {"MELODY", 50}, {"FULL", 20}}

const (
	maxAttempts   = int32(len(roundStages))
	maxGuessRunes = 255
)

type GuessResult struct {
	BattleID       string `json:"battleId"`
	RoundID        string `json:"roundId"`
	Attempt        int32  `json:"attempt"`
	Correct        bool   `json:"correct"`
	Result         string `json:"result"`
	BasePoints     int32  `json:"basePoints"`
	StreakBonus    int32  `json:"streakBonus"`
	PointsAwarded  int32  `json:"pointsAwarded"`
	Stage          string `json:"stage"`
	PlayerFinished bool   `json:"playerFinished"`
	RoundFinished  bool   `json:"roundFinished"`
	BattleStatus   string `json:"battleStatus"`
}

func (l *Lobby) SubmitGuess(ctx context.Context, userID, roundID pgtype.UUID, guess string, expectedAttempt int32) (GuessResult, error) {
	guess = strings.TrimSpace(guess)
	if !userID.Valid || !roundID.Valid || expectedAttempt < 1 || expectedAttempt > maxAttempts || utf8.RuneCountInString(guess) > maxGuessRunes || normalizeTitle(guess) == "" {
		return GuessResult{}, invalidInput("provide a song title and an expectedAttempt between 1 and 4")
	}
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return GuessResult{}, fmt.Errorf("begin guess: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	b, err := q.LockRoundBattleForParticipant(ctx, dbgen.LockRoundBattleForParticipantParams{RoundID: roundID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return GuessResult{}, ErrNotFound
	}
	if err != nil {
		return GuessResult{}, fmt.Errorf("lock battle for guess: %w", err)
	}
	r, err := q.LockRoundForBattle(ctx, dbgen.LockRoundForBattleParams{ID: roundID, BattleID: b.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return GuessResult{}, ErrNotFound
	}
	if err != nil {
		return GuessResult{}, fmt.Errorf("lock guess round: %w", err)
	}
	isHost := b.HostID == userID
	if b.Status != "ONGOING" || r.Status != "GUESSING" || r.RoundNumber != b.CurrentRound || playerSide(r, isHost).finished {
		if err := timeExpiredError(ctx, q, roundID, userID, expectedAttempt); err != nil {
			return GuessResult{}, err
		}
		return GuessResult{}, ErrRoundUnavailable
	}
	now, err := q.CurrentTime(ctx)
	if err != nil {
		return GuessResult{}, fmt.Errorf("read database time: %w", err)
	}
	expired, _, err := expireStages(ctx, q, b, r, isHost, now.Time)
	if err != nil {
		return GuessResult{}, err
	}
	if expired > 0 {
		if err := tx.Commit(ctx); err != nil {
			return GuessResult{}, fmt.Errorf("commit expired stages: %w", err)
		}
		return GuessResult{}, ErrTimeExpired
	}
	attempts, err := q.CountRoundGuesses(ctx, dbgen.CountRoundGuessesParams{RoundID: roundID, UserID: userID})
	if err != nil {
		return GuessResult{}, fmt.Errorf("count round guesses: %w", err)
	}
	attempt := attempts + 1
	if attempt > maxAttempts || expectedAttempt != attempt {
		if err := timeExpiredError(ctx, q, roundID, userID, expectedAttempt); err != nil {
			return GuessResult{}, err
		}
		return GuessResult{}, ErrStaleAttempt
	}
	targetID := playerSide(r, isHost).target
	if !targetID.Valid {
		return GuessResult{}, errors.New("round has inconsistent player progress")
	}
	title, err := q.GetRoundAnswer(ctx, targetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return GuessResult{}, ErrRoundAudioUnavailable
	}
	if err != nil {
		return GuessResult{}, fmt.Errorf("read round answer: %w", err)
	}
	history, err := q.ListBattleRounds(ctx, dbgen.ListBattleRoundsParams{BattleID: b.ID, UserID: userID})
	if err != nil {
		return GuessResult{}, fmt.Errorf("read battle progress: %w", err)
	}
	if len(history) != int(b.RoundCount) {
		return GuessResult{}, errors.New("battle has an incomplete round set")
	}
	a := attemptInput{number: attempt, guess: guess, correct: matchesTitle(guess, title)}
	if a.correct {
		a.streakBonus = streakBonus(history, r.RoundNumber, isHost)
	}
	if err := touchBattle(ctx, q, b.ID); err != nil {
		return GuessResult{}, err
	}
	result, _, err := recordAttempt(ctx, q, b, r, isHost, a, now.Time)
	if err != nil {
		return GuessResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GuessResult{}, fmt.Errorf("commit guess: %w", err)
	}
	return result, nil
}

type side struct {
	stage     string
	finished  bool
	target    pgtype.UUID
	startedAt pgtype.Timestamptz
}

func playerSide(r dbgen.Round, isHost bool) side {
	if isHost {
		return side{r.HostStage, r.HostFinished, r.GuestSongID, r.HostStageStartedAt}
	}
	return side{r.GuestStage, r.GuestFinished, r.HostSongID, r.GuestStageStartedAt}
}

type attemptInput struct {
	number      int32
	guess       string
	correct     bool
	timedOut    bool
	streakBonus int32
}

func recordAttempt(ctx context.Context, q *dbgen.Queries, b dbgen.Battle, r dbgen.Round, isHost bool, a attemptInput, nextStageStart time.Time) (GuessResult, dbgen.Round, error) {
	stage := playerSide(r, isHost).stage
	if stage != roundStages[a.number-1].name {
		return GuessResult{}, r, errors.New("round has inconsistent player progress")
	}
	userID := b.GuestID
	if isHost {
		userID = b.HostID
	}
	result := GuessResult{BattleID: b.ID.String(), RoundID: r.ID.String(), Attempt: a.number, Correct: a.correct,
		Result: "INCORRECT", Stage: stage, BattleStatus: b.Status, PlayerFinished: a.correct || a.number == maxAttempts}
	if a.timedOut {
		result.Result = "TIMED_OUT"
	}
	if a.correct {
		result.Result = "CORRECT"
		result.BasePoints = roundStages[a.number-1].points
		result.StreakBonus = a.streakBonus
		result.PointsAwarded = result.BasePoints + result.StreakBonus
	}
	if !result.PlayerFinished {
		result.Stage = roundStages[a.number].name
	}
	count, err := q.InsertRoundGuess(ctx, dbgen.InsertRoundGuessParams{RoundID: r.ID, UserID: userID, Attempt: a.number, Guess: a.guess,
		Correct: a.correct, PointsAwarded: result.PointsAwarded, TimedOut: a.timedOut})
	if err != nil {
		return GuessResult{}, r, fmt.Errorf("save round guess: %w", err)
	}
	if count != 1 {
		return GuessResult{}, r, errors.New("guess was not saved")
	}
	updated, err := q.UpdateRoundPlayerProgress(ctx, dbgen.UpdateRoundPlayerProgressParams{ID: r.ID, IsHost: isHost, Stage: result.Stage,
		Points: result.PointsAwarded, Finished: result.PlayerFinished, StageStartedAt: pgtype.Timestamptz{Time: nextStageStart, Valid: true}})
	if err != nil {
		return GuessResult{}, r, fmt.Errorf("save player progress: %w", err)
	}
	result.RoundFinished = updated.HostFinished && updated.GuestFinished
	if result.RoundFinished {
		count, err := q.FinishRound(ctx, r.ID)
		if err != nil {
			return GuessResult{}, r, fmt.Errorf("finish round: %w", err)
		}
		if count != 1 {
			return GuessResult{}, r, errors.New("round could not finish")
		}
		abandoned, err := roundAbandoned(ctx, q, r.ID, a)
		if err != nil {
			return GuessResult{}, r, err
		}
		if abandoned {
			if err := endBattle(ctx, q, dbgen.EndBattleParams{ID: b.ID, Status: "CANCELLED", EndReason: "TIMEOUT"}); err != nil {
				return GuessResult{}, r, err
			}
			result.BattleStatus = "CANCELLED"
			return result, updated, nil
		}
		result.BattleStatus, err = advanceBattle(ctx, q, b)
		if err != nil {
			return GuessResult{}, r, err
		}
	}
	return result, updated, nil
}

func roundAbandoned(ctx context.Context, q *dbgen.Queries, roundID pgtype.UUID, last attemptInput) (bool, error) {
	if !last.timedOut {
		return false, nil
	}
	played, err := q.CountPlayedRoundGuesses(ctx, roundID)
	if err != nil {
		return false, fmt.Errorf("count played round guesses: %w", err)
	}
	return played == 0, nil
}

func timeExpiredError(ctx context.Context, q *dbgen.Queries, roundID, userID pgtype.UUID, attempt int32) error {
	timedOut, err := q.GuessTimedOut(ctx, dbgen.GuessTimedOutParams{RoundID: roundID, UserID: userID, Attempt: attempt})
	if err != nil {
		return fmt.Errorf("read timed out attempt: %w", err)
	}
	if timedOut {
		return ErrTimeExpired
	}
	return nil
}

func streakBonus(rounds []dbgen.ListBattleRoundsRow, current int32, isHost bool) int32 {
	var streak int32
	for i := len(rounds) - 1; i >= 0; i-- {
		r := rounds[i]
		if r.RoundNumber >= current {
			continue
		}
		points := r.HostPoints
		if !isHost {
			points = r.GuestPoints
		}
		if r.Status != "FINISHED" || points == 0 {
			break
		}
		streak++
		if streak == 2 {
			break
		}
	}
	return streak * 10
}

func advanceBattle(ctx context.Context, q *dbgen.Queries, b dbgen.Battle) (string, error) {
	if b.CurrentRound < b.RoundCount {
		count, err := q.ActivateRound(ctx, dbgen.ActivateRoundParams{BattleID: b.ID, RoundNumber: b.CurrentRound + 1})
		if err != nil {
			return "", fmt.Errorf("activate next round: %w", err)
		}
		if count != 1 {
			return "", errors.New("next round could not be activated")
		}
		count, err = q.AdvanceBattle(ctx, dbgen.AdvanceBattleParams{ID: b.ID, CurrentRound: b.CurrentRound})
		if err != nil {
			return "", fmt.Errorf("advance battle: %w", err)
		}
		if count != 1 {
			return "", errors.New("battle could not advance")
		}
		return "ONGOING", nil
	}
	totals, err := q.GetBattleTotals(ctx, b.ID)
	if err != nil {
		return "", fmt.Errorf("read final scores: %w", err)
	}
	if totals.Rounds != b.RoundCount || totals.FinishedRounds != totals.Rounds || !b.GuestID.Valid {
		return "", errors.New("battle has incomplete final scores")
	}
	var winner pgtype.UUID
	if totals.HostTotal > totals.GuestTotal {
		winner = b.HostID
	} else if totals.GuestTotal > totals.HostTotal {
		winner = b.GuestID
	}
	count, err := q.FinishBattle(ctx, dbgen.FinishBattleParams{ID: b.ID, WinnerID: winner})
	if err != nil {
		return "", fmt.Errorf("finish battle: %w", err)
	}
	if count != 1 {
		return "", errors.New("battle could not finish")
	}
	return "FINISHED", nil
}

func touchBattle(ctx context.Context, q *dbgen.Queries, id pgtype.UUID) error {
	count, err := q.TouchBattle(ctx, id)
	if err != nil {
		return fmt.Errorf("record battle activity: %w", err)
	}
	if count != 1 {
		return errors.New("battle disappeared while recording activity")
	}
	return nil
}
