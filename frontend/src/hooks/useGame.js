import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api/client.js';

const STORAGE_KEY = 'connect4.gameId';

function loadGameId() {
  try {
    return localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

function saveGameId(id) {
  try {
    localStorage.setItem(STORAGE_KEY, id);
  } catch {
    // Storage unavailable (private mode etc.) — the game still works, it just won't resume.
  }
}

// useGame owns the current game and talks to the API. On first load it
// resumes the last game played in this browser, or starts a new one.
export function useGame() {
  const [game, setGame] = useState(null);
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);
  const started = useRef(false);

  const run = useCallback(async (fetchGame) => {
    setBusy(true);
    setError(null);
    try {
      const g = await fetchGame();
      setGame(g);
      saveGameId(g.id);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }, []);

  const newGame = useCallback(() => run(api.createGame), [run]);

  const play = useCallback(
    (column) => {
      if (!game || busy || game.status !== 'in_progress') return;
      run(() => api.playMove(game.id, column));
    },
    [game, busy, run],
  );

  useEffect(() => {
    // Guard against StrictMode running this effect twice in development.
    if (started.current) return;
    started.current = true;
    const id = loadGameId();
    run(() => (id ? api.getGame(id).catch(() => api.createGame()) : api.createGame()));
  }, [run]);

  return { game, error, busy, newGame, play };
}
