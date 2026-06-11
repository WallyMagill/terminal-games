package engine

import "testing"

func TestWinByRevealingAllSafeCells(t *testing.T) {
	g := newWithMines(t, 3, 3, [][2]int{{0, 0}})
	g.Reveal(2, 2) // zero region floods everything except the mine → instant win
	if g.Status() != Won {
		t.Fatalf("status = %v, want Won", g.Status())
	}
}

func TestWinAutoFlagsRemainingMines(t *testing.T) {
	g := newWithMines(t, 3, 3, [][2]int{{0, 0}})
	g.Reveal(2, 2)
	c, _ := g.CellAt(0, 0)
	if c.State != Flagged {
		t.Errorf("mine state after win = %v, want Flagged", c.State)
	}
	if g.MinesRemaining() != 0 {
		t.Errorf("MinesRemaining after win = %d, want 0", g.MinesRemaining())
	}
}

func TestWinAutoFlagsQuestionedMine(t *testing.T) {
	g := newWithMines(t, 3, 3, [][2]int{{0, 0}})
	g.CycleFlag(0, 0) // → Flagged
	g.CycleFlag(0, 0) // → Questioned
	g.Reveal(2, 2)
	c, _ := g.CellAt(0, 0)
	if c.State != Flagged {
		t.Errorf("questioned mine after win = %v, want Flagged", c.State)
	}
	if g.MinesRemaining() != 0 {
		t.Errorf("MinesRemaining after win = %d, want 0", g.MinesRemaining())
	}
}

func TestWinDoesNotRequireFlags(t *testing.T) {
	g := cornerBoard(t)
	// Reveal every safe cell directly; place zero flags.
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if c, _ := g.CellAt(x, y); !c.Mine {
				g.Reveal(x, y)
			}
		}
	}
	if g.Status() != Won {
		t.Fatalf("status = %v, want Won (flags are not required)", g.Status())
	}
}

func TestActionsAfterWinAreNoOps(t *testing.T) {
	g := newWithMines(t, 3, 3, [][2]int{{0, 0}})
	g.Reveal(2, 2)
	if g.Reveal(0, 0) || g.CycleFlag(1, 1) || g.Chord(1, 1) {
		t.Error("all actions after win must be no-ops")
	}
}

func TestWinViaChord(t *testing.T) {
	// Review finding: every other win test wins via Reveal; a dropped
	// checkWin() inside Chord would regress silently without this.
	g := newWithMines(t, 3, 3, [][2]int{{0, 0}})
	g.Reveal(1, 1)    // the "1"
	g.CycleFlag(0, 0) // flag the mine
	g.Chord(1, 1)     // opens every remaining safe cell
	if g.Status() != Won {
		t.Fatalf("status = %v, want Won (win must be detectable via chord)", g.Status())
	}
	// The pre-flagged mine must not be double-counted by autoFlagMines.
	if g.MinesRemaining() != 0 {
		t.Errorf("MinesRemaining after win with pre-flagged mine = %d, want 0", g.MinesRemaining())
	}
}
