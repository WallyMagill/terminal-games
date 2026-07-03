package engine

import "testing"

func TestDrawMovesThreeCardsToWaste(t *testing.T) {
	g, err := New(testRNG())
	if err != nil {
		t.Fatal(err)
	}
	if !g.Draw() {
		t.Fatal("Draw should change state with a full stock")
	}
	if g.StockCount() != 21 {
		t.Fatalf("stock count = %d, want 21", g.StockCount())
	}
	if got := len(g.WasteVisible()); got != 3 {
		t.Fatalf("visible waste = %d, want 3", got)
	}
	if g.Moves() != 1 {
		t.Fatalf("moves = %d, want 1", g.Moves())
	}
}

func TestDrawRemainderAndRecycle(t *testing.T) {
	g, err := New(testRNG())
	if err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		if !g.Draw() {
			t.Fatalf("draw %d changed=false, want true", i+1)
		}
	}
	if g.StockCount() != 0 {
		t.Fatalf("stock count after 8 draws = %d, want 0", g.StockCount())
	}
	beforeRecycle := append([]Card(nil), g.WasteVisible()...)
	if !g.Draw() {
		t.Fatal("draw on empty stock with waste should recycle")
	}
	if g.StockCount() != 24 {
		t.Fatalf("stock after recycle = %d, want 24", g.StockCount())
	}
	if got := len(g.WasteVisible()); got != 0 {
		t.Fatalf("waste after recycle = %d, want 0", got)
	}
	if len(beforeRecycle) == 0 {
		t.Fatal("waste before recycle unexpectedly empty")
	}
	if g.Moves() != 9 {
		t.Fatalf("moves after recycle = %d, want 9", g.Moves())
	}
}

func TestRecyclePreservesDrawThreeOrderAcrossPasses(t *testing.T) {
	cards := []Card{
		{Rank: Two, Suit: Spades},
		{Rank: Three, Suit: Spades},
		{Rank: Four, Suit: Spades},
		{Rank: Five, Suit: Spades},
		{Rank: Six, Suit: Spades},
		{Rank: Seven, Suit: Spades},
	}
	g := &Game{stock: append([]Card(nil), cards...), status: Playing}

	g.Draw()
	firstPassFirstDraw := g.WasteVisible()
	g.Draw()
	firstPassSecondDraw := g.WasteVisible()
	g.Draw()
	g.Draw()
	secondPassFirstDraw := g.WasteVisible()

	if !sameCards(firstPassFirstDraw, secondPassFirstDraw) {
		t.Fatalf("first draw after recycle = %v, want first-pass first draw %v", secondPassFirstDraw, firstPassFirstDraw)
	}
	if sameCards(firstPassSecondDraw, secondPassFirstDraw) {
		t.Fatalf("first draw after recycle repeated second draw %v; want stock order preserved", secondPassFirstDraw)
	}
}

func sameCards(a, b []Card) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDrawReturnsFalseWhenStockAndWasteAreEmpty(t *testing.T) {
	g := &Game{status: Playing}
	if g.Draw() {
		t.Fatal("Draw returned true with empty stock and waste")
	}
	if g.Moves() != 0 {
		t.Fatalf("moves = %d, want 0", g.Moves())
	}
}

func TestDrawReturnsFalseWhenWon(t *testing.T) {
	g, err := New(testRNG())
	if err != nil {
		t.Fatal(err)
	}
	g.status = Won
	if g.Draw() {
		t.Fatal("Draw returned true after game was won")
	}
	if g.StockCount() != 24 {
		t.Fatalf("stock count = %d, want 24", g.StockCount())
	}
}

func TestWasteVisibleDoesNotExposeMutableState(t *testing.T) {
	g, err := New(testRNG())
	if err != nil {
		t.Fatal(err)
	}
	g.Draw()
	waste := g.WasteVisible()
	waste[0] = Card{Rank: King, Suit: Clubs}
	if g.WasteVisible()[0] == (Card{Rank: King, Suit: Clubs}) {
		t.Fatal("WasteVisible returned mutable backing storage")
	}
}
