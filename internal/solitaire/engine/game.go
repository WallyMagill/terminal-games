package engine

import (
	"errors"
	"math/rand"
)

var ErrNilRNG = errors.New("rng must not be nil")

type Status uint8

const (
	Playing Status = iota
	Won
)

type Game struct {
	stock       []Card
	waste       []Card
	tableau     [7][]TableauCard
	foundations [4][]Card
	moves       int
	status      Status
}

func New(rng *rand.Rand) (*Game, error) {
	if rng == nil {
		return nil, ErrNilRNG
	}
	deck := standardDeck()
	shuffle(deck, rng)
	g := &Game{status: Playing}
	next := 0
	for col := range len(g.tableau) {
		g.tableau[col] = make([]TableauCard, col+1)
		for row := range col + 1 {
			g.tableau[col][row] = TableauCard{
				Card:   deck[next],
				FaceUp: row == col,
			}
			next++
		}
	}
	g.stock = append(g.stock, deck[next:]...)
	return g, nil
}

func (g *Game) Draw() bool {
	if g.status == Won {
		return false
	}
	if len(g.stock) == 0 {
		if len(g.waste) == 0 {
			return false
		}
		// Classic flip: rebuild the stock so every pass re-deals the same
		// triples with the same tops. Walk the waste bottom-up in chunks of 3
		// (the original draw groups) and stack the chunks in reverse order —
		// the first-drawn group ends up back on top.
		n := len(g.waste)
		for start := ((n - 1) / 3) * 3; start >= 0; start -= 3 {
			end := start + 3
			if end > n {
				end = n
			}
			g.stock = append(g.stock, g.waste[start:end]...)
		}
		g.waste = nil
		g.moves++
		return true
	}
	n := 3
	if len(g.stock) < n {
		n = len(g.stock)
	}
	start := len(g.stock) - n
	g.waste = append(g.waste, g.stock[start:]...)
	g.stock = g.stock[:start]
	g.moves++
	return true
}

func (g *Game) Moves() int      { return g.moves }
func (g *Game) Status() Status  { return g.status }
func (g *Game) StockCount() int { return len(g.stock) }

func (g *Game) TableauLen(col int) int {
	if col < 0 || col >= len(g.tableau) {
		return 0
	}
	return len(g.tableau[col])
}
func (g *Game) WasteVisible() []Card {
	start := len(g.waste) - 3
	if start < 0 {
		start = 0
	}
	return append([]Card(nil), g.waste[start:]...)
}

func (g *Game) FoundationTop(s Suit) (Card, bool) {
	if s > Clubs || len(g.foundations[s]) == 0 {
		return Card{}, false
	}
	pile := g.foundations[s]
	return pile[len(pile)-1], true
}

func (g *Game) Tableau(col int) []TableauCard {
	if col < 0 || col >= len(g.tableau) {
		return nil
	}
	return append([]TableauCard(nil), g.tableau[col]...)
}
