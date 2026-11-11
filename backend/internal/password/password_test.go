package password

import (
	"errors"
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	h, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected format: %s", h)
	}
	if strings.Contains(h, "correct horse") {
		t.Fatal("hash contains the plaintext password")
	}

	ok, err := Verify("correct horse battery staple", h)
	if err != nil || !ok {
		t.Fatalf("correct password: ok=%v err=%v", ok, err)
	}
	ok, err = Verify("correct horse battery stapler", h)
	if err != nil || ok {
		t.Fatalf("wrong password: ok=%v err=%v", ok, err)
	}
}

func TestSaltMakesHashesUnique(t *testing.T) {
	a, _ := Hash("same password")
	b, _ := Hash("same password")
	if a == b {
		t.Fatal("hashing the same password twice gave identical hashes; salt is not random")
	}
	saltA, saltB := strings.Split(a, "$")[4], strings.Split(b, "$")[4]
	if saltA == saltB {
		t.Fatal("salts should differ")
	}
	for _, h := range []string{a, b} {
		if ok, _ := Verify("same password", h); !ok {
			t.Fatal("both hashes should verify")
		}
	}
}

func TestMalformedHashes(t *testing.T) {
	good, _ := Hash("pw")
	parts := strings.Split(good, "$")
	bad := []string{
		"",
		"plaintext",
		"$2a$12$bcrypthashlookslikethis",
		strings.Replace(good, "argon2id", "argon2i", 1),
		strings.Replace(good, "v=19", "v=16", 1),
		"$argon2id$v=19$m=0,t=2,p=1$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=99999999,t=2,p=1$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=19456,t=2,p=1$!!notbase64!!$" + parts[5],
		"$argon2id$v=19$m=19456,t=2,p=1$" + parts[4] + "$",
	}
	for _, h := range bad {
		if _, err := Verify("pw", h); !errors.Is(err, ErrMalformed) {
			t.Errorf("Verify(%q) err = %v, want ErrMalformed", h, err)
		}
	}
}

func TestNeedsRehash(t *testing.T) {
	current, _ := Hash("pw")
	if NeedsRehash(current) {
		t.Fatal("a fresh hash should not need rehashing")
	}
	weaker := strings.Replace(current, "t=2", "t=1", 1)
	if !NeedsRehash(weaker) {
		t.Fatal("a hash with fewer iterations should need rehashing")
	}
	if !NeedsRehash("garbage") {
		t.Fatal("a malformed hash should need rehashing")
	}
}
