import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { api, getToken, setToken } from '../api/client.js';

const PlayerContext = createContext(null);

// PlayerProvider holds the signed-in player ({id, name, stats}) or null.
// Signing in means picking a name; the server returns a token we keep in
// localStorage and send with every request.
export function PlayerProvider({ children }) {
  const [player, setPlayer] = useState(null);
  const [loading, setLoading] = useState(() => Boolean(getToken()));

  const refresh = useCallback(async () => {
    if (!getToken()) {
      setPlayer(null);
      setLoading(false);
      return;
    }
    try {
      setPlayer(await api.me());
    } catch (e) {
      if (e.status === 401) {
        // Token no longer valid (e.g. database was reset).
        setToken(null);
        setPlayer(null);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const register = useCallback(async (name) => {
    const { player: created, token } = await api.register(name);
    setToken(token);
    setPlayer(created);
    return created;
  }, []);

  const value = useMemo(() => ({ player, loading, register, refresh }), [player, loading, register, refresh]);
  return <PlayerContext.Provider value={value}>{children}</PlayerContext.Provider>;
}

export function usePlayer() {
  return useContext(PlayerContext);
}
