// Package engine implements classic minesweeper rules with no UI dependencies.
package engine

import (
	"errors"
	"math/rand"
)

type CellState uint8

const (
	Hidden CellState = iota
	Flagged
	Questioned
	Revealed
)

type Status uint8

const (
	NotStarted Status = iota
	Playing
	Won
	Lost
)

type Cell struct {
	Mine     bool
	State    CellState
	Adjacent int
}

var (
	ErrBoardTooSmall = errors.New("board must be at least 2x2")
	ErrMineCount     = errors.New("mines must be between 1 and width*height-2")
)

// Game is a single minesweeper round. Mines are placed lazily on the first
// Reveal so the first click is never a mine.
type Game struct {
	width, height, mines int
	cells                []Cell
	status               Status
	minesPlaced          bool
	// rng is math/rand v1 by choice: rand.New(rand.NewSource(seed)) is not
	// deprecated and int64 seeding keeps tests ergonomic. v2 offers nothing
	// this package needs.
	rng        *rand.Rand
	flags      int
	revealed   int
	hitX, hitY int
	mineHit    bool
}

func New(width, height, mines int, rng *rand.Rand) (*Game, error) {
	if width < 2 || height < 2 {
		return nil, ErrBoardTooSmall
	}
	if mines < 1 || mines > width*height-2 {
		return nil, ErrMineCount
	}
	return &Game{
		width:  width,
		height: height,
		mines:  mines,
		cells:  make([]Cell, width*height),
		rng:    rng,
	}, nil
}

// Width, Height, and Mines return the board configuration passed to New.
// Status returns the current game lifecycle state.
func (g *Game) Width() int     { return g.width }
func (g *Game) Height() int    { return g.height }
func (g *Game) Mines() int     { return g.mines }
func (g *Game) Status() Status { return g.status }

// MinesRemaining is the HUD counter: total mines minus placed flags.
// It can go negative when the player over-flags.
func (g *Game) MinesRemaining() int { return g.mines - g.flags }

// CellAt returns a snapshot copy of the cell, or ok=false out of bounds.
func (g *Game) CellAt(x, y int) (Cell, bool) {
	if !g.inBounds(x, y) {
		return Cell{}, false
	}
	return g.cells[y*g.width+x], true
}

// HitMine reports which mine ended the game after a loss.
func (g *Game) HitMine() (x, y int, ok bool) {
	return g.hitX, g.hitY, g.mineHit
}

func (g *Game) inBounds(x, y int) bool {
	return x >= 0 && x < g.width && y >= 0 && y < g.height
}

func (g *Game) at(x, y int) *Cell { return &g.cells[y*g.width+x] }

func (g *Game) eachNeighbor(x, y int, fn func(nx, ny int)) {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if nx, ny := x+dx, y+dy; g.inBounds(nx, ny) {
				fn(nx, ny)
			}
		}
	}
}

// placeMines fills the board on first reveal, excluding the clicked cell so
// the first click is never a mine (classic rule — it may still be a number).
func (g *Game) placeMines(safeX, safeY int) {
	safe := safeY*g.width + safeX
	candidates := make([]int, 0, len(g.cells)-1)
	for i := range g.cells {
		if i != safe {
			candidates = append(candidates, i)
		}
	}
	g.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	for _, idx := range candidates[:g.mines] {
		g.cells[idx].Mine = true
	}
	g.computeAdjacency()
	g.minesPlaced = true
}

func (g *Game) computeAdjacency() {
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			n := 0
			g.eachNeighbor(x, y, func(nx, ny int) {
				if g.at(nx, ny).Mine {
					n++
				}
			})
			g.at(x, y).Adjacent = n
		}
	}
}
