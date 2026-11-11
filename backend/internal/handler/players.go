package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"connect4/internal/store"
)

const (
	minNameLen       = 2
	maxNameLen       = 20
	recentGamesLimit = 20
	leaderboardLimit = 20
)

// Letters and digits from any language, plus space, underscore, dot and
// hyphen. Must start with a letter or digit.
var nameRE = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} _.\-]*$`)

var errNameLength = errors.New("name must be 2 to 20 characters")
var errNameChars = errors.New("name can only use letters, numbers, spaces and _ . -")

// validateName trims name and checks it is acceptable as a display name.
func validateName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ") // trim and collapse inner whitespace
	if n := utf8.RuneCountInString(name); n < minNameLen || n > maxNameLen {
		return "", errNameLength
	}
	if !nameRE.MatchString(name) {
		return "", errNameChars
	}
	return name, nil
}

type profile struct {
	*store.Player
	Stats store.Stats `json:"stats"`
}

func (h *Handler) createPlayer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, `request body must be {"name": "..."}`)
		return
	}
	name, err := validateName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	token, hash, err := newToken()
	if err != nil {
		h.serverError(w, err)
		return
	}
	p, err := h.store.CreatePlayer(r.Context(), name, hash)
	if errors.Is(err, store.ErrNameTaken) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"player": profile{Player: p},
		"token":  token,
	})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	p, ok := h.requirePlayer(w, r)
	if !ok {
		return
	}
	h.writeProfile(w, r, p)
}

func (h *Handler) getPlayer(w http.ResponseWriter, r *http.Request) {
	p, err := h.store.PlayerByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	h.writeProfile(w, r, p)
}

func (h *Handler) writeProfile(w http.ResponseWriter, r *http.Request, p *store.Player) {
	stats, err := h.store.PlayerStats(r.Context(), p.ID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile{Player: p, Stats: stats})
}

func (h *Handler) myGames(w http.ResponseWriter, r *http.Request) {
	p, ok := h.requirePlayer(w, r)
	if !ok {
		return
	}
	games, err := h.store.PlayerGames(r.Context(), p.ID, recentGamesLimit)
	if err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"games": games})
}

func (h *Handler) leaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := h.store.Leaderboard(r.Context(), leaderboardLimit)
	if err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"players": entries})
}
