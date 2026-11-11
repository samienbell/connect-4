# API

All responses are JSON. Errors look like `{"error": "message"}`.

## Game object

```json
{
  "id": "7f1c…",
  "board": [[0,0,0,0,0,0,0], …6 rows…],
  "currentPlayer": 1,
  "status": "in_progress",
  "winner": 0,
  "moveCount": 0,
  "createdAt": "2026-11-11T15:00:00Z",
  "updatedAt": "2026-11-11T15:00:00Z"
}
```

- `board[row][col]`: 6 rows × 7 columns, row 0 is the **top**. `0` = empty, `1` = Red, `2` = Yellow.
- `status`: `in_progress`, `won` or `draw`.
- `winner`: `1` or `2` when `status` is `won`, otherwise `0`.

## Endpoints

| Method | Path                     | Body              | Success | Errors |
|--------|--------------------------|-------------------|---------|--------|
| POST   | `/api/games`             | none              | 201 game | |
| GET    | `/api/games/{id}`        | none              | 200 game | 404 |
| POST   | `/api/games/{id}/moves`  | `{"column": 0-6}` | 200 game | 400 bad column, 404, 409 column full / game over / concurrent move |
| GET    | `/healthz`               | none              | 200 (process is up) | |
| GET    | `/readyz`                | none              | 200 (database reachable) | 503 |

Moves are always played by `currentPlayer`; two players share one screen.
