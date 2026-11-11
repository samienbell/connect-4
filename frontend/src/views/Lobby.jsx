import { useEffect, useState } from 'react';
import { api } from '../api/client.js';
import { navigate } from '../router.jsx';
import { usePlayer } from '../hooks/usePlayer.jsx';
import AuthForm from '../components/AuthForm.jsx';
import PasswordForm from '../components/PasswordForm.jsx';
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

  const errorMessage = error && (
    <p className="error" role="alert">
      {error}
    </p>
  );

  // Signed out: online play (behind sign-in) and local play are separate sections.
  if (!loading && !player) {
    return (
      <div className="lobby">
        <section className="panel">
          <h2>Online play</h2>
          <p className="hint">
            Play against a friend on another device. Sign in to start a game, keep your stats and appear on the
            leaderboard.
          </p>
          <AuthForm initialMode="register" />
        </section>

        <section className="panel">
          <h2>Local play</h2>
          <p className="hint">Two players take turns on this device. No account needed; games don't count toward stats.</p>
          <div className="actions">
            <button type="button" onClick={() => start('local')} disabled={busy}>
              New local game
            </button>
          </div>
          {errorMessage}
        </section>

        <Leaderboard />
      </div>
    );
  }

  return (
    <div className="lobby">
      <section className="panel">
        <h2>Play</h2>
        {loading ? (
          <p className="hint">Loading…</p>
        ) : (
          <p>
            Playing as <strong>{player.name}</strong>
          </p>
        )}
        <div className="actions">
          <button type="button" onClick={() => start('online')} disabled={!player || busy}>
            New online game
          </button>
          <button type="button" className="secondary" onClick={() => start('local')} disabled={busy}>
            New local game
          </button>
        </div>
        {errorMessage}
      </section>

      {player && !player.hasPassword && (
        <section className="panel notice">
          <h2>Set a password</h2>
          <p>
            Your account was created before passwords existed, so it only lives in this browser. Set a password to keep
            the name <strong>{player.name}</strong> and sign in from other devices.
          </p>
          <PasswordForm />
        </section>
      )}

      {player && <StatsCard stats={player.stats} />}
      {player && <RecentGames player={player} />}
      <Leaderboard highlightId={player?.id} />

      {player?.hasPassword && (
        <section className="panel">
          <details>
            <summary>Change password</summary>
            <PasswordForm />
          </details>
        </section>
      )}
    </div>
  );
}
