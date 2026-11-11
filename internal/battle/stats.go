package battle

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

type Stats struct {
	Played       int32 `json:"played"`
	Wins         int32 `json:"wins"`
	Losses       int32 `json:"losses"`
	Draws        int32 `json:"draws"`
	BestScore    int32 `json:"bestScore"`
	AverageScore int32 `json:"averageScore"`
}

func (l *Lobby) Stats(ctx context.Context, userID pgtype.UUID) (Stats, error) {
	row, err := dbgen.New(l.pool).GetPlayerStats(ctx, userID)
	if err != nil {
		return Stats{}, fmt.Errorf("read player stats: %w", err)
	}
	return Stats{Played: row.Played, Wins: row.Wins, Losses: row.Losses, Draws: row.Draws,
		BestScore: row.BestScore, AverageScore: row.AverageScore}, nil
}
