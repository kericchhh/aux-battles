package battle

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func (l *Lobby) OpenRoundAudio(ctx context.Context, userID, roundID pgtype.UUID) (*os.File, os.FileInfo, error) {
	audio, err := dbgen.New(l.pool).GetRoundAudioForParticipant(ctx, dbgen.GetRoundAudioForParticipantParams{UserID: userID, RoundID: roundID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read round audio: %w", err)
	}
	if audio.BattleStatus != "ONGOING" || audio.RoundStatus != "GUESSING" || audio.CurrentRound != audio.RoundNumber {
		return nil, nil, ErrRoundUnavailable
	}
	if audio.SongStatus.String != "READY" || audio.AudioPath == "" {
		return nil, nil, ErrRoundAudioUnavailable
	}
	file, info, err := l.media.OpenAudio(audio.AudioPath)
	if err != nil {
		return nil, nil, withCause(ErrRoundAudioUnavailable, err)
	}
	return file, info, nil
}
