package engine

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestFirstRevealNeverMine(t *testing.T) {
	// Densest legal board: any non-excluded cell is almost certainly a mine,
	// so 50 seeds give overwhelming coverage of the exclusion logic.
	for seed := int64(0); seed < 50; seed++ {
		g, err := New(4, 4, 14, rand.New(rand.NewSource(seed)))
		if err != nil {
			t.Fatal(err)
		}
		g.Reveal(2, 1)
		c, _ := g.CellAt(2, 1)
		if c.Mine {
			t.Fatalf("seed %d: first revealed cell is a mine", seed)
		}
		if g.Status() == Lost {
			t.Fatalf("seed %d: lost on first reveal", seed)
		}
	}
}

func TestPlacementCountAndAdjacency(t *testing.T) {
	g, _ := New(9, 9, 10, rand.New(rand.NewSource(42)))
	g.Reveal(4, 4)

	mineCount := 0
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			c, _ := g.CellAt(x, y)
			if c.Mine {
				mineCount++
			}
			want := 0
			g.eachNeighbor(x, y, func(nx, ny int) {
				if g.at(nx, ny).Mine { // white-box: eachNeighbor guarantees bounds
					want++
				}
			})
			if c.Adjacent != want {
				t.Errorf("cell (%d,%d) adjacent = %d, want %d", x, y, c.Adjacent, want)
			}
		}
	}
	if mineCount != 10 {
		t.Errorf("mine count = %d, want 10", mineCount)
	}
}

// Audit finding: without this, deleting the rng.Shuffle call in placeMines
// survived the suite — placement would silently become deterministic (the
// first N board indices), identical for every seed.
func TestPlacementVariesAcrossSeeds(t *testing.T) {
	layouts := map[string]bool{}
	for seed := int64(0); seed < 20; seed++ {
		g, err := New(9, 9, 10, rand.New(rand.NewSource(seed)))
		if err != nil {
			t.Fatal(err)
		}
		g.Reveal(4, 4)
		sig := ""
		for y := 0; y < 9; y++ {
			for x := 0; x < 9; x++ {
				c, _ := g.CellAt(x, y)
				if c.Mine {
					sig += fmt.Sprintf("(%d,%d)", x, y)
				}
			}
		}
		layouts[sig] = true
	}
	if len(layouts) < 2 {
		t.Fatalf("20 seeds produced %d distinct mine layouts; placement ignores the RNG", len(layouts))
	}
}

// Audit finding: lazy placement must run exactly once. Without the
// minesPlaced latch every subsequent Reveal re-places mines, growing the
// mine count and silently rewriting revealed numbers; no other engine test
// performs a second Reveal on a game built via New.
func TestSecondRevealDoesNotReplaceMines(t *testing.T) {
	g, err := New(9, 9, 10, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	countMines := func() int {
		n := 0
		for y := 0; y < 9; y++ {
			for x := 0; x < 9; x++ {
				if c, _ := g.CellAt(x, y); c.Mine {
					n++
				}
			}
		}
		return n
	}
	g.Reveal(0, 0)
	if got := countMines(); got != 10 {
		t.Fatalf("after first reveal: %d mines on board, want 10", got)
	}
	// Reveal a second hidden safe cell; placement must NOT run again.
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			if c, _ := g.CellAt(x, y); c.State == Hidden && !c.Mine {
				g.Reveal(x, y)
				if got := countMines(); got != 10 {
					t.Fatalf("after second reveal at (%d,%d): %d mines on board, want 10 (mines were re-placed)", x, y, got)
				}
				return
			}
		}
	}
	t.Fatal("no hidden safe cell found for the second reveal")
}

// Audit finding: a mine's Adjacent must count neighbors only, never itself.
// TestPlacementCountAndAdjacency computes its expectation with the same
// eachNeighbor under test, so a self-including iterator inflated both sides
// identically and survived.
func TestMineAdjacentDoesNotCountItself(t *testing.T) {
	g := newWithMines(t, 3, 3, [][2]int{{0, 0}})
	c, _ := g.CellAt(0, 0)
	if c.Adjacent != 0 {
		t.Errorf("lone mine Adjacent = %d, want 0 (must not count itself)", c.Adjacent)
	}
}

func TestRevealStartsGame(t *testing.T) {
	g, _ := New(9, 9, 10, rand.New(rand.NewSource(7)))
	if g.Status() != NotStarted {
		t.Fatal("expected NotStarted before first reveal")
	}
	g.Reveal(0, 0)
	if g.Status() != Playing && g.Status() != Won {
		t.Errorf("status after first reveal = %v, want Playing (or Won)", g.Status())
	}
}
