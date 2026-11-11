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

// Player is 0 for an empty cell, or 1 (Red) / 2 (Yellow) for a disc or seat.
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

// Mode says who may make moves.
type Mode string

const (
	// ModeLocal: two people share one screen; anyone with the link can move.
	ModeLocal Mode = "local"
	// ModeOnline: each seat belongs to a signed-in player who may only move on their turn.
	ModeOnline Mode = "online"
)

// Board is indexed [row][col]; row 0 is the top row.
type Board [Rows][Cols]Player

var (
	ErrInvalidColumn      = errors.New("column must be between 0 and 6")
	ErrColumnFull         = errors.New("column is full")
	ErrGameOver           = errors.New("game is already over")
	ErrWaitingForOpponent = errors.New("waiting for an opponent to join")
)

// PlayerRef identifies the player sitting in a seat.
type PlayerRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Game struct {
	ID            string     `json:"id"`
	Mode          Mode       `json:"mode"`
	Board         Board      `json:"board"`
	CurrentPlayer Player     `json:"currentPlayer"`
	Status        Status     `json:"status"`
	Winner        Player     `json:"winner"` // 0 when there is no winner
	MoveCount     int        `json:"moveCount"`
	Player1       *PlayerRef `json:"player1"` // Red; nil for anonymous local games
	Player2       *PlayerRef `json:"player2"` // Yellow; nil until someone joins an online game
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	FinishedAt    *time.Time `json:"finishedAt"`
}

func New(mode Mode) *Game {
	return &Game{Mode: mode, CurrentPlayer: Player1, Status: StatusInProgress}
}

// Waiting reports whether an online game still needs its second player.
func (g *Game) Waiting() bool {
	return g.Mode == ModeOnline && g.Player2 == nil
}

// SeatOf returns the seat (Player1 or Player2) held by playerID, or Empty.
func (g *Game) SeatOf(playerID string) Player {
	switch {
	case playerID == "":
		return Empty
	case g.Player1 != nil && g.Player1.ID == playerID:
		return Player1
	case g.Player2 != nil && g.Player2.ID == playerID:
		return Player2
	}
	return Empty
}

// Play drops the current player's disc into col and returns the row it landed in.
func (g *Game) Play(col int) (int, error) {
	if g.Status != StatusInProgress {
		return -1, ErrGameOver
	}
	if g.Waiting() {
		return -1, ErrWaitingForOpponent
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
