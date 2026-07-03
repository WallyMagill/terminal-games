package engine

import "testing"

func TestCloneDoesNotAliasMutableSlices(t *testing.T) {
	g := &Game{status: Playing}
	g.stock = []Card{{Rank: Ace, Suit: Spades}}
	g.waste = []Card{{Rank: Two, Suit: Hearts}}
	g.tableau[0] = []TableauCard{{Card: Card{Rank: Three, Suit: Clubs}, FaceUp: true}}
	g.foundations[Diamonds] = []Card{{Rank: Ace, Suit: Diamonds}}
	g.moves = 12

	clone := g.Clone()
	clone.stock[0] = Card{Rank: King, Suit: Clubs}
	clone.waste[0] = Card{Rank: Queen, Suit: Clubs}
	clone.tableau[0][0] = TableauCard{Card: Card{Rank: Jack, Suit: Clubs}, FaceUp: false}
	clone.foundations[Diamonds][0] = Card{Rank: Ten, Suit: Clubs}
	clone.moves = 99

	if g.stock[0].Rank != Ace || g.waste[0].Rank != Two || g.tableau[0][0].Card.Rank != Three || g.foundations[Diamonds][0].Rank != Ace || g.moves != 12 {
		t.Fatalf("clone mutated original: original=%+v clone=%+v", g, clone)
	}
}

func TestCloneCopiesStatus(t *testing.T) {
	g := &Game{status: Won}

	clone := g.Clone()
	clone.status = Playing

	if g.status != Won {
		t.Fatalf("clone status mutation changed original status: original=%v clone=%v", g.status, clone.status)
	}
	if clone.status != Playing {
		t.Fatalf("clone status = %v, want %v after mutation", clone.status, Playing)
	}
}
