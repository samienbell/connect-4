# API

All responses are JSON. Errors look like `{"error": "message"}`.

## Accounts and sessions

Register or log in with a name and password. Either returns a session
**token**; send it on later requests as

```
Authorization: Bearer <token>
```

Endpoints marked 🔒 require a token.

- **Names** are 2–20 characters (letters, numbers, space, `_ . -`) and unique
  regardless of case.
- **Passwords** are 8–128 characters and can't equal the name. They're stored
  as **Argon2id** hashes (19 MiB, 2 iterations, 1 lane, the OWASP minimum) with
  a random 16-byte salt per password, in PHC format:
  `$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>`. If the parameters are raised
  later, a user's hash is upgraded on their next successful login.
- **Sessions** last 30 days. The database stores only a SHA-256 hash of each
  token. Logging out ends that session; changing your password ends all your
  other sessions.
- **Throttling:** after 5 wrong passwords in a row, the next attempt must wait
  30 s, then 1, 2, 4, 8 and at most 15 minutes. While waiting, login returns
  `429` with a `Retry-After` header, even for the right password. A successful
  login resets the count. The count lives in the database, so it applies
  across all backend replicas.
- A wrong password and an unknown name give the same `401` message and take
  the same time, so login doesn't reveal which names exist. (Names are public
  on the leaderboard anyway.)
- Players created before accounts existed have no password (`hasPassword:
  false`). They stay signed in on their original browser and can set a
  password with `PUT /api/me/password`.

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
{ "id": "…", "name": "Alice", "createdAt": "…", "hasPassword": true,
  "stats": { "played": 10, "wins": 6, "losses": 3, "draws": 1, "winRate": 0.6 } }
```

Stats count **finished online games** only.

## Endpoints

### Accounts

| Method | Path | Body | Success | Errors |
|---|---|---|---|---|
| POST | `/api/auth/register` | `{"name", "password"}` | 201 `{"player": Profile, "token"}` | 400 invalid name/password, 409 name taken |
| POST | `/api/auth/login` | `{"name", "password"}` | 200 `{"player": Profile, "token"}` | 400, 401 incorrect name or password, 429 too many attempts |
| POST 🔒 | `/api/auth/logout` | | 204 | 401 |
| PUT 🔒 | `/api/me/password` | `{"currentPassword", "newPassword"}` | 204 (other sessions signed out) | 400 invalid new password, 401, 403 wrong current password |

`currentPassword` is ignored for players who don't have a password yet.

### Players

| Method | Path | Body | Success | Errors |
|---|---|---|---|---|
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
