package engine

import (
	"errors"
	"math/rand"
)

var ErrNilRNG = errors.New("rng must not be nil")

type Dir int

const (
	DirUp Dir = iota
	DirDown
	DirLeft
	DirRight
)

type Game struct {
	board [Size][Size]int
	score int
	won   bool
	over  bool
	rng   *rand.Rand
}

type MoveResult struct {
	Changed bool
	Score   int
	Won     bool // true only if this move first reached >= 2048
	Over    bool
}

func New(rng *rand.Rand) (*Game, error) {
	if rng == nil {
		return nil, ErrNilRNG
	}
	g := &Game{rng: rng}
	g.spawn()
	g.spawn()
	g.over = g.isDead()
	return g, nil
}

func (g *Game) Board() [Size][Size]int { return g.board }
func (g *Game) Score() int             { return g.score }
func (g *Game) Won() bool              { return g.won }
func (g *Game) Over() bool             { return g.over }

func (g *Game) HighestTile() int {
	best := 0
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			if g.board[r][c] > best {
				best = g.board[r][c]
			}
		}
	}
	return best
}

func (g *Game) isDead() bool {
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			if g.board[r][c] == 0 {
				return false
			}
			v := g.board[r][c]
			if c+1 < Size && g.board[r][c+1] == v {
				return false
			}
			if r+1 < Size && g.board[r+1][c] == v {
				return false
			}
		}
	}
	return true
}
