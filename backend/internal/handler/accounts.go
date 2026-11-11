package handler

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"connect4/internal/password"
	"connect4/internal/store"
)

const (
	sessionTTL = 30 * 24 * time.Hour

	minPasswordLen = 8
	maxPasswordLen = 128 // bounds hashing work per request

	// After this many wrong passwords in a row, each further attempt must
	// wait: 30s, then 1m, 2m, 4m, 8m, capped at 15m. A correct password resets it.
	freeLoginAttempts = 5
	maxLoginDelay     = 15 * time.Minute
)

var (
	errPasswordLength   = fmt.Errorf("password must be %d to %d characters", minPasswordLen, maxPasswordLen)
	errPasswordIsName   = errors.New("password can't be the same as your name")
	errWrongCredentials = errors.New("incorrect name or password")
)

func validatePassword(pw, name string) error {
	if n := utf8.RuneCountInString(pw); n < minPasswordLen || n > maxPasswordLen {
		return errPasswordLength
	}
	if strings.EqualFold(strings.TrimSpace(pw), name) {
		return errPasswordIsName
	}
	return nil
}

// loginDelay is how long to wait after the last failure before another
// attempt is allowed, given the number of consecutive failures.
func loginDelay(failures int) time.Duration {
	if failures < freeLoginAttempts {
		return 0
	}
	shift := failures - freeLoginAttempts
	if shift > 5 {
		shift = 5
	}
	return min(30*time.Second<<shift, maxLoginDelay)
}

// dummyHash is verified against when a login names an unknown player, so
// the response takes as long as a real check and doesn't reveal which
// names exist through timing.
var dummyHash = sync.OnceValue(func() string {
	h, _ := password.Hash("dummy password for timing")
	return h
})

type credentialsRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// signedIn is the response to register and login.
type signedIn struct {
	Player profile `json:"player"`
	Token  string  `json:"token"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, `request body must be {"name": "...", "password": "..."}`)
		return
	}
	name, err := validateName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePassword(req.Password, name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := password.Hash(req.Password)
	if err != nil {
		h.serverError(w, err)
		return
	}
	token, tokenHash, err := newToken()
	if err != nil {
		h.serverError(w, err)
		return
	}
	p, err := h.store.Register(r.Context(), name, hash, tokenHash, time.Now().Add(sessionTTL))
	if errors.Is(err, store.ErrNameTaken) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, signedIn{Player: profile{Player: p}, Token: token})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil || req.Name == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "enter your name and password")
		return
	}
	if utf8.RuneCountInString(req.Password) > maxPasswordLen {
		writeError(w, http.StatusUnauthorized, errWrongCredentials.Error())
		return
	}
	ctx := r.Context()
	name := strings.Join(strings.Fields(req.Name), " ")

	creds, err := h.store.CredentialsByName(ctx, name)
	if errors.Is(err, store.ErrNotFound) {
		_, _ = password.Verify(req.Password, dummyHash())
		writeError(w, http.StatusUnauthorized, errWrongCredentials.Error())
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}

	if creds.LastFailedLogin != nil {
		wait := loginDelay(creds.FailedLogins) - time.Since(*creds.LastFailedLogin)
		if wait > 0 {
			secs := int(math.Ceil(wait.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			writeError(w, http.StatusTooManyRequests,
				fmt.Sprintf("too many failed attempts; try again in %s", humanSeconds(secs)))
			return
		}
	}

	if creds.PasswordHash == nil {
		// Player from before accounts existed: no password to log in with.
		_, _ = password.Verify(req.Password, dummyHash())
		writeError(w, http.StatusUnauthorized, errWrongCredentials.Error())
		return
	}

	ok, err := password.Verify(req.Password, *creds.PasswordHash)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if !ok {
		if err := h.store.RecordLoginFailure(ctx, creds.ID); err != nil {
			h.serverError(w, err)
			return
		}
		writeError(w, http.StatusUnauthorized, errWrongCredentials.Error())
		return
	}

	if creds.FailedLogins > 0 {
		if err := h.store.ResetLoginFailures(ctx, creds.ID); err != nil {
			h.serverError(w, err)
			return
		}
	}
	if password.NeedsRehash(*creds.PasswordHash) {
		// Upgrade to the current hashing parameters now that we know the password.
		if newHash, err := password.Hash(req.Password); err == nil {
			if err := h.store.SetPassword(ctx, creds.ID, newHash); err != nil {
				h.log.Warn("rehash password", "err", err)
			}
		}
	}

	token, tokenHash, err := newToken()
	if err != nil {
		h.serverError(w, err)
		return
	}
	if err := h.store.CreateSession(ctx, creds.ID, tokenHash, time.Now().Add(sessionTTL)); err != nil {
		h.serverError(w, err)
		return
	}
	stats, err := h.store.PlayerStats(ctx, creds.ID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	p := creds.Player
	writeJSON(w, http.StatusOK, signedIn{Player: profile{Player: &p, Stats: stats}, Token: token})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	_, tokenHash, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteSession(r.Context(), tokenHash); err != nil {
		h.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// changePassword sets a new password. Players who already have one must
// confirm it; players from before accounts set their first one. All other
// sessions are signed out.
func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	p, tokenHash, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, `request body must be {"currentPassword": "...", "newPassword": "..."}`)
		return
	}
	ctx := r.Context()

	creds, err := h.store.CredentialsByID(ctx, p.ID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if creds.PasswordHash != nil {
		ok := false
		if utf8.RuneCountInString(req.CurrentPassword) <= maxPasswordLen {
			if ok, err = password.Verify(req.CurrentPassword, *creds.PasswordHash); err != nil {
				h.serverError(w, err)
				return
			}
		}
		if !ok {
			// 403, not 401: the session is fine, the confirmation isn't.
			writeError(w, http.StatusForbidden, "current password is incorrect")
			return
		}
	}
	if err := validatePassword(req.NewPassword, p.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := password.Hash(req.NewPassword)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if err := h.store.SetPassword(ctx, p.ID, hash); err != nil {
		h.serverError(w, err)
		return
	}
	if err := h.store.DeleteOtherSessions(ctx, p.ID, tokenHash); err != nil {
		h.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func humanSeconds(secs int) string {
	if secs == 1 {
		return "1 second"
	}
	if secs < 60 {
		return fmt.Sprintf("%d seconds", secs)
	}
	mins := (secs + 59) / 60
	if mins == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", mins)
}
