CREATE TABLE IF NOT EXISTS games (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    board          JSONB       NOT NULL,
    current_player SMALLINT    NOT NULL CHECK (current_player IN (1, 2)),
    status         TEXT        NOT NULL DEFAULT 'in_progress'
                               CHECK (status IN ('in_progress', 'won', 'draw')),
    winner         SMALLINT    CHECK (winner IN (1, 2)),
    move_count     INT         NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS moves (
    game_id     UUID        NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    move_number INT         NOT NULL,
    player      SMALLINT    NOT NULL CHECK (player IN (1, 2)),
    row_idx     SMALLINT    NOT NULL CHECK (row_idx BETWEEN 0 AND 5),
    col_idx     SMALLINT    NOT NULL CHECK (col_idx BETWEEN 0 AND 6),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (game_id, move_number)
);
