package battle

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

func TestTimeoutEndingPenalizesOnlyTheStallingPlayer(t *testing.T) {
	var host, guest, song pgtype.UUID
	for value, id := range map[string]*pgtype.UUID{
		"00000000-0000-4000-8000-000000000001": &host,
		"00000000-0000-4000-8000-000000000002": &guest,
		"00000000-0000-4000-8000-000000000003": &song,
	} {
		if err := id.Scan(value); err != nil {
			t.Fatal(err)
		}
	}
	numbered := func(n int32, hostSong, guestSong pgtype.UUID, hostDone, guestDone bool) dbgen.ListBattleRoundsRow {
		return dbgen.ListBattleRoundsRow{RoundNumber: n, HostSongID: hostSong, GuestSongID: guestSong, HostFinished: hostDone, GuestFinished: guestDone}
	}
	round := func(hostSong, guestSong pgtype.UUID, hostDone, guestDone bool) dbgen.ListBattleRoundsRow {
		return numbered(1, hostSong, guestSong, hostDone, guestDone)
	}
	for _, tc := range []struct {
		name    string
		status  string
		current int32
		rounds  []dbgen.ListBattleRoundsRow
		want    string
		winner  pgtype.UUID
	}{
		{"lobby", "PENDING", 1, nil, "CANCELLED", pgtype.UUID{}},
		{"selecting without rounds", "SELECTING", 1, nil, "CANCELLED", pgtype.UUID{}},
		{"guest lineup incomplete", "SELECTING", 1, []dbgen.ListBattleRoundsRow{
			numbered(1, song, song, false, false), numbered(2, song, pgtype.UUID{}, false, false)}, "FINISHED", host},
		{"later round nobody finished", "ONGOING", 2, []dbgen.ListBattleRoundsRow{
			numbered(1, song, song, true, false), numbered(2, song, song, false, false), numbered(3, song, song, true, false)}, "CANCELLED", pgtype.UUID{}},
		{"later round guest finished", "ONGOING", 2, []dbgen.ListBattleRoundsRow{
			numbered(1, song, song, true, false), numbered(2, song, song, false, true), numbered(3, song, song, true, false)}, "FINISHED", guest},
		{"no lineups", "SELECTING", 1, []dbgen.ListBattleRoundsRow{round(pgtype.UUID{}, pgtype.UUID{}, false, false)}, "CANCELLED", pgtype.UUID{}},
		{"host lineup", "SELECTING", 1, []dbgen.ListBattleRoundsRow{round(song, pgtype.UUID{}, false, false)}, "FINISHED", host},
		{"guest lineup", "SELECTING", 1, []dbgen.ListBattleRoundsRow{round(pgtype.UUID{}, song, false, false)}, "FINISHED", guest},
		{"nobody finished", "ONGOING", 1, []dbgen.ListBattleRoundsRow{round(song, song, false, false)}, "CANCELLED", pgtype.UUID{}},
		{"host finished", "ONGOING", 1, []dbgen.ListBattleRoundsRow{round(song, song, true, false)}, "FINISHED", host},
		{"guest finished", "ONGOING", 1, []dbgen.ListBattleRoundsRow{round(song, song, false, true)}, "FINISHED", guest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := dbgen.Battle{ID: song, HostID: host, GuestID: guest, Status: tc.status, CurrentRound: tc.current, RoundCount: 3}
			end := timeoutEnding(b, tc.rounds)
			if end.Status != tc.want || end.WinnerID != tc.winner || end.EndReason != "TIMEOUT" || end.ID != b.ID {
				t.Fatalf("ending=%+v, want %s winner %v", end, tc.want, tc.winner)
			}
		})
	}
}

func TestParticipantViewReportsEndReason(t *testing.T) {
	var host pgtype.UUID
	if err := host.Scan("00000000-0000-4000-8000-000000000001"); err != nil {
		t.Fatal(err)
	}
	b := dbgen.Battle{ID: host, HostID: host, RoundCount: 1, CurrentRound: 1, Status: "ONGOING"}
	if view := participantView(b, nil, host); view.EndReason != nil || view.Outcome != nil {
		t.Fatalf("running battle view=%+v", view)
	}
	b.Status, b.EndReason = "CANCELLED", pgtype.Text{String: "TIMEOUT", Valid: true}
	view := participantView(b, nil, host)
	if view.EndReason == nil || *view.EndReason != "TIMEOUT" || view.Outcome != nil {
		t.Fatalf("cancelled battle view=%+v", view)
	}
}
