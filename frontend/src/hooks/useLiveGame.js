import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client.js';

// Prefer whichever copy of the game is newer, so a slow response can't
// overwrite a fresher live update (or the other way round).
function newer(prev, next) {
  if (!prev || prev.id !== next.id) return next;
  if (next.moveCount !== prev.moveCount) return next.moveCount > prev.moveCount ? next : prev;
  return new Date(next.updatedAt) >= new Date(prev.updatedAt) ? next : prev;
}

// useLiveGame loads a game and keeps it up to date through the server's
// event stream. `connection` is 'connecting', 'open' or 'lost'; the browser
// reconnects automatically after 'lost'.
export function useLiveGame(gameId) {
  const [game, setGame] = useState(null);
  const [error, setError] = useState(null);
  const [connection, setConnection] = useState('connecting');

  useEffect(() => {
    let cancelled = false;
    let source = null;
    setGame(null);
    setError(null);
    setConnection('connecting');

    api
      .getGame(gameId)
      .then((g) => {
        if (cancelled) return;
        setGame(g);
        source = new EventSource(api.gameEventsUrl(gameId));
        source.addEventListener('game', (e) => setGame((prev) => newer(prev, JSON.parse(e.data))));
        source.onopen = () => setConnection('open');
        source.onerror = () => setConnection('lost');
      })
      .catch((e) => {
        if (!cancelled) setError(e.status === 404 ? 'Game not found.' : e.message);
      });

    return () => {
      cancelled = true;
      source?.close();
    };
  }, [gameId]);

  // Apply a game returned by one of our own requests (join, move).
  const update = useCallback((g) => setGame((prev) => newer(prev, g)), []);

  return { game, error, connection, update };
}
