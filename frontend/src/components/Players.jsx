import { COLORS, isWaiting, seatName } from '../lib/players.js';

function Seat({ game, seat, mySeat, showTurn }) {
  const empty = seat === 2 && isWaiting(game);
  const active = showTurn && game.status === 'in_progress' && !isWaiting(game) && game.currentPlayer === seat;
  return (
    <div className={`seat${active ? ' active' : ''}`}>
      <span className={`token p${seat}`} aria-hidden="true" />
      <span className="seat-name">
        {empty ? <em>Open seat</em> : seatName(game, seat)}
        {mySeat === seat && <span className="you"> (you)</span>}
      </span>
      <span className="sr-only">plays {COLORS[seat]}</span>
    </div>
  );
}

// Players shows both seats and, if showTurn, highlights whose turn it is.
export default function Players({ game, mySeat, showTurn = true }) {
  return (
    <div className="players">
      <Seat game={game} seat={1} mySeat={mySeat} showTurn={showTurn} />
      <span className="vs">vs</span>
      <Seat game={game} seat={2} mySeat={mySeat} showTurn={showTurn} />
    </div>
  );
}
