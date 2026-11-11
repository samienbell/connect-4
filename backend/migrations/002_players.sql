-- Players are identified by a random token held in their browser. Only a
-- SHA-256 hash of the token is stored.
CREATE TABLE IF NOT EXISTS players (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    token_hash BYTEA       NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Names are unique regardless of case, so "alice" can't pose as "Alice".
CREATE UNIQUE INDEX IF NOT EXISTS players_name_key ON players (lower(name));

ALTER TABLE games
    ADD COLUMN IF NOT EXISTS mode        TEXT NOT NULL DEFAULT 'local' CHECK (mode IN ('local', 'online')),
    ADD COLUMN IF NOT EXISTS player1_id  UUID REFERENCES players (id),
    ADD COLUMN IF NOT EXISTS player2_id  UUID REFERENCES players (id),
    ADD COLUMN IF NOT EXISTS finished_at TIMESTAMPTZ;

ALTER TABLE games
    ADD CONSTRAINT games_distinct_players CHECK (player2_id IS NULL OR player2_id <> player1_id);

CREATE INDEX IF NOT EXISTS games_player1_idx ON games (player1_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS games_player2_idx ON games (player2_id, updated_at DESC);
