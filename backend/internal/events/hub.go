// Package events tells open browser connections when a game changes.
//
// Every backend replica runs one Hub. Writes to a game call pg_notify with the
// game's ID (see store.notify); each Hub LISTENs on that channel and wakes the
// local subscribers for that game. Going through Postgres means a move made on
// one replica reaches viewers connected to any other replica.
package events

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

type Hub struct {
	channel string
	log     *slog.Logger

	mu   sync.Mutex
	subs map[string]map[chan struct{}]struct{}

	done      chan struct{}
	closeOnce sync.Once
}

func NewHub(channel string, log *slog.Logger) *Hub {
	return &Hub{
		channel: channel,
		log:     log,
		subs:    make(map[string]map[chan struct{}]struct{}),
		done:    make(chan struct{}),
	}
}

// Subscribe returns a channel that receives a signal whenever gameID changes,
// and a function to stop listening. Signals are coalesced: several quick
// changes may arrive as one, so subscribers should reload the current state
// rather than count signals.
func (h *Hub) Subscribe(gameID string) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if h.subs[gameID] == nil {
		h.subs[gameID] = make(map[chan struct{}]struct{})
	}
	h.subs[gameID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.subs[gameID], ch)
		if len(h.subs[gameID]) == 0 {
			delete(h.subs, gameID)
		}
	}
}

// Publish wakes every local subscriber of gameID.
func (h *Hub) Publish(gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[gameID] {
		signal(ch)
	}
}

func (h *Hub) publishAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, set := range h.subs {
		for ch := range set {
			signal(ch)
		}
	}
}

func signal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default: // a signal is already pending
	}
}

// Done is closed when the server shuts down, so long-lived streams can end.
func (h *Hub) Done() <-chan struct{} { return h.done }

// Close ends all streams. Safe to call more than once.
func (h *Hub) Close() { h.closeOnce.Do(func() { close(h.done) }) }

// Listen forwards Postgres notifications to subscribers until ctx is
// cancelled, reconnecting if the database connection drops.
func (h *Hub) Listen(ctx context.Context, databaseURL string) {
	for {
		err := h.listenOnce(ctx, databaseURL)
		if ctx.Err() != nil {
			return
		}
		h.log.Warn("event listener disconnected, reconnecting", "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (h *Hub) listenOnce(ctx context.Context, databaseURL string) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{h.channel}.Sanitize()); err != nil {
		return err
	}
	h.log.Info("listening for game events", "channel", h.channel)
	// Games may have changed while we were disconnected; let everyone reload.
	h.publishAll()

	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		h.Publish(n.Payload)
	}
}
