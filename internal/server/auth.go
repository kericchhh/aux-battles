package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kericchhh/aux-battles/internal/dbgen"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)
var errUnauthenticated = errors.New("unauthenticated")

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (in *registerRequest) normalizeAndValidate() error {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	if !usernamePattern.MatchString(in.Username) {
		return errors.New("invalid username")
	}
	address, err := mail.ParseAddress(in.Email)
	if err != nil || address.Address != in.Email || len(in.Email) > 255 {
		return errors.New("invalid email")
	}
	if len(in.Password) > 1024 {
		return errors.New("password is too long")
	}
	if utf8.RuneCountInString(in.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

type registerResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (s *server) registerUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var input registerRequest
	if !readJSON(w, r, &input) {
		return
	}
	if err := input.normalizeAndValidate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	passwordHash, err := argon2id.CreateHash(input.Password, argon2id.DefaultParams)
	if err != nil {
		s.logger.Error("hash password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not register user")
		return
	}
	user, err := s.queries.CreateUser(r.Context(), dbgen.CreateUserParams{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSONError(w, http.StatusConflict, "username or email unavailable")
			return
		}
		s.logger.Error("create user", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not register user")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.MarshalWrite(w, registerResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
	}); err != nil {
		s.logger.Error("write registration response", "error", err)
	}
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (s *server) loginUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var input loginRequest
	if !readJSON(w, r, &input) {
		return
	}
	identifier := strings.ToLower(strings.TrimSpace(input.Identifier))
	if identifier == "" || len(identifier) > 255 || input.Password == "" || len(input.Password) > 1024 {
		writeJSONError(w, http.StatusBadRequest, "identifier and password are required")
		return
	}
	user, err := s.queries.GetUserByIdentifier(r.Context(), identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		s.logger.Error("find user for login", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not log in")
		return
	}
	match, err := argon2id.ComparePasswordAndHash(input.Password, user.PasswordHash)
	if err != nil {
		s.logger.Error("verify password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not log in")
		return
	}
	if !match {
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, tokenHash, err := newSessionToken()
	if err != nil {
		s.logger.Error("generate session token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not login")
		return
	}
	const sessionDuration = 7 * 24 * time.Hour
	expiresAt := time.Now().Add(sessionDuration)
	err = s.queries.CreateSession(r.Context(), dbgen.CreateSessionParams{
		TokenHash: tokenHash,
		UserID:    user.ID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		s.logger.Error("create session", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not log in")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.sessionCookieName(),
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, registerResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
	}); err != nil {
		s.logger.Error("write login response", "error", err)
	}
}

func (s *server) logoutUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if cookie, err := r.Cookie(s.sessionCookieName()); err == nil {
		sum := sha256.Sum256([]byte(cookie.Value))
		if err := s.queries.DeleteSession(r.Context(), hex.EncodeToString(sum[:])); err != nil {
			s.logger.Error("delete session", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "could not log out")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.sessionCookieName(),
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) currentUser(r *http.Request) (dbgen.GetValidSessionRow, error) {
	cookie, err := r.Cookie(s.sessionCookieName())
	if err != nil || cookie.Value == "" {
		return dbgen.GetValidSessionRow{}, errUnauthenticated
	}
	sum := sha256.Sum256([]byte(cookie.Value))
	user, err := s.queries.GetValidSession(r.Context(), hex.EncodeToString(sum[:]))
	if errors.Is(err, pgx.ErrNoRows) {
		return dbgen.GetValidSessionRow{}, errUnauthenticated
	}
	return user, err
}

func (s *server) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	user, err := s.currentUser(r)
	if errors.Is(err, errUnauthenticated) {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	if err != nil {
		s.logger.Error("get current user", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "could not get user")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.MarshalWrite(w, struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}); err != nil {
		s.logger.Error("write current user response", "error", err)
	}
}

func (s *server) sessionCookieName() string {
	if s.secureCookies {
		return "__Host-aux_session"
	}
	return "aux_session"
}

func newSessionToken() (token, tokenHash string, err error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(random[:])
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}
