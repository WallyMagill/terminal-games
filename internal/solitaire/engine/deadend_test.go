package engine

import "testing"

// shuffleOnlyGame has exactly one legal card move (8♠ from 9♥ onto 9♦) which
// reveals nothing — a pure reshuffle. Empty piles accept only kings; there are
// no kings, no aces, and no stock or waste.
func shuffleOnlyGame() *Game {
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{
		{Card: Card{Suit: Hearts, Rank: Nine}, FaceUp: true},
		{Card: Card{Suit: Spades, Rank: Eight}, FaceUp: true},
	}
	g.tableau[1] = []TableauCard{
		{Card: Card{Suit: Diamonds, Rank: Nine}, FaceUp: true},
	}
	return g
}

func TestProductiveMovesExcludesShuffles(t *testing.T) {
	g := shuffleOnlyGame()
	if len(g.LegalMoves()) == 0 {
		t.Fatal("expected at least the 8♠→9♦ shuffle to be legal")
	}
	if got := g.ProductiveMoves(); len(got) != 0 {
		t.Fatalf("shuffle-only position must have no productive moves, got %d", len(got))
	}
}

func TestProductiveMovesIncludesFoundationRevealAndDraw(t *testing.T) {
	// Ace on a tableau pile: foundation move is productive.
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Clubs, Rank: Ace}, FaceUp: true}}
	if len(g.ProductiveMoves()) == 0 {
		t.Fatal("ace to foundation must be productive")
	}

	// Revealing move: face-down card under a movable card.
	g = &Game{status: Playing}
	g.tableau[0] = []TableauCard{
		{Card: Card{Suit: Clubs, Rank: Five}, FaceUp: false},
		{Card: Card{Suit: Spades, Rank: Eight}, FaceUp: true},
	}
	g.tableau[1] = []TableauCard{{Card: Card{Suit: Diamonds, Rank: Nine}, FaceUp: true}}
	moves := g.ProductiveMoves()
	if len(moves) == 0 {
		t.Fatal("revealing 8♠→9♦ must be productive")
	}

	// Waste-source move is productive even to a tableau pile.
	g = &Game{status: Playing}
	g.waste = []Card{{Suit: Spades, Rank: Eight}}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Diamonds, Rank: Nine}, FaceUp: true}}
	if len(g.ProductiveMoves()) == 0 {
		t.Fatal("draw-pile card onto tableau must be productive")
	}
}

func TestDeadEndTrueWhenNothingReachable(t *testing.T) {
	// No stock, no waste, only a shuffle available.
	if g := shuffleOnlyGame(); !g.DeadEnd() {
		t.Fatal("shuffle-only position with empty deck must be a dead end")
	}

	// Full deck cycle with no productive tops: three unplayable cards cycle
	// forever with Q♦ as the only reachable top.
	g := shuffleOnlyGame()
	g.stock = []Card{
		{Suit: Clubs, Rank: Five},
		{Suit: Hearts, Rank: Queen},
		{Suit: Diamonds, Rank: Queen},
	}
	if !g.DeadEnd() {
		t.Fatal("cycling deck with no playable top must be a dead end")
	}
}

func TestDeadEndFalseWhenCardBuriedInDeck(t *testing.T) {
	// Ace is not the current waste top but surfaces after one draw.
	g := shuffleOnlyGame()
	g.stock = []Card{
		{Suit: Clubs, Rank: Five},
		{Suit: Clubs, Rank: Ace},
	}
	if g.DeadEnd() {
		t.Fatal("ace reachable via draw must not be a dead end")
	}
}

func TestDeadEndFalseWhenCardSurfacesOnlyAfterPermutedRecycle(t *testing.T) {
	// 7 cards: stock+waste ≢ 0 mod 3, so this engine's reverse-chunk recycle
	// permutes the deck each cycle. The ace at index 4 is never a waste top
	// in cycle 1; it surfaces only after a recycle reorders the deck.
	// Length-based cycle detection would wrongly report a dead end here.
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Hearts, Rank: Five}, FaceUp: true}}
	g.stock = []Card{
		{Suit: Clubs, Rank: Ten},
		{Suit: Diamonds, Rank: Ten},
		{Suit: Hearts, Rank: Ten},
		{Suit: Spades, Rank: Ten},
		{Suit: Clubs, Rank: Ace},
		{Suit: Diamonds, Rank: Nine},
		{Suit: Clubs, Rank: Queen},
	}
	if g.DeadEnd() {
		t.Fatal("ace reachable only via permuted recycle must not be a dead end")
	}
}

func TestDeadEndFalseWhenProductiveNowOrWon(t *testing.T) {
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Clubs, Rank: Ace}, FaceUp: true}}
	if g.DeadEnd() {
		t.Fatal("position with immediate productive move is not dead")
	}
	won := &Game{status: Won}
	if won.DeadEnd() {
		t.Fatal("won game is never a dead end")
	}
}

func TestDeadEndDoesNotMutate(t *testing.T) {
	g := shuffleOnlyGame()
	g.stock = []Card{
		{Suit: Clubs, Rank: Five},
		{Suit: Hearts, Rank: Queen},
		{Suit: Diamonds, Rank: Queen},
	}
	g.DeadEnd()
	if g.Moves() != 0 || g.StockCount() != 3 || len(g.waste) != 0 {
		t.Fatalf("DeadEnd mutated game: moves=%d stock=%d waste=%d", g.Moves(), g.StockCount(), len(g.waste))
	}
}
