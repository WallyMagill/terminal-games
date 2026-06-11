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

// Board fixture used across reveal/chord tests (4x4, mines at corners):
//
//   - 1  .  .        x→
//     1  1  .  .     y
//     .  .  1  1     ↓
//     .  .  1  *
func cornerBoard(t *testing.T) *Game {
	return newWithMines(t, 4, 4, [][2]int{{0, 0}, {3, 3}})
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
	// The connected zero region plus its numbered border opens; on this board
	// that is everything except the two mines.
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			c, _ := g.CellAt(x, y)
			if c.Mine {
				if c.State == Revealed {
					t.Errorf("mine (%d,%d) revealed by flood", x, y)
				}
				continue
			}
			if c.State != Revealed {
				t.Errorf("safe cell (%d,%d) not revealed by flood", x, y)
			}
		}
	}
}

func TestFloodSkipsFlaggedCells(t *testing.T) {
	g := cornerBoard(t)
	g.CycleFlag(3, 3) // flag a safe cell in the flood region
	g.Reveal(3, 0)
	if got := stateAt(t, g, 3, 3); got != Flagged {
		t.Errorf("flagged cell state = %v, want Flagged (flood must not open flags)", got)
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
