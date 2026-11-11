import { useId, useState } from 'react';
import { usePlayer } from '../hooks/usePlayer.jsx';

// AuthForm lets a visitor sign in or create an account. `onDone` runs after
// either succeeds (e.g. to join a game straight away).
export default function AuthForm({ onDone, initialMode = 'login' }) {
  const { signIn } = usePlayer();
  const id = useId();
  const [mode, setMode] = useState(initialMode);
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);

  const registering = mode === 'register';
  const mismatch = registering && confirm !== '' && confirm !== password;

  const switchMode = (next) => {
    setMode(next);
    setError(null);
    setPassword('');
    setConfirm('');
  };

  const submit = async (e) => {
    e.preventDefault();
    if (registering && password !== confirm) {
      setError("Passwords don't match.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const player = await signIn(mode, name, password);
      await onDone?.(player);
    } catch (err) {
      setError(err.message.charAt(0).toUpperCase() + err.message.slice(1));
      setBusy(false);
    }
  };

  return (
    <form className="auth-form" onSubmit={submit}>
      <div className="tabs" role="tablist">
        <button type="button" role="tab" aria-selected={!registering} className={!registering ? 'on' : ''} onClick={() => switchMode('login')}>
          Sign in
        </button>
        <button type="button" role="tab" aria-selected={registering} className={registering ? 'on' : ''} onClick={() => switchMode('register')}>
          Create account
        </button>
      </div>

      <label htmlFor={`${id}-name`}>Name</label>
      <input
        id={`${id}-name`}
        value={name}
        onChange={(e) => setName(e.target.value)}
        maxLength={20}
        autoComplete="username"
        required
      />
      {registering && <p className="hint">2–20 letters, numbers, spaces, _ . or -. Shown on the leaderboard.</p>}

      <label htmlFor={`${id}-password`}>Password</label>
      <input
        id={`${id}-password`}
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        autoComplete={registering ? 'new-password' : 'current-password'}
        minLength={registering ? 8 : undefined}
        maxLength={128}
        required
      />
      {registering && <p className="hint">At least 8 characters.</p>}

      {registering && (
        <>
          <label htmlFor={`${id}-confirm`}>Confirm password</label>
          <input
            id={`${id}-confirm`}
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
            maxLength={128}
            aria-invalid={mismatch}
            required
          />
          {mismatch && <p className="error">Passwords don't match.</p>}
        </>
      )}

      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      <button type="submit" disabled={busy || mismatch}>
        {busy ? 'Please wait…' : registering ? 'Create account' : 'Sign in'}
      </button>
    </form>
  );
}
