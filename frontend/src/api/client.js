// Thin wrapper around the backend REST API. All paths are relative so the
// same build works behind the Vite dev proxy and behind nginx.

const TOKEN_KEY = 'connect4.token';

export function getToken() {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setToken(token) {
  try {
    if (token) localStorage.setItem(TOKEN_KEY, token);
    else localStorage.removeItem(TOKEN_KEY);
  } catch {
    // Storage unavailable (private mode etc.): sign-in lasts until the tab closes.
  }
}

export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.status = status;
  }
}

async function request(path, { method = 'GET', body } = {}) {
  const headers = {};
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`/api${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new ApiError(data.error || `Request failed (${res.status})`, res.status);
  return data;
}

const enc = encodeURIComponent;

export const api = {
  register: (name) => request('/players', { method: 'POST', body: { name } }),
  me: () => request('/me'),
  myGames: () => request('/me/games'),
  leaderboard: () => request('/leaderboard'),

  createGame: (mode) => request('/games', { method: 'POST', body: { mode } }),
  getGame: (id) => request(`/games/${enc(id)}`),
  joinGame: (id) => request(`/games/${enc(id)}/join`, { method: 'POST' }),
  playMove: (id, column) => request(`/games/${enc(id)}/moves`, { method: 'POST', body: { column } }),
  getMoves: (id) => request(`/games/${enc(id)}/moves`),
  gameEventsUrl: (id) => `/api/games/${enc(id)}/events`,
};
