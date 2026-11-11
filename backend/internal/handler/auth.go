package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"connect4/internal/store"
)

// Players have no password. On sign-up the server returns a random token,
// the browser keeps it in localStorage and sends it as
// "Authorization: Bearer <token>". Only its SHA-256 hash is stored.

var errBadToken = errors.New("invalid token")

func newToken() (token string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// player returns the signed-in caller, or nil if the request has no token.
// A token that is present but unknown returns errBadToken.
func (h *Handler) player(r *http.Request) (*store.Player, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, nil
	}
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		return nil, errBadToken
	}
	p, err := h.store.PlayerByTokenHash(r.Context(), hashToken(token))
	if errors.Is(err, store.ErrNotFound) {
		return nil, errBadToken
	}
	return p, err
}

// optionalPlayer is player() that writes the error response itself. ok is
// false when the handler should stop.
func (h *Handler) optionalPlayer(w http.ResponseWriter, r *http.Request) (p *store.Player, ok bool) {
	p, err := h.player(r)
	switch {
	case errors.Is(err, errBadToken):
		writeError(w, http.StatusUnauthorized, "your sign-in is no longer valid; choose a name again")
		return nil, false
	case err != nil:
		h.serverError(w, err)
		return nil, false
	}
	return p, true
}

// requirePlayer is optionalPlayer that also rejects anonymous callers.
func (h *Handler) requirePlayer(w http.ResponseWriter, r *http.Request) (*store.Player, bool) {
	p, ok := h.optionalPlayer(w, r)
	if ok && p == nil {
		writeError(w, http.StatusUnauthorized, "choose a name first")
		return nil, false
	}
	return p, ok
}
