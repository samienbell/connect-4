// Package handler exposes the game over a JSON HTTP API.
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"connect4/internal/game"
	"connect4/internal/store"
)

type Handler struct {
	store *store.Store
	log   *slog.Logger
}

func New(s *store.Store, log *slog.Logger) *Handler {
	return &Handler{store: s, log: log}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("POST /api/games", h.createGame)
	mux.HandleFunc("GET /api/games/{id}", h.getGame)
	mux.HandleFunc("POST /api/games/{id}/moves", h.playMove)
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

func (h *Handler) createGame(w http.ResponseWriter, r *http.Request) {
	g := game.New()
	if err := h.store.CreateGame(r.Context(), g); err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (h *Handler) getGame(w http.ResponseWriter, r *http.Request) {
	g, err := h.store.GetGame(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

type moveRequest struct {
	Column *int `json:"column"`
}

func (h *Handler) playMove(w http.ResponseWriter, r *http.Request) {
	var req moveRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Column == nil {
		writeError(w, http.StatusBadRequest, `request body must be {"column": 0-6}`)
		return
	}
	col := *req.Column

	g, err := h.store.GetGame(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}

	player := g.CurrentPlayer
	row, err := g.Play(col)
	switch {
	case errors.Is(err, game.ErrInvalidColumn):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, game.ErrColumnFull), errors.Is(err, game.ErrGameOver):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		h.serverError(w, err)
		return
	}

	move := store.Move{Number: g.MoveCount, Player: player, Row: row, Col: col}
	if err := h.store.SaveMove(r.Context(), g, move); err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
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
