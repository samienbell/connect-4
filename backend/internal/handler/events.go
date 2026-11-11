package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"connect4/internal/game"
)

const heartbeatInterval = 25 * time.Second

// gameEvents streams the game as Server-Sent Events: one "game" event with the
// current state straight away, then another every time it changes. Anyone with
// the game's link may watch, so no sign-in is needed (EventSource can't send
// an Authorization header anyway).
func (h *Handler) gameEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Subscribe before reading the game so no change can slip in between.
	changed, unsubscribe := h.hub.Subscribe(id)
	defer unsubscribe()

	g, ok := h.loadGame(w, r)
	if !ok {
		return
	}

	// The server's WriteTimeout would cut the stream off; lift it for this response.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		h.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no") // tell nginx not to buffer
	w.WriteHeader(http.StatusOK)

	send := func(g *game.Game) bool {
		data, err := json.Marshal(g)
		if err != nil {
			h.log.Error("encode game event", "err", err)
			return false
		}
		if _, err := fmt.Fprintf(w, "event: game\ndata: %s\n\n", data); err != nil {
			return false
		}
		return rc.Flush() == nil
	}

	if !send(g) {
		return
	}
	lastUpdate := g.UpdatedAt

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-h.hub.Done():
			return // server shutting down; the browser will reconnect elsewhere
		case <-heartbeat.C:
			// A comment line keeps proxies from closing an idle connection.
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil || rc.Flush() != nil {
				return
			}
		case <-changed:
			g, err := h.store.GetGame(r.Context(), id)
			if err != nil {
				return
			}
			if g.UpdatedAt.Equal(lastUpdate) {
				continue
			}
			lastUpdate = g.UpdatedAt
			if !send(g) {
				return
			}
		}
	}
}
