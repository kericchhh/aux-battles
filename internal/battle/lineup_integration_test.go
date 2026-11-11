package battle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
)

func lineupFixture(t *testing.T, rounds int32) (*Lobby, context.Context, []pgtype.UUID, pgtype.UUID, string) {
	t.Helper()
	l, ctx, users := lobbyFixture(t)
	root := t.TempDir()
	l = New(l.pool, media.New(root))
	id, err := l.Create(ctx, users[0], rounds, 0)
	if err != nil {
		t.Fatal(err)
	}
	view, err := l.Get(ctx, users[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Join(ctx, users[1], view.InviteCode); err != nil {
		t.Fatal(err)
	}
	return l, ctx, users, id, root
}

func lineupSong(t *testing.T, l *Lobby, ctx context.Context, root string, ready bool) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := l.pool.QueryRow(ctx, `INSERT INTO songs(title,artist,genre,duration_seconds) VALUES('Cleared fixture','Test','Test',12) RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := l.pool.Exec(context.Background(), `DELETE FROM battles WHERE id IN (SELECT battle_id FROM rounds WHERE host_song_id=$1 OR guest_song_id=$1)`, id); err != nil {
			t.Error(err)
		}
		if _, err := l.pool.Exec(context.Background(), `DELETE FROM songs WHERE id=$1`, id); err != nil {
			t.Error(err)
		}
	})
	if !ready {
		return id
	}
	dir := filepath.Join(root, "songs", id.String(), "attempt-1")
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, 5)
	for _, name := range []string{"full.mp3", "drums.mp3", "bass.mp3", "other.mp3", "vocals.mp3"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("audio fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	if _, err := l.pool.Exec(ctx, `UPDATE songs SET status='READY',full_song_path=$2,drums_path=$3,bass_path=$4,melody_path=$5,vocals_path=$6 WHERE id=$1`, id, paths[0], paths[1], paths[2], paths[3], paths[4]); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestConcurrentLineupsStartOnlyFirstRoundAndPreserveOrder(t *testing.T) {
	l, ctx, u, id, root := lineupFixture(t, 2)
	songs := []pgtype.UUID{lineupSong(t, l, ctx, root, true), lineupSong(t, l, ctx, root, true)}
	results := make(chan struct {
		result LineupResult
		err    error
	}, 2)
	start := make(chan struct{})
	for i, user := range u[:2] {
		choice := []pgtype.UUID{songs[i], songs[1-i]}
		go func() {
			<-start
			r, e := l.SubmitLineup(ctx, user, id, choice)
			results <- struct {
				result LineupResult
				err    error
			}{r, e}
		}()
	}
	close(start)
	statuses := map[string]int{}
	for range 2 {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.result.BattleID != id.String() {
			t.Fatalf("result=%+v", r.result)
		}
		statuses[r.result.Status]++
	}
	if statuses["WAITING_ON_OPPONENT"] != 1 || statuses["BATTLE_STARTED"] != 1 {
		t.Fatalf("results=%v", statuses)
	}
	rounds, err := dbgen.New(l.pool).ListBattleRounds(ctx, dbgen.ListBattleRoundsParams{BattleID: id, UserID: u[0]})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range rounds {
		if r.HostSongID != songs[i] || r.GuestSongID != songs[1-i] {
			t.Fatalf("round %d order=%+v", i, r)
		}
		want := "WAITING"
		if i == 0 {
			want = "GUESSING"
		}
		if r.Status != want {
			t.Fatalf("round %d status=%s", i, r.Status)
		}
	}
	for _, user := range u[:2] {
		v, err := l.Get(ctx, user, id)
		if err != nil {
			t.Fatal(err)
		}
		if v.Status != "ONGOING" || v.CurrentRound != 1 || !v.MyLineupLocked || !v.OpponentLineupLocked || v.Round == nil || v.Round.MyStage != "DRUM" || v.Round.MyAttempts != 0 {
			t.Fatalf("started view=%+v", v)
		}
		if _, err := l.SubmitLineup(ctx, user, id, songs); !errors.Is(err, ErrLineupConflict) {
			t.Fatalf("resubmit active battle=%v", err)
		}
	}
}

func TestLineupValidationLeavesNoPartialChoices(t *testing.T) {
	for _, tc := range []struct {
		name string
		want error
	}{
		{"outsider", ErrNotFound}, {"wrong count", ErrInvalidInput}, {"duplicate", ErrInvalidInput},
		{"missing song", ErrSongNotFound}, {"processing song", ErrSongNotReady}, {"missing audio", ErrAudioUnavailable},
		{"empty audio", ErrAudioUnavailable}, {"symlink outside media", ErrAudioUnavailable}, {"pending battle", ErrLineupConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l, ctx, u, id, root := lineupFixture(t, 2)
			songs := []pgtype.UUID{lineupSong(t, l, ctx, root, true), lineupSong(t, l, ctx, root, true)}
			user := u[0]
			switch tc.name {
			case "outsider":
				user = u[2]
			case "wrong count":
				songs = songs[:1]
			case "duplicate":
				songs[1] = songs[0]
			case "missing song":
				songs[1] = pgtype.UUID{Valid: true}
			case "processing song":
				songs[1] = lineupSong(t, l, ctx, root, false)
			case "missing audio", "empty audio", "symlink outside media":
				path := filepath.Join(root, "songs", songs[1].String(), "attempt-1", "bass.mp3")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if tc.name == "empty audio" {
					if err := os.WriteFile(path, nil, 0600); err != nil {
						t.Fatal(err)
					}
				}
				if tc.name == "symlink outside media" {
					outside := filepath.Join(t.TempDir(), "private.mp3")
					if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(outside, path); err != nil {
						t.Fatal(err)
					}
				}
			case "pending battle":
				var err error
				id, err = l.Create(ctx, u[0], 2, 0)
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := l.SubmitLineup(ctx, user, id, songs); !errors.Is(err, tc.want) {
				t.Fatalf("submit=%v, want %v", err, tc.want)
			}
			var selected int
			if err := l.pool.QueryRow(ctx, `SELECT count(*) FROM rounds WHERE battle_id=$1 AND (host_song_id IS NOT NULL OR guest_song_id IS NOT NULL)`, id).Scan(&selected); err != nil {
				t.Fatal(err)
			}
			if selected != 0 {
				t.Fatalf("partial lineup after failure: %d", selected)
			}
		})
	}
}

func TestConcurrentSubmissionsFromSamePlayerDoNotOverwriteLineup(t *testing.T) {
	l, ctx, u, id, root := lineupFixture(t, 1)
	songs := []pgtype.UUID{lineupSong(t, l, ctx, root, true), lineupSong(t, l, ctx, root, true)}
	type result struct {
		song  pgtype.UUID
		value LineupResult
		err   error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for _, song := range songs {
		go func() {
			<-start
			r, e := l.SubmitLineup(ctx, u[0], id, []pgtype.UUID{song})
			results <- result{song, r, e}
		}()
	}
	close(start)
	var winner pgtype.UUID
	successes, conflicts := 0, 0
	for range 2 {
		r := <-results
		if r.err == nil {
			successes++
			winner = r.song
			if r.value.Status != "WAITING_ON_OPPONENT" {
				t.Fatalf("status=%s", r.value.Status)
			}
		} else if errors.Is(r.err, ErrLineupConflict) {
			conflicts++
		} else {
			t.Fatal(r.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	var saved pgtype.UUID
	if err := l.pool.QueryRow(ctx, `SELECT host_song_id FROM rounds WHERE battle_id=$1`, id).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if saved != winner {
		t.Fatalf("saved=%v, winning submission=%v", saved, winner)
	}
	host, err := l.Get(ctx, u[0], id)
	if err != nil {
		t.Fatal(err)
	}
	guest, err := l.Get(ctx, u[1], id)
	if err != nil {
		t.Fatal(err)
	}
	if !host.MyLineupLocked || host.OpponentLineupLocked || guest.MyLineupLocked || !guest.OpponentLineupLocked || host.Round != nil || guest.Round != nil {
		t.Fatalf("waiting views: host=%+v guest=%+v", host, guest)
	}
}

func TestFailedStartRollsBackLineupAndRoundActivation(t *testing.T) {
	l, ctx, u, id, root := lineupFixture(t, 2)
	songs := []pgtype.UUID{lineupSong(t, l, ctx, root, true), lineupSong(t, l, ctx, root, true)}
	if _, err := l.SubmitLineup(ctx, u[1], id, songs); err != nil {
		t.Fatal(err)
	}
	if _, err := l.pool.Exec(ctx, `UPDATE battles SET current_round=2 WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SubmitLineup(ctx, u[0], id, songs); err == nil {
		t.Fatal("submission succeeded despite invalid starting round")
	}
	rounds, err := dbgen.New(l.pool).ListBattleRounds(ctx, dbgen.ListBattleRoundsParams{BattleID: id, UserID: u[0]})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range rounds {
		if r.HostSongID.Valid || r.GuestSongID != songs[i] || r.Status != "WAITING" {
			t.Fatalf("rollback round=%+v", r)
		}
	}
	view, err := l.Get(ctx, u[0], id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "SELECTING" || view.MyLineupLocked || !view.OpponentLineupLocked {
		t.Fatalf("rollback view=%+v", view)
	}
}

func TestLineupRejectsUnmatchableTitlesWithoutLocking(t *testing.T) {
	l, ctx, users, id, root := lineupFixture(t, 1)
	song := lineupSong(t, l, ctx, root, true)
	for _, title := range []string{"!!!", "\u0301"} {
		if _, err := l.pool.Exec(ctx, `UPDATE songs SET title=$2 WHERE id=$1`, song, title); err != nil {
			t.Fatal(err)
		}
		if _, err := l.SubmitLineup(ctx, users[0], id, []pgtype.UUID{song}); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("title=%q lineup error=%v", title, err)
		}
		view, err := l.Get(ctx, users[0], id)
		if err != nil {
			t.Fatal(err)
		}
		if view.MyLineupLocked || view.Status != "SELECTING" {
			t.Fatalf("rejected lineup persisted: %+v", view)
		}
	}
	if _, err := l.pool.Exec(ctx, `UPDATE songs SET title='(Untitled)' WHERE id=$1`, song); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SubmitLineup(ctx, users[0], id, []pgtype.UUID{song}); err != nil {
		t.Fatalf("literal bracketed title rejected: %v", err)
	}
}
