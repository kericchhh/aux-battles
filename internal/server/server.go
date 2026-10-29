package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"golang.org/x/time/rate"
)

type server struct {
	queries       *dbgen.Queries
	pool          *pgxpool.Pool
	logger        *slog.Logger
	secureCookies bool
}

func New(queries *dbgen.Queries, pool *pgxpool.Pool, logger *slog.Logger, secureCookies bool) http.Handler {
	loginLimit := newIPLimiter(rate.Every(12*time.Second), 5)
	registerLimit := newIPLimiter(rate.Every(time.Minute), 3)
	app := &server{queries: queries, pool: pool, logger: logger, secureCookies: secureCookies}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", app.health)
	mux.HandleFunc("GET /songs", app.listSongs)
	mux.HandleFunc("POST /songs", app.requireAuth(app.uploadSong))
	mux.HandleFunc("GET /songs/{id}", app.getSong)
	mux.HandleFunc("GET /songs/{id}/status", app.requireAuth(app.getSongStatus))
	mux.HandleFunc("POST /users", registerLimit.wrap(app.registerUser))
	mux.HandleFunc("POST /users/login", loginLimit.wrap(app.loginUser))
	mux.HandleFunc("POST /users/logout", app.logoutUser)
	mux.HandleFunc("GET /users/me", app.getCurrentUser)
	return logRequests(logger, mux)
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok\n"))
}
