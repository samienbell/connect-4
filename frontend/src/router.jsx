// A tiny client-side router: three pages don't justify a dependency.
//   /             lobby
//   /game/:id     play or watch a game
//   /replay/:id   step through a finished game
import { useEffect, useState } from 'react';

const ROUTES = [
  { name: 'game', pattern: /^\/game\/([^/]+)\/?$/ },
  { name: 'replay', pattern: /^\/replay\/([^/]+)\/?$/ },
];

function match(pathname) {
  for (const route of ROUTES) {
    const m = pathname.match(route.pattern);
    if (m) return { name: route.name, id: decodeURIComponent(m[1]) };
  }
  return { name: 'lobby' };
}

export function navigate(path) {
  if (path === window.location.pathname) return;
  window.history.pushState(null, '', path);
  window.dispatchEvent(new PopStateEvent('popstate'));
  window.scrollTo(0, 0);
}

export function useRoute() {
  const [route, setRoute] = useState(() => match(window.location.pathname));
  useEffect(() => {
    const onPop = () => setRoute(match(window.location.pathname));
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, []);
  return route;
}

export function Link({ to, children, ...props }) {
  const onClick = (e) => {
    // Let the browser handle new-tab clicks.
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    e.preventDefault();
    navigate(to);
  };
  return (
    <a href={to} onClick={onClick} {...props}>
      {children}
    </a>
  );
}
