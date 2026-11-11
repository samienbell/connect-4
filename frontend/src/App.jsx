import { Link, useRoute } from './router.jsx';
import { PlayerProvider, usePlayer } from './hooks/usePlayer.jsx';
import Lobby from './views/Lobby.jsx';
import GameView from './views/GameView.jsx';
import ReplayView from './views/ReplayView.jsx';

function PlayerBadge() {
  const { player, logout } = usePlayer();
  if (!player) return null;
  return (
    <div className="badge-name">
      <span>{player.name}</span>
      <button type="button" className="secondary small" onClick={logout}>
        Sign out
      </button>
    </div>
  );
}

function Page() {
  const route = useRoute();
  if (route.name === 'game') return <GameView key={route.id} gameId={route.id} />;
  if (route.name === 'replay') return <ReplayView key={route.id} gameId={route.id} />;
  return <Lobby />;
}

export default function App() {
  return (
    <PlayerProvider>
      <div className="app">
        <header className="topbar">
          <Link to="/" className="brand">
            Connect 4
          </Link>
          <PlayerBadge />
        </header>
        <main>
          <Page />
        </main>
      </div>
    </PlayerProvider>
  );
}
