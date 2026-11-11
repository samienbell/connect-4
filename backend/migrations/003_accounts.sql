-- Accounts: players get a password, and sign-ins become sessions so a player
-- can be signed in on several devices and sign out of one.

CREATE TABLE sessions (
    token_hash BYTEA       PRIMARY KEY, -- SHA-256 of the token held by the browser
    player_id  UUID        NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_player_idx ON sessions (player_id);

-- Keep existing browsers signed in: their old token becomes a session.
INSERT INTO sessions (token_hash, player_id, expires_at)
SELECT token_hash, id, now() + interval '30 days' FROM players;

ALTER TABLE players
    DROP COLUMN token_hash,
    -- Argon2id hash in PHC format (includes the per-password salt).
    -- NULL for players created before accounts existed.
    ADD COLUMN password_hash     TEXT,
    -- Consecutive failed logins, used to slow down password guessing.
    ADD COLUMN failed_logins     INT NOT NULL DEFAULT 0,
    ADD COLUMN last_failed_login TIMESTAMPTZ;
