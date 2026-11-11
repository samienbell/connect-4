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

// Signing in (register or login) creates a session: a random token the
// browser keeps in localStorage and sends as "Authorization: Bearer <token>".
// The sessions table stores only the token's SHA-256 hash, so a database leak
// doesn't hand out working sign-ins. (A plain hash is enough here because the
// token is 256 random bits; passwords, which people choose, get Argon2id.)

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

// session returns the signed-in caller and their session's token hash, or
// nils if the request has no token. A token that is present but unknown or
// expired returns errBadToken.
func (h *Handler) session(r *http.Request) (*store.Player, []byte, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, nil, nil
	}
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		return nil, nil, errBadToken
	}
	tokenHash := hashToken(token)
	p, err := h.store.PlayerBySession(r.Context(), tokenHash)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, errBadToken
	}
	if err != nil {
		return nil, nil, err
	}
	return p, tokenHash, nil
}

// optionalSession is session() that writes the error response itself. ok is
// false when the handler should stop.
func (h *Handler) optionalSession(w http.ResponseWriter, r *http.Request) (p *store.Player, tokenHash []byte, ok bool) {
	p, tokenHash, err := h.session(r)
	switch {
	case errors.Is(err, errBadToken):
		writeError(w, http.StatusUnauthorized, "your session has expired; please sign in again")
		return nil, nil, false
	case err != nil:
		h.serverError(w, err)
		return nil, nil, false
	}
	return p, tokenHash, true
}

// requireSession is optionalSession that also rejects anonymous callers.
func (h *Handler) requireSession(w http.ResponseWriter, r *http.Request) (*store.Player, []byte, bool) {
	p, tokenHash, ok := h.optionalSession(w, r)
	if ok && p == nil {
		writeError(w, http.StatusUnauthorized, "please sign in first")
		return nil, nil, false
	}
	return p, tokenHash, ok
}

func (h *Handler) optionalPlayer(w http.ResponseWriter, r *http.Request) (*store.Player, bool) {
	p, _, ok := h.optionalSession(w, r)
	return p, ok
}

func (h *Handler) requirePlayer(w http.ResponseWriter, r *http.Request) (*store.Player, bool) {
	p, _, ok := h.requireSession(w, r)
	return p, ok
}
