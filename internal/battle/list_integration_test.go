package battle

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestListShowsOnlyOwnBattlesActiveFirst(t *testing.T) {
	l, ctx, u := lobbyFixture(t)
	finished, err := l.Create(ctx, u[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Forfeit(ctx, u[0], finished); err != nil {
		t.Fatal(err)
	}
	joined, err := l.Create(ctx, u[1], 2)
	if err != nil {
		t.Fatal(err)
	}
	var code string
	if err := l.pool.QueryRow(ctx, `SELECT invite_code FROM battles WHERE id=$1`, joined).Scan(&code); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Join(ctx, u[0], code); err != nil {
		t.Fatal(err)
	}
	pending, err := l.Create(ctx, u[0], 3)
	if err != nil {
		t.Fatal(err)
	}
	stranger, err := l.Create(ctx, u[2], 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.pool.Exec(ctx, `UPDATE battles SET last_activity_at = now() - interval '1 hour' WHERE id=$1`, pending); err != nil {
		t.Fatal(err)
	}

	list, err := l.List(ctx, u[0])
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(list))
	for i, s := range list {
		ids[i] = s.ID
		if s.ID == stranger.String() {
			t.Fatal("listed another player's battle")
		}
	}
	if len(list) != 3 || ids[0] != joined.String() || ids[1] != pending.String() || ids[2] != finished.String() {
		t.Fatalf("order = %v, want joined, pending, finished", ids)
	}
	if list[0].Opponent == nil || list[0].InviteCode != nil || list[0].Status != "SELECTING" || list[0].Rounds != 2 {
		t.Fatalf("joined summary = %+v", list[0])
	}
	if list[1].Opponent != nil || list[1].InviteCode == nil || len(*list[1].InviteCode) != 6 {
		t.Fatalf("pending host summary = %+v", list[1])
	}
	if list[2].Status != "CANCELLED" || list[2].EndReason == nil || *list[2].EndReason != "FORFEIT" || list[2].Outcome != nil {
		t.Fatalf("cancelled summary = %+v", list[2])
	}

	guestList, err := l.List(ctx, u[1])
	if err != nil {
		t.Fatal(err)
	}
	if len(guestList) != 1 || guestList[0].Opponent == nil || guestList[0].InviteCode != nil {
		t.Fatalf("host-of-joined list = %+v", guestList)
	}
}

func TestListIsCapped(t *testing.T) {
	l, ctx, u := lobbyFixture(t)
	for range maxListedBattles + 2 {
		if _, err := l.Create(ctx, u[2], 1); err != nil {
			t.Fatal(err)
		}
	}
	list, err := l.List(ctx, u[2])
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != maxListedBattles {
		t.Fatalf("listed %d battles, want %d", len(list), maxListedBattles)
	}
}

func TestViewRevealsSongsOnlyAfterBothPlayersFinish(t *testing.T) {
	l, users, id, _ := activeMatch(t, 2)
	ctx := t.Context()
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Me.Username == "" || view.Opponent == nil || view.Opponent.Username == view.Me.Username {
		t.Fatalf("players me=%+v opponent=%+v", view.Me, view.Opponent)
	}
	var roundID pgtype.UUID
	if err := roundID.Scan(view.Round.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 1); err != nil {
		t.Fatal(err)
	}
	for _, user := range users[:2] {
		view, err := l.Get(ctx, user, id)
		if err != nil {
			t.Fatal(err)
		}
		if len(view.CompletedRounds) != 0 || view.MyScore != 0 || view.OpponentScore != 0 {
			t.Fatalf("revealed before the round finished: %+v", view)
		}
		list, err := l.List(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].MyScore != 0 || list[0].OpponentScore != 0 {
			t.Fatalf("listed scores before the round finished: %+v", list)
		}
	}
	if _, err := l.SubmitGuess(ctx, users[1], roundID, "Host selection", 1); err != nil {
		t.Fatal(err)
	}
	host, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	guest, err := l.Get(ctx, users[1], id)
	if err != nil {
		t.Fatal(err)
	}
	if len(host.CompletedRounds) != 1 || host.CompletedRounds[0].MySong.Title != "Host selection" || host.CompletedRounds[0].OpponentSong.Title != "Guest selection" || host.CompletedRounds[0].MyPoints != 100 {
		t.Fatalf("host completed = %+v", host.CompletedRounds)
	}
	if len(guest.CompletedRounds) != 1 || guest.CompletedRounds[0].MySong.Title != "Guest selection" || guest.CompletedRounds[0].OpponentPoints != 100 {
		t.Fatalf("guest completed = %+v", guest.CompletedRounds)
	}
	list, err := l.List(ctx, users[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].MyScore != 100 || list[0].OpponentScore != 100 || list[0].CurrentRound != 2 || list[0].Opponent == nil {
		t.Fatalf("list after round one = %+v", list)
	}
}

func TestListSumsScoresAcrossRounds(t *testing.T) {
	l, users, id, _ := activeMatch(t, 2)
	ctx := t.Context()
	for round := range 2 {
		view, err := l.Get(ctx, users[0], id)
		if err != nil {
			t.Fatal(err)
		}
		var roundID pgtype.UUID
		if err := roundID.Scan(view.Round.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 1); err != nil {
			t.Fatal(err)
		}
		attempt := int32(1)
		if round == 0 {
			if _, err := l.SubmitGuess(ctx, users[1], roundID, "wrong answer", attempt); err != nil {
				t.Fatal(err)
			}
			attempt++
		}
		if _, err := l.SubmitGuess(ctx, users[1], roundID, "Host selection", attempt); err != nil {
			t.Fatal(err)
		}
	}
	for i, user := range users[:2] {
		view, err := l.Get(ctx, user, id)
		if err != nil {
			t.Fatal(err)
		}
		list, err := l.List(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		if view.Status != "FINISHED" || len(view.CompletedRounds) != 2 || view.MyScore == view.OpponentScore {
			t.Fatalf("player %d view = %+v", i, view)
		}
		var mine, theirs int32
		for _, r := range view.CompletedRounds {
			mine += r.MyPoints
			theirs += r.OpponentPoints
		}
		if len(list) != 1 || list[0].MyScore != mine || list[0].OpponentScore != theirs || list[0].MyScore != view.MyScore {
			t.Fatalf("player %d list = %+v, want %d-%d", i, list, mine, theirs)
		}
	}
}

func TestScoresIncludeInterruptedRoundAfterForfeit(t *testing.T) {
	l, users, id, _ := activeMatch(t, 2)
	ctx := t.Context()
	if _, err := l.SubmitGuess(ctx, users[0], roundIDFor(t, l, ctx, users[0], id), "Guest selection", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Forfeit(ctx, users[1], id); err != nil {
		t.Fatal(err)
	}
	for i, user := range users[:2] {
		view, err := l.Get(ctx, user, id)
		if err != nil {
			t.Fatal(err)
		}
		list, err := l.List(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		mine, theirs, result := int32(100), int32(0), "WIN"
		if i == 1 {
			mine, theirs, result = theirs, mine, "LOSS"
		}
		if view.Status != "FINISHED" || view.MyScore != mine || view.OpponentScore != theirs || len(view.CompletedRounds) != 0 ||
			view.Outcome == nil || *view.Outcome != result || view.EndReason == nil || *view.EndReason != "FORFEIT" {
			t.Fatalf("player %d view = %+v", i, view)
		}
		if len(list) != 1 || list[0].MyScore != mine || list[0].OpponentScore != theirs ||
			list[0].Outcome == nil || *list[0].Outcome != result || list[0].EndReason == nil || *list[0].EndReason != "FORFEIT" {
			t.Fatalf("player %d list = %+v", i, list)
		}
	}
}
