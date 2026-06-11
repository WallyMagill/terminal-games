package engine

import "testing"

func TestFlagCycle(t *testing.T) {
	g := cornerBoard(t)
	steps := []CellState{Flagged, Questioned, Hidden, Flagged}
	for i, want := range steps {
		if !g.CycleFlag(1, 1) {
			t.Fatalf("step %d: CycleFlag returned false", i)
		}
		if got := stateAt(t, g, 1, 1); got != want {
			t.Fatalf("step %d: state = %v, want %v", i, got, want)
		}
	}
}

func TestMinesRemainingTracksFlags(t *testing.T) {
	g := cornerBoard(t) // 2 mines
	if g.MinesRemaining() != 2 {
		t.Fatalf("initial MinesRemaining = %d, want 2", g.MinesRemaining())
	}
	g.CycleFlag(0, 0)
	g.CycleFlag(1, 0)
	g.CycleFlag(2, 0) // over-flagged: counter goes negative
	if g.MinesRemaining() != -1 {
		t.Errorf("MinesRemaining = %d, want -1", g.MinesRemaining())
	}
	g.CycleFlag(2, 0) // flag → question: count returns
	if g.MinesRemaining() != 0 {
		t.Errorf("MinesRemaining = %d, want 0", g.MinesRemaining())
	}
}

func TestFlagOnRevealedCellIsNoOp(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1)
	if g.CycleFlag(1, 1) {
		t.Error("CycleFlag on revealed cell should be a no-op")
	}
	if got := stateAt(t, g, 1, 1); got != Revealed {
		t.Errorf("state = %v, want Revealed", got)
	}
}

func TestFlagOutOfBoundsIsNoOp(t *testing.T) {
	g := cornerBoard(t)
	if g.CycleFlag(9, 9) {
		t.Error("out-of-bounds CycleFlag should be a no-op")
	}
}
