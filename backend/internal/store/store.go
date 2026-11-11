// Package store persists games and moves in PostgreSQL.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"connect4/internal/game"
	"connect4/migrations"
)

var (
	ErrNotFound = errors.New("game not found")
	ErrConflict = errors.New("game was updated by another request; reload and try again")
)

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Store struct {
	pool *pgxpool.Pool
}

// Move is one row of the moves history table.
type Move struct {
	Number int
	Player game.Player
	Row    int
	Col    int
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

// Migrate applies every embedded .sql file in filename order.
func (s *Store) Migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		sql, err := fs.ReadFile(migrations.FS, e.Name())
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
	}
	return nil
}

func (s *Store) CreateGame(ctx context.Context, g *game.Game) error {
	board, err := json.Marshal(g.Board)
	if err != nil {
		return err
	}
	return s.pool.QueryRow(ctx, `
		INSERT INTO games (board, current_player, status, move_count)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, created_at, updated_at`,
		string(board), int16(g.CurrentPlayer), string(g.Status), g.MoveCount,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

func (s *Store) GetGame(ctx context.Context, id string) (*game.Game, error) {
	if !uuidRE.MatchString(id) {
		return nil, ErrNotFound
	}
	var (
		g       game.Game
		board   []byte
		current int16
		status  string
		winner  *int16
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, board, current_player, status, winner, move_count, created_at, updated_at
		FROM games WHERE id = $1`, id,
	).Scan(&g.ID, &board, &current, &status, &winner, &g.MoveCount, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(board, &g.Board); err != nil {
		return nil, fmt.Errorf("decode board: %w", err)
	}
	g.CurrentPlayer = game.Player(current)
	g.Status = game.Status(status)
	if winner != nil {
		g.Winner = game.Player(*winner)
	}
	return &g, nil
}

// SaveMove stores the game's new state and records the move. g must already
// have had the move applied. It returns ErrConflict if another request saved a
// move to this game first.
func (s *Store) SaveMove(ctx context.Context, g *game.Game, m Move) error {
	board, err := json.Marshal(g.Board)
	if err != nil {
		return err
	}
	var winner *int16
	if g.Winner != game.Empty {
		w := int16(g.Winner)
		winner = &w
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			UPDATE games
			SET board = $2, current_player = $3, status = $4, winner = $5,
			    move_count = $6, updated_at = now()
			WHERE id = $1 AND move_count = $7
			RETURNING updated_at`,
			g.ID, string(board), int16(g.CurrentPlayer), string(g.Status), winner,
			g.MoveCount, g.MoveCount-1,
		).Scan(&g.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO moves (game_id, move_number, player, row_idx, col_idx)
			VALUES ($1, $2, $3, $4, $5)`,
			g.ID, m.Number, int16(m.Player), int16(m.Row), int16(m.Col),
		)
		return err
	})
}
