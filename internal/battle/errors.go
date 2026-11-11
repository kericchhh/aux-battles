package battle

import "fmt"

type ErrorKind uint8

const (
	InvalidInput ErrorKind = iota + 1
	NotFound
	Conflict
	Unavailable
)

type Error struct {
	Kind    ErrorKind
	Message string
	cause   error
	reason  *Error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

func (e *Error) Is(target error) bool { return e.reason != nil && target == e.reason }

var (
	ErrInvalidInput          = &Error{Kind: InvalidInput, Message: "invalid battle input"}
	ErrNotFound              = &Error{Kind: NotFound, Message: "battle not found"}
	ErrUnavailable           = &Error{Kind: Conflict, Message: "battle is full, already started, or belongs to you"}
	ErrInviteExhausted       = &Error{Kind: Unavailable, Message: "could not allocate an invite code; retry"}
	ErrBattleEnded           = &Error{Kind: Conflict, Message: "battle has already ended"}
	ErrRoundUnavailable      = &Error{Kind: Conflict, Message: "round is not accepting this operation"}
	ErrStaleAttempt          = &Error{Kind: Conflict, Message: "attempt already submitted or stale; refresh the battle"}
	ErrTimeExpired           = &Error{Kind: Conflict, Message: "time ran out on this stem; refresh the battle"}
	ErrRoundAudioUnavailable = &Error{Kind: Conflict, Message: "round audio is unavailable"}
	ErrLineupConflict        = &Error{Kind: Conflict, Message: "battle is not accepting this lineup"}
	ErrSongNotFound          = &Error{Kind: NotFound, Message: "a selected song was not found"}
	ErrSongNotReady          = &Error{Kind: Conflict, Message: "a selected song is not ready"}
	ErrAudioUnavailable      = &Error{Kind: Conflict, Message: "a selected song has unavailable audio"}
)

func invalidInput(message string) error {
	return &Error{Kind: InvalidInput, Message: message, reason: ErrInvalidInput}
}

func withCause(reason *Error, cause error) error {
	return &Error{Kind: reason.Kind, Message: reason.Message, cause: cause, reason: reason}
}
