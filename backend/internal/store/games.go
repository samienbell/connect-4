package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"connect4/internal/game"
)

// Move is one entry in a game's history.
type Move struct {
	Number   int         `json:"number"` // 1-based
	Player   game.Player `json:"player"`
	Row      int         `json:"row"` // 0 is the top row
	Col      int         `json:"col"`
	PlayedAt time.Time   `json:"playedAt"`
}

const gameSelect = `
	SELECT g.id::text, g.mode, g.board, g.current_player, g.status, g.winner, g.move_count,
	       g.created_at, g.updated_at, g.finished_at,
	       g.player1_id::text, p1.name, g.player2_id::text, p2.name
	FROM games g
	LEFT JOIN players p1 ON p1.id = g.player1_id
	LEFT JOIN players p2 ON p2.id = g.player2_id`

func scanGame(row pgx.Row) (*game.Game, error) {
	var (
		g                          game.Game
		mode, status               string
		board                      []byte
		current                    int16
		winner                     *int16
		p1ID, p1Name, p2ID, p2Name *string
	)
	err := row.Scan(&g.ID, &mode, &board, &current, &status, &winner, &g.MoveCount,
		&g.CreatedAt, &g.UpdatedAt, &g.FinishedAt, &p1ID, &p1Name, &p2ID, &p2Name)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(board, &g.Board); err != nil {
		return nil, fmt.Errorf("decode board: %w", err)
	}
	g.Mode = game.Mode(mode)
	g.CurrentPlayer = game.Player(current)
	g.Status = game.Status(status)
	if winner != nil {
		g.Winner = game.Player(*winner)
	}
	g.Player1 = playerRef(p1ID, p1Name)
	g.Player2 = playerRef(p2ID, p2Name)
	return &g, nil
}

func playerRef(id, name *string) *game.PlayerRef {
	if id == nil {
		return nil
	}
	ref := &game.PlayerRef{ID: *id}
	if name != nil {
		ref.Name = *name
	}
	return ref
}

func (s *Store) CreateGame(ctx context.Context, g *game.Game) error {
	board, err := json.Marshal(g.Board)
	if err != nil {
		return err
	}
	var player1 *string
	if g.Player1 != nil {
		player1 = &g.Player1.ID
	}
	return s.pool.QueryRow(ctx, `
		INSERT INTO games (mode, board, current_player, status, move_count, player1_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, created_at, updated_at`,
		string(g.Mode), string(board), int16(g.CurrentPlayer), string(g.Status), g.MoveCount, player1,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

func (s *Store) GetGame(ctx context.Context, id string) (*game.Game, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	g, err := scanGame(s.pool.QueryRow(ctx, gameSelect+" WHERE g.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return g, err
}

// PlayerGames returns the player's most recently active games, newest first.
func (s *Store) PlayerGames(ctx context.Context, playerID string, limit int) ([]*game.Game, error) {
	rows, err := s.pool.Query(ctx, gameSelect+`
		WHERE g.player1_id = $1 OR g.player2_id = $1
		ORDER BY g.updated_at DESC
		LIMIT $2`, playerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	games := []*game.Game{}
	for rows.Next() {
		g, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

// JoinGame puts playerID in the empty second seat of an online game. It
// returns ErrCannotJoin if the seat was taken first or the game isn't joinable.
func (s *Store) JoinGame(ctx context.Context, gameID, playerID string) error {
	if !validID(gameID) {
		return ErrNotFound
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE games SET player2_id = $2, updated_at = now()
			WHERE id = $1 AND mode = 'online' AND player2_id IS NULL AND player1_id <> $2`,
			gameID, playerID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrCannotJoin
		}
		return notify(ctx, tx, gameID)
	})
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
			SET board = $2, current_player = $3, status = $4, winner = $5, move_count = $6,
			    updated_at = now(),
			    finished_at = CASE WHEN $4 = 'in_progress' THEN NULL ELSE now() END
			WHERE id = $1 AND move_count = $7
			RETURNING updated_at, finished_at`,
			g.ID, string(board), int16(g.CurrentPlayer), string(g.Status), winner,
			g.MoveCount, g.MoveCount-1,
		).Scan(&g.UpdatedAt, &g.FinishedAt)
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
		if err != nil {
			return err
		}
		return notify(ctx, tx, g.ID)
	})
}

// ListMoves returns a game's moves in the order they were played.
func (s *Store) ListMoves(ctx context.Context, gameID string) ([]Move, error) {
	if !validID(gameID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		SELECT move_number, player, row_idx, col_idx, created_at
		FROM moves WHERE game_id = $1 ORDER BY move_number`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	moves := []Move{}
	for rows.Next() {
		var (
			m                Move
			player, row, col int16
		)
		if err := rows.Scan(&m.Number, &player, &row, &col, &m.PlayedAt); err != nil {
			return nil, err
		}
		m.Player, m.Row, m.Col = game.Player(player), int(row), int(col)
		moves = append(moves, m)
	}
	return moves, rows.Err()
}

// notify tells every backend replica that the game changed. Postgres delivers
// it only if the surrounding transaction commits.
func notify(ctx context.Context, tx pgx.Tx, gameID string) error {
	_, err := tx.Exec(ctx, "SELECT pg_notify($1, $2)", EventsChannel, gameID)
	return err
}
