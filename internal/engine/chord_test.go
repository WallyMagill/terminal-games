package engine

import "testing"

func TestChordWithCorrectFlagsRevealsNeighbors(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1)    // the "2"
	g.CycleFlag(0, 0) // flag both real mines
	g.CycleFlag(2, 2)
	if !g.Chord(1, 1) {
		t.Fatal("chord with matching flags should change the board")
	}
	// All unflagged neighbors of (1,1) open. (2,0) and (0,2) are zero cells,
	// so the chord cascades through their zero regions too.
	wantRevealed := [][2]int{
		{0, 1}, {1, 0}, {2, 0}, {2, 1}, {0, 2}, {1, 2}, // direct neighbors
		{3, 0}, {3, 1}, {0, 3}, {1, 3}, // cascade via the zero cells
	}
	for _, p := range wantRevealed {
		if got := stateAt(t, g, p[0], p[1]); got != Revealed {
			t.Errorf("(%d,%d) state = %v, want Revealed", p[0], p[1], got)
		}
	}
	// The cascade must not cross the numbered border around the (2,2) mine.
	for _, p := range [][2]int{{2, 3}, {3, 2}, {3, 3}} {
		if got := stateAt(t, g, p[0], p[1]); got != Hidden {
			t.Errorf("(%d,%d) state = %v, want Hidden", p[0], p[1], got)
		}
	}
	if g.Status() == Lost {
		t.Error("correct chord must not lose")
	}
}

func TestChordRevealsQuestionedNeighbors(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1)
	g.CycleFlag(0, 0)
	g.CycleFlag(2, 2)
	g.CycleFlag(1, 0) // → Flagged... cycle on to Questioned:
	g.CycleFlag(1, 0)
	g.Chord(1, 1)
	if got := stateAt(t, g, 1, 0); got != Revealed {
		t.Errorf("questioned neighbor state = %v, want Revealed (\"?\" counts as hidden)", got)
	}
}

func TestChordWithWrongFlagLoses(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1)
	g.CycleFlag(1, 0) // wrong: safe cell flagged
	g.CycleFlag(0, 2) // wrong: safe cell flagged — count now matches the "2"
	if !g.Chord(1, 1) {
		t.Fatal("chord should proceed when flag count matches")
	}
	if g.Status() != Lost {
		t.Errorf("status = %v, want Lost (chord opened a real mine)", g.Status())
	}
	if _, _, ok := g.HitMine(); !ok {
		t.Error("HitMine should report the chord-opened mine")
	}
}

func TestChordNoOps(t *testing.T) {
	g := cornerBoard(t)
	if g.Chord(1, 1) {
		t.Error("chord on hidden cell should be a no-op")
	}
	g.Reveal(1, 1)
	if g.Chord(1, 1) {
		t.Error("chord with zero flags placed should be a no-op")
	}
	g.CycleFlag(0, 0)
	if g.Chord(1, 1) {
		t.Error("chord with insufficient flags should be a no-op")
	}
	g.Reveal(3, 0) // zero cell
	if g.Chord(3, 0) {
		t.Error("chord on a zero cell should be a no-op")
	}
	if g.Chord(7, 7) {
		t.Error("out-of-bounds chord should be a no-op")
	}
}
