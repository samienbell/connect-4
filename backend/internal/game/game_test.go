package game

import (
	"errors"
	"testing"
)

func play(t *testing.T, g *Game, cols ...int) {
	t.Helper()
	for i, c := range cols {
		if _, err := g.Play(c); err != nil {
			t.Fatalf("move %d (col %d): %v", i+1, c, err)
		}
	}
}

func TestWins(t *testing.T) {
	tests := []struct {
		name  string
		moves []int
	}{
		{"horizontal", []int{0, 0, 1, 1, 2, 2, 3}},
		{"vertical", []int{0, 1, 0, 1, 0, 1, 0}},
		{"diagonal up-right", []int{0, 1, 1, 2, 2, 3, 2, 3, 3, 5, 3}},
		{"diagonal up-left", []int{6, 5, 5, 4, 4, 3, 4, 3, 3, 1, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(ModeLocal)
			play(t, g, tt.moves...)
			if g.Status != StatusWon || g.Winner != Player1 {
				t.Fatalf("got status=%s winner=%d, want won by player 1", g.Status, g.Winner)
			}
		})
	}
}

func TestDraw(t *testing.T) {
	// Fill the board with a pattern that has no four in a row, leaving the
	// top-left cell for the final move. Counting rows from the bottom (r) the
	// pattern is 1 + ((c/2 + r) % 2): pairs of columns, alternating each row.
	g := New(ModeLocal)
	for row := 0; row < Rows; row++ {
		for col := 0; col < Cols; col++ {
			r := Rows - 1 - row
			g.Board[row][col] = Player(1 + (col/2+r)%2)
		}
	}
	g.Board[0][0] = Empty
	g.MoveCount = MaxMoves - 1
	g.CurrentPlayer = Player(1 + (0+Rows-1)%2) // the colour the pattern expects there

	if _, err := g.Play(0); err != nil {
		t.Fatal(err)
	}
	if g.Status != StatusDraw || g.Winner != Empty {
		t.Fatalf("got status=%s winner=%d, want draw", g.Status, g.Winner)
	}
}

func TestTurnsAlternate(t *testing.T) {
	g := New(ModeLocal)
	play(t, g, 0, 0, 1, 1, 2, 2)
	if g.Status != StatusInProgress {
		t.Fatalf("status = %s, want in_progress", g.Status)
	}
	if g.CurrentPlayer != Player1 {
		t.Fatalf("current player = %d, want 1", g.CurrentPlayer)
	}
}

func TestDiscLandsOnBottom(t *testing.T) {
	g := New(ModeLocal)
	row, err := g.Play(3)
	if err != nil || row != Rows-1 {
		t.Fatalf("got row=%d err=%v, want row=%d", row, err, Rows-1)
	}
	if row, _ = g.Play(3); row != Rows-2 {
		t.Fatalf("second disc row = %d, want %d", row, Rows-2)
	}
}

func TestInvalidColumn(t *testing.T) {
	g := New(ModeLocal)
	for _, c := range []int{-1, Cols} {
		if _, err := g.Play(c); !errors.Is(err, ErrInvalidColumn) {
			t.Errorf("col %d: err = %v, want ErrInvalidColumn", c, err)
		}
	}
}

func TestColumnFull(t *testing.T) {
	g := New(ModeLocal)
	play(t, g, 0, 0, 0, 0, 0, 0)
	if _, err := g.Play(0); !errors.Is(err, ErrColumnFull) {
		t.Fatalf("err = %v, want ErrColumnFull", err)
	}
}

func TestGameOver(t *testing.T) {
	g := New(ModeLocal)
	play(t, g, 0, 0, 1, 1, 2, 2, 3)
	if _, err := g.Play(4); !errors.Is(err, ErrGameOver) {
		t.Fatalf("err = %v, want ErrGameOver", err)
	}
}

func TestOnlineGameWaitsForOpponent(t *testing.T) {
	g := New(ModeOnline)
	g.Player1 = &PlayerRef{ID: "a", Name: "Alice"}
	if !g.Waiting() {
		t.Fatal("new online game should be waiting")
	}
	if _, err := g.Play(0); !errors.Is(err, ErrWaitingForOpponent) {
		t.Fatalf("err = %v, want ErrWaitingForOpponent", err)
	}
	g.Player2 = &PlayerRef{ID: "b", Name: "Bob"}
	if g.Waiting() {
		t.Fatal("game with two players should not be waiting")
	}
	if _, err := g.Play(0); err != nil {
		t.Fatalf("move after join: %v", err)
	}
}

func TestSeatOf(t *testing.T) {
	g := New(ModeOnline)
	g.Player1 = &PlayerRef{ID: "a"}
	g.Player2 = &PlayerRef{ID: "b"}
	for id, want := range map[string]Player{"a": Player1, "b": Player2, "c": Empty, "": Empty} {
		if got := g.SeatOf(id); got != want {
			t.Errorf("SeatOf(%q) = %d, want %d", id, got, want)
		}
	}
}
