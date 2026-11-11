const COLS = 7;
const PLAYER_CLASS = { 1: 'p1', 2: 'p2' };

// The board is rendered column by column so a whole column is one button:
// click anywhere in it to drop a disc. board[row][col], row 0 is the top.
// `highlight` ({row, col}) rings one disc, e.g. the last move in a replay.
export default function Board({ board, disabled, onPlay, highlight }) {
  return (
    <div className="board">
      {Array.from({ length: COLS }, (_, col) => {
        const full = board[0][col] !== 0;
        return (
          <button
            key={col}
            type="button"
            className="column"
            disabled={disabled || full || !onPlay}
            onClick={() => onPlay?.(col)}
            aria-label={`Drop disc in column ${col + 1}`}
          >
            {board.map((row, r) => {
              const isLast = highlight && highlight.row === r && highlight.col === col;
              return <span key={r} className={`cell ${PLAYER_CLASS[row[col]] ?? ''}${isLast ? ' last' : ''}`} />;
            })}
          </button>
        );
      })}
    </div>
  );
}
