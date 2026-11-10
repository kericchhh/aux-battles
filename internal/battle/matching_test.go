package battle

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
)

func TestTitleMatching(t *testing.T) {
	for _, tc := range []struct {
		guess, title string
		want         bool
	}{
		{"  CAFE  ", "Café (Remastered)", true},
		{"hello", "Hello feat. Artist", true},
		{"hello", "Hello ft. Artist", true},
		{"hello", "Hello featuring Artist", true},
		{"a", "A Feat of Clay", false},
		{"a feat of clay", "A Feat of Clay", true},
		{"the", "The Feat of Strength", false},
		{"fifty", "Fifty ft Woman", false},
		{"hello", "Hello (feat. Artist)", true},
		{"hello", "Hello FEATURING Artist", true},
		{"hello", "Hello - ft. Artist", true},
		{"hello", "Hello feat. Artist", true},
		{"hello", "Hello feat Artist", false},
		{"hello", "Hello feat.Artist", false},
		{"50", "50 Ft. Queenie", false},
		{"50 ft queenie", "50 Ft. Queenie", true},
		{"now", "Now Featuring You", false},
		{"hey", "Hey ft. Artist", false},
		{"hey ft. artist", "Hey ft. Artist", true},
		{"left behind", "Left Behind", true},
		{"gift", "Gift", true},
		{"hallo", "Hello", true},
		{"halo", "Hello", false},
		{"one", "One", true},
		{"on", "One", false},
		{"東京", "東京", true},
		{"(Something)", "(Something)", true},
		{"Something", "(Something)", true},
		{"[Untitled]", "[Untitled]", true},
		{"Ft. Something", "Ft. Something", true},
		{"Feat. Artist", "Feat. Artist", true},
		{"711", "7/11", true},
		{"ab", "A.B", true},
		{"", "", false},
		{"!!!", "(Live)", false},
		{"different", "Hello", false},
	} {
		if got := matchesTitle(tc.guess, tc.title); got != tc.want {
			t.Errorf("match(%q,%q)=%t, want %t", tc.guess, tc.title, got, tc.want)
		}
	}
}

func TestGuessRejectsInvalidInputBeforeDatabase(t *testing.T) {
	l := New(nil, media.New(""))
	id := pgtype.UUID{Valid: true}
	for _, tc := range []struct {
		guess   string
		attempt int32
	}{
		{"", 1}, {"!!!", 1}, {"\u0301", 1}, {strings.Repeat("a", 256), 1}, {"title", 0}, {"title", 5},
	} {
		if _, err := l.SubmitGuess(context.Background(), id, id, tc.guess, tc.attempt); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("guess=%q attempt=%d error=%v", tc.guess, tc.attempt, err)
		}
	}
}

func TestStreakBonus(t *testing.T) {
	rounds := []dbgen.ListBattleRoundsRow{
		{RoundNumber: 1, Status: "FINISHED", HostPoints: 100, GuestPoints: 0},
		{RoundNumber: 2, Status: "FINISHED", HostPoints: 110, GuestPoints: 20},
		{RoundNumber: 3, Status: "GUESSING"},
	}
	for _, tc := range []struct {
		round int32
		host  bool
		want  int32
	}{
		{1, true, 0}, {2, true, 10}, {3, true, 20}, {3, false, 10}, {2, false, 0},
	} {
		if got := streakBonus(rounds, tc.round, tc.host); got != tc.want {
			t.Errorf("bonus=%d, want %d", got, tc.want)
		}
	}
}
