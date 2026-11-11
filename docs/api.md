# API

All responses are JSON. Errors look like `{"error": "message"}`.

## Signing in

There are no passwords. `POST /api/players` with a display name returns a
random **token**; send it on later requests as

```
Authorization: Bearer <token>
```

The server stores only a SHA-256 hash of the token. Names are 2–20 characters
and unique regardless of case. Endpoints marked 🔒 require a token.

## Objects

### Game

```json
{
  "id": "7f1c…",
  "mode": "online",
  "board": [[0,0,0,0,0,0,0], …6 rows…],
  "currentPlayer": 1,
  "status": "in_progress",
  "winner": 0,
  "moveCount": 0,
  "player1": { "id": "…", "name": "Alice" },
  "player2": null,
  "createdAt": "2026-11-11T15:00:00Z",
  "updatedAt": "2026-11-11T15:00:00Z",
  "finishedAt": null
}
```

- `mode`: `local` (two people on one screen; anyone with the link may move) or
  `online` (each seat belongs to a player who may only move on their turn).
- `board[row][col]`: 6 rows × 7 columns, row 0 is the **top**. `0` = empty, `1` = Red, `2` = Yellow.
- `status`: `in_progress`, `won` or `draw`. An online game with `player2: null` is waiting for an opponent.
- `winner`: `1` or `2` when `status` is `won`, otherwise `0`.
- `player1` is Red and always the creator (null for anonymous local games). `player2` is Yellow.

### Move

```json
{ "number": 1, "player": 1, "row": 5, "col": 3, "playedAt": "…" }
```

### Profile

```json
{ "id": "…", "name": "Alice", "createdAt": "…",
  "stats": { "played": 10, "wins": 6, "losses": 3, "draws": 1, "winRate": 0.6 } }
```

Stats count **finished online games** only.

## Endpoints

### Players

| Method | Path | Body | Success | Errors |
|---|---|---|---|---|
| POST | `/api/players` | `{"name": "Alice"}` | 201 `{"player": Profile, "token": "…"}` | 400 invalid name, 409 name taken |
| GET 🔒 | `/api/me` | | 200 Profile | 401 |
| GET 🔒 | `/api/me/games` | | 200 `{"games": [Game]}`, 20 most recent | 401 |
| GET | `/api/players/{id}` | | 200 Profile | 404 |
| GET | `/api/leaderboard` | | 200 `{"players": [{rank, id, name, ...stats}]}`, top 20 | |

The leaderboard is ordered by wins, then fewest losses, then most games played.

### Games

| Method | Path | Body | Success | Errors |
|---|---|---|---|---|
| POST | `/api/games` | `{"mode": "local" \| "online"}` | 201 Game | 400, 401 (online needs a token) |
| GET | `/api/games/{id}` | | 200 Game | 404 |
| POST 🔒 | `/api/games/{id}/join` | | 200 Game | 400 not online, 404, 409 full |
| POST | `/api/games/{id}/moves` | `{"column": 0-6}` | 200 Game | 400 bad column, 401/403 (online: not signed in / not a player), 404, 409 not your turn / column full / game over / waiting for opponent / concurrent move |
| GET | `/api/games/{id}/moves` | | 200 `{"moves": [Move]}` in order | 404 |
| GET | `/api/games/{id}/events` | | `text/event-stream` | 404 |

#### Live updates

`/api/games/{id}/events` is a [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
stream. It sends one `game` event with the current Game immediately, then
another whenever the game changes (a player joins or moves). A comment line
(`: ping`) is sent every 25 seconds to keep proxies from closing the connection.
No token is needed, so anyone with the link can watch.

Behind the scenes every write runs `pg_notify('game_events', <game id>)`, and
each backend replica `LISTEN`s on that channel, so updates reach viewers no
matter which replica they're connected to.

### Health

| Method | Path | Success | Errors |
|---|---|---|---|
| GET | `/healthz` | 200 (process is up) | |
| GET | `/readyz` | 200 (database reachable) | 503 |
