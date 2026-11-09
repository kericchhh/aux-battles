package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kericchhh/aux-battles/internal/battle"
)

func TestBattleErrorResponsesKeepInternalDetailsPrivate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		status  int
		message string
		logged  bool
	}{
		{"invalid input", battle.ErrInvalidInput, 400, "invalid battle input", false},
		{"not found", battle.ErrSongNotFound, 404, "a selected song was not found", false},
		{"conflict", battle.ErrLineupConflict, 409, "battle is not accepting this lineup", false},
		{"unavailable", battle.ErrInviteExhausted, 503, "could not allocate an invite code; retry", false},
		{"wrapped public error", fmt.Errorf("private context: %w", battle.ErrNotFound), 404, "battle not found", false},
		{"internal error", errors.New("private database details"), 500, "could not complete battle operation", true},
		{"unknown category", &battle.Error{Kind: 99, Message: "private details"}, 500, "could not complete battle operation", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			s := &server{logger: slog.New(slog.NewJSONHandler(&logs, nil))}
			response := httptest.NewRecorder()
			s.writeBattleError(response, httptest.NewRequest(http.MethodPut, "/battles/id/lineup", nil), tc.err)
			var body struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != tc.status || body.Message != tc.message {
				t.Fatalf("response=%d %q", response.Code, body.Message)
			}
			if strings.Contains(response.Body.String(), "private") {
				t.Fatal("response leaked internal details")
			}
			if (logs.Len() > 0) != tc.logged {
				t.Fatalf("logs=%s", logs.String())
			}
		})
	}
}
