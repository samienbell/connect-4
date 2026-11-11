// Thin wrapper around the backend REST API. All paths are relative so the
// same build works behind the Vite dev proxy and behind nginx.

async function request(path, options = {}) {
  const res = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.error || `Request failed (${res.status})`);
  }
  return body;
}

export const api = {
  createGame: () => request('/games', { method: 'POST' }),
  getGame: (id) => request(`/games/${encodeURIComponent(id)}`),
  playMove: (id, column) =>
    request(`/games/${encodeURIComponent(id)}/moves`, {
      method: 'POST',
      body: JSON.stringify({ column }),
    }),
};
