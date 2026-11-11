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
		if _, err := l.Create(context.Background(), pgtype.UUID{}, n, 0); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("rounds %d: %v", n, err)
		}
	}
	for _, seconds := range []int32{-15, 1, 10, 20, 60} {
		if _, err := l.Create(context.Background(), pgtype.UUID{}, 3, seconds); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("guessSeconds %d: %v", seconds, err)
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
	players := dbgen.GetBattlePlayersRow{HostUsername: "hana", GuestUsername: pgtype.Text{String: "gus", Valid: true}}
	rows := []dbgen.ListBattleRoundsRow{{ID: song, RoundNumber: 1, HostSongID: song, GuestSongID: song, HostPoints: 100, GuestPoints: 50, HostStage: "DRUM", GuestStage: "MELODY", HostFinished: true, GuestFinished: false, Status: "GUESSING", MyAttempts: 3}}
	songs := []dbgen.ListFinishedRoundSongsRow{{RoundNumber: 1, HostSongTitle: "Host pick", HostSongArtist: "H", GuestSongTitle: "Guest pick", GuestSongArtist: "G"}}
	view := participantView(b, players, rows, songs, guest)
	if view.MyScore != 0 || view.OpponentScore != 0 || !view.MyLineupLocked || !view.OpponentLineupLocked || view.Round == nil {
		t.Fatalf("guest view=%+v", view)
	}
	b.Status = "CANCELLED"
	if ended := participantView(b, players, rows, songs, guest); ended.MyScore != 50 || ended.OpponentScore != 100 {
		t.Fatalf("ended guest view=%+v", ended)
	}
	if view.Me.Username != "gus" || view.Opponent == nil || view.Opponent.Username != "hana" {
		t.Fatalf("guest players me=%+v opponent=%+v", view.Me, view.Opponent)
	}
	if view.Round.MyStage != "MELODY" || view.Round.MyFinished || !view.Round.OpponentFinished || view.Round.MyAttempts != 3 {
		t.Fatalf("guest round=%+v", view.Round)
	}
	if len(view.CompletedRounds) != 0 {
		t.Fatalf("unfinished round revealed=%+v", view.CompletedRounds)
	}
	b.Status = "FINISHED"
	rows[0].Status = "FINISHED"
	for _, tc := range []struct {
		winner pgtype.UUID
		want   string
	}{{host, "LOSS"}, {guest, "WIN"}, {pgtype.UUID{}, "DRAW"}} {
		b.WinnerID = tc.winner
		view = participantView(b, players, rows, songs, guest)
		if view.Outcome == nil || *view.Outcome != tc.want || view.Round != nil || len(view.CompletedRounds) != 1 {
			t.Fatalf("finished guest view=%+v", view)
		}
	}
	want := RoundResult{Number: 1, MyPoints: 50, OpponentPoints: 100, MySong: Song{Title: "Guest pick", Artist: "G"}, OpponentSong: Song{Title: "Host pick", Artist: "H"}}
	if view.CompletedRounds[0] != want {
		t.Fatalf("guest result=%+v, want %+v", view.CompletedRounds[0], want)
	}
	hostView := participantView(b, players, rows, songs, host)
	if hostView.Me.Username != "hana" || hostView.Opponent.Username != "gus" || hostView.CompletedRounds[0].MySong.Title != "Host pick" || hostView.CompletedRounds[0].OpponentSong.Title != "Guest pick" {
		t.Fatalf("host view me=%+v opponent=%+v results=%+v", hostView.Me, hostView.Opponent, hostView.CompletedRounds)
	}
}

func TestParticipantViewBeforeOpponentJoins(t *testing.T) {
	var host pgtype.UUID
	if err := host.Scan("00000000-0000-4000-8000-000000000001"); err != nil {
		t.Fatal(err)
	}
	b := dbgen.Battle{ID: host, HostID: host, RoundCount: 3, Status: "PENDING"}
	view := participantView(b, dbgen.GetBattlePlayersRow{HostUsername: "hana"}, nil, nil, host)
	if view.Me.Username != "hana" || view.Opponent != nil || view.CompletedRounds == nil {
		t.Fatalf("pending view=%+v", view)
	}
}
