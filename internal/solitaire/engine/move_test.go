package engine

import "testing"

func fixtureGame() *Game {
	return &Game{
		tableau: [7][]TableauCard{
			{{Card: Card{Rank: King, Suit: Spades}, FaceUp: true}},
			{{Card: Card{Rank: Queen, Suit: Hearts}, FaceUp: true}, {Card: Card{Rank: Jack, Suit: Clubs}, FaceUp: true}},
			{{Card: Card{Rank: Ace, Suit: Hearts}, FaceUp: true}},
		},
		waste:  []Card{{Rank: Two, Suit: Hearts}},
		status: Playing,
	}
}

func TestWasteMovesToFoundation(t *testing.T) {
	g := fixtureGame()
	g.waste = []Card{{Rank: Ace, Suit: Hearts}}
	if !g.Move(Location{Kind: Waste}, Location{Kind: Foundation, Pile: int(Hearts)}) {
		t.Fatal("ace of hearts should move to empty hearts foundation")
	}
	top, ok := g.FoundationTop(Hearts)
	if !ok || top != (Card{Rank: Ace, Suit: Hearts}) {
		t.Fatalf("foundation top = %+v/%v, want A hearts", top, ok)
	}
	if len(g.waste) != 0 {
		t.Fatalf("waste length = %d, want 0", len(g.waste))
	}
	if g.Moves() != 1 {
		t.Fatalf("moves = %d, want 1", g.Moves())
	}
}

func TestTableauStackMovesDescendingAlternating(t *testing.T) {
	g := fixtureGame()
	if !g.Move(Location{Kind: Tableau, Pile: 1, Index: 0}, Location{Kind: Tableau, Pile: 0}) {
		t.Fatal("Q hearts / J clubs stack should move onto K spades")
	}
	if len(g.tableau[0]) != 3 {
		t.Fatalf("dest length = %d, want 3", len(g.tableau[0]))
	}
	if len(g.tableau[1]) != 0 {
		t.Fatalf("source length = %d, want 0", len(g.tableau[1]))
	}
	if g.Moves() != 1 {
		t.Fatalf("moves = %d, want 1", g.Moves())
	}
}

func TestEmptyTableauAcceptsOnlyKing(t *testing.T) {
	g := fixtureGame()
	g.tableau[4] = nil
	if g.Move(Location{Kind: Tableau, Pile: 1, Index: 0}, Location{Kind: Tableau, Pile: 4}) {
		t.Fatal("Q-led stack must not move to empty tableau")
	}
	if !g.Move(Location{Kind: Tableau, Pile: 0, Index: 0}, Location{Kind: Tableau, Pile: 4}) {
		t.Fatal("King should move to empty tableau")
	}
	if g.Moves() != 1 {
		t.Fatalf("moves = %d, want 1", g.Moves())
	}
}

func TestInvalidMoveDoesNotIncrementMoves(t *testing.T) {
	g := fixtureGame()
	if g.Move(Location{Kind: Waste}, Location{Kind: Tableau, Pile: 2}) {
		t.Fatal("2 hearts cannot move onto A hearts")
	}
	if g.Moves() != 0 {
		t.Fatalf("moves = %d, want 0", g.Moves())
	}
	if len(g.waste) != 1 || g.waste[0] != (Card{Rank: Two, Suit: Hearts}) {
		t.Fatalf("waste after invalid move = %+v, want 2 hearts", g.waste)
	}
}

func TestMoveFlipsExposedTableauCard(t *testing.T) {
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{
		{Card: Card{Rank: Nine, Suit: Clubs}, FaceUp: false},
		{Card: Card{Rank: Eight, Suit: Hearts}, FaceUp: true},
	}
	g.tableau[1] = []TableauCard{{Card: Card{Rank: Nine, Suit: Spades}, FaceUp: true}}
	if !g.Move(Location{Kind: Tableau, Pile: 0, Index: 1}, Location{Kind: Tableau, Pile: 1}) {
		t.Fatal("8 hearts should move onto 9 spades")
	}
	if !g.tableau[0][0].FaceUp {
		t.Fatal("exposed tableau card should flip face up")
	}
}

func TestFoundationMovesToTableau(t *testing.T) {
	g := &Game{status: Playing}
	g.foundations[Spades] = []Card{{Rank: Ace, Suit: Spades}, {Rank: Two, Suit: Spades}, {Rank: Three, Suit: Spades}}
	g.tableau[0] = []TableauCard{{Card: Card{Rank: Four, Suit: Hearts}, FaceUp: true}}
	if !g.Move(Location{Kind: Foundation, Pile: int(Spades)}, Location{Kind: Tableau, Pile: 0}) {
		t.Fatal("3 spades should move onto 4 hearts")
	}
	top, ok := g.FoundationTop(Spades)
	if !ok || top != (Card{Rank: Two, Suit: Spades}) {
		t.Fatalf("foundation top = %+v/%v, want 2 spades", top, ok)
	}
	got := g.tableau[0][len(g.tableau[0])-1].Card
	if got != (Card{Rank: Three, Suit: Spades}) {
		t.Fatalf("tableau top = %+v, want 3 spades", got)
	}
}

func TestAutoFoundationMovesSingleTopCardToMatchingFoundation(t *testing.T) {
	g := fixtureGame()
	g.foundations[Hearts] = []Card{{Rank: Ace, Suit: Hearts}}
	if !g.AutoFoundation(Location{Kind: Waste}) {
		t.Fatal("2 hearts should auto-move to hearts foundation")
	}
	top, ok := g.FoundationTop(Hearts)
	if !ok || top != (Card{Rank: Two, Suit: Hearts}) {
		t.Fatalf("foundation top = %+v/%v, want 2 hearts", top, ok)
	}
}

func TestInvalidLocationsAreSafeNoOps(t *testing.T) {
	g := fixtureGame()
	if g.Move(Location{Kind: Tableau, Pile: -1}, Location{Kind: Foundation, Pile: int(Hearts)}) {
		t.Fatal("invalid source pile should not move")
	}
	if g.Move(Location{Kind: Waste}, Location{Kind: Foundation, Pile: 9}) {
		t.Fatal("invalid destination pile should not move")
	}
	if g.Move(Location{Kind: Stock}, Location{Kind: Tableau, Pile: 0}) {
		t.Fatal("stock is not a valid move source")
	}
	if g.Moves() != 0 {
		t.Fatalf("moves = %d, want 0", g.Moves())
	}
}
