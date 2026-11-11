// Helpers for describing seats and results.

export const COLORS = { 1: 'Red', 2: 'Yellow' };

// seatOf returns 1 or 2 if `player` sits in that seat of `game`, else 0.
export function seatOf(game, player) {
  if (!player) return 0;
  if (game.player1?.id === player.id) return 1;
  if (game.player2?.id === player.id) return 2;
  return 0;
}

// seatName is the player's name in online games, or the colour otherwise.
export function seatName(game, seat) {
  if (game.mode === 'online') {
    const p = seat === 1 ? game.player1 : game.player2;
    if (p) return p.name;
  }
  return COLORS[seat];
}

export function isWaiting(game) {
  return game.mode === 'online' && !game.player2;
}

// statusText describes the game from the viewer's point of view.
export function statusText(game, mySeat) {
  const online = game.mode === 'online';
  if (game.status === 'won') {
    if (online && mySeat) return mySeat === game.winner ? 'You win!' : `${seatName(game, game.winner)} wins.`;
    return `${seatName(game, game.winner)} wins!`;
  }
  if (game.status === 'draw') return "It's a draw.";
  if (isWaiting(game)) return mySeat === 1 ? 'Waiting for an opponent to join…' : 'This game needs a second player.';
  if (online && mySeat) {
    return mySeat === game.currentPlayer ? 'Your turn' : `Waiting for ${seatName(game, game.currentPlayer)}…`;
  }
  return `${seatName(game, game.currentPlayer)}'s turn`;
}

// statusSeat is whose colour to show next to the status text.
export function statusSeat(game) {
  if (game.status === 'won') return game.winner;
  if (game.status === 'draw' || isWaiting(game)) return 0;
  return game.currentPlayer;
}

export function formatDate(iso) {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
}
