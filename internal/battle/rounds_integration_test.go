package battle

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func activeMatch(t *testing.T, n int32) (*Lobby, []pgtype.UUID, pgtype.UUID, string) {
	t.Helper()
	l, ctx, users, id, root := lineupFixture(t, n)
	for i, user := range users[:2] {
		songs := make([]pgtype.UUID, n)
		for j := range songs {
			songs[j] = lineupSong(t, l, ctx, root, true)
			title := "Host selection"
			if i == 1 {
				title = "Guest selection"
			}
			if _, err := l.pool.Exec(ctx, `UPDATE songs SET title=$2 WHERE id=$1`, songs[j], title); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := l.SubmitLineup(ctx, user, id, songs); err != nil {
			t.Fatal(err)
		}
	}
	return l, users, id, root
}

func TestFullMatchStreakCapResetAndWinner(t *testing.T) {
	l, users, id, _ := activeMatch(t, 4)
	ctx := t.Context()
	hostExpected := []int32{100, 110, 120, 120}
	guestExpected := []int32{20, 0, 20, 30}
	for round := range 4 {
		view, err := l.Get(ctx, users[0], id)
		if err != nil {
			t.Fatal(err)
		}
		if view.CurrentRound != int32(round+1) || view.Round == nil {
			t.Fatalf("view=%+v", view)
		}
		var roundID pgtype.UUID
		if err := roundID.Scan(view.Round.ID); err != nil {
			t.Fatal(err)
		}
		host, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 1)
		if err != nil {
			t.Fatal(err)
		}
		if host.PointsAwarded != hostExpected[round] || host.BasePoints != 100 || host.RoundFinished || !host.PlayerFinished {
			t.Fatalf("host round %d=%+v", round+1, host)
		}
		if _, err := l.SubmitGuess(ctx, users[0], roundID, "Guest selection", 1); !errors.Is(err, ErrRoundUnavailable) {
			t.Fatalf("finished player=%v", err)
		}
		for attempt := int32(1); attempt <= 4; attempt++ {
			guess := "wrong answer"
			if attempt == 4 && round != 1 {
				guess = "Host selection"
			}
			guest, err := l.SubmitGuess(ctx, users[1], roundID, guess, attempt)
			if err != nil {
				t.Fatal(err)
			}
			if attempt < 4 && (guest.PointsAwarded != 0 || guest.PlayerFinished || guest.RoundFinished) {
				t.Fatalf("incorrect guess=%+v", guest)
			}
			if attempt == 4 && (guest.PointsAwarded != guestExpected[round] || !guest.RoundFinished || !guest.PlayerFinished) {
				t.Fatalf("guest round %d=%+v", round+1, guest)
			}
		}
	}
	host, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	guest, err := l.Get(ctx, users[1], id)
	if err != nil {
		t.Fatal(err)
	}
	if host.Status != "FINISHED" || host.MyScore != 450 || host.OpponentScore != 70 || host.Outcome == nil || *host.Outcome != "WIN" || guest.Outcome == nil || *guest.Outcome != "LOSS" || host.Round != nil {
		t.Fatalf("final host=%+v guest=%+v", host, guest)
	}
	var recorded int32
	if err := l.pool.QueryRow(ctx, `SELECT sum(points_awarded)::integer FROM guesses g JOIN rounds r ON r.id=g.round_id WHERE r.battle_id=$1 AND g.user_id=$2`, id, users[0]).Scan(&recorded); err != nil {
		t.Fatal(err)
	}
	if recorded != host.MyScore {
		t.Fatalf("recorded=%d displayed=%d", recorded, host.MyScore)
	}
}

func TestConcurrentGuessesConsumeOneAttemptAndFinishDraw(t *testing.T) {
	l, users, id, _ := activeMatch(t, 1)
	ctx := t.Context()
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	var roundID pgtype.UUID
	if err := roundID.Scan(view.Round.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SubmitGuess(ctx, users[2], roundID, "Guest selection", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider=%v", err)
	}
	start, results := make(chan struct{}), make(chan error, 2)
	for range 2 {
		go func() { <-start; _, err := l.SubmitGuess(ctx, users[0], roundID, "wrong", 1); results <- err }()
	}
	close(start)
	success, conflicts := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, ErrStaleAttempt) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
	if _, err := l.SubmitGuess(ctx, users[1], roundID, "wrong", 1); err != nil {
		t.Fatal(err)
	}
	type answer struct {
		result GuessResult
		err    error
	}
	finish := make(chan answer, 2)
	for i, user := range users[:2] {
		title := "Guest selection"
		if i == 1 {
			title = "Host selection"
		}
		go func() { r, err := l.SubmitGuess(ctx, user, roundID, title, 2); finish <- answer{r, err} }()
	}
	finished := 0
	for range 2 {
		a := <-finish
		if a.err != nil {
			t.Fatal(a.err)
		}
		if a.result.RoundFinished {
			finished++
		}
		if a.result.BasePoints != 75 {
			t.Fatalf("answer=%+v", a.result)
		}
	}
	final, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if finished != 1 || final.Outcome == nil || *final.Outcome != "DRAW" || final.MyScore != 75 || final.OpponentScore != 75 {
		t.Fatalf("finished=%d final=%+v", finished, final)
	}
}

func TestRoundAudioUsesOpponentStageAndRejectsUnavailableRounds(t *testing.T) {
	l, users, id, root := activeMatch(t, 2)
	ctx := t.Context()
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	var current, future pgtype.UUID
	if err := current.Scan(view.Round.ID); err != nil {
		t.Fatal(err)
	}
	if err := l.pool.QueryRow(ctx, `SELECT id FROM rounds WHERE battle_id=$1 AND round_number=2`, id).Scan(&future); err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.OpenRoundAudio(ctx, users[2], current); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider audio=%v", err)
	}
	if _, _, err := l.OpenRoundAudio(ctx, users[0], future); !errors.Is(err, ErrRoundUnavailable) {
		t.Fatalf("future audio=%v", err)
	}
	var songID pgtype.UUID
	if err := l.pool.QueryRow(ctx, `SELECT guest_song_id FROM rounds WHERE id=$1`, current).Scan(&songID); err != nil {
		t.Fatal(err)
	}
	for attempt, name := range []string{"drums.mp3", "bass.mp3", "other.mp3", "full.mp3"} {
		path := filepath.Join(root, "songs", songID.String(), "attempt-1", name)
		if err := os.WriteFile(path, []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		file, _, err := l.OpenRoundAudio(ctx, users[0], current)
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(file)
		file.Close()
		if readErr != nil || string(data) != name {
			t.Fatalf("stage %d audio=%q error=%v", attempt+1, data, readErr)
		}
		if _, err := l.SubmitGuess(ctx, users[0], current, "wrong answer", int32(attempt+1)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := l.SubmitGuess(ctx, users[1], current, "Host selection", 1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.OpenRoundAudio(ctx, users[0], current); !errors.Is(err, ErrRoundUnavailable) {
		t.Fatalf("old round audio=%v", err)
	}
}

func TestGuessRollbackWhenNextRoundCannotActivate(t *testing.T) {
	l, users, id, _ := activeMatch(t, 2)
	ctx := t.Context()
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
	if _, err := l.pool.Exec(ctx, `UPDATE rounds SET status='GUESSING' WHERE battle_id=$1 AND round_number=2`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SubmitGuess(ctx, users[1], roundID, "Host selection", 1); err == nil {
		t.Fatal("guess advanced an inconsistent battle")
	}
	var attempts int32
	var finished bool
	if err := l.pool.QueryRow(ctx, `SELECT guest_finished,(SELECT count(*)::integer FROM guesses WHERE round_id=$1 AND user_id=$2) FROM rounds WHERE id=$1`, roundID, users[1]).Scan(&finished, &attempts); err != nil {
		t.Fatal(err)
	}
	if finished || attempts != 0 {
		t.Fatalf("failed transaction persisted: finished=%t attempts=%d", finished, attempts)
	}
	if _, err := l.pool.Exec(ctx, `UPDATE rounds SET status='WAITING' WHERE battle_id=$1 AND round_number=2`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SubmitGuess(ctx, users[1], roundID, "Host selection", 1); err != nil {
		t.Fatalf("retry after rollback=%v", err)
	}
}

func TestRoundAudioRejectsMissingAndUnsafeFiles(t *testing.T) {
	l, users, id, root := activeMatch(t, 1)
	ctx := t.Context()
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	var roundID pgtype.UUID
	if err := roundID.Scan(view.Round.ID); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.mp3")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "songs", "missing.mp3"), outside} {
		if _, err := l.pool.Exec(ctx, `UPDATE songs SET drums_path=$2 WHERE id=(SELECT guest_song_id FROM rounds WHERE id=$1)`, roundID, path); err != nil {
			t.Fatal(err)
		}
		file, _, err := l.OpenRoundAudio(ctx, users[0], roundID)
		if file != nil {
			file.Close()
			t.Fatal("opened unavailable audio")
		}
		if !errors.Is(err, ErrRoundAudioUnavailable) {
			t.Fatalf("audio error=%v", err)
		}
	}
	if _, err := l.pool.Exec(ctx, `UPDATE songs SET status='PROCESSING' WHERE id=(SELECT guest_song_id FROM rounds WHERE id=$1)`, roundID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.OpenRoundAudio(ctx, users[0], roundID); !errors.Is(err, ErrRoundAudioUnavailable) {
		t.Fatalf("not-ready audio error=%v", err)
	}
	if _, err := l.SubmitGuess(ctx, users[0], roundID, "anything", 1); !errors.Is(err, ErrRoundAudioUnavailable) {
		t.Fatalf("not-ready guess error=%v", err)
	}
}
