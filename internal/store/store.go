package store

import (
	"context"
	"embed"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

//go:embed migrations/*.sql
var migrationFS embed.FS

type Store struct {
	pool *pgxpool.Pool
}

type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

func Connect(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	s := &Store{pool: pool}
	if err := s.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Migrate(ctx context.Context) error {
	sqlBytes, err := migrationFS.ReadFile("migrations/001_users.sql")
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, string(sqlBytes))
	return err
}

func (s *Store) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO users (username, password_hash) VALUES ($1, $2)`, username, passwordHash)
	return err
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `SELECT id, username, password_hash FROM users WHERE username = $1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetAppEnabled(ctx context.Context, stem string) (bool, bool, error) {
	var enabled bool
	err := s.pool.QueryRow(ctx, `SELECT enabled FROM app_preferences WHERE stem = $1`, stem).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, err
	}
	return enabled, true, nil
}

func (s *Store) SetAppEnabled(ctx context.Context, stem string, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_preferences (stem, enabled, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (stem) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()
	`, stem, enabled)
	return err
}

func (s *Store) ListAppPreferences(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT stem, enabled FROM app_preferences`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var stem string
		var enabled bool
		if err := rows.Scan(&stem, &enabled); err != nil {
			return nil, err
		}
		out[stem] = enabled
	}
	return out, rows.Err()
}
