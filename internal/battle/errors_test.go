package battle

import (
	"errors"
	"fmt"
	"testing"
)

func TestBattleErrorsPreserveReasonAndInternalCause(t *testing.T) {
	cause := errors.New("private media path")
	wrapped := fmt.Errorf("operation context: %w", withCause(ErrAudioUnavailable, cause))
	if !errors.Is(wrapped, ErrAudioUnavailable) || !errors.Is(wrapped, cause) || errors.Is(wrapped, ErrSongNotReady) {
		t.Fatal("error chain lost its specific reason or cause")
	}
	var failure *Error
	if !errors.As(wrapped, &failure) || failure.Kind != Conflict || failure.Message != ErrAudioUnavailable.Message {
		t.Fatalf("public error=%+v", failure)
	}
	input := invalidInput("choose exactly two songs")
	if !errors.Is(input, ErrInvalidInput) || input.Error() != "choose exactly two songs" {
		t.Fatalf("input error=%v", input)
	}
}
