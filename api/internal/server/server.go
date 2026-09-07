package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ry023/semdiff-sample-app/api/internal/database"
)

const (
	sessionCookie = "todo_session"
	sessionTTL    = 7 * 24 * time.Hour
)

type Server struct {
	store        *database.Store
	cookieSecure bool
}

type contextKey string

const userContextKey contextKey = "user"

func New(store *database.Store, cookieSecure bool) http.Handler {
	s := &Server{store: store, cookieSecure: cookieSecure}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.health)
	mux.HandleFunc("/api/auth/register", s.register)
	mux.HandleFunc("/api/auth/login", s.login)
	mux.HandleFunc("/api/auth/logout", s.logout)
	mux.HandleFunc("/api/auth/me", s.me)
	mux.HandleFunc("/api/todos", s.todos)
	mux.HandleFunc("/api/todos/", s.todoByID)

	return s.withJSON(s.withUser(mux))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var input credentialsRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if len(input.Username) < 3 || len(input.Username) > 50 {
		writeError(w, http.StatusBadRequest, "username must be between 3 and 50 characters")
		return
	}
	if len(input.Password) < 4 || len(input.Password) > 100 {
		writeError(w, http.StatusBadRequest, "password must be between 4 and 100 characters")
		return
	}

	user, err := s.store.CreateUser(r.Context(), input.Username, hashPassword(input.Password))
	if errors.Is(err, database.ErrConflict) {
		writeError(w, http.StatusConflict, "username is already registered")
		return
	}
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var input credentialsRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	user, storedHash, err := s.store.FindUserByUsername(r.Context(), strings.TrimSpace(input.Username))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		writeServerError(w, err)
		return
	}
	if !passwordMatches(storedHash, input.Password) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := newToken()
	if err != nil {
		writeServerError(w, err)
		return
	}
	if err := s.store.CreateSession(r.Context(), token, user.ID, time.Now().Add(sessionTTL)); err != nil {
		writeServerError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(sessionTTL),
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if err := s.store.DeleteSession(r.Context(), cookie.Value); err != nil {
			writeServerError(w, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type todoRequest struct {
	Title     *string `json:"title"`
	Completed *bool   `json:"completed"`
}

func (s *Server) todos(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		todos, err := s.store.ListTodos(r.Context(), user.ID)
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, todos)
	case http.MethodPost:
		var input todoRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		if input.Title == nil {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		title := strings.TrimSpace(*input.Title)
		if title == "" || len(title) > 200 {
			writeError(w, http.StatusBadRequest, "title must be between 1 and 200 characters")
			return
		}
		todo, err := s.store.CreateTodo(r.Context(), user.ID, title)
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, todo)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) todoByID(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	idText := strings.TrimPrefix(r.URL.Path, "/api/todos/")
	todoID, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || todoID < 1 {
		writeError(w, http.StatusBadRequest, "invalid todo id")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var input todoRequest
		if !decodeJSON(w, r, &input) {
			return
		}
		if input.Title == nil && input.Completed == nil {
			writeError(w, http.StatusBadRequest, "title or completed is required")
			return
		}
		if input.Title != nil {
			title := strings.TrimSpace(*input.Title)
			if title == "" || len(title) > 200 {
				writeError(w, http.StatusBadRequest, "title must be between 1 and 200 characters")
				return
			}
			input.Title = &title
		}
		todo, err := s.store.UpdateTodo(r.Context(), user.ID, todoID, input.Title, input.Completed)
		if errors.Is(err, database.ErrNotFound) {
			writeError(w, http.StatusNotFound, "todo not found")
			return
		}
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, todo)
	case http.MethodDelete:
		err := s.store.DeleteTodo(r.Context(), user.ID, todoID)
		if errors.Is(err, database.ErrNotFound) {
			writeError(w, http.StatusNotFound, "todo not found")
			return
		}
		if err != nil {
			writeServerError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodPatch, http.MethodDelete)
	}
}

func (s *Server) withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookie); err == nil && cookie.Value != "" {
			if user, err := s.store.FindUserBySession(r.Context(), cookie.Value); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), userContextKey, user))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func currentUser(ctx context.Context) (database.User, bool) {
	user, ok := ctx.Value(userContextKey).(database.User)
	return user, ok
}

func hashPassword(password string) string {
	digest := sha256.Sum256([]byte(password))
	return hex.EncodeToString(digest[:])
}

func passwordMatches(storedHash, password string) bool {
	return storedHash != "" && storedHash == hashPassword(password)
}

func newToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeServerError(w http.ResponseWriter, err error) {
	fmt.Printf("internal server error: %v\n", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
