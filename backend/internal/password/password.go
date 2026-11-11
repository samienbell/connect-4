// Package password hashes and verifies passwords with Argon2id.
//
// Argon2id is deliberately slow and memory-hard, so guessing passwords from a
// stolen hash is expensive. Every hash gets its own random salt, so two users
// with the same password get different hashes and precomputed (rainbow table)
// attacks don't work.
//
// Hashes are stored in the standard PHC string format, which records the
// salt and parameters alongside the hash:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<base64 salt>$<base64 hash>
//
// Because the parameters travel with each hash, Default can be raised later;
// NeedsRehash tells the caller to upgrade a user's hash on their next login.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32 // bytes
	KeyLength   uint32 // bytes
}

// Default follows the OWASP Password Storage Cheat Sheet minimum for
// Argon2id: 19 MiB of memory, 2 iterations, 1 degree of parallelism.
var Default = Params{Memory: 19 * 1024, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32}

// maxMemory rejects stored hashes that would need an absurd amount of memory.
const maxMemory = 1 << 20 // 1 GiB in KiB

var ErrMalformed = errors.New("password: malformed hash")

// slots limits how many hashes run at once. Each one needs Params.Memory, so
// without a cap a burst of logins could exhaust the container's memory.
var slots = make(chan struct{}, 4)

var b64 = base64.RawStdEncoding

// Hash returns a salted Argon2id hash of pw in PHC string format.
func Hash(pw string) (string, error) {
	p := Default
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := derive(pw, salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify reports whether pw matches encoded, re-deriving the hash with the
// salt and parameters stored in encoded. The comparison is constant-time.
func Verify(pw, encoded string) (bool, error) {
	p, salt, key, err := decode(encoded)
	if err != nil {
		return false, err
	}
	other := derive(pw, salt, p.Iterations, p.Memory, p.Parallelism, uint32(len(key)))
	return subtle.ConstantTimeCompare(key, other) == 1, nil
}

// NeedsRehash reports whether encoded was made with parameters other than
// Default (for example, before Default was strengthened).
func NeedsRehash(encoded string) bool {
	p, _, _, err := decode(encoded)
	if err != nil {
		return true
	}
	return p != Default
}

func derive(pw string, salt []byte, iterations, memory uint32, parallelism uint8, keyLen uint32) []byte {
	slots <- struct{}{}
	defer func() { <-slots }()
	return argon2.IDKey([]byte(pw), salt, iterations, memory, parallelism, keyLen)
}

func decode(encoded string) (p Params, salt, key []byte, err error) {
	// "$argon2id$v=19$m=..,t=..,p=..$salt$key" splits into 6 parts, the first empty.
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return p, nil, nil, ErrMalformed
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, nil, nil, ErrMalformed
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return p, nil, nil, ErrMalformed
	}
	if p.Memory == 0 || p.Memory > maxMemory || p.Iterations == 0 || p.Parallelism == 0 {
		return p, nil, nil, ErrMalformed
	}
	if salt, err = b64.DecodeString(parts[4]); err != nil || len(salt) == 0 {
		return p, nil, nil, ErrMalformed
	}
	if key, err = b64.DecodeString(parts[5]); err != nil || len(key) == 0 {
		return p, nil, nil, ErrMalformed
	}
	p.SaltLength, p.KeyLength = uint32(len(salt)), uint32(len(key))
	return p, salt, key, nil
}
