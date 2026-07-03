package engine

import (
	"math/rand"
	"testing"
)

// FuzzGameOps drives a random sequence of Reveal/CycleFlag/Chord calls —
// at in-range and out-of-range coordinates — against small boards of
// varying size and mine count, checking engine invariants after every
// single operation:
//
//   - the engine never panics
//   - MinesRemaining == Mines() - (number of Flagged cells)
//   - status never changes once Won or Lost, and the board does not
//     mutate after a terminal status
//   - the revealed-cell count never decreases
//   - a revealed cell is never a mine unless the game is Lost
//   - every cell state is from the legal set
//   - once started, exactly Mines() mines are on the board
//   - on Won: every safe cell is Revealed and every mine is Flagged
//
// Input encoding: data[0..2] pick width, height, mine count; each following
// 3-byte chunk is one op: (op selector, x selector, y selector). Coordinate
// selectors map into [-2, dim+2) so out-of-bounds calls are exercised on
// every run.
func FuzzGameOps(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0})                                  // 2x2/1, reveal origin
	f.Add([]byte{7, 7, 9, 0, 4, 4, 1, 3, 3, 2, 4, 4})                // 9x9/10: reveal, flag, chord
	f.Add([]byte{2, 2, 13, 1, 2, 2, 1, 2, 2, 0, 2, 2, 2, 2, 2})      // dense 4x4/14
	f.Add([]byte{3, 0, 4, 0, 0, 0, 0, 200, 200, 1, 255, 0, 2, 0, 9}) // OOB coords
	f.Add([]byte{0, 0, 1, 0, 0, 0, 0, 3, 0, 0, 0, 3, 0, 3, 3})       // 2x2/2: forced loss path

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 3 {
			t.Skip()
		}
		w := 2 + int(data[0])%7 // 2..8
		h := 2 + int(data[1])%7 // 2..8
		mines := 1 + int(data[2])%(w*h-2)
		seed := int64(len(data))*1_000_003 + int64(data[0])<<16 + int64(data[1])<<8 + int64(data[2])
		g, err := New(w, h, mines, rand.New(rand.NewSource(seed)))
		if err != nil {
			t.Fatalf("New(%d,%d,%d) unexpected error: %v", w, h, mines, err)
		}

		snapshot := func() []Cell {
			s := make([]Cell, 0, w*h)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					c, ok := g.CellAt(x, y)
					if !ok {
						t.Fatalf("CellAt(%d,%d) reported out of bounds inside the board", x, y)
					}
					s = append(s, c)
				}
			}
			return s
		}

		prevRevealed := 0
		for i := 3; i+2 < len(data); i += 3 {
			op := data[i] % 3
			x := int(data[i+1])%(w+4) - 2 // [-2, w+2)
			y := int(data[i+2])%(h+4) - 2 // [-2, h+2)

			statusBefore := g.Status()
			var boardBefore []Cell
			if statusBefore == Won || statusBefore == Lost {
				boardBefore = snapshot()
			}

			var changed bool
			switch op {
			case 0:
				changed = g.Reveal(x, y)
			case 1:
				changed = g.CycleFlag(x, y)
			case 2:
				changed = g.Chord(x, y)
			}

			// Terminal status is sticky and the board frozen.
			if statusBefore == Won || statusBefore == Lost {
				if g.Status() != statusBefore {
					t.Fatalf("op %d at (%d,%d): status changed %v -> %v after game over", op, x, y, statusBefore, g.Status())
				}
				if changed {
					t.Fatalf("op %d at (%d,%d): reported change after game over", op, x, y)
				}
				for j, c := range snapshot() {
					if c != boardBefore[j] {
						t.Fatalf("op %d at (%d,%d): cell %d mutated after game over: %+v -> %+v", op, x, y, j, boardBefore[j], c)
					}
				}
			}

			// Per-cell invariants and counters.
			flags, revealed, mineCount := 0, 0, 0
			for yy := 0; yy < h; yy++ {
				for xx := 0; xx < w; xx++ {
					c, _ := g.CellAt(xx, yy)
					switch c.State {
					case Hidden, Flagged, Questioned, Revealed:
					default:
						t.Fatalf("cell (%d,%d) has illegal state %d", xx, yy, c.State)
					}
					if c.State == Flagged {
						flags++
					}
					if c.State == Revealed {
						revealed++
						if c.Mine && g.Status() != Lost {
							t.Fatalf("cell (%d,%d): revealed mine while status %v", xx, yy, g.Status())
						}
					}
					if c.Mine {
						mineCount++
					}
					if c.Adjacent < 0 || c.Adjacent > 8 {
						t.Fatalf("cell (%d,%d) adjacent = %d out of range", xx, yy, c.Adjacent)
					}
				}
			}
			if got := g.MinesRemaining(); got != mines-flags {
				t.Fatalf("MinesRemaining = %d, want %d (mines %d - flags %d)", got, mines-flags, mines, flags)
			}
			if g.Status() != NotStarted && mineCount != mines {
				t.Fatalf("board has %d mines, want %d (status %v)", mineCount, mines, g.Status())
			}
			if revealed < prevRevealed {
				t.Fatalf("revealed count decreased %d -> %d", prevRevealed, revealed)
			}
			prevRevealed = revealed
			if g.Status() == Won {
				if revealed != w*h-mines {
					t.Fatalf("Won with %d revealed, want %d", revealed, w*h-mines)
				}
				if flags != mines {
					t.Fatalf("Won with %d flags, want all %d mines flagged", flags, mines)
				}
			}
		}
	})
}
