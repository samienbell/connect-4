import { useEffect, useState } from 'react';
import { api } from '../api/client.js';
import { Link } from '../router.jsx';
import { COLORS, formatDate, isWaiting, seatOf } from '../lib/players.js';

function describe(game, player) {
  const seat = seatOf(game, player);
  const online = game.mode === 'online';
  const opponent = seat === 1 ? game.player2 : game.player1;

  const title = !online ? 'Game on this device' : opponent ? `vs ${opponent.name}` : 'Waiting for opponent';

  let result;
  let tone = '';
  if (game.status === 'won') {
    if (online) {
      result = game.winner === seat ? 'Won' : 'Lost';
      tone = game.winner === seat ? 'win' : 'loss';
    } else {
      result = `${COLORS[game.winner]} won`;
    }
  } else if (game.status === 'draw') {
    result = 'Draw';
  } else if (isWaiting(game)) {
    result = 'Waiting';
  } else if (online) {
    result = game.currentPlayer === seat ? 'Your turn' : 'Their turn';
    tone = game.currentPlayer === seat ? 'turn' : '';
  } else {
    result = 'In progress';
  }
  return { title, result, tone };
}

export default function RecentGames({ player }) {
  const [games, setGames] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;
    api
      .myGames()
      .then((d) => !cancelled && setGames(d.games))
      .catch((e) => !cancelled && setError(e.message));
    return () => {
      cancelled = true;
    };
  }, [player.id]);

  return (
    <section className="panel">
      <h2>Your games</h2>
      {error && <p className="error">{error}</p>}
      {!games && !error && <p className="hint">Loading…</p>}
      {games?.length === 0 && <p className="hint">No games yet.</p>}
      {games?.length > 0 && (
        <ul className="game-list">
          {games.map((g) => {
            const { title, result, tone } = describe(g, player);
            const finished = g.status !== 'in_progress';
            return (
              <li key={g.id}>
                <div>
                  <div className="title">{title}</div>
                  <div className="meta">
                    {formatDate(g.updatedAt)} · {g.moveCount} moves
                  </div>
                </div>
                <span className={`badge ${tone}`}>{result}</span>
                <div className="links">
                  {!finished && <Link to={`/game/${g.id}`}>Open</Link>}
                  {g.moveCount > 0 && <Link to={`/replay/${g.id}`}>Replay</Link>}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
