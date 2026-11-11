// StatsCard shows a player's record in finished online games.
export default function StatsCard({ stats }) {
  const items = [
    ['Played', stats.played],
    ['Wins', stats.wins],
    ['Losses', stats.losses],
    ['Draws', stats.draws],
    ['Win rate', stats.played ? `${Math.round(stats.winRate * 100)}%` : '–'],
  ];
  return (
    <section className="panel">
      <h2>Your stats</h2>
      <dl className="stats">
        {items.map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      <p className="hint">Counts finished online games only.</p>
    </section>
  );
}
