package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kericchhh/aux-battles/internal/battle"
	"github.com/kericchhh/aux-battles/internal/dbgen"
	"github.com/kericchhh/aux-battles/internal/media"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
)

type server struct {
	battles       *battle.Lobby
	queries       *dbgen.Queries
	pool          *pgxpool.Pool
	logger        *slog.Logger
	secureCookies bool
}

func New(queries *dbgen.Queries, pool *pgxpool.Pool, logger *slog.Logger, secureCookies bool) http.Handler {
	loginLimit := newIPLimiter(rate.Every(12*time.Second), 5)
	registerLimit := newIPLimiter(rate.Every(time.Minute), 3)
	app := &server{queries: queries, pool: pool, battles: battle.New(pool, media.FromEnv()), logger: logger, secureCookies: secureCookies}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", app.health)
	mux.HandleFunc("POST /battles", app.requireAuth(app.createBattle))
	mux.HandleFunc("POST /battles/join", app.requireAuth(app.joinBattle))
	mux.HandleFunc("GET /battles/{battleId}", app.requireAuth(app.getBattle))
	mux.HandleFunc("PUT /battles/{battleId}/lineup", app.requireAuth(app.submitLineup))
	mux.HandleFunc("GET /songs", app.listSongs)
	mux.HandleFunc("POST /songs", app.requireAuth(app.uploadSong))
	mux.HandleFunc("GET /songs/{id}", app.getSong)
	mux.HandleFunc("GET /songs/{id}/status", app.requireAuth(app.getSongStatus))
	mux.HandleFunc("GET /songs/{id}/audio/{part}", app.requireAuth(app.getSongAudio))
	mux.HandleFunc("POST /users", registerLimit.wrap(app.registerUser))
	mux.HandleFunc("POST /users/login", loginLimit.wrap(app.loginUser))
	mux.HandleFunc("POST /users/logout", app.logoutUser)
	mux.HandleFunc("GET /users/me", app.getCurrentUser)
	mux.Handle("GET /metrics", promhttp.Handler())
	return logRequests(logger, mux)
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok\n"))
}
