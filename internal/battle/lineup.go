package battle

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

type LineupResult struct {
	BattleID string `json:"battleId"`
	Status   string `json:"status"`
}

func (l *Lobby) SubmitLineup(ctx context.Context, userID, battleID pgtype.UUID, songIDs []pgtype.UUID) (LineupResult, error) {
	if len(songIDs) < 1 || len(songIDs) > MaxRounds {
		return LineupResult{}, invalidInput("choose between 1 and 7 songs")
	}
	seen := make(map[pgtype.UUID]bool, len(songIDs))
	for _, id := range songIDs {
		if !id.Valid {
			return LineupResult{}, invalidInput("invalid song ID")
		}
		if seen[id] {
			return LineupResult{}, invalidInput("each round must use a different song")
		}
		seen[id] = true
	}
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return LineupResult{}, fmt.Errorf("begin lineup submission: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	b, err := q.LockBattleForParticipant(ctx, dbgen.LockBattleForParticipantParams{ID: battleID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return LineupResult{}, ErrNotFound
	}
	if err != nil {
		return LineupResult{}, fmt.Errorf("lock battle for lineup: %w", err)
	}
	if b.Status != "SELECTING" {
		return LineupResult{}, ErrLineupConflict
	}
	if len(songIDs) != int(b.RoundCount) {
		return LineupResult{}, invalidInput(fmt.Sprintf("choose exactly %d songs", b.RoundCount))
	}
	rounds, err := q.ListBattleRounds(ctx, dbgen.ListBattleRoundsParams{BattleID: battleID, UserID: userID})
	if err != nil {
		return LineupResult{}, fmt.Errorf("read lineup rounds: %w", err)
	}
	if len(rounds) != int(b.RoundCount) {
		return LineupResult{}, errors.New("battle has an incomplete round set")
	}
	isHost := b.HostID == userID
	ownCount, opponentCount := 0, 0
	for i, r := range rounds {
		if r.RoundNumber != int32(i+1) || r.Status != "WAITING" {
			return LineupResult{}, errors.New("battle has inconsistent lineup rounds")
		}
		own, opponent := r.HostSongID, r.GuestSongID
		if !isHost {
			own, opponent = opponent, own
		}
		if own.Valid {
			ownCount++
		}
		if opponent.Valid {
			opponentCount++
		}
	}
	if ownCount == len(rounds) {
		return LineupResult{}, ErrLineupConflict
	}
	if ownCount != 0 || (opponentCount != 0 && opponentCount != len(rounds)) {
		return LineupResult{}, errors.New("battle contains a partially saved lineup")
	}
	songs, err := q.GetLineupSongs(ctx, songIDs)
	if err != nil {
		return LineupResult{}, fmt.Errorf("read selected songs: %w", err)
	}
	if len(songs) != len(songIDs) {
		return LineupResult{}, ErrSongNotFound
	}
	for _, song := range songs {
		if normalizeTitle(song.Title) == "" {
			return LineupResult{}, invalidInput("a selected song has an unmatchable title")
		}
		if song.Status != "READY" {
			return LineupResult{}, ErrSongNotReady
		}
		if err := ctx.Err(); err != nil {
			return LineupResult{}, err
		}
		for _, path := range []pgtype.Text{song.FullSongPath, song.DrumsPath, song.BassPath, song.MelodyPath, song.VocalsPath} {
			if !path.Valid || path.String == "" {
				return LineupResult{}, withCause(ErrAudioUnavailable, fmt.Errorf("song %s has an empty audio path", song.ID.String()))
			}
			file, _, err := l.media.OpenAudio(path.String)
			if err != nil {
				return LineupResult{}, withCause(ErrAudioUnavailable, fmt.Errorf("song %s: %w", song.ID.String(), err))
			}
			if err := file.Close(); err != nil {
				return LineupResult{}, fmt.Errorf("close selected song audio: %w", err)
			}
		}
	}
	for i, r := range rounds {
		var count int64
		if isHost {
			count, err = q.SetHostRoundSong(ctx, dbgen.SetHostRoundSongParams{ID: r.ID, SongID: songIDs[i]})
		} else {
			count, err = q.SetGuestRoundSong(ctx, dbgen.SetGuestRoundSongParams{ID: r.ID, SongID: songIDs[i]})
		}
		if err != nil {
			return LineupResult{}, fmt.Errorf("save lineup song: %w", err)
		}
		if count != 1 {
			return LineupResult{}, errors.New("round changed while saving lineup")
		}
	}
	status := "WAITING_ON_OPPONENT"
	if opponentCount == len(rounds) {
		count, err := q.ActivateFirstRound(ctx, battleID)
		if err != nil {
			return LineupResult{}, fmt.Errorf("activate first round: %w", err)
		}
		if count != 1 {
			return LineupResult{}, errors.New("first round could not be activated")
		}
		count, err = q.StartBattle(ctx, battleID)
		if err != nil {
			return LineupResult{}, fmt.Errorf("start battle: %w", err)
		}
		if count != 1 {
			return LineupResult{}, errors.New("battle changed while starting")
		}
		status = "BATTLE_STARTED"
	}
	if err := tx.Commit(ctx); err != nil {
		return LineupResult{}, fmt.Errorf("commit lineup: %w", err)
	}
	return LineupResult{BattleID: battleID.String(), Status: status}, nil
}
