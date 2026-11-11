import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { api, getToken, setToken } from '../api/client.js';

const PlayerContext = createContext(null);

// PlayerProvider holds the signed-in player ({id, name, hasPassword, stats})
// or null. Registering or logging in returns a session token, which we keep
// in localStorage and send with every request.
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
        // Session expired or signed out elsewhere.
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

  const signIn = useCallback(async (mode, name, password) => {
    const call = mode === 'register' ? api.register : api.login;
    const { player: signedIn, token } = await call(name, password);
    setToken(token);
    setPlayer(signedIn);
    return signedIn;
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } catch {
      // Even if the server call fails, forget the session locally.
    }
    setToken(null);
    setPlayer(null);
  }, []);

  const changePassword = useCallback(
    async (currentPassword, newPassword) => {
      await api.changePassword(currentPassword, newPassword);
      await refresh(); // hasPassword may have changed
    },
    [refresh],
  );

  const value = useMemo(
    () => ({ player, loading, signIn, logout, changePassword, refresh }),
    [player, loading, signIn, logout, changePassword, refresh],
  );
  return <PlayerContext.Provider value={value}>{children}</PlayerContext.Provider>;
}

export function usePlayer() {
  return useContext(PlayerContext);
}
