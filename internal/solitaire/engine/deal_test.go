package engine

import (
	"errors"
	"math/rand/v2"
	"testing"
)

func testRNG() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestNewRequiresRNG(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, ErrNilRNG) {
		t.Fatalf("New(nil) error = %v, want ErrNilRNG", err)
	}
}

func TestNewDealsKlondikeTableau(t *testing.T) {
	g, err := New(testRNG())
	if err != nil {
		t.Fatal(err)
	}
	if g.StockCount() != 24 {
		t.Fatalf("stock count = %d, want 24", g.StockCount())
	}
	seen := map[Card]bool{}
	for col := range 7 {
		cards := g.Tableau(col)
		if len(cards) != col+1 {
			t.Fatalf("tableau %d length = %d, want %d", col, len(cards), col+1)
		}
		for i, c := range cards {
			if seen[c.Card] {
				t.Fatalf("duplicate card in deal: %+v", c.Card)
			}
			seen[c.Card] = true
			wantFaceUp := i == len(cards)-1
			if c.FaceUp != wantFaceUp {
				t.Fatalf("tableau %d card %d FaceUp = %v, want %v", col, i, c.FaceUp, wantFaceUp)
			}
		}
	}
	if len(seen) != 28 {
		t.Fatalf("tableau unique cards = %d, want 28", len(seen))
	}
}

func TestNewInitialState(t *testing.T) {
	g, err := New(testRNG())
	if err != nil {
		t.Fatal(err)
	}
	if g.Status() != Playing {
		t.Fatalf("status = %v, want Playing", g.Status())
	}
	if g.Moves() != 0 {
		t.Fatalf("moves = %d, want 0", g.Moves())
	}
	if len(g.WasteVisible()) != 0 {
		t.Fatalf("visible waste = %d, want 0", len(g.WasteVisible()))
	}
	for suit := Spades; suit <= Clubs; suit++ {
		if top, ok := g.FoundationTop(suit); ok || top != (Card{}) {
			t.Fatalf("FoundationTop(%v) = %+v/%v, want zero/false", suit, top, ok)
		}
	}
}

func TestSnapshotAccessorsDoNotExposeMutableState(t *testing.T) {
	g, err := New(testRNG())
	if err != nil {
		t.Fatal(err)
	}
	tableau := g.Tableau(0)
	tableau[0].FaceUp = false
	if !g.Tableau(0)[0].FaceUp {
		t.Fatal("Tableau returned mutable backing storage")
	}
	if got := g.Tableau(-1); got != nil {
		t.Fatalf("Tableau(-1) = %v, want nil", got)
	}
	if got := g.Tableau(7); got != nil {
		t.Fatalf("Tableau(7) = %v, want nil", got)
	}
}
