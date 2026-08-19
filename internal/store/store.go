package store

import (
	"context"
	"embed"
	"errors"

	"github.com/ArminDashti/local-apps-manager-api/internal/runmode"
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

type AppPreference struct {
	LocalEnabled   bool
	DockerEnabled  bool
	PublicEnabled  bool
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
	for _, name := range []string{
		"migrations/001_users.sql",
		"migrations/002_app_run_mode.sql",
		"migrations/003_app_mode_flags.sql",
	} {
		sqlBytes, err := migrationFS.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, string(sqlBytes)); err != nil {
			return err
		}
	}
	return nil
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

func (s *Store) GetAppPreference(ctx context.Context, stem string) (AppPreference, bool, error) {
	var pref AppPreference
	err := s.pool.QueryRow(ctx, `
		SELECT local_enabled, docker_enabled, public_enabled
		FROM app_preferences WHERE stem = $1
	`, stem).Scan(&pref.LocalEnabled, &pref.DockerEnabled, &pref.PublicEnabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AppPreference{}, false, nil
		}
		return AppPreference{}, false, err
	}
	return pref, true, nil
}

func (s *Store) SetModeEnabled(ctx context.Context, stem string, mode runmode.Mode, enabled bool) error {
	pref, _, err := s.GetAppPreference(ctx, stem)
	if err != nil {
		return err
	}
	switch mode {
	case runmode.Local:
		pref.LocalEnabled = enabled
	case runmode.LocalDocker:
		pref.DockerEnabled = enabled
	case runmode.Server:
		pref.PublicEnabled = enabled
	default:
		return errors.New("invalid mode")
	}
	return s.SetAppPreference(ctx, stem, pref)
}

func (s *Store) SetAppPreference(ctx context.Context, stem string, pref AppPreference) error {
	legacyMode := runmode.Default()
	legacyEnabled := false
	if pref.PublicEnabled {
		legacyMode = runmode.Server
		legacyEnabled = true
	} else if pref.DockerEnabled {
		legacyMode = runmode.LocalDocker
		legacyEnabled = true
	} else if pref.LocalEnabled {
		legacyMode = runmode.Local
		legacyEnabled = true
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_preferences (stem, local_enabled, docker_enabled, public_enabled, enabled, run_mode, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (stem) DO UPDATE SET
			local_enabled = EXCLUDED.local_enabled,
			docker_enabled = EXCLUDED.docker_enabled,
			public_enabled = EXCLUDED.public_enabled,
			enabled = EXCLUDED.enabled,
			run_mode = EXCLUDED.run_mode,
			updated_at = NOW()
	`, stem, pref.LocalEnabled, pref.DockerEnabled, pref.PublicEnabled, legacyEnabled, string(legacyMode))
	return err
}

func (s *Store) ListAppPreferences(ctx context.Context) (map[string]AppPreference, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT stem, local_enabled, docker_enabled, public_enabled FROM app_preferences
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]AppPreference{}
	for rows.Next() {
		var stem string
		var pref AppPreference
		if err := rows.Scan(&stem, &pref.LocalEnabled, &pref.DockerEnabled, &pref.PublicEnabled); err != nil {
			return nil, err
		}
		out[stem] = pref
	}
	return out, rows.Err()
}
