package battle

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
)

func TestLobbyRejectsInvalidInputBeforeDatabaseAccess(t *testing.T) {
	l := New(nil, media.New(""))
	for _, n := range []int32{-1, 0, 8, 100} {
		if _, err := l.Create(context.Background(), pgtype.UUID{}, n); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("rounds %d: %v", n, err)
		}
	}
	for _, code := range []string{"", "ABC", "ABCDEFG", "ABC-12", "ＡＢＣ１２３"} {
		if _, err := l.Join(context.Background(), pgtype.UUID{}, code); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("code %q: %v", code, err)
		}
	}
}

func TestParticipantViewUsesPlayerPerspective(t *testing.T) {
	var host, guest, song pgtype.UUID
	if err := host.Scan("00000000-0000-4000-8000-000000000001"); err != nil {
		t.Fatal(err)
	}
	if err := guest.Scan("00000000-0000-4000-8000-000000000002"); err != nil {
		t.Fatal(err)
	}
	if err := song.Scan("00000000-0000-4000-8000-000000000003"); err != nil {
		t.Fatal(err)
	}
	b := dbgen.Battle{ID: host, HostID: host, GuestID: guest, RoundCount: 1, CurrentRound: 1, Status: "ONGOING"}
	rows := []dbgen.ListBattleRoundsRow{{ID: song, RoundNumber: 1, HostSongID: song, GuestSongID: song, HostPoints: 100, GuestPoints: 50, HostStage: "DRUM", GuestStage: "MELODY", HostFinished: true, GuestFinished: false, Status: "GUESSING", MyAttempts: 3}}
	view := participantView(b, rows, guest)
	if view.MyScore != 50 || view.OpponentScore != 100 || !view.MyLineupLocked || !view.OpponentLineupLocked || view.Round == nil {
		t.Fatalf("guest view=%+v", view)
	}
	if view.Round.MyStage != "MELODY" || view.Round.MyFinished || !view.Round.OpponentFinished || view.Round.MyAttempts != 3 {
		t.Fatalf("guest round=%+v", view.Round)
	}
	b.Status = "FINISHED"
	rows[0].Status = "FINISHED"
	for _, tc := range []struct {
		winner pgtype.UUID
		want   string
	}{{host, "LOSS"}, {guest, "WIN"}, {pgtype.UUID{}, "DRAW"}} {
		b.WinnerID = tc.winner
		view = participantView(b, rows, guest)
		if view.Outcome == nil || *view.Outcome != tc.want || view.Round != nil || view.PreviousRound == nil || view.PreviousRound.MyPoints != 50 {
			t.Fatalf("finished guest view=%+v", view)
		}
	}
}
