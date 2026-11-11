import { useState } from 'react';
import { api } from '../api/client.js';
import { Link } from '../router.jsx';
import { usePlayer } from '../hooks/usePlayer.jsx';
import { useLiveGame } from '../hooks/useLiveGame.js';
import { isWaiting, seatOf, statusSeat, statusText } from '../lib/players.js';
import Board from '../components/Board.jsx';
import Players from '../components/Players.jsx';
import ShareLink from '../components/ShareLink.jsx';
import NameForm from '../components/NameForm.jsx';

export default function GameView({ gameId }) {
  const { player, loading } = usePlayer();
  const { game, error, connection, update } = useLiveGame(gameId);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState(null);

  if (error) {
    return (
      <section className="panel">
        <p className="error">{error}</p>
        <Link to="/">Back to lobby</Link>
      </section>
    );
  }
  if (!game || loading) return <p className="status">Loading…</p>;

  const act = async (request) => {
    setBusy(true);
    setActionError(null);
    try {
      update(await request());
    } catch (e) {
      setActionError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const join = () => act(() => api.joinGame(game.id));

  const online = game.mode === 'online';
  const mySeat = seatOf(game, player);
  const waiting = isWaiting(game);
  const inProgress = game.status === 'in_progress';
  const canMove = inProgress && !waiting && (!online || mySeat === game.currentPlayer);
  const spectating = online && !waiting && mySeat === 0;
  const seat = statusSeat(game);

  return (
    <section className="game-view">
      <Players game={game} mySeat={mySeat} />

      <p className="status" role="status" aria-live="polite">
        {seat > 0 && <span className={`token p${seat}`} aria-hidden="true" />}
        {statusText(game, mySeat)}
      </p>

      {waiting && mySeat === 1 && <ShareLink gameId={game.id} />}

      {waiting && mySeat === 0 && (
        <div className="panel join">
          {player ? (
            <button type="button" onClick={join} disabled={busy}>
              Join as Yellow
            </button>
          ) : (
            <>
              <p>Choose a name to join this game as Yellow.</p>
              <NameForm submitLabel="Join game" onDone={join} />
            </>
          )}
        </div>
      )}

      <Board board={game.board} disabled={busy || !canMove} onPlay={(col) => act(() => api.playMove(game.id, col))} />

      {spectating && <p className="hint">You're watching this game.</p>}
      {online && inProgress && connection === 'lost' && <p className="hint">Connection lost, reconnecting…</p>}
      {actionError && (
        <p className="error" role="alert">
          {actionError}
        </p>
      )}

      <div className="actions">
        <Link to="/" className="button secondary">
          Lobby
        </Link>
        {!inProgress && (
          <Link to={`/replay/${game.id}`} className="button">
            Watch replay
          </Link>
        )}
      </div>
    </section>
  );
}
