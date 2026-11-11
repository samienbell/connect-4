package handler

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		pw, name string
		want     error
	}{
		{"hunter22!", "Alice", nil},
		{"short", "Alice", errPasswordLength},
		{strings.Repeat("a", maxPasswordLen+1), "Alice", errPasswordLength},
		{strings.Repeat("é", maxPasswordLen), "Alice", nil}, // counts characters, not bytes
		{"Alice123", "Alice123", errPasswordIsName},
		{"alice123", "Alice123", errPasswordIsName},
	}
	for _, tt := range tests {
		if err := validatePassword(tt.pw, tt.name); !errors.Is(err, tt.want) {
			t.Errorf("validatePassword(%q, %q) = %v, want %v", tt.pw, tt.name, err, tt.want)
		}
	}
}

func TestLoginDelay(t *testing.T) {
	tests := map[int]time.Duration{
		0:   0,
		4:   0,
		5:   30 * time.Second,
		6:   time.Minute,
		7:   2 * time.Minute,
		9:   8 * time.Minute,
		10:  maxLoginDelay,
		500: maxLoginDelay, // must not overflow
	}
	for failures, want := range tests {
		if got := loginDelay(failures); got != want {
			t.Errorf("loginDelay(%d) = %v, want %v", failures, got, want)
		}
	}
}

func TestHumanSeconds(t *testing.T) {
	for secs, want := range map[int]string{1: "1 second", 2: "2 seconds", 59: "59 seconds", 60: "1 minute", 61: "2 minutes", 900: "15 minutes"} {
		if got := humanSeconds(secs); got != want {
			t.Errorf("humanSeconds(%d) = %q, want %q", secs, got, want)
		}
	}
}
