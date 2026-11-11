package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Player struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"createdAt"`
	HasPassword bool      `json:"hasPassword"` // false for players from before accounts
}

// Credentials is what the login check needs. Never send it to clients.
type Credentials struct {
	Player
	PasswordHash    *string // nil if the player has no password yet
	FailedLogins    int
	LastFailedLogin *time.Time
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

// execer is satisfied by both the pool and a transaction.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Register creates a player with a password and signs them in with a new
// session, in one transaction.
func (s *Store) Register(ctx context.Context, name, passwordHash string, tokenHash []byte, expires time.Time) (*Player, error) {
	p := Player{Name: name, HasPassword: true}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO players (name, password_hash) VALUES ($1, $2)
			RETURNING id::text, created_at`, name, passwordHash,
		).Scan(&p.ID, &p.CreatedAt)
		if err != nil {
			return err
		}
		return insertSession(ctx, tx, p.ID, tokenHash, expires)
	})
	if isUniqueViolation(err) {
		return nil, ErrNameTaken
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

const playerColumns = "p.id::text, p.name, p.created_at, p.password_hash IS NOT NULL"

func scanPlayer(row pgx.Row) (*Player, error) {
	var p Player
	err := row.Scan(&p.ID, &p.Name, &p.CreatedAt, &p.HasPassword)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) PlayerByID(ctx context.Context, id string) (*Player, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	return scanPlayer(s.pool.QueryRow(ctx, "SELECT "+playerColumns+" FROM players p WHERE p.id = $1", id))
}

// PlayerBySession returns the player signed in with this session token, if
// the session exists and hasn't expired.
func (s *Store) PlayerBySession(ctx context.Context, tokenHash []byte) (*Player, error) {
	return scanPlayer(s.pool.QueryRow(ctx, `
		SELECT `+playerColumns+`
		FROM sessions s JOIN players p ON p.id = s.player_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash))
}

const credentialColumns = playerColumns + ", p.password_hash, p.failed_logins, p.last_failed_login"

func scanCredentials(row pgx.Row) (*Credentials, error) {
	var c Credentials
	err := row.Scan(&c.ID, &c.Name, &c.CreatedAt, &c.HasPassword, &c.PasswordHash, &c.FailedLogins, &c.LastFailedLogin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CredentialsByName looks a player up by name, ignoring case.
func (s *Store) CredentialsByName(ctx context.Context, name string) (*Credentials, error) {
	return scanCredentials(s.pool.QueryRow(ctx,
		"SELECT "+credentialColumns+" FROM players p WHERE lower(p.name) = lower($1)", name))
}

func (s *Store) CredentialsByID(ctx context.Context, id string) (*Credentials, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	return scanCredentials(s.pool.QueryRow(ctx,
		"SELECT "+credentialColumns+" FROM players p WHERE p.id = $1", id))
}

func (s *Store) SetPassword(ctx context.Context, playerID, passwordHash string) error {
	_, err := s.pool.Exec(ctx, "UPDATE players SET password_hash = $2 WHERE id = $1", playerID, passwordHash)
	return err
}

func (s *Store) RecordLoginFailure(ctx context.Context, playerID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE players SET failed_logins = failed_logins + 1, last_failed_login = now()
		WHERE id = $1`, playerID)
	return err
}

func (s *Store) ResetLoginFailures(ctx context.Context, playerID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE players SET failed_logins = 0, last_failed_login = NULL
		WHERE id = $1 AND failed_logins <> 0`, playerID)
	return err
}

// CreateSession signs a player in. It also clears out their expired sessions.
func (s *Store) CreateSession(ctx context.Context, playerID string, tokenHash []byte, expires time.Time) error {
	if _, err := s.pool.Exec(ctx,
		"DELETE FROM sessions WHERE player_id = $1 AND expires_at <= now()", playerID); err != nil {
		return err
	}
	return insertSession(ctx, s.pool, playerID, tokenHash, expires)
}

func insertSession(ctx context.Context, db execer, playerID string, tokenHash []byte, expires time.Time) error {
	_, err := db.Exec(ctx,
		"INSERT INTO sessions (token_hash, player_id, expires_at) VALUES ($1, $2, $3)",
		tokenHash, playerID, expires)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE token_hash = $1", tokenHash)
	return err
}

// DeleteOtherSessions signs the player out everywhere except the session
// identified by keepTokenHash.
func (s *Store) DeleteOtherSessions(ctx context.Context, playerID string, keepTokenHash []byte) error {
	_, err := s.pool.Exec(ctx,
		"DELETE FROM sessions WHERE player_id = $1 AND token_hash <> $2", playerID, keepTokenHash)
	return err
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
