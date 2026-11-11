const NAMES = { 1: 'Red', 2: 'Yellow' };

export default function Status({ game }) {
  let text;
  let player = null;

  if (game.status === 'won') {
    player = game.winner;
    text = `${NAMES[player]} wins!`;
  } else if (game.status === 'draw') {
    text = "It's a draw.";
  } else {
    player = game.currentPlayer;
    text = `${NAMES[player]}'s turn`;
  }

  return (
    <p className="status" role="status" aria-live="polite">
      {player && <span className={`token p${player}`} aria-hidden="true" />}
      {text}
    </p>
  );
}
