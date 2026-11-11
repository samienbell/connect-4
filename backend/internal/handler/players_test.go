package handler

import (
	"bytes"
	"errors"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		in, want string
		err      error
	}{
		{"Alice", "Alice", nil},
		{"  Bob  ", "Bob", nil},
		{"Mary   Jane", "Mary Jane", nil},
		{"José_99", "José_99", nil},
		{"李雷", "李雷", nil},
		{"a", "", errNameLength},
		{"   ", "", errNameLength},
		{"abcdefghijklmnopqrstu", "", errNameLength}, // 21 characters
		{"<script>", "", errNameChars},
		{"-dash", "", errNameChars},
		{"semi;colon", "", errNameChars},
	}
	for _, tt := range tests {
		got, err := validateName(tt.in)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("validateName(%q) = %q, %v; want %q, %v", tt.in, got, err, tt.want, tt.err)
		}
	}
}

func TestTokens(t *testing.T) {
	tok1, hash1, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	tok2, _, _ := newToken()
	if tok1 == tok2 {
		t.Fatal("two tokens should not be equal")
	}
	if len(tok1) < 40 {
		t.Fatalf("token too short: %d chars", len(tok1))
	}
	if !bytes.Equal(hash1, hashToken(tok1)) {
		t.Fatal("hashToken should match the hash returned by newToken")
	}
}
