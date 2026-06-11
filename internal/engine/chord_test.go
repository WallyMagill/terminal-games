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

// Audit finding: a losing chord stops at the FIRST mine hit; cells after it
// in neighbor order stay untouched and HitMine reports the first mine, not
// the last. Pins the mid-chord stop-on-loss.
func TestChordStopsAtFirstMineHit(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1)    // the "2"
	g.CycleFlag(1, 0) // two wrong flags
	g.CycleFlag(0, 2)
	g.Chord(1, 1) // first neighbor (0,0) is a mine → Lost immediately
	if g.Status() != Lost {
		t.Fatal("setup: expected Lost")
	}
	if x, y, ok := g.HitMine(); !ok || x != 0 || y != 0 {
		t.Errorf("HitMine = (%d,%d,%v), want (0,0,true) — first mine hit", x, y, ok)
	}
	// (2,0) follows the mine in neighbor order and would flood-cascade if
	// the chord kept going; (2,2) is the second mine.
	if got := stateAt(t, g, 2, 0); got != Hidden {
		t.Errorf("(2,0) state = %v, want Hidden (chord must stop at first mine)", got)
	}
	if got := stateAt(t, g, 2, 2); got != Hidden {
		t.Errorf("second mine (2,2) state = %v, want Hidden", got)
	}
}

// Pins the post-loss freeze for Chord. Dropping only the entry Lost guard
// is equivalent at HEAD — the mid-chord stop neutralizes every iteration —
// but this test pins the pair contract so the entry guard and the mid-chord
// stop cannot both regress silently.
func TestChordAfterLossOnChordableNumberIsNoOp(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1)    // the "2"
	g.CycleFlag(0, 0) // correct flag
	g.CycleFlag(2, 1) // wrong flag on safe cell — count matches the "2"
	g.Reveal(2, 2)    // hit the unflagged mine: Lost
	if g.Status() != Lost {
		t.Fatal("setup: expected Lost")
	}
	if g.Chord(1, 1) {
		t.Fatal("chord after loss must be a no-op")
	}
	if got := stateAt(t, g, 1, 0); got != Hidden {
		t.Errorf("(1,0) state = %v, want Hidden (board must not mutate after loss)", got)
	}
}

// Audit finding: the changed-result contract. A chord whose neighbors are
// all flagged or already revealed changes nothing and must return false.
func TestChordWithNothingLeftToRevealReturnsFalse(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1)
	g.CycleFlag(0, 0)
	g.CycleFlag(2, 2)
	if !g.Chord(1, 1) {
		t.Fatal("first chord should change the board")
	}
	if g.Chord(1, 1) {
		t.Error("second chord changes nothing and must return false")
	}
}

// Audit finding: chording fires only when the flag count EQUALS the number.
// Excess flags must be a no-op just like insufficient flags — a relaxed >=
// guard would open the unflagged mine when one of three flags is wrong.
func TestChordExcessFlagsIsNoOp(t *testing.T) {
	g := cornerBoard(t)
	g.Reveal(1, 1)    // the "2"
	g.CycleFlag(0, 0) // both real mines flagged
	g.CycleFlag(2, 2)
	g.CycleFlag(1, 0) // third flag — excess
	if g.Chord(1, 1) {
		t.Error("chord with excess flags (3 around a 2) must be a no-op")
	}
	for _, p := range [][2]int{{0, 1}, {2, 0}, {2, 1}, {0, 2}, {1, 2}} {
		if got := stateAt(t, g, p[0], p[1]); got != Hidden {
			t.Errorf("(%d,%d) state = %v, want Hidden after excess-flag chord", p[0], p[1], got)
		}
	}
	if g.Status() != Playing {
		t.Errorf("status = %v, want Playing", g.Status())
	}
}

// Audit finding: chording a cell that is not revealed must be a no-op even
// when its adjacent flags happen to match its number. TestChordNoOps chords
// a hidden cell only with zero flags placed, so the flag-count gate masked
// a dropped State != Revealed precondition.
func TestChordOnHiddenCellWithMatchingFlagsIsNoOp(t *testing.T) {
	g := cornerBoard(t)
	g.CycleFlag(0, 0) // both mines flagged; (1,1) is the hidden "2"
	g.CycleFlag(2, 2)
	if g.Chord(1, 1) {
		t.Fatal("chord on a hidden cell must be a no-op even with matching flags")
	}
	for _, p := range [][2]int{{1, 0}, {2, 0}, {0, 1}, {2, 1}, {0, 2}, {1, 2}} {
		if got := stateAt(t, g, p[0], p[1]); got != Hidden {
			t.Errorf("(%d,%d) state = %v, want Hidden", p[0], p[1], got)
		}
	}
}

// Audit finding: chording a revealed ZERO cell must be a no-op even when it
// still has a hidden neighbor (reachable by flagging inside a flood region,
// then cycling the flag away). Pins the Adjacent == 0 precondition.
func TestChordOnRevealedZeroCellIsNoOp(t *testing.T) {
	g := cornerBoard(t)
	g.CycleFlag(2, 0) // flag inside the top-right zero region
	g.Reveal(3, 0)    // flood stops at the flag
	g.CycleFlag(2, 0) // flag → question
	g.CycleFlag(2, 0) // question → hidden: (3,0) is a revealed zero with a hidden neighbor
	if g.Chord(3, 0) {
		t.Fatal("chord on a revealed zero cell must be a no-op")
	}
	if got := stateAt(t, g, 2, 0); got != Hidden {
		t.Errorf("(2,0) state = %v, want Hidden", got)
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
