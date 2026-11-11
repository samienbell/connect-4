package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Player struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// Stats summarises a player's finished online games.
type Stats struct {
	Played  int     `json:"played"`
	Wins    int     `json:"wins"`
	Losses  int     `json:"losses"`
	Draws   int     `json:"draws"`
	WinRate float64 `json:"winRate"` // 0 to 1
}

type LeaderboardEntry struct {
	Rank int    `json:"rank"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Stats
}

func (s *Store) CreatePlayer(ctx context.Context, name string, tokenHash []byte) (*Player, error) {
	p := Player{Name: name}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO players (name, token_hash) VALUES ($1, $2)
		RETURNING id::text, created_at`, name, tokenHash,
	).Scan(&p.ID, &p.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return nil, ErrNameTaken
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) PlayerByTokenHash(ctx context.Context, tokenHash []byte) (*Player, error) {
	return s.getPlayer(ctx, "token_hash = $1", tokenHash)
}

func (s *Store) PlayerByID(ctx context.Context, id string) (*Player, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	return s.getPlayer(ctx, "id = $1", id)
}

func (s *Store) getPlayer(ctx context.Context, where string, arg any) (*Player, error) {
	var p Player
	err := s.pool.QueryRow(ctx,
		"SELECT id::text, name, created_at FROM players WHERE "+where, arg,
	).Scan(&p.ID, &p.Name, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// resultsCTE turns every finished online game into two rows, one per seat,
// each labelled win, loss or draw from that player's point of view.
const resultsCTE = `
	WITH finished AS (
		SELECT player1_id, player2_id, status, winner
		FROM games
		WHERE mode = 'online' AND status IN ('won', 'draw') AND player2_id IS NOT NULL
	), results AS (
		SELECT player1_id AS player_id,
		       CASE WHEN status = 'draw' THEN 'draw' WHEN winner = 1 THEN 'win' ELSE 'loss' END AS result
		FROM finished
		UNION ALL
		SELECT player2_id,
		       CASE WHEN status = 'draw' THEN 'draw' WHEN winner = 2 THEN 'win' ELSE 'loss' END
		FROM finished
	)`

const statsColumns = `
	count(*)                                  AS played,
	count(*) FILTER (WHERE r.result = 'win')  AS wins,
	count(*) FILTER (WHERE r.result = 'loss') AS losses,
	count(*) FILTER (WHERE r.result = 'draw') AS draws`

func (s *Store) PlayerStats(ctx context.Context, playerID string) (Stats, error) {
	var st Stats
	err := s.pool.QueryRow(ctx,
		resultsCTE+" SELECT "+statsColumns+" FROM results r WHERE r.player_id = $1", playerID,
	).Scan(&st.Played, &st.Wins, &st.Losses, &st.Draws)
	st.WinRate = winRate(st)
	return st, err
}

// Leaderboard ranks players by wins, then fewest losses, then most games.
func (s *Store) Leaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	rows, err := s.pool.Query(ctx, resultsCTE+`
		SELECT p.id::text, p.name,`+statsColumns+`
		FROM results r
		JOIN players p ON p.id = r.player_id
		GROUP BY p.id, p.name
		ORDER BY wins DESC, losses ASC, played DESC, p.name ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []LeaderboardEntry{}
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Played, &e.Wins, &e.Losses, &e.Draws); err != nil {
			return nil, err
		}
		e.Rank = len(entries) + 1
		e.WinRate = winRate(e.Stats)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func winRate(s Stats) float64 {
	if s.Played == 0 {
		return 0
	}
	return float64(s.Wins) / float64(s.Played)
}
