package engine

import "testing"

func TestWinWhenAllFoundationsComplete(t *testing.T) {
	g := &Game{status: Playing}
	for suit := Spades; suit <= Clubs; suit++ {
		topRank := King
		if suit == Spades {
			topRank = Queen
		}
		for rank := Ace; rank <= topRank; rank++ {
			g.foundations[suit] = append(g.foundations[suit], Card{Rank: rank, Suit: suit})
		}
	}
	g.tableau[0] = []TableauCard{{Card: Card{Rank: King, Suit: Spades}, FaceUp: true}}
	if !g.Move(Location{Kind: Tableau, Pile: 0, Index: 0}, Location{Kind: Foundation, Pile: int(Spades)}) {
		t.Fatal("king of spades should complete spades foundation")
	}
	if g.Status() != Won {
		t.Fatalf("status = %v, want Won", g.Status())
	}
}
