package server

import (
	"log/slog"
	"net/http"

	"github.com/kericchhh/aux-battles/internal/dbgen"
)

type server struct {
	queries       *dbgen.Queries
	logger        *slog.Logger
	secureCookies bool
}

func New(queries *dbgen.Queries, logger *slog.Logger, secureCookies bool) http.Handler {
	app := &server{queries: queries, logger: logger, secureCookies: secureCookies}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", app.health)
	mux.HandleFunc("GET /songs", app.listSongs)
	mux.HandleFunc("GET /songs/{id}", app.getSong)
	mux.HandleFunc("POST /users", app.registerUser)
	mux.HandleFunc("POST /users/login", app.loginUser)
	return logRequests(logger, mux)
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok\n"))
}
