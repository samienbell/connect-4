// Package game holds the Connect 4 rules. It does no I/O, so it is easy to test.
package game

import (
	"errors"
	"time"
)

const (
	Rows     = 6
	Cols     = 7
	ToWin    = 4
	MaxMoves = Rows * Cols
)

// Player is 0 for an empty cell, or 1 / 2 for a disc.
type Player int

const (
	Empty   Player = 0
	Player1 Player = 1
	Player2 Player = 2
)

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusWon        Status = "won"
	StatusDraw       Status = "draw"
)

// Board is indexed [row][col]; row 0 is the top row.
type Board [Rows][Cols]Player

var (
	ErrInvalidColumn = errors.New("column must be between 0 and 6")
	ErrColumnFull    = errors.New("column is full")
	ErrGameOver      = errors.New("game is already over")
)

type Game struct {
	ID            string    `json:"id"`
	Board         Board     `json:"board"`
	CurrentPlayer Player    `json:"currentPlayer"`
	Status        Status    `json:"status"`
	Winner        Player    `json:"winner"` // 0 when there is no winner
	MoveCount     int       `json:"moveCount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func New() *Game {
	return &Game{CurrentPlayer: Player1, Status: StatusInProgress}
}

// Play drops the current player's disc into col and returns the row it landed in.
func (g *Game) Play(col int) (int, error) {
	if g.Status != StatusInProgress {
		return -1, ErrGameOver
	}
	if col < 0 || col >= Cols {
		return -1, ErrInvalidColumn
	}
	for row := Rows - 1; row >= 0; row-- {
		if g.Board[row][col] != Empty {
			continue
		}
		p := g.CurrentPlayer
		g.Board[row][col] = p
		g.MoveCount++
		switch {
		case g.Board.connects(row, col):
			g.Status = StatusWon
			g.Winner = p
		case g.MoveCount == MaxMoves:
			g.Status = StatusDraw
		default:
			g.CurrentPlayer = Player1 + Player2 - p
		}
		return row, nil
	}
	return -1, ErrColumnFull
}

// connects reports whether the disc at (row, col) is part of a line of ToWin.
func (b *Board) connects(row, col int) bool {
	p := b[row][col]
	for _, d := range [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}} {
		n := 1 + b.count(row, col, d[0], d[1], p) + b.count(row, col, -d[0], -d[1], p)
		if n >= ToWin {
			return true
		}
	}
	return false
}

// count returns how many consecutive discs of p lie beyond (row, col) in direction (dr, dc).
func (b *Board) count(row, col, dr, dc int, p Player) int {
	n := 0
	for r, c := row+dr, col+dc; r >= 0 && r < Rows && c >= 0 && c < Cols && b[r][c] == p; r, c = r+dr, c+dc {
		n++
	}
	return n
}
