package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (User, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	if err != nil {
		if isConstraintError(err) {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Username: username}, nil
}

func (s *Store) FindUserByUsername(ctx context.Context, username string) (User, string, error) {
	var user User
	var passwordHash string
	err := s.db.QueryRowContext(ctx,
		"SELECT id, username, password_hash FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	if err != nil {
		return User{}, "", err
	}
	return user, passwordHash, nil
}

func (s *Store) FindUserBySession(ctx context.Context, token string) (User, error) {
	var user User
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username
		FROM users u
		JOIN sessions s ON s.user_id = u.id
		WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP
	`, token).Scan(&user.ID, &user.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Store) CreateSession(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt.UTC(),
	)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (s *Store) ListTodos(ctx context.Context, userID int64) ([]Todo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, completed, created_at, updated_at
		FROM todos
		WHERE user_id = ?
		ORDER BY completed ASC, created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := make([]Todo, 0)
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return todos, nil
}

func (s *Store) CreateTodo(ctx context.Context, userID int64, title string) (Todo, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO todos (user_id, title) VALUES (?, ?)",
		userID, title,
	)
	if err != nil {
		return Todo{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Todo{}, err
	}
	return s.GetTodo(ctx, userID, id)
}

func (s *Store) GetTodo(ctx context.Context, userID, todoID int64) (Todo, error) {
	var todo Todo
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, completed, created_at, updated_at
		FROM todos
		WHERE id = ? AND user_id = ?
	`, todoID, userID).Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Todo{}, ErrNotFound
	}
	if err != nil {
		return Todo{}, err
	}
	return todo, nil
}

func (s *Store) UpdateTodo(ctx context.Context, userID, todoID int64, title *string, completed *bool) (Todo, error) {
	current, err := s.GetTodo(ctx, userID, todoID)
	if err != nil {
		return Todo{}, err
	}
	if title != nil {
		current.Title = *title
	}
	if completed != nil {
		current.Completed = *completed
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE todos
		SET title = ?, completed = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, current.Title, current.Completed, todoID, userID)
	if err != nil {
		return Todo{}, err
	}
	return s.GetTodo(ctx, userID, todoID)
}

func (s *Store) DeleteTodo(ctx context.Context, userID, todoID int64) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM todos WHERE id = ? AND user_id = ?",
		todoID, userID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func isConstraintError(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") || contains(err.Error(), "constraint failed"))
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
