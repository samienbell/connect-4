import { useEffect, useState } from 'react';
import { api } from '../api/client.js';
import { navigate } from '../router.jsx';
import { usePlayer } from '../hooks/usePlayer.jsx';
import NameForm from '../components/NameForm.jsx';
import StatsCard from '../components/StatsCard.jsx';
import RecentGames from '../components/RecentGames.jsx';
import Leaderboard from '../components/Leaderboard.jsx';

export default function Lobby() {
  const { player, loading, refresh } = usePlayer();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);

  // Stats may have changed since we last looked (e.g. a game just ended).
  useEffect(() => {
    refresh();
  }, [refresh]);

  const start = async (mode) => {
    setBusy(true);
    setError(null);
    try {
      const game = await api.createGame(mode);
      navigate(`/game/${game.id}`);
    } catch (e) {
      setError(e.message);
      setBusy(false);
    }
  };

  return (
    <div className="lobby">
      <section className="panel">
        <h2>Play</h2>
        {loading ? (
          <p className="hint">Loading…</p>
        ) : player ? (
          <p>
            Playing as <strong>{player.name}</strong>
          </p>
        ) : (
          <>
            <p className="hint">Choose a name to play online, keep your stats and appear on the leaderboard.</p>
            <NameForm />
          </>
        )}
        <div className="actions">
          <button type="button" onClick={() => start('online')} disabled={!player || busy}>
            New online game
          </button>
          <button type="button" className="secondary" onClick={() => start('local')} disabled={busy}>
            Play on this device
          </button>
        </div>
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
      </section>

      {player && <StatsCard stats={player.stats} />}
      {player && <RecentGames player={player} />}
      <Leaderboard highlightId={player?.id} />
    </div>
  );
}
