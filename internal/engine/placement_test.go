package engine

import (
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
				if n, _ := g.CellAt(nx, ny); n.Mine {
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

func TestRevealStartsGame(t *testing.T) {
	g, _ := New(9, 9, 10, rand.New(rand.NewSource(7)))
	g.Reveal(0, 0)
	if g.Status() != Playing && g.Status() != Won {
		t.Errorf("status after first reveal = %v, want Playing (or Won)", g.Status())
	}
}
