// Package store persists players, games and moves in PostgreSQL.
package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"connect4/migrations"
)

// EventsChannel is the Postgres NOTIFY channel that carries the ID of every
// game that changes. See internal/events.
const EventsChannel = "game_events"

// migrationLockID is an arbitrary key for pg_advisory_lock so that only one
// replica runs migrations at a time.
const migrationLockID int64 = 4_242_001

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("game was updated by another request; reload and try again")
	ErrNameTaken  = errors.New("that name is already taken")
	ErrCannotJoin = errors.New("this game can no longer be joined")
)

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// validID reports whether id looks like a UUID. Checking first turns
// malformed IDs into ErrNotFound instead of a Postgres syntax error.
func validID(id string) bool { return uuidRE.MatchString(id) }

type Store struct {
	pool *pgxpool.Pool
}

// Open connects to PostgreSQL, retrying while the database starts up.
func Open(ctx context.Context, url string, log *slog.Logger) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	const attempts = 30
	for i := 1; ; i++ {
		err = pool.Ping(ctx)
		if err == nil {
			return &Store{pool: pool}, nil
		}
		if i == attempts {
			pool.Close()
			return nil, fmt.Errorf("connect to database: %w", err)
		}
		log.Warn("database not ready, retrying", "attempt", i, "err", err)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Migrate applies, in filename order, every embedded .sql file that isn't yet
// recorded in schema_migrations. Each file runs in its own transaction.
func (s *Store) Migrate(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID) //nolint:errcheck

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		var applied bool
		if err := conn.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", name,
		).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return err
		}
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(sql)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", name)
			return err
		})
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}
