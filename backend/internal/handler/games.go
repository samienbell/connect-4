package handler

import (
	"errors"
	"net/http"

	"connect4/internal/game"
	"connect4/internal/store"
)

// loadGame fetches the game named in the URL, writing 404/500 itself.
func (h *Handler) loadGame(w http.ResponseWriter, r *http.Request) (*game.Game, bool) {
	g, err := h.store.GetGame(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "game not found")
		return nil, false
	}
	if err != nil {
		h.serverError(w, err)
		return nil, false
	}
	return g, true
}

func (h *Handler) createGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode game.Mode `json:"mode"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, `request body must be {"mode": "local" | "online"}`)
		return
	}
	if req.Mode == "" {
		req.Mode = game.ModeLocal
	}
	if req.Mode != game.ModeLocal && req.Mode != game.ModeOnline {
		writeError(w, http.StatusBadRequest, `mode must be "local" or "online"`)
		return
	}

	p, ok := h.optionalPlayer(w, r)
	if !ok {
		return
	}
	if req.Mode == game.ModeOnline && p == nil {
		writeError(w, http.StatusUnauthorized, "choose a name to play online")
		return
	}

	g := game.New(req.Mode)
	if p != nil {
		// The creator always plays Red. For local games this just lists the
		// game under their recent games; it doesn't count toward stats.
		g.Player1 = &game.PlayerRef{ID: p.ID, Name: p.Name}
	}
	if err := h.store.CreateGame(r.Context(), g); err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (h *Handler) getGame(w http.ResponseWriter, r *http.Request) {
	if g, ok := h.loadGame(w, r); ok {
		writeJSON(w, http.StatusOK, g)
	}
}

func (h *Handler) joinGame(w http.ResponseWriter, r *http.Request) {
	p, ok := h.requirePlayer(w, r)
	if !ok {
		return
	}
	g, ok := h.loadGame(w, r)
	if !ok {
		return
	}
	switch {
	case g.Mode != game.ModeOnline:
		writeError(w, http.StatusBadRequest, "only online games can be joined")
		return
	case g.SeatOf(p.ID) != game.Empty:
		writeJSON(w, http.StatusOK, g) // already in this game
		return
	case g.Player2 != nil:
		writeError(w, http.StatusConflict, "this game already has two players")
		return
	}

	if err := h.store.JoinGame(r.Context(), g.ID, p.ID); err != nil {
		if errors.Is(err, store.ErrCannotJoin) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		h.serverError(w, err)
		return
	}
	h.getGame(w, r)
}

func (h *Handler) listMoves(w http.ResponseWriter, r *http.Request) {
	g, ok := h.loadGame(w, r)
	if !ok {
		return
	}
	moves, err := h.store.ListMoves(r.Context(), g.ID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"moves": moves})
}

func (h *Handler) playMove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Column *int `json:"column"`
	}
	if err := decodeJSON(w, r, &req); err != nil || req.Column == nil {
		writeError(w, http.StatusBadRequest, `request body must be {"column": 0-6}`)
		return
	}
	col := *req.Column

	g, ok := h.loadGame(w, r)
	if !ok {
		return
	}

	if g.Mode == game.ModeOnline {
		p, ok := h.requirePlayer(w, r)
		if !ok {
			return
		}
		seat := g.SeatOf(p.ID)
		if seat == game.Empty {
			writeError(w, http.StatusForbidden, "you are not a player in this game")
			return
		}
		if g.Status == game.StatusInProgress && !g.Waiting() && seat != g.CurrentPlayer {
			writeError(w, http.StatusConflict, "it's not your turn")
			return
		}
	}

	player := g.CurrentPlayer
	row, err := g.Play(col)
	switch {
	case errors.Is(err, game.ErrInvalidColumn):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, game.ErrColumnFull),
		errors.Is(err, game.ErrGameOver),
		errors.Is(err, game.ErrWaitingForOpponent):
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
