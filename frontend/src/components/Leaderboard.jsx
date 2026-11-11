import { useEffect, useState } from 'react';
import { api } from '../api/client.js';

export default function Leaderboard({ highlightId }) {
  const [players, setPlayers] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;
    api
      .leaderboard()
      .then((d) => !cancelled && setPlayers(d.players))
      .catch((e) => !cancelled && setError(e.message));
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <section className="panel">
      <h2>Leaderboard</h2>
      {error && <p className="error">{error}</p>}
      {!players && !error && <p className="hint">Loading…</p>}
      {players?.length === 0 && <p className="hint">No finished online games yet. Be the first!</p>}
      {players?.length > 0 && (
        <div className="table-wrap">
          <table className="leaderboard">
            <thead>
              <tr>
                <th scope="col">#</th>
                <th scope="col">Player</th>
                <th scope="col" title="Wins">W</th>
                <th scope="col" title="Losses">L</th>
                <th scope="col" title="Draws">D</th>
                <th scope="col">Win %</th>
              </tr>
            </thead>
            <tbody>
              {players.map((p) => (
                <tr key={p.id} className={p.id === highlightId ? 'me' : undefined}>
                  <td>{p.rank}</td>
                  <td className="name">{p.name}</td>
                  <td>{p.wins}</td>
                  <td>{p.losses}</td>
                  <td>{p.draws}</td>
                  <td>{Math.round(p.winRate * 100)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
