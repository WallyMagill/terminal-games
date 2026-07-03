package engine

import "testing"

func TestLegalMovesIncludesDrawAndRecycle(t *testing.T) {
	g := &Game{status: Playing, stock: []Card{{Rank: Ace, Suit: Spades}}, waste: nil}
	moves := g.LegalMoves()
	if !hasMoveKind(moves, MoveDraw) {
		t.Fatalf("LegalMoves missing draw move: %+v", moves)
	}

	g = &Game{status: Playing, waste: []Card{{Rank: Ace, Suit: Spades}}, stock: nil}
	moves = g.LegalMoves()
	if !hasMoveKind(moves, MoveRecycle) {
		t.Fatalf("LegalMoves missing recycle move: %+v", moves)
	}
}

func TestLegalMovesIncludesWasteToFoundationAndTableau(t *testing.T) {
	g := &Game{status: Playing}
	g.waste = []Card{{Rank: Ace, Suit: Spades}}
	g.tableau[0] = []TableauCard{{Card: Card{Rank: Two, Suit: Hearts}, FaceUp: true}}

	moves := g.LegalMoves()
	if !hasMove(moves, MoveWasteToFoundation, Location{Kind: Waste}, Location{Kind: Foundation, Pile: int(Spades)}) {
		t.Fatalf("LegalMoves missing waste ace to foundation: %+v", moves)
	}

	g.waste = []Card{{Rank: Ace, Suit: Spades}, {Rank: Ace, Suit: Clubs}}
	moves = g.LegalMoves()
	if !hasMove(moves, MoveWasteToFoundation, Location{Kind: Waste}, Location{Kind: Foundation, Pile: int(Clubs)}) {
		t.Fatalf("LegalMoves should use top waste card only: %+v", moves)
	}
}

func TestLegalMovesIncludesTableauStackThatRevealsCard(t *testing.T) {
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{
		{Card: Card{Rank: Nine, Suit: Clubs}, FaceUp: false},
		{Card: Card{Rank: Eight, Suit: Hearts}, FaceUp: true},
	}
	g.tableau[1] = []TableauCard{{Card: Card{Rank: Nine, Suit: Clubs}, FaceUp: true}}

	moves := g.LegalMoves()
	wantSrc := Location{Kind: Tableau, Pile: 0, Index: 1}
	wantDst := Location{Kind: Tableau, Pile: 1}
	if !hasRevealingMove(moves, MoveTableauToTableau, wantSrc, wantDst) {
		t.Fatalf("LegalMoves missing revealing stack move: %+v", moves)
	}
}

func TestPlayableSourcesSkipNonPlayableLocations(t *testing.T) {
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{{Card: Card{Rank: Five, Suit: Spades}, FaceUp: false}}
	g.tableau[1] = []TableauCard{{Card: Card{Rank: King, Suit: Hearts}, FaceUp: true}}
	g.stock = []Card{{Rank: Ace, Suit: Spades}}

	sources := g.PlayableSources()
	if containsLocation(sources, Location{Kind: Tableau, Pile: 0, Index: 0}) {
		t.Fatalf("face-down tableau card should not be a playable source: %+v", sources)
	}
	if !containsLocation(sources, Location{Kind: Tableau, Pile: 1, Index: 0}) {
		t.Fatalf("face-up tableau king should be a playable source: %+v", sources)
	}
	if !containsLocation(sources, Location{Kind: Stock}) {
		t.Fatalf("stock with cards should be a playable source: %+v", sources)
	}
}

func TestLegalMovesFromIncludesFoundationToTableauAndCopiesCards(t *testing.T) {
	g := &Game{status: Playing}
	g.foundations[Spades] = []Card{{Rank: Ace, Suit: Spades}, {Rank: Two, Suit: Spades}, {Rank: Three, Suit: Spades}}
	g.tableau[0] = []TableauCard{{Card: Card{Rank: Four, Suit: Hearts}, FaceUp: true}}

	moves := g.LegalMovesFrom(Location{Kind: Foundation, Pile: int(Spades)})
	if !hasMove(moves, MoveFoundationToTableau, Location{Kind: Foundation, Pile: int(Spades)}, Location{Kind: Tableau, Pile: 0}) {
		t.Fatalf("LegalMovesFrom missing foundation to tableau move: %+v", moves)
	}
	moves[0].Cards[0] = Card{Rank: King, Suit: Clubs}
	top, ok := g.FoundationTop(Spades)
	if !ok || top != (Card{Rank: Three, Suit: Spades}) {
		t.Fatalf("move cards alias foundation storage; top = %+v/%v", top, ok)
	}
}

func TestApplyMoveDelegatesDrawAndCardMoves(t *testing.T) {
	g := &Game{status: Playing, stock: []Card{{Rank: Ace, Suit: Spades}}}
	if !g.ApplyMove(Move{Kind: MoveDraw}) {
		t.Fatal("ApplyMove should draw for MoveDraw")
	}
	if g.StockCount() != 0 || len(g.waste) != 1 {
		t.Fatalf("after draw stock=%d waste=%d, want stock=0 waste=1", g.StockCount(), len(g.waste))
	}

	g = &Game{status: Playing, waste: []Card{{Rank: Ace, Suit: Hearts}}}
	if !g.ApplyMove(Move{
		Kind:        MoveWasteToFoundation,
		Source:      Location{Kind: Waste},
		Destination: Location{Kind: Foundation, Pile: int(Hearts)},
	}) {
		t.Fatal("ApplyMove should delegate card moves to Move")
	}
	top, ok := g.FoundationTop(Hearts)
	if !ok || top != (Card{Rank: Ace, Suit: Hearts}) {
		t.Fatalf("foundation top = %+v/%v, want ace hearts", top, ok)
	}
}

func hasMoveKind(moves []Move, kind MoveKind) bool {
	for _, move := range moves {
		if move.Kind == kind {
			return true
		}
	}
	return false
}

func hasMove(moves []Move, kind MoveKind, src Location, dst Location) bool {
	for _, move := range moves {
		if move.Kind == kind && sameLocationForTest(move.Source, src) && sameLocationForTest(move.Destination, dst) {
			return true
		}
	}
	return false
}

func hasRevealingMove(moves []Move, kind MoveKind, src Location, dst Location) bool {
	for _, move := range moves {
		if move.Kind == kind && move.RevealsCard && sameLocationForTest(move.Source, src) && sameLocationForTest(move.Destination, dst) {
			return true
		}
	}
	return false
}

func containsLocation(locs []Location, want Location) bool {
	for _, loc := range locs {
		if sameLocationForTest(loc, want) {
			return true
		}
	}
	return false
}

func sameLocationForTest(a Location, b Location) bool {
	return a.Kind == b.Kind && a.Pile == b.Pile && a.Index == b.Index
}
