import { useId, useState } from 'react';
import { usePlayer } from '../hooks/usePlayer.jsx';

// PasswordForm changes the signed-in player's password, or sets the first
// one for players created before accounts existed.
export default function PasswordForm() {
  const { player, changePassword } = usePlayer();
  const id = useId();
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);
  const [done, setDone] = useState(false);

  const settingFirst = !player.hasPassword;
  const mismatch = confirm !== '' && confirm !== next;

  const submit = async (e) => {
    e.preventDefault();
    if (next !== confirm) {
      setError("Passwords don't match.");
      return;
    }
    setBusy(true);
    setError(null);
    setDone(false);
    try {
      await changePassword(current, next);
      setCurrent('');
      setNext('');
      setConfirm('');
      setDone(true);
    } catch (err) {
      setError(err.message.charAt(0).toUpperCase() + err.message.slice(1));
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className="auth-form" onSubmit={submit}>
      {/* Lets password managers associate the new password with this account. */}
      <input type="text" autoComplete="username" value={player.name} readOnly hidden />

      {!settingFirst && (
        <>
          <label htmlFor={`${id}-current`}>Current password</label>
          <input
            id={`${id}-current`}
            type="password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            autoComplete="current-password"
            required
          />
        </>
      )}
      <label htmlFor={`${id}-new`}>{settingFirst ? 'Password' : 'New password'}</label>
      <input
        id={`${id}-new`}
        type="password"
        value={next}
        onChange={(e) => setNext(e.target.value)}
        autoComplete="new-password"
        minLength={8}
        maxLength={128}
        required
      />
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
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {done && (
        <p className="success" role="status">
          Password saved. Any other devices have been signed out.
        </p>
      )}
      <button type="submit" disabled={busy || mismatch}>
        {busy ? 'Saving…' : settingFirst ? 'Set password' : 'Change password'}
      </button>
    </form>
  );
}
