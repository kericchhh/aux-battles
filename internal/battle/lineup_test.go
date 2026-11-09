package battle

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/media"
)

func TestLineupRejectsInvalidChoicesBeforeDatabaseAccess(t *testing.T) {
	l := New(nil, media.New(""))
	id := pgtype.UUID{Valid: true}
	for _, ids := range [][]pgtype.UUID{nil, {}, {{}}, {id, id}, make([]pgtype.UUID, 8)} {
		if _, err := l.SubmitLineup(context.Background(), id, id, ids); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("choices=%v error=%v", ids, err)
		}
	}
}
