import Board from './components/Board.jsx';
import Status from './components/Status.jsx';
import { useGame } from './hooks/useGame.js';

export default function App() {
  const { game, error, busy, newGame, play } = useGame();

  return (
    <main className="app">
      <h1>Connect 4</h1>

      {game ? (
        <>
          <Status game={game} />
          <Board
            board={game.board}
            disabled={busy || game.status !== 'in_progress'}
            onPlay={play}
          />
        </>
      ) : (
        <p className="status">Loading…</p>
      )}

      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}

      <button type="button" className="new-game" onClick={newGame} disabled={busy}>
        New game
      </button>
    </main>
  );
}
