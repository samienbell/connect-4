import { useState } from 'react';

// ShareLink shows the game's URL with a copy button.
export default function ShareLink({ gameId }) {
  const url = `${window.location.origin}/game/${gameId}`;
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard blocked: the text is selectable in the input instead.
    }
  };

  return (
    <div className="share">
      <label htmlFor="share-url">Send this link to your opponent</label>
      <div className="row">
        <input id="share-url" value={url} readOnly onFocus={(e) => e.target.select()} />
        <button type="button" onClick={copy}>
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
    </div>
  );
}
