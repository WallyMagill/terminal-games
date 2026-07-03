package engine

import "testing"

// newWithMines builds a started game with mines at exact coordinates.
func newWithMines(t *testing.T, w, h int, mines [][2]int) *Game {
	t.Helper()
	g, err := New(w, h, len(mines), testRNG())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mines {
		g.at(m[0], m[1]).Mine = true
	}
	g.computeAdjacency()
	g.minesPlaced = true
	g.status = Playing
	return g
}

func stateAt(t *testing.T, g *Game, x, y int) CellState {
	t.Helper()
	c, ok := g.CellAt(x, y)
	if !ok {
		t.Fatalf("CellAt(%d,%d) out of bounds", x, y)
	}
	return c.State
}

// Board fixture used across reveal/chord tests (4x4, mines on the diagonal).
// Note the two zero regions — top-right {(2,0),(3,0)} and bottom-left
// {(0,2),(0,3)} — are NOT connected; the numbered border stops the flood.
//
//	M  1  .  .        x→
//	1  2  1  1     y
//	.  1  M  1     ↓
//	.  1  1  1
//
// M = mine, . = zero (adj=0).
func cornerBoard(t *testing.T) *Game {
	return newWithMines(t, 4, 4, [][2]int{{0, 0}, {2, 2}})
}

func TestRevealNumberedCellRevealsOnlyIt(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1) // the "2"
	if got := stateAt(t, g, 1, 1); got != Revealed {
		t.Errorf("(1,1) state = %v, want Revealed", got)
	}
	for _, p := range [][2]int{{0, 1}, {1, 0}, {2, 1}, {1, 2}} {
		if got := stateAt(t, g, p[0], p[1]); got != Hidden {
			t.Errorf("(%d,%d) state = %v, want Hidden", p[0], p[1], got)
		}
	}
}

func TestRevealZeroFloodsRegion(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(3, 0) // zero cell, top-right
	// The connected zero region {(2,0),(3,0)} opens with its numbered border;
	// the disconnected bottom-left zero region must stay hidden — floods do
	// not leak through numbered borders.
	wantRevealed := [][2]int{{2, 0}, {3, 0}, {1, 0}, {1, 1}, {2, 1}, {3, 1}}
	wantHidden := [][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}, {3, 2}, {3, 3}}
	for _, p := range wantRevealed {
		if got := stateAt(t, g, p[0], p[1]); got != Revealed {
			t.Errorf("(%d,%d) state = %v, want Revealed", p[0], p[1], got)
		}
	}
	for _, p := range wantHidden {
		if got := stateAt(t, g, p[0], p[1]); got != Hidden {
			t.Errorf("(%d,%d) state = %v, want Hidden (flood must not cross borders)", p[0], p[1], got)
		}
	}
}

func TestFloodSkipsFlaggedCells(t *testing.T) {
	g := cornerBoard(t)
	g.CycleFlag(2, 0) // flag a ZERO cell inside the flood region
	g.Reveal(3, 0)
	if got := stateAt(t, g, 2, 0); got != Flagged {
		t.Errorf("flagged cell state = %v, want Flagged (flood must not open flags)", got)
	}
	// The flag also blocks expansion through it: cells reachable only via
	// (2,0) stay hidden.
	for _, p := range [][2]int{{1, 0}, {1, 1}} {
		if got := stateAt(t, g, p[0], p[1]); got != Hidden {
			t.Errorf("(%d,%d) state = %v, want Hidden (flag blocks flood expansion)", p[0], p[1], got)
		}
	}
}

func TestRevealMineLoses(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(0, 0)
	if g.Status() != Lost {
		t.Fatalf("status = %v, want Lost", g.Status())
	}
	if x, y, ok := g.HitMine(); !ok || x != 0 || y != 0 {
		t.Errorf("HitMine = (%d,%d,%v), want (0,0,true)", x, y, ok)
	}
}

// Audit finding: the changed-result contract — revealing a mine mutates the
// board to Lost and must return true like any other board change.
func TestRevealMineReturnsTrue(t *testing.T) {
	g := cornerBoard(t)
	if !g.Reveal(0, 0) {
		t.Error("revealing a mine changes the board and must return true")
	}
}

// Audit finding: the committed loss tests use the symmetric coordinate
// (0,0) or check only the ok flag, so transposed hit coordinates in Reveal,
// Chord, and the HitMine accessor all survived. An asymmetric mine position
// pins all three sites.
func TestHitMineCoordinatesNotTransposed(t *testing.T) {
	g := newWithMines(t, 4, 4, [][2]int{{2, 1}})
	g.Reveal(2, 1) // reveal the mine directly
	if x, y, ok := g.HitMine(); !ok || x != 2 || y != 1 {
		t.Errorf("Reveal: HitMine = (%d,%d,%v), want (2,1,true)", x, y, ok)
	}

	g2 := newWithMines(t, 4, 4, [][2]int{{2, 1}})
	g2.Reveal(1, 0)    // a "1"
	g2.CycleFlag(0, 0) // wrong flag, count matches
	g2.Chord(1, 0)     // opens (2,1): Lost
	if g2.Status() != Lost {
		t.Fatal("setup: expected Lost via chord")
	}
	if x, y, ok := g2.HitMine(); !ok || x != 2 || y != 1 {
		t.Errorf("Chord: HitMine = (%d,%d,%v), want (2,1,true)", x, y, ok)
	}
}

// Audit finding (round 3): every committed test queries HitMine only after a
// loss, so a mutant hardcoding the ok result to true (board.go HitMine:
// `return g.hitX, g.hitY, g.mineHit` -> `return g.hitX, g.hitY, true`)
// survives the entire repo suite. HitMine's contract is that ok is false
// until a mine actually ends the game — including before the first reveal,
// during play, and after a WIN.
func TestHitMineNotOKWithoutALoss(t *testing.T) {
	// Fresh, unstarted game.
	g, err := New(4, 4, 2, testRNG())
	if err != nil {
		t.Fatal(err)
	}
	if x, y, ok := g.HitMine(); ok {
		t.Errorf("HitMine = (%d,%d,%v) before any reveal, want ok=false", x, y, ok)
	}

	// Mid-game: a safe reveal on a known board.
	g = newWithMines(t, 4, 4, [][2]int{{0, 0}, {2, 2}})
	g.Reveal(3, 0) // zero region top-right, safe
	if g.Status() != Playing {
		t.Fatalf("status = %v, want Playing", g.Status())
	}
	if x, y, ok := g.HitMine(); ok {
		t.Errorf("HitMine = (%d,%d,%v) while playing, want ok=false", x, y, ok)
	}

	// Won game: reveal every safe cell.
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if (x == 0 && y == 0) || (x == 2 && y == 2) {
				continue
			}
			g.Reveal(x, y)
		}
	}
	if g.Status() != Won {
		t.Fatalf("status = %v, want Won", g.Status())
	}
	if x, y, ok := g.HitMine(); ok {
		t.Errorf("HitMine = (%d,%d,%v) after a win, want ok=false", x, y, ok)
	}
}

func TestRevealEdgeCases(t *testing.T) {
	g := cornerBoard(t)
	g.CycleFlag(1, 1)
	if g.Reveal(1, 1) {
		t.Error("reveal of flagged cell should be a no-op")
	}
	g.CycleFlag(2, 1) // → Flagged
	g.CycleFlag(2, 1) // → Questioned
	if !g.Reveal(2, 1) {
		t.Error("questioned cells must be revealable (only flags block)")
	}
	if !g.Reveal(3, 3) {
		t.Error("expected reveal to change board")
	}
	if g.Reveal(3, 3) {
		t.Error("re-reveal of revealed cell should be a no-op")
	}
	if g.Reveal(-1, 5) {
		t.Error("out-of-bounds reveal should be a no-op")
	}
}

func TestActionsAfterLossAreNoOps(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(0, 0) // lose
	if g.Reveal(3, 3) || g.CycleFlag(3, 3) || g.Chord(1, 1) {
		t.Error("all actions after loss must be no-ops")
	}
}
