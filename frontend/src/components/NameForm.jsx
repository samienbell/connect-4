import { useState } from 'react';
import { usePlayer } from '../hooks/usePlayer.jsx';

// NameForm signs the visitor in by choosing a display name. `onDone` runs
// after a successful sign-up (e.g. to join a game straight away).
export default function NameForm({ onDone, submitLabel = 'Save name' }) {
  const { register } = usePlayer();
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);

  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const player = await register(name);
      await onDone?.(player);
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className="name-form" onSubmit={submit}>
      <label htmlFor="player-name">Your name</label>
      <div className="row">
        <input
          id="player-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          maxLength={20}
          autoComplete="nickname"
          placeholder="2–20 characters"
          required
        />
        <button type="submit" disabled={busy || name.trim().length < 2}>
          {submitLabel}
        </button>
      </div>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
    </form>
  );
}
