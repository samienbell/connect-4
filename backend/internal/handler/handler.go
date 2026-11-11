// Package handler exposes the game over a JSON HTTP API. See docs/api.md.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"connect4/internal/events"
	"connect4/internal/store"
)

type Handler struct {
	store *store.Store
	hub   *events.Hub
	log   *slog.Logger
}

func New(s *store.Store, hub *events.Hub, log *slog.Logger) *Handler {
	return &Handler{store: s, hub: hub, log: log}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)

	mux.HandleFunc("POST /api/players", h.createPlayer)
	mux.HandleFunc("GET /api/players/{id}", h.getPlayer)
	mux.HandleFunc("GET /api/me", h.me)
	mux.HandleFunc("GET /api/me/games", h.myGames)
	mux.HandleFunc("GET /api/leaderboard", h.leaderboard)

	mux.HandleFunc("POST /api/games", h.createGame)
	mux.HandleFunc("GET /api/games/{id}", h.getGame)
	mux.HandleFunc("POST /api/games/{id}/join", h.joinGame)
	mux.HandleFunc("GET /api/games/{id}/moves", h.listMoves)
	mux.HandleFunc("POST /api/games/{id}/moves", h.playMove)
	mux.HandleFunc("GET /api/games/{id}/events", h.gameEvents)
	return mux
}

// healthz is the liveness probe: the process is up.
func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz is the readiness probe: the database is reachable.
func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// decodeJSON reads a small JSON body into v. An empty body leaves v unchanged.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	err := json.NewDecoder(r.Body).Decode(v)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.log.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
