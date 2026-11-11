import { useEffect, useMemo, useState } from 'react';
import { api } from '../api/client.js';
import { Link } from '../router.jsx';
import { usePlayer } from '../hooks/usePlayer.jsx';
import { COLORS, seatOf, statusSeat, statusText } from '../lib/players.js';
import Board from '../components/Board.jsx';
import Players from '../components/Players.jsx';

const ROWS = 6;
const COLS = 7;
const STEP_MS = 700;

// buildFrames returns the board after each move: frames[0] is empty,
// frames[n] is the position after move n.
function buildFrames(moves) {
  const frames = [Array.from({ length: ROWS }, () => Array(COLS).fill(0))];
  for (const m of moves) {
    const next = frames[frames.length - 1].map((row) => row.slice());
    next[m.row][m.col] = m.player;
    frames.push(next);
  }
  return frames;
}

export default function ReplayView({ gameId }) {
  const { player } = usePlayer();
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [step, setStep] = useState(0);
  const [playing, setPlaying] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.all([api.getGame(gameId), api.getMoves(gameId)])
      .then(([game, { moves }]) => {
        if (cancelled) return;
        setData({ game, moves });
        setStep(0);
        setPlaying(moves.length > 0);
      })
      .catch((e) => !cancelled && setError(e.status === 404 ? 'Game not found.' : e.message));
    return () => {
      cancelled = true;
    };
  }, [gameId]);

  const frames = useMemo(() => (data ? buildFrames(data.moves) : []), [data]);
  const total = data ? data.moves.length : 0;

  // Auto-advance while playing.
  useEffect(() => {
    if (!playing) return undefined;
    if (step >= total) {
      setPlaying(false);
      return undefined;
    }
    const t = setTimeout(() => setStep((s) => Math.min(s + 1, total)), STEP_MS);
    return () => clearTimeout(t);
  }, [playing, step, total]);

  const goTo = (s) => {
    setPlaying(false);
    setStep(Math.max(0, Math.min(total, s)));
  };
  const togglePlay = () => {
    if (playing) {
      setPlaying(false);
      return;
    }
    if (step >= total) setStep(0);
    setPlaying(true);
  };

  // Keyboard: ← → step, Home/End jump, space play/pause.
  useEffect(() => {
    const onKey = (e) => {
      const tag = e.target.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
      const keys = { ArrowLeft: step - 1, ArrowRight: step + 1, Home: 0, End: total };
      if (e.key in keys) {
        e.preventDefault();
        goTo(keys[e.key]);
      } else if (e.key === ' ' && tag !== 'BUTTON' && tag !== 'A') {
        e.preventDefault();
        togglePlay();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });

  if (error) {
    return (
      <section className="panel">
        <p className="error">{error}</p>
        <Link to="/">Back to lobby</Link>
      </section>
    );
  }
  if (!data) return <p className="status">Loading…</p>;

  const { game, moves } = data;
  const mySeat = seatOf(game, player);
  const lastMove = step > 0 ? moves[step - 1] : null;
  const atEnd = step === total;

  const caption =
    step === 0 ? 'Start of game' : `Move ${step} of ${total}: ${COLORS[lastMove.player]} in column ${lastMove.col + 1}`;

  const endSeat = statusSeat(game);

  return (
    <section className="game-view">
      <h2 className="replay-title">Replay</h2>
      <Players game={game} mySeat={mySeat} showTurn={false} />

      <p className="status" aria-live="polite">
        {atEnd && game.status !== 'in_progress' ? (
          <>
            {endSeat > 0 && <span className={`token p${endSeat}`} aria-hidden="true" />}
            {statusText(game, mySeat)}
          </>
        ) : (
          caption
        )}
      </p>

      <Board board={frames[step]} disabled highlight={lastMove} />

      <div className="replay-controls">
        <button type="button" className="secondary" onClick={() => goTo(0)} disabled={step === 0} aria-label="First move">
          ⏮
        </button>
        <button type="button" className="secondary" onClick={() => goTo(step - 1)} disabled={step === 0} aria-label="Previous move">
          ◀
        </button>
        <button type="button" onClick={togglePlay} disabled={total === 0} className="play">
          {playing ? 'Pause' : atEnd ? 'Replay' : 'Play'}
        </button>
        <button type="button" className="secondary" onClick={() => goTo(step + 1)} disabled={atEnd} aria-label="Next move">
          ▶
        </button>
        <button type="button" className="secondary" onClick={() => goTo(total)} disabled={atEnd} aria-label="Last move">
          ⏭
        </button>
      </div>
      <input
        type="range"
        className="scrubber"
        min={0}
        max={total}
        value={step}
        onChange={(e) => goTo(Number(e.target.value))}
        aria-label="Move"
      />
      <p className="hint">Tip: use ← → to step, space to play or pause.</p>

      <div className="actions">
        <Link to="/" className="button secondary">
          Lobby
        </Link>
        {game.status === 'in_progress' && (
          <Link to={`/game/${game.id}`} className="button">
            Back to game
          </Link>
        )}
      </div>
    </section>
  );
}
