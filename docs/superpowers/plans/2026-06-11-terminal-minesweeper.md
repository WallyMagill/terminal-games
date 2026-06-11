# Terminal Minesweeper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `mines`, a polished keyboard-only classic minesweeper TUI in Go, per the approved spec at `docs/superpowers/specs/2026-06-11-terminal-minesweeper-design.md`.

**Architecture:** Pure game engine (`internal/engine`, zero TUI imports, deterministic via injected RNG) + Bubble Tea presentation layer (`internal/ui`) + JSON best-times persistence (`internal/store`). The engine is a mutate-behind-methods state machine; the UI reads cell snapshots and owns the timer.

**Tech Stack:** Go 1.26, Bubble Tea (TUI framework), Lip Gloss (styling), stdlib `testing` with table-driven tests.

---

## File structure

| File | Responsibility |
|---|---|
| `go.mod` | Module `github.com/wallymagill/mines` |
| `main.go` | Entry point: build store, run Bubble Tea program |
| `internal/engine/board.go` | `Cell`/`CellState`/`Status` types, `Game` struct, constructor + validation, mine placement, adjacency |
| `internal/engine/actions.go` | `Reveal` (+ flood fill), `CycleFlag`, `Chord` |
| `internal/engine/game.go` | Win detection, auto-flag on win, status accessors |
| `internal/engine/*_test.go` | White-box table-driven tests (same package) |
| `internal/store/store.go` | `Scores` map, `Store` with Load/Save/RecordIfBest, `DefaultPath` |
| `internal/store/store_test.go` | Round-trip, corruption, atomic-write tests |
| `internal/ui/presets.go` | Beginner/Intermediate/Expert preset definitions |
| `internal/ui/styles.go` | Lip Gloss styles, cell glyphs, number palette |
| `internal/ui/game.go` | Game screen model: cursor, key routing, timer, HUD, board render |
| `internal/ui/menu.go` | Menu model: preset list, custom form, best-times display |
| `internal/ui/app.go` | Root model: menu↔game routing, window size, too-small guard |
| `internal/ui/*_test.go` | Update-function tests + View golden test |

Engine API surface (locked here, used by all later tasks):

```go
func New(width, height, mines int, rng *rand.Rand) (*Game, error)
func (g *Game) Width() int / Height() int / Mines() int / Status() Status
func (g *Game) MinesRemaining() int            // mines - flags, may go negative
func (g *Game) CellAt(x, y int) (Cell, bool)   // snapshot copy, false if out of bounds
func (g *Game) HitMine() (x, y int, ok bool)   // the mine that ended the game
func (g *Game) Reveal(x, y int) bool           // true if board changed
func (g *Game) CycleFlag(x, y int) bool
func (g *Game) Chord(x, y int) bool
```

Conventions for every task below:
- Run tests from the repo root: `/Users/wallymagill/personal-projects/terminal-game`.
- Engine tests are **white-box** (package `engine`) so they can construct exact boards.
- Commit after every green test run. Conventional commit format, no attribution footer (disabled globally).

---

### Task 1: Project scaffolding

**Files:**
- Create: `go.mod` (via command)
- Create: `.gitignore`
- Create: `main.go` (temporary stub, replaced in Task 14)

- [ ] **Step 1: Init module and fetch dependencies**

```bash
cd /Users/wallymagill/personal-projects/terminal-game
go mod init github.com/wallymagill/mines
go get github.com/charmbracelet/bubbletea@v1
go get github.com/charmbracelet/lipgloss@v1
```

Expected: `go.mod` and `go.sum` created; bubbletea v1.x and lipgloss v1.x listed.

- [ ] **Step 2: Create `.gitignore`**

```gitignore
mines
*.test
coverage.out
```

- [ ] **Step 3: Create stub `main.go`**

```go
package main

import "fmt"

func main() {
	fmt.Println("mines: TUI wiring lands in Task 14")
}
```

- [ ] **Step 4: Verify it builds**

Run: `go build ./... && go run .`
Expected: prints `mines: TUI wiring lands in Task 14`

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum .gitignore main.go
git commit -m "chore: scaffold Go module with bubbletea and lipgloss"
```

---

### Task 2: Engine types and constructor validation

**Files:**
- Create: `internal/engine/board.go`
- Test: `internal/engine/board_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/engine/board_test.go`:

```go
package engine

import (
	"errors"
	"math/rand"
	"testing"
)

func testRNG() *rand.Rand { return rand.New(rand.NewSource(1)) }

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name    string
		w, h    int
		mines   int
		wantErr error
	}{
		{"valid beginner", 9, 9, 10, nil},
		{"valid minimum board", 2, 2, 1, nil},
		{"valid max mines", 4, 4, 14, nil},
		{"width too small", 1, 5, 1, ErrBoardTooSmall},
		{"height too small", 5, 1, 1, ErrBoardTooSmall},
		{"zero mines", 4, 4, 0, ErrMineCount},
		{"negative mines", 4, 4, -1, ErrMineCount},
		{"too many mines", 4, 4, 15, ErrMineCount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := New(tt.w, tt.h, tt.mines, testRNG())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("New(%d,%d,%d) error = %v, want %v", tt.w, tt.h, tt.mines, err, tt.wantErr)
			}
			if tt.wantErr == nil && g == nil {
				t.Fatal("expected non-nil game")
			}
		})
	}
}

func TestNewInitialState(t *testing.T) {
	g, err := New(9, 9, 10, testRNG())
	if err != nil {
		t.Fatal(err)
	}
	if g.Status() != NotStarted {
		t.Errorf("status = %v, want NotStarted", g.Status())
	}
	if g.Width() != 9 || g.Height() != 9 || g.Mines() != 10 {
		t.Errorf("dims = %dx%d/%d, want 9x9/10", g.Width(), g.Height(), g.Mines())
	}
	if g.MinesRemaining() != 10 {
		t.Errorf("MinesRemaining = %d, want 10", g.MinesRemaining())
	}
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			c, ok := g.CellAt(x, y)
			if !ok {
				t.Fatalf("CellAt(%d,%d) out of bounds", x, y)
			}
			if c.State != Hidden || c.Mine {
				t.Fatalf("cell (%d,%d) = %+v, want hidden non-mine", x, y, c)
			}
		}
	}
}

func TestCellAtOutOfBounds(t *testing.T) {
	g, _ := New(4, 4, 2, testRNG())
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {4, 0}, {0, 4}} {
		if _, ok := g.CellAt(p[0], p[1]); ok {
			t.Errorf("CellAt(%d,%d) ok = true, want false", p[0], p[1])
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -v`
Expected: compile error — `undefined: New`, `undefined: ErrBoardTooSmall`, etc.

- [ ] **Step 3: Write the implementation**

`internal/engine/board.go`:

```go
// Package engine implements classic minesweeper rules with no UI dependencies.
package engine

import (
	"errors"
	"math/rand"
)

type CellState uint8

const (
	Hidden CellState = iota
	Flagged
	Questioned
	Revealed
)

type Status uint8

const (
	NotStarted Status = iota
	Playing
	Won
	Lost
)

type Cell struct {
	Mine     bool
	State    CellState
	Adjacent int
}

var (
	ErrBoardTooSmall = errors.New("board must be at least 2x2")
	ErrMineCount     = errors.New("mines must be between 1 and width*height-2")
)

// Game is a single minesweeper round. Mines are placed lazily on the first
// Reveal so the first click is never a mine.
type Game struct {
	width, height, mines int
	cells                []Cell
	status               Status
	minesPlaced          bool
	rng                  *rand.Rand
	flags                int
	revealed             int
	hitX, hitY           int
	mineHit              bool
}

func New(width, height, mines int, rng *rand.Rand) (*Game, error) {
	if width < 2 || height < 2 {
		return nil, ErrBoardTooSmall
	}
	if mines < 1 || mines > width*height-2 {
		return nil, ErrMineCount
	}
	return &Game{
		width:  width,
		height: height,
		mines:  mines,
		cells:  make([]Cell, width*height),
		rng:    rng,
	}, nil
}

func (g *Game) Width() int     { return g.width }
func (g *Game) Height() int    { return g.height }
func (g *Game) Mines() int     { return g.mines }
func (g *Game) Status() Status { return g.status }

// MinesRemaining is the HUD counter: total mines minus placed flags.
// It can go negative when the player over-flags.
func (g *Game) MinesRemaining() int { return g.mines - g.flags }

// CellAt returns a snapshot copy of the cell, or ok=false out of bounds.
func (g *Game) CellAt(x, y int) (Cell, bool) {
	if !g.inBounds(x, y) {
		return Cell{}, false
	}
	return g.cells[y*g.width+x], true
}

// HitMine reports which mine ended the game after a loss.
func (g *Game) HitMine() (x, y int, ok bool) {
	return g.hitX, g.hitY, g.mineHit
}

func (g *Game) inBounds(x, y int) bool {
	return x >= 0 && x < g.width && y >= 0 && y < g.height
}

func (g *Game) at(x, y int) *Cell { return &g.cells[y*g.width+x] }

func (g *Game) eachNeighbor(x, y int, fn func(nx, ny int)) {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if nx, ny := x+dx, y+dy; g.inBounds(nx, ny) {
				fn(nx, ny)
			}
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -v`
Expected: `TestNewValidation`, `TestNewInitialState`, `TestCellAtOutOfBounds` all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat: engine types, constructor validation, cell snapshots"
```

> **Execution amendment (review-driven):** `New` additionally rejects a nil
> rng with `ErrNilRNG` so the engine cannot panic on first reveal, and the
> exported identifiers carry doc comments.

---

### Task 3: Lazy mine placement and first-click safety

**Files:**
- Modify: `internal/engine/board.go` (add `placeMines`, `computeAdjacency`)
- Create: `internal/engine/actions.go` (minimal `Reveal` that only places mines — fleshed out in Task 4)
- Test: `internal/engine/placement_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/engine/placement_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -v`
Expected: compile error — `g.Reveal undefined`.

- [ ] **Step 3: Write the implementation**

Append to `internal/engine/board.go`:

```go
// placeMines fills the board on first reveal, excluding the clicked cell so
// the first click is never a mine (classic rule — it may still be a number).
func (g *Game) placeMines(safeX, safeY int) {
	safe := safeY*g.width + safeX
	candidates := make([]int, 0, len(g.cells)-1)
	for i := range g.cells {
		if i != safe {
			candidates = append(candidates, i)
		}
	}
	g.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	for _, idx := range candidates[:g.mines] {
		g.cells[idx].Mine = true
	}
	g.computeAdjacency()
	g.minesPlaced = true
}

func (g *Game) computeAdjacency() {
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			n := 0
			g.eachNeighbor(x, y, func(nx, ny int) {
				if g.at(nx, ny).Mine {
					n++
				}
			})
			g.at(x, y).Adjacent = n
		}
	}
}
```

Create `internal/engine/actions.go` (placement trigger only; flood fill lands in Task 4):

```go
package engine

// Reveal uncovers the cell at (x, y). Returns true if the board changed.
func (g *Game) Reveal(x, y int) bool {
	if g.status == Won || g.status == Lost || !g.inBounds(x, y) {
		return false
	}
	c := g.at(x, y)
	if c.State == Revealed || c.State == Flagged {
		return false
	}
	if !g.minesPlaced {
		g.placeMines(x, y)
		g.status = Playing
	}
	c.State = Revealed
	g.revealed++
	return true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -v`
Expected: all tests PASS (Task 2 tests still green).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat: lazy mine placement with first-click safety"
```

---

### Task 4: Flood fill, loss on mine, reveal edge cases

**Files:**
- Modify: `internal/engine/actions.go` (replace `Reveal` body, add `floodReveal`)
- Create: `internal/engine/game.go` (`checkWin` stub — full win logic in Task 7)
- Test: `internal/engine/reveal_test.go`

- [ ] **Step 1: Write the test helper and failing tests**

`internal/engine/reveal_test.go`. The helper builds an exact board so behavior tests are fully deterministic:

```go
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
//	*  1  .  .        x→
//	1  2  1  1     y
//	.  1  *  1     ↓
//	.  1  1  1
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
```

Note: `CycleFlag` and `Chord` don't exist yet — add minimal stubs in this task so the package compiles; their real behavior is Tasks 5–6.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -v`
Expected: compile errors for `CycleFlag`/`Chord`/`computeAdjacency` absence resolved only after Step 3; then `TestRevealZeroFloodsRegion` FAILS (no flood fill yet).

- [ ] **Step 3: Write the implementation**

Replace `Reveal` in `internal/engine/actions.go` and add flood fill plus temporary stubs:

```go
package engine

// Reveal uncovers the cell at (x, y). Zero cells flood-fill their region.
// Returns true if the board changed.
func (g *Game) Reveal(x, y int) bool {
	if g.status == Won || g.status == Lost || !g.inBounds(x, y) {
		return false
	}
	c := g.at(x, y)
	if c.State == Revealed || c.State == Flagged {
		return false
	}
	if !g.minesPlaced {
		g.placeMines(x, y)
		g.status = Playing
	}
	if c.Mine {
		c.State = Revealed
		g.hitX, g.hitY, g.mineHit = x, y, true
		g.status = Lost
		return true
	}
	g.floodReveal(x, y)
	g.checkWin()
	return true
}

// floodReveal opens (x,y); if it has zero adjacent mines it BFS-expands
// through the connected zero region and its numbered border. Flags block.
func (g *Game) floodReveal(x, y int) {
	type pt struct{ x, y int }
	queue := []pt{{x, y}}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		c := g.at(p.x, p.y)
		if c.State == Revealed || c.State == Flagged || c.Mine {
			continue
		}
		c.State = Revealed
		g.revealed++
		if c.Adjacent == 0 {
			g.eachNeighbor(p.x, p.y, func(nx, ny int) {
				queue = append(queue, pt{nx, ny})
			})
		}
	}
}

// CycleFlag is implemented in Task 5.
func (g *Game) CycleFlag(x, y int) bool {
	if g.status == Won || g.status == Lost || !g.inBounds(x, y) {
		return false
	}
	c := g.at(x, y)
	switch c.State {
	case Hidden:
		c.State = Flagged
		g.flags++
		return true
	case Flagged:
		c.State = Questioned
		g.flags--
		return true
	case Questioned:
		c.State = Hidden
		return true
	}
	return false
}

// Chord is implemented in Task 6.
func (g *Game) Chord(x, y int) bool {
	if g.status == Won || g.status == Lost || !g.inBounds(x, y) {
		return false
	}
	return false
}
```

(The flag cycle is needed by this task's tests, so it lands complete here; Task 5 adds its dedicated tests. The `Chord` stub only carries the game-over guard.)

Create `internal/engine/game.go`:

```go
package engine

// checkWin promotes the game to Won when every safe cell is revealed.
// Full behavior (auto-flagging remaining mines) is completed in Task 7.
func (g *Game) checkWin() {
	if g.revealed == g.width*g.height-g.mines {
		g.status = Won
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -v`
Expected: all PASS, including all Task 2–3 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat: flood-fill reveal, loss on mine, reveal edge cases"
```

---

### Task 5: Flag cycle and mine counter

**Files:**
- Modify: `internal/engine/actions.go` (behavior already landed in Task 4 — this task locks it with tests)
- Test: `internal/engine/flag_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/engine/flag_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — expect PASS (characterization)**

Run: `go test ./internal/engine/ -run TestFlag -v && go test ./internal/engine/ -run TestMinesRemaining -v`
Expected: PASS — the implementation landed with Task 4. These tests pin the contract; if any FAIL, fix `CycleFlag` (not the tests) until the cycle is hidden → flagged → questioned → hidden with the counter only counting flags.

- [ ] **Step 3: Commit**

```bash
git add internal/engine/flag_test.go
git commit -m "test: lock flag cycle and mine counter contract"
```

---

### Task 6: Chord

**Files:**
- Modify: `internal/engine/actions.go` (replace the `Chord` stub)
- Test: `internal/engine/chord_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/engine/chord_test.go` (uses `cornerBoard` from `reveal_test.go` — mines at (0,0) and (2,2); cell (1,1) reads "2"):

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run TestChord -v`
Expected: `TestChordWithCorrectFlagsRevealsNeighbors`, `TestChordRevealsQuestionedNeighbors`, `TestChordWithWrongFlagLoses` FAIL (stub returns false); `TestChordNoOps` passes by accident.

- [ ] **Step 3: Write the implementation**

Replace the `Chord` stub in `internal/engine/actions.go`:

```go
// Chord opens all non-flagged neighbors of a revealed numbered cell when the
// adjacent flag count equals its number. Questioned cells count as hidden and
// are opened. A wrongly placed flag means a mine gets opened — that is a loss,
// faithful to the original game.
func (g *Game) Chord(x, y int) bool {
	if g.status == Won || g.status == Lost || !g.inBounds(x, y) {
		return false
	}
	c := g.at(x, y)
	if c.State != Revealed || c.Adjacent == 0 {
		return false
	}
	flagged := 0
	g.eachNeighbor(x, y, func(nx, ny int) {
		if g.at(nx, ny).State == Flagged {
			flagged++
		}
	})
	if flagged != c.Adjacent {
		return false
	}
	changed := false
	g.eachNeighbor(x, y, func(nx, ny int) {
		if g.status == Lost {
			return // a mine already ended the game mid-chord
		}
		n := g.at(nx, ny)
		if n.State == Flagged || n.State == Revealed {
			return
		}
		if n.Mine {
			n.State = Revealed
			g.hitX, g.hitY, g.mineHit = nx, ny, true
			g.status = Lost
			changed = true
			return
		}
		g.floodReveal(nx, ny)
		changed = true
	})
	if g.status != Lost {
		g.checkWin()
	}
	return changed
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat: chording with wrong-flag loss, questioned cells open"
```

---

### Task 7: Win detection and auto-flag on win

**Files:**
- Modify: `internal/engine/game.go` (complete `checkWin`, add `autoFlagMines`)
- Test: `internal/engine/win_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/engine/win_test.go`:

```go
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
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run TestWin -v`
Expected: `TestWinAutoFlagsRemainingMines` and `TestWinAutoFlagsQuestionedMine` FAIL (no auto-flag yet); the others PASS.

- [ ] **Step 3: Write the implementation**

Replace `internal/engine/game.go`:

```go
package engine

// checkWin promotes the game to Won when every safe cell is revealed and
// auto-flags any remaining unflagged mines (classic behavior: the counter
// reads zero on a win).
func (g *Game) checkWin() {
	if g.revealed != g.width*g.height-g.mines {
		return
	}
	g.status = Won
	g.autoFlagMines()
}

func (g *Game) autoFlagMines() {
	for i := range g.cells {
		c := &g.cells[i]
		if c.Mine && c.State != Flagged {
			c.State = Flagged
			g.flags++
		}
	}
}
```

- [ ] **Step 4: Run full engine suite with coverage**

Run: `go test ./internal/engine/ -cover`
Expected: all PASS, coverage ≥ 90%.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat: win detection with auto-flagged mines"
```

---

### Task 8: Best-times store

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/store/store_test.go` (black-box, package `store_test`):

```go
package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wallymagill/mines/internal/store"
)

func tempStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mines", "scores.json")
	return store.New(path), path
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	s, _ := tempStore(t)
	if got := s.Load(); len(got) != 0 {
		t.Errorf("Load() = %v, want empty", got)
	}
}

func TestLoadCorruptedFileReturnsEmpty(t *testing.T) {
	s, path := tempStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.Load(); len(got) != 0 {
		t.Errorf("Load() on corrupt file = %v, want empty", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	s, _ := tempStore(t)
	want := store.Scores{"beginner": 42, "expert": 311}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	got := s.Load()
	if got["beginner"] != 42 || got["expert"] != 311 || len(got) != 2 {
		t.Errorf("round trip = %v, want %v", got, want)
	}
}

func TestSaveLeavesNoTempFile(t *testing.T) {
	s, path := tempStore(t)
	if err := s.Save(store.Scores{"beginner": 9}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file left behind after atomic save")
	}
}

func TestRecordIfBest(t *testing.T) {
	s, _ := tempStore(t)
	tests := []struct {
		name     string
		secs     int
		wantSave bool
		wantBest int
	}{
		{"first time always records", 100, true, 100},
		{"slower time ignored", 150, false, 100},
		{"equal time ignored", 100, false, 100},
		{"faster time records", 60, true, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved, err := s.RecordIfBest("beginner", tt.secs)
			if err != nil {
				t.Fatal(err)
			}
			if saved != tt.wantSave {
				t.Errorf("saved = %v, want %v", saved, tt.wantSave)
			}
			if got := s.Load()["beginner"]; got != tt.wantBest {
				t.Errorf("best = %d, want %d", got, tt.wantBest)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/ -v`
Expected: compile error — package `store` does not exist.

- [ ] **Step 3: Write the implementation**

`internal/store/store.go`:

```go
// Package store persists best times as JSON in the user config directory.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Scores maps a preset key ("beginner", "intermediate", "expert") to the
// best completion time in seconds.
type Scores map[string]int

type Store struct {
	path string
}

func New(path string) *Store { return &Store{path: path} }

// DefaultPath is os.UserConfigDir()/mines/scores.json
// (~/Library/Application Support/mines/scores.json on macOS).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mines", "scores.json"), nil
}

// Load returns saved scores. A missing or corrupted file yields empty scores —
// score I/O must never take the game down.
func (s *Store) Load() Scores {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return Scores{}
	}
	var sc Scores
	if err := json.Unmarshal(data, &sc); err != nil || sc == nil {
		return Scores{}
	}
	return sc
}

// Save writes scores atomically (temp file + rename).
func (s *Store) Save(sc Scores) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// RecordIfBest saves secs under key if it beats the current best.
// It reports whether a new record was written.
func (s *Store) RecordIfBest(key string, secs int) (bool, error) {
	sc := s.Load()
	if best, ok := sc[key]; ok && best <= secs {
		return false, nil
	}
	sc[key] = secs
	return true, s.Save(sc)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/ -v -cover`
Expected: all PASS, coverage ≥ 80% (DefaultPath is exercised only in production).

- [ ] **Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat: atomic JSON best-times store with record-if-best"
```

> **Execution amendment (audit-driven):** `Load` drops entries whose value
> lies outside [0, 999]. The UI timer starts at 0 and caps at 999, so no
> real game can produce a value beyond that range — such entries only come
> from hand edits or valid-JSON corruption. Previously they loaded
> verbatim: a `{"beginner": -3}` file rendered `best: -03` in the menu and,
> because `RecordIfBest` compares against the stored best, permanently
> blocked every legitimate future record for that preset. The boundary
> values 0 (sub-second win) and 999 (timer cap) are reachable in play and
> load untouched.

> **Execution amendment (audit-driven):** `Save` removes `path+".tmp"`
> (best effort) when the rename step fails — e.g. the scores path occupied
> by a directory, or the parent directory's write bit lost between the two
> calls. Atomicity already held (a failed save never touched an existing
> good file), but the temp file was orphaned on that path; the standard
> atomic-write idiom cleans it up, and `TestSaveLeavesNoTempFile` already
> pinned the no-stray-temp intent for the success path. The rename error
> is still returned to the caller either way.

---

### Task 9: UI presets and styles

Pure constants — no behavior, so no TDD cycle; verified by compilation and used by every later UI task.

**Files:**
- Create: `internal/ui/presets.go`
- Create: `internal/ui/styles.go`

- [ ] **Step 1: Create `internal/ui/presets.go`**

```go
// Package ui implements the Bubble Tea presentation layer.
package ui

// Preset describes a board configuration. Key is the store key for best
// times; it is empty for custom boards, which never record times.
type Preset struct {
	Name   string
	Key    string
	Width  int
	Height int
	Mines  int
}

var presets = []Preset{
	{Name: "Beginner", Key: "beginner", Width: 9, Height: 9, Mines: 10},
	{Name: "Intermediate", Key: "intermediate", Width: 16, Height: 16, Mines: 40},
	{Name: "Expert", Key: "expert", Width: 30, Height: 16, Mines: 99},
}
```

- [ ] **Step 2: Create `internal/ui/styles.go`**

```go
package ui

import "github.com/charmbracelet/lipgloss"

// Cell glyphs are two columns wide so the board reads roughly square.
const (
	glyphHidden   = "░░"
	glyphEmpty    = "· "
	glyphFlag     = "⚑ "
	glyphQuestion = "? "
	glyphMine     = "✱ "
	glyphWrong    = "✕ " // a flag that was not a mine, shown on loss
)

// boardChromeRows: HUD (1) + blank (1) + blank (1) + status (1) + margins (2).
const boardChromeRows = 6

// Classic Windows minesweeper number palette.
var numberStyles = map[int]lipgloss.Style{
	1: lipgloss.NewStyle().Foreground(lipgloss.Color("12")), // blue
	2: lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // green
	3: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // red
	4: lipgloss.NewStyle().Foreground(lipgloss.Color("4")),  // navy
	5: lipgloss.NewStyle().Foreground(lipgloss.Color("1")),  // maroon
	6: lipgloss.NewStyle().Foreground(lipgloss.Color("6")),  // teal
	7: lipgloss.NewStyle().Foreground(lipgloss.Color("15")), // white
	8: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),  // gray
}

var (
	styleHidden = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleFlag   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleMine   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleCursor = lipgloss.NewStyle().Reverse(true)
	styleHUD    = lipgloss.NewStyle().Bold(true)
	styleFaint  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: success (unused-variable rules don't apply to package-level declarations).

- [ ] **Step 4: Commit**

```bash
git add internal/ui/
git commit -m "feat: ui presets, glyphs, and classic color palette"
```

---

### Task 10: Game model — cursor, key routing, engine actions

**Files:**
- Create: `internal/ui/game.go`
- Test: `internal/ui/game_test.go`

The game model wraps an `engine.Game`. Keys: `hjkl`/arrows move (clamped to the board), `space`/`enter` reveal — or chord when the cursor sits on a revealed number, `f` cycles flags, `c` chords explicitly, `n` restarts with the same preset, `?` toggles a help overlay. Quit/menu keys are handled by the root model (Task 14).

- [ ] **Step 1: Write the failing tests**

`internal/ui/game_test.go`:

```go
package ui

import (
	"math/rand"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/engine"
	"github.com/wallymagill/mines/internal/store"
)

var testPreset = Preset{Name: "Test", Key: "", Width: 4, Height: 4, Mines: 2}

func testGameModel(t *testing.T) gameModel {
	t.Helper()
	s := store.New(t.TempDir() + "/scores.json")
	m, err := newGameModel(testPreset, s, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	if s == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestCursorMovementClamps(t *testing.T) {
	m := testGameModel(t)
	tests := []struct {
		name   string
		keys   []string
		wantX  int
		wantY  int
	}{
		{"starts at origin, h/k clamp", []string{"h", "k"}, 0, 0},
		{"l and j move", []string{"l", "l", "j"}, 2, 1},
		{"clamps at right/bottom edge", []string{"l", "l", "l", "l", "l", "j", "j", "j", "j"}, 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testGameModel(t)
			for _, k := range tt.keys {
				m, _ = m.Update(key(k))
			}
			if m.cursorX != tt.wantX || m.cursorY != tt.wantY {
				t.Errorf("cursor = (%d,%d), want (%d,%d)", m.cursorX, m.cursorY, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestSpaceRevealsAndStartsGame(t *testing.T) {
	m := testGameModel(t)
	m, cmd := m.Update(key(" "))
	c, _ := m.game.CellAt(0, 0)
	if c.State != engine.Revealed {
		t.Error("space should reveal the cursor cell")
	}
	if m.game.Status() == engine.NotStarted {
		t.Error("game should have started")
	}
	if !m.running || cmd == nil {
		t.Error("first reveal must start the timer (running=true, tick cmd returned)")
	}
}

func TestFKeyCyclesFlag(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key("f"))
	c, _ := m.game.CellAt(0, 0)
	if c.State != engine.Flagged {
		t.Errorf("state = %v, want Flagged", c.State)
	}
	if m.game.MinesRemaining() != 1 {
		t.Errorf("MinesRemaining = %d, want 1", m.game.MinesRemaining())
	}
}

func TestNRestartsGame(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" "))
	m, _ = m.Update(key("n"))
	if m.game.Status() != engine.NotStarted {
		t.Error("n should produce a fresh, unstarted game")
	}
	if m.elapsed != 0 || m.running {
		t.Error("n should reset the timer")
	}
}

func TestHelpOverlayToggles(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key("?"))
	if !m.showHelp {
		t.Error("? should open help")
	}
	m, _ = m.Update(key("j"))
	if m.showHelp {
		t.Error("any key should close help")
	}
	if m.cursorY != 0 {
		t.Error("the help-dismissing key must not also act on the board")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -v`
Expected: compile error — `undefined: newGameModel`.

- [ ] **Step 3: Write the implementation**

`internal/ui/game.go`:

```go
package ui

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/engine"
	"github.com/wallymagill/mines/internal/store"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type gameModel struct {
	game     *engine.Game
	preset   Preset
	scores   *store.Store
	rng      *rand.Rand
	cursorX  int
	cursorY  int
	elapsed  int
	running  bool
	recorded bool
	newBest  bool
	saveErr  bool
	showHelp bool
}

func newGameModel(p Preset, scores *store.Store, rng *rand.Rand) (gameModel, error) {
	g, err := engine.New(p.Width, p.Height, p.Mines, rng)
	if err != nil {
		return gameModel{}, err
	}
	return gameModel{game: g, preset: p, scores: scores, rng: rng}, nil
}

func (m gameModel) Update(msg tea.Msg) (gameModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.running && m.game.Status() == engine.Playing {
			if m.elapsed < 999 {
				m.elapsed++
			}
			return m, tickCmd()
		}
		m.running = false
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg.String())
	}
	return m, nil
}

func (m gameModel) handleKey(k string) (gameModel, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}
	switch k {
	case "up", "k":
		if m.cursorY > 0 {
			m.cursorY--
		}
	case "down", "j":
		if m.cursorY < m.game.Height()-1 {
			m.cursorY++
		}
	case "left", "h":
		if m.cursorX > 0 {
			m.cursorX--
		}
	case "right", "l":
		if m.cursorX < m.game.Width()-1 {
			m.cursorX++
		}
	case " ", "enter":
		return m.activate(false)
	case "c":
		return m.activate(true)
	case "f":
		m.game.CycleFlag(m.cursorX, m.cursorY)
	case "n":
		fresh, err := newGameModel(m.preset, m.scores, m.rng)
		if err != nil {
			return m, nil // preset was already valid; keep current game
		}
		return fresh, nil
	case "?":
		m.showHelp = true
	}
	return m, nil
}

// activate runs reveal-or-chord at the cursor, starts the timer on the first
// reveal, and records a best time on a win. chordOnly forces chord (the "c"
// key); otherwise a revealed numbered cell chords and anything else reveals.
func (m gameModel) activate(chordOnly bool) (gameModel, tea.Cmd) {
	wasNotStarted := m.game.Status() == engine.NotStarted
	cell, ok := m.game.CellAt(m.cursorX, m.cursorY)
	if !ok {
		return m, nil
	}
	if chordOnly || cell.State == engine.Revealed {
		m.game.Chord(m.cursorX, m.cursorY)
	} else {
		m.game.Reveal(m.cursorX, m.cursorY)
	}
	var cmd tea.Cmd
	if wasNotStarted && m.game.Status() != engine.NotStarted && !m.running {
		m.running = true
		cmd = tickCmd()
	}
	m = m.recordIfWon()
	return m, cmd
}

func (m gameModel) recordIfWon() gameModel {
	if m.game.Status() != engine.Won || m.preset.Key == "" || m.recorded {
		return m
	}
	m.recorded = true
	isBest, err := m.scores.RecordIfBest(m.preset.Key, m.elapsed)
	if err != nil {
		m.saveErr = true
		return m
	}
	m.newBest = isBest
	return m
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/game.go internal/ui/game_test.go
git commit -m "feat: game model with cursor, key routing, and timer start"
```

> **Execution amendment (review-driven):** `tickMsg` became `struct{ gen int }`
> with a `tickGen` generation counter on the model — a stale tick surviving an
> `n` restart otherwise creates a second tick loop (2× timer, corrupted best
> times). `n` bumps the generation; mismatched ticks are dropped without
> rescheduling. activate() starts the timer only when status is exactly
> `Playing`. Chord routing, arrow keys, and up/left movement gained tests.

---

### Task 11: Game model — timer ticks and best-time recording

**Files:**
- Modify: `internal/ui/game.go` (already complete — this task locks timer/score behavior with tests)
- Test: `internal/ui/game_timer_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/ui/game_timer_test.go`:

```go
package ui

import (
	"math/rand"
	"testing"
	"time"

	"github.com/wallymagill/mines/internal/store"
)

func TestTickIncrementsOnlyWhilePlaying(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(tickMsg(time.Time{})) // before first reveal: no-op
	if m.elapsed != 0 {
		t.Errorf("elapsed before start = %d, want 0", m.elapsed)
	}
	m, _ = m.Update(key(" ")) // start
	m, cmd := m.Update(tickMsg(time.Time{}))
	if m.elapsed != 1 {
		t.Errorf("elapsed after one tick = %d, want 1", m.elapsed)
	}
	if cmd == nil {
		t.Error("tick while playing must schedule the next tick")
	}
}

func TestTimerCapsAt999(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" "))
	m.elapsed = 999
	m, _ = m.Update(tickMsg(time.Time{}))
	if m.elapsed != 999 {
		t.Errorf("elapsed = %d, want capped at 999", m.elapsed)
	}
}

func TestWinRecordsBestTimeForPreset(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	// 2x2 board, 1 mine: first reveal places the mine elsewhere; revealing
	// the three safe cells wins. Find them via the engine state.
	p := Preset{Name: "Tiny", Key: "beginner", Width: 2, Height: 2, Mines: 1}
	m, err := newGameModel(p, s, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	m.elapsed = 7 // pretend 7 seconds passed
	// Reveal every cell that isn't the mine.
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			m.cursorX, m.cursorY = x, y
			if c, _ := m.game.CellAt(x, y); !c.Mine {
				m, _ = m.Update(key(" "))
			}
		}
	}
	// Note: elapsed was set before the first reveal; activate() does not
	// reset it, so the recorded time is 7.
	if got := s.Load()["beginner"]; got != 7 {
		t.Errorf("recorded best = %d, want 7", got)
	}
	if !m.newBest {
		t.Error("first win should be flagged as a new best")
	}
}

func TestCustomGamesDoNotRecordTimes(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	p := Preset{Name: "Custom", Key: "", Width: 2, Height: 2, Mines: 1}
	m, err := newGameModel(p, s, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			m.cursorX, m.cursorY = x, y
			if c, _ := m.game.CellAt(x, y); !c.Mine {
				m, _ = m.Update(key(" "))
			}
		}
	}
	if got := s.Load(); len(got) != 0 {
		t.Errorf("custom game recorded a time: %v", got)
	}
}
```

There is a subtlety the first test must respect: on a 2×2 board with 1 mine, mines are placed on the *first* reveal — `CellAt(...).Mine` is meaningless before that. The loop above works because the first iteration reveals (0,0) (triggering placement that excludes it), and later iterations check `Mine` against the now-placed board. If the implementation under test breaks this, fix the implementation, not the loop.

- [ ] **Step 2: Run tests — expect mostly PASS (characterization), fix any failures**

Run: `go test ./internal/ui/ -v`
Expected: PASS — Task 10 landed this behavior. These tests pin the timer and recording contract. If `TestWinRecordsBestTimeForPreset` fails because placement made (0,0) a mine under seed 3, change the seed in the test until first-reveal placement yields a winnable loop (the engine guarantees (0,0) is safe, so the loop is correct for any seed).

- [ ] **Step 3: Commit**

```bash
git add internal/ui/game_timer_test.go
git commit -m "test: lock timer tick and best-time recording contract"
```

---

### Task 12: Game view rendering + golden test

**Files:**
- Modify: `internal/ui/game.go` (add `View` and render helpers)
- Test: `internal/ui/game_view_test.go`

Note on the spec: it names `teatest` for the smoke test. A direct `View()` golden test gives the same signal with zero extra dependencies, because `go test` runs without a TTY so Lip Gloss renders plain text (no ANSI codes). This is the one deliberate deviation from the spec.

- [ ] **Step 1: Write the failing tests**

`internal/ui/game_view_test.go`:

```go
package ui

import (
	"strings"
	"testing"
)

func TestViewGoldenInitialBoard(t *testing.T) {
	m := testGameModel(t) // 4x4, 2 mines, nothing revealed
	got := m.View()
	want := strings.Join([]string{
		" 🚩 002   🙂   ⏱ 000",
		"",
		"[░░]░░ ░░ ░░",
		" ░░ ░░ ░░ ░░",
		" ░░ ░░ ░░ ░░",
		" ░░ ░░ ░░ ░░",
		" hjkl move · spc reveal · f flag · c chord · n new · ? help · m menu · q quit",
	}, "\n")
	if got != want {
		t.Errorf("View() mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestViewShowsRevealedNumbersAndEmpty(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" ")) // reveal at (0,0)
	v := m.View()
	if strings.Count(v, "░") == 16*2 {
		t.Error("view should show fewer hidden cells after a reveal")
	}
}

func TestViewLostShowsMinesAndStatus(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" ")) // start: places mines
	// Move the cursor onto a mine and reveal it.
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if c, _ := m.game.CellAt(x, y); c.Mine {
				m.cursorX, m.cursorY = x, y
			}
		}
	}
	m, _ = m.Update(key(" "))
	v := m.View()
	if !strings.Contains(v, "✱") {
		t.Error("lost view must expose mines")
	}
	if !strings.Contains(v, "💀") {
		t.Error("lost view must show the dead face")
	}
	if !strings.Contains(v, "n new") {
		t.Error("lost view must show restart hint")
	}
}

func TestViewHelpOverlay(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key("?"))
	v := m.View()
	if !strings.Contains(v, "chord") || !strings.Contains(v, "flag") {
		t.Error("help overlay should list key bindings")
	}
}
```

The exact golden string in `TestViewGoldenInitialBoard` depends on spacing decisions in `View()`. **Write the implementation first if needed, run it, eyeball the output for correctness against the layout below, then paste the verified output into the test.** That is an acceptable golden-test workflow — the eyeball step is the test design.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestView -v`
Expected: compile error — `m.View undefined`.

- [ ] **Step 3: Write the implementation**

Append to `internal/ui/game.go`:

```go
import (
	"fmt"
	"strings"
	// ...existing imports
)

func (m gameModel) View() string {
	var b strings.Builder
	b.WriteString(m.hudView())
	b.WriteString("\n\n")
	b.WriteString(m.boardView())
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	return b.String()
}

func (m gameModel) hudView() string {
	face := "🙂"
	switch m.game.Status() {
	case engine.Won:
		face = "😎"
	case engine.Lost:
		face = "💀"
	}
	return styleHUD.Render(fmt.Sprintf(" 🚩 %03d   %s   ⏱ %03d",
		m.game.MinesRemaining(), face, m.elapsed))
}

func (m gameModel) boardView() string {
	var b strings.Builder
	over := m.game.Status() == engine.Won || m.game.Status() == engine.Lost
	for y := 0; y < m.game.Height(); y++ {
		for x := 0; x < m.game.Width(); x++ {
			onCursor := x == m.cursorX && y == m.cursorY && !over
			// Cursor brackets live in the separator columns.
			switch {
			case onCursor:
				b.WriteString("[")
			case x > 0 && m.cursorY == y && m.cursorX == x-1 && !over:
				b.WriteString("]")
			default:
				b.WriteString(" ")
			}
			b.WriteString(m.cellView(x, y, onCursor))
		}
		if m.cursorY == y && m.cursorX == m.game.Width()-1 && !over {
			b.WriteString("]")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m gameModel) cellView(x, y int, onCursor bool) string {
	cell, _ := m.game.CellAt(x, y)
	lost := m.game.Status() == engine.Lost
	glyph, style := m.cellGlyph(cell, lost)
	if onCursor {
		return styleCursor.Render(glyph)
	}
	return style.Render(glyph)
}

func (m gameModel) cellGlyph(cell engine.Cell, lost bool) (string, lipgloss.Style) {
	switch {
	case lost && cell.Mine && cell.State != engine.Flagged:
		return glyphMine, styleMine
	case lost && !cell.Mine && cell.State == engine.Flagged:
		return glyphWrong, styleMine
	case cell.State == engine.Flagged:
		return glyphFlag, styleFlag
	case cell.State == engine.Questioned:
		return glyphQuestion, styleFlag
	case cell.State != engine.Revealed:
		return glyphHidden, styleHidden
	case cell.Adjacent == 0:
		return glyphEmpty, styleFaint
	default:
		return fmt.Sprintf("%d ", cell.Adjacent), numberStyles[cell.Adjacent]
	}
}

func (m gameModel) statusLine() string {
	if m.showHelp {
		return styleFaint.Render(strings.Join([]string{
			"",
			" hjkl / arrows  move cursor",
			" space / enter  reveal (chord on a revealed number)",
			" f              flag → ? → clear",
			" c              chord",
			" n              new game   m  menu   q  quit",
			"",
			" any key to close help",
		}, "\n"))
	}
	switch m.game.Status() {
	case engine.Won:
		s := fmt.Sprintf(" 😎 Cleared in %ds", m.elapsed)
		if m.newBest {
			s += " — NEW BEST!"
		}
		if m.saveErr {
			s += styleErr.Render("  (could not save score)")
		}
		return s + styleFaint.Render("  · n new · m menu")
	case engine.Lost:
		return " 💀 Boom." + styleFaint.Render("  · n new · m menu")
	default:
		return styleFaint.Render(" hjkl move · spc reveal · f flag · c chord · n new · ? help · m menu · q quit")
	}
}
```

Also add `"github.com/charmbracelet/lipgloss"` to the import block (needed by `cellGlyph`'s return type).

- [ ] **Step 4: Calibrate the golden string**

Run: `go test ./internal/ui/ -run TestViewGolden -v`
If the golden test fails on exact spacing: print the actual output with `t.Logf("%q", got)`, verify by eye that it matches the intended layout (HUD line, blank, 4 board rows with cursor brackets at (0,0), blank, key-hint line), then update the `want` literal to the verified actual. Commit only after the eyeball check.

- [ ] **Step 5: Run all UI tests**

Run: `go test ./internal/ui/ -v`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/
git commit -m "feat: game view with HUD, board glyphs, status line, help overlay"
```

> **Execution amendment (audit-driven):** the HUD's mine counter clamps its
> *display* at -99, classic-minesweeper style — `%03d` emits four characters
> at -100 and below (reachable by mass flagging on Intermediate/Expert),
> widening the HUD by one column and shifting the face and timer mid-game.
> `MinesRemaining()` itself still goes as negative as flags allow; only the
> rendered field saturates. The positive side needs no clamp: presets and
> the three-digit custom form cap mines at 999, which `%03d` shows whole.

---

### Task 13: Menu model and custom board form

**Files:**
- Create: `internal/ui/menu.go`
- Test: `internal/ui/menu_test.go`

The menu lists Beginner / Intermediate / Expert / Custom / Quit with best times inline. Selecting Custom opens a 3-field form (width, height, mines). The menu communicates upward via a `startGameMsg` command; the root model (Task 14) consumes it.

- [ ] **Step 1: Write the failing tests**

`internal/ui/menu_test.go`:

```go
package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

func testMenu() menuModel {
	return newMenuModel(store.Scores{"beginner": 42}, 120, 40)
}

// drain runs a returned command and hands back the message it produces.
func drain(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestMenuNavigationClamps(t *testing.T) {
	m := testMenu()
	m, _ = m.Update(key("k"))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", m.cursor)
	}
	for i := 0; i < 10; i++ {
		m, _ = m.Update(key("j"))
	}
	if m.cursor != 4 {
		t.Errorf("cursor = %d, want 4 (Quit, clamped)", m.cursor)
	}
}

func TestMenuSelectPresetEmitsStartGame(t *testing.T) {
	m := testMenu()
	m, _ = m.Update(key("j")) // Intermediate
	_, cmd := m.Update(key("enter"))
	msg, ok := drain(cmd).(startGameMsg)
	if !ok {
		t.Fatalf("msg = %T, want startGameMsg", drain(cmd))
	}
	if msg.preset.Name != "Intermediate" {
		t.Errorf("preset = %s, want Intermediate", msg.preset.Name)
	}
}

func TestMenuQuit(t *testing.T) {
	m := testMenu()
	for i := 0; i < 4; i++ {
		m, _ = m.Update(key("j"))
	}
	_, cmd := m.Update(key("enter"))
	if msg := drain(cmd); msg == nil {
		t.Fatal("selecting Quit should return tea.Quit")
	} else if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("msg = %T, want tea.QuitMsg", msg)
	}
}

func TestCustomFormFlow(t *testing.T) {
	m := testMenu()
	for i := 0; i < 3; i++ {
		m, _ = m.Update(key("j")) // Custom
	}
	m, _ = m.Update(key("enter"))
	if m.form == nil {
		t.Fatal("enter on Custom should open the form")
	}
	// Type width 12, height 8, mines 20.
	for _, k := range []string{"1", "2", "enter", "8", "enter", "2", "0"} {
		m, _ = m.Update(key(k))
	}
	_, cmd := m.Update(key("enter"))
	msg, ok := drain(cmd).(startGameMsg)
	if !ok {
		t.Fatalf("msg = %T, want startGameMsg", drain(cmd))
	}
	p := msg.preset
	if p.Width != 12 || p.Height != 8 || p.Mines != 20 || p.Key != "" {
		t.Errorf("preset = %+v, want 12x8/20 with empty key", p)
	}
}

func TestCustomFormValidation(t *testing.T) {
	tests := []struct {
		name    string
		entries []string // width, height, mines
		wantErr string
	}{
		{"empty fields", []string{"", "", ""}, "required"},
		{"board too small", []string{"1", "9", "5"}, "at least 2x2"},
		{"too many mines", []string{"4", "4", "15"}, "mines must be"},
		{"too big for terminal", []string{"99", "9", "10"}, "too large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMenuModel(store.Scores{}, 80, 24)
			for i := 0; i < 3; i++ {
				m, _ = m.Update(key("j"))
			}
			m, _ = m.Update(key("enter")) // open form
			for _, field := range tt.entries {
				for _, ch := range field {
					m, _ = m.Update(key(string(ch)))
				}
				m, _ = m.Update(key("enter"))
			}
			if m.err == "" || !strings.Contains(m.err, tt.wantErr) {
				t.Errorf("err = %q, want containing %q", m.err, tt.wantErr)
			}
			if m.form == nil {
				t.Error("form should stay open on validation error")
			}
		})
	}
}

func TestCustomFormEscReturnsToList(t *testing.T) {
	m := testMenu()
	for i := 0; i < 3; i++ {
		m, _ = m.Update(key("j"))
	}
	m, _ = m.Update(key("enter"))
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.form != nil {
		t.Error("esc should close the form")
	}
}

func TestMenuViewShowsBestTimes(t *testing.T) {
	m := testMenu()
	v := m.View()
	if !strings.Contains(v, "042") {
		t.Error("menu should show the beginner best time")
	}
	if !strings.Contains(v, "—") {
		t.Error("menu should show an em dash for missing best times")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestMenu -v`
Expected: compile error — `undefined: newMenuModel`, `undefined: startGameMsg`.

- [ ] **Step 3: Write the implementation**

`internal/ui/menu.go`:

```go
package ui

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/engine"
	"github.com/wallymagill/mines/internal/store"
)

// startGameMsg is emitted by the menu and consumed by the root model.
type startGameMsg struct{ preset Preset }

func startGame(p Preset) tea.Cmd {
	return func() tea.Msg { return startGameMsg{preset: p} }
}

const (
	itemCustom = 3
	itemQuit   = 4
)

type menuModel struct {
	cursor int
	scores store.Scores
	form   *customForm
	err    string
	termW  int
	termH  int
}

func newMenuModel(scores store.Scores, termW, termH int) menuModel {
	return menuModel{scores: scores, termW: termW, termH: termH}
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.form != nil {
		return m.updateForm(keyMsg)
	}
	switch keyMsg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < itemQuit {
			m.cursor++
		}
	case "enter", " ":
		switch {
		case m.cursor < len(presets):
			return m, startGame(presets[m.cursor])
		case m.cursor == itemCustom:
			f := newCustomForm()
			m.form = &f
			m.err = ""
		default:
			return m, tea.Quit
		}
	case "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

type customForm struct {
	fields [3]string // width, height, mines
	active int
}

func newCustomForm() customForm { return customForm{} }

func (m menuModel) updateForm(keyMsg tea.KeyMsg) (menuModel, tea.Cmd) {
	switch keyMsg.String() {
	case "esc":
		m.form = nil
		m.err = ""
	case "tab", "down":
		m.form.active = (m.form.active + 1) % 3
	case "shift+tab", "up":
		m.form.active = (m.form.active + 2) % 3
	case "backspace":
		if s := m.form.fields[m.form.active]; s != "" {
			m.form.fields[m.form.active] = s[:len(s)-1]
		}
	case "enter":
		if m.form.active < 2 {
			m.form.active++
			return m, nil
		}
		p, err := m.form.preset(m.termW, m.termH)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.form = nil
		m.err = ""
		return m, startGame(p)
	default:
		s := keyMsg.String()
		if len(s) == 1 && s[0] >= '0' && s[0] <= '9' && len(m.form.fields[m.form.active]) < 3 {
			m.form.fields[m.form.active] += s
		}
	}
	return m, nil
}

// preset validates the form. Rule checks are delegated to engine.New (DRY);
// the terminal-fit check is presentation-layer knowledge and lives here.
func (f *customForm) preset(termW, termH int) (Preset, error) {
	w, err1 := strconv.Atoi(f.fields[0])
	h, err2 := strconv.Atoi(f.fields[1])
	mines, err3 := strconv.Atoi(f.fields[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return Preset{}, errors.New("all fields are required")
	}
	if _, err := engine.New(w, h, mines, rand.New(rand.NewSource(0))); err != nil {
		return Preset{}, err
	}
	if termW > 0 && (w*2+2 > termW || h+boardChromeRows > termH) {
		return Preset{}, fmt.Errorf("board too large for terminal (max %d×%d)",
			(termW-2)/2, termH-boardChromeRows)
	}
	return Preset{Name: "Custom", Width: w, Height: h, Mines: mines}, nil
}

func (m menuModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(" MINES") + "\n\n")
	if m.form != nil {
		return b.String() + m.formView()
	}
	for i, p := range presets {
		b.WriteString(m.menuRow(i, fmt.Sprintf("%-13s %d×%d · %d mines  best: %s",
			p.Name, p.Width, p.Height, p.Mines, m.bestFor(p.Key))))
	}
	b.WriteString(m.menuRow(itemCustom, "Custom…"))
	b.WriteString(m.menuRow(itemQuit, "Quit"))
	b.WriteString("\n" + styleFaint.Render(" ↑↓ move · enter select"))
	if m.err != "" {
		b.WriteString("\n" + styleErr.Render(" "+m.err))
	}
	return b.String()
}

func (m menuModel) menuRow(i int, label string) string {
	prefix := "   "
	if m.cursor == i {
		prefix = " > "
		return styleHUD.Render(prefix+label) + "\n"
	}
	return prefix + label + "\n"
}

func (m menuModel) bestFor(key string) string {
	if secs, ok := m.scores[key]; ok {
		return fmt.Sprintf("%03d", secs)
	}
	return "—"
}

func (m menuModel) formView() string {
	labels := []string{"width", "height", "mines"}
	var b strings.Builder
	b.WriteString(" Custom board\n\n")
	for i, l := range labels {
		cursor := " "
		if m.form.active == i {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf(" %s %-7s %s\n", cursor, l+":", m.form.fields[i]))
	}
	b.WriteString("\n" + styleFaint.Render(" tab/enter next · enter on mines starts · esc back"))
	if m.err != "" {
		b.WriteString("\n" + styleErr.Render(" "+m.err))
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/menu.go internal/ui/menu_test.go
git commit -m "feat: menu with presets, best times, validated custom form"
```

> **Execution amendment (review-driven):** the form became a value
> (`form customForm` + `showForm bool`) instead of `*customForm` — pointer
> mutation through a value receiver aliased old model copies. `itemCustom`/
> `itemQuit` are derived from `len(presets)`; editing clears stale validation
> errors; the fit check requires both terminal dims and clamps printed maxes.

> **Execution amendment (audit-driven):** the preset row format pinned above
> padded only the name (`%-13s`), leaving the dims segment ragged — Beginner's
> `9×9` is two columns narrower than `16×16`/`30×16`, so its `best:` field
> started two columns early. The dims are now formatted separately and padded
> to the widest preset (`%-5s`; `×` is a single rune and column), aligning the
> `best:` column across all rows with or without recorded times.

---

### Task 14: App root model and main wiring

**Files:**
- Create: `internal/ui/app.go`
- Modify: `main.go` (replace stub)
- Test: `internal/ui/app_test.go`

The root model owns the menu↔game routing, global quit keys, window size, and the too-small guard. It is the only exported type in `ui`.

- [ ] **Step 1: Write the failing tests**

`internal/ui/app_test.go`:

```go
package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

func testApp(t *testing.T) App {
	t.Helper()
	a := NewApp(store.New(t.TempDir() + "/scores.json"))
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return model.(App)
}

func TestAppStartsOnMenu(t *testing.T) {
	a := testApp(t)
	if !strings.Contains(a.View(), "Beginner") {
		t.Error("initial view should be the menu")
	}
}

func TestAppRoutesStartGameToGameView(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	if a.view != viewGame {
		t.Error("startGameMsg should switch to the game view")
	}
	if !strings.Contains(a.View(), "🚩") {
		t.Error("game view should render the HUD")
	}
}

func TestAppMenuKeyReturnsToMenu(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	model, _ = a.Update(key("m"))
	a = model.(App)
	if a.view != viewMenu {
		t.Error("m should return to the menu")
	}
}

func TestAppCtrlCQuits(t *testing.T) {
	a := testApp(t)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c should produce tea.QuitMsg")
	}
}

func TestAppTooSmallTerminal(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[2]}) // Expert: needs 62 cols
	a = model.(App)
	model, _ = a.Update(tea.WindowSizeMsg{Width: 40, Height: 40})
	a = model.(App)
	if !strings.Contains(a.View(), "too small") {
		t.Error("undersized terminal should show the too-small message")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestApp -v`
Expected: compile error — `undefined: NewApp`.

- [ ] **Step 3: Write the implementation**

`internal/ui/app.go`:

```go
package ui

import (
	"fmt"
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

type view int

const (
	viewMenu view = iota
	viewGame
)

// App is the Bubble Tea root model: routes between menu and game, owns
// global keys (quit, back-to-menu) and terminal size.
type App struct {
	view   view
	menu   menuModel
	game   gameModel
	scores *store.Store
	termW  int
	termH  int
}

func NewApp(scores *store.Store) App {
	return App{
		menu:   newMenuModel(scores.Load(), 0, 0),
		scores: scores,
	}
}

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.termW, a.termH = msg.Width, msg.Height
		a.menu.termW, a.menu.termH = msg.Width, msg.Height
		return a, nil
	case startGameMsg:
		gm, err := newGameModel(msg.preset, a.scores,
			rand.New(rand.NewSource(time.Now().UnixNano())))
		if err != nil {
			a.menu.err = err.Error()
			return a, nil
		}
		// Thread the tick generation across rounds: a tick loop from the
		// previous game may still be in flight, and a fresh model at gen 0
		// would mistake it for its own.
		gm.tickGen = a.game.tickGen + 1
		a.game = gm
		a.view = viewGame
		return a, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		if a.view == viewGame {
			switch msg.String() {
			case "m", "esc":
				a.view = viewMenu
				a.menu.scores = a.scores.Load() // refresh best times
				a.menu.err = ""
				return a, nil
			case "q":
				return a, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	if a.view == viewGame {
		a.game, cmd = a.game.Update(msg)
	} else {
		a.menu, cmd = a.menu.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	if a.view != viewGame {
		return "\n" + a.menu.View() + "\n"
	}
	needW := a.game.game.Width()*2 + 2
	needH := a.game.game.Height() + boardChromeRows
	if a.termW > 0 && (a.termW < needW || a.termH < needH) {
		return fmt.Sprintf(
			"\n Terminal too small — need %d×%d, have %d×%d.\n Resize, or press m for menu.\n",
			needW, needH, a.termW, a.termH)
	}
	return "\n" + a.game.View() + "\n"
}
```

Note: non-key messages (timer ticks) fall through the switch to the active sub-model — that is what keeps the clock running.

Replace `main.go`:

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
	"github.com/wallymagill/mines/internal/ui"
)

func main() {
	path, err := store.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mines: cannot determine config dir:", err)
		os.Exit(1)
	}
	app := ui.NewApp(store.New(path))
	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "mines:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Run all tests and build**

Run: `go test ./... && go build -o mines .`
Expected: all PASS, binary builds.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/app.go internal/ui/app_test.go main.go
git commit -m "feat: root model routing and program entry point"
```

> **Execution amendment (review-driven):** the too-small guard's width math
> was wrong in this plan — the renderer emits 3 columns per cell plus the
> cursor's trailing bracket, so a width-W board needs `3W+1` columns, not
> `2W+2` (Expert wrapped on an 80-column terminal with no guard). A shared
> `boardCols()` helper in styles.go now feeds both the App guard and the
> custom-form fit check. Global m/esc/q interception also skips while the
> help overlay is open, honoring "any key to close help".

> **Execution amendment (audit-driven):** the too-small guard now sizes the
> whole frame, not just the board grid — the chrome (HUD, status line) was
> still truncated on terminals the guard accepted. `frameCols(w) =
> max(boardCols(w), chromeCols)` and `frameRows(h) = h + boardChromeRows`
> are shared by the App guard and the custom-form fit check. Rather than
> demanding the 77-column key hint everywhere, the chrome adapts to the
> terminal (the spec sketch shows compact in-frame hints): `fitLine` swaps
> the full status/win lines for compact variants when the last
> `WindowSizeMsg` width can't show them whole, and the help overlay — the
> canonical full key list — was reflowed to the same budget, so
> `chromeCols` is 30 and Beginner stays playable from 30×14 up, including
> 80×24. `boardChromeRows` was also corrected from 6 to 5: the rendered
> layout has a single interior blank (margin, HUD, blank, board, status,
> margin), so exact-fit terminals are now accepted and the custom form
> allows one more row of board. The too-small screen itself was stacked
> into four short lines (≤ 20 columns, 6 rows) so it fits every terminal
> that triggers it, entering the too-small state closes an open help
> overlay, and all keys except m/esc/q are ignored while it shows — the
> advertised "press m for menu" now always works on the first press.

---

### Task 15: Final verification

**Files:** none created — verification only.

- [ ] **Step 1: Format, vet, full test suite with coverage**

```bash
gofmt -l . && go vet ./...
go mod tidy && git diff --exit-code go.mod go.sum
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1
```

(`go mod tidy` must be a no-op: Task 1 pre-pinned bubbletea/lipgloss before any import used them, which is fine mid-branch but must be clean at merge.)

Expected: `gofmt -l` prints nothing, `go vet` clean, all tests PASS, total coverage ≥ 80%. If coverage falls short, `go tool cover -html=coverage.out` shows the gaps — add engine/store tests first (they're the meaningful ones), never View-string assertions for coverage's sake.

- [ ] **Step 2: Build and play (manual verification checklist)**

```bash
go build -o mines . && ./mines
```

Verify each spec behavior by playing:
- [ ] Menu shows three presets with best times ("—" initially), Custom, Quit
- [ ] Beginner: first reveal never a mine; zero cells cascade open
- [ ] Numbers show classic colors; cursor visible; hjkl + arrows both work
- [ ] `f` cycles flag → ? → clear; mine counter tracks flags (goes negative when over-flagged)
- [ ] Chord: flag a number's mines, press space on the number — neighbors open; wrong flag + chord = loss
- [ ] Loss: all mines shown, wrong flags marked ✕, dead face, timer stops
- [ ] Win: remaining mines auto-flag, counter reads 000, win line; on a preset, best time saved
- [ ] Best time appears in menu after `m`; survives quit + relaunch
- [ ] Custom form: validation messages for bad input; valid board plays; no time recorded
- [ ] Expert fits a full-screen terminal; shrinking the window shows the too-small message
- [ ] `?` overlay opens/closes; `n` restarts; `q` quits cleanly (terminal restored)
- [ ] `~/Library/Application Support/mines/scores.json` exists and is valid JSON after a preset win

- [ ] **Step 3: Install on PATH**

```bash
go install
which mines && mines
```

Expected: `mines` resolves in `$GOPATH/bin` (or `$HOME/go/bin`) and launches.

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "chore: final verification pass for v1"
```

---

## Execution handoff

Plan complete. Execute with superpowers:subagent-driven-development (fresh subagent per task, review between tasks) or superpowers:executing-plans (inline with checkpoints). Tasks 2–8 are engine/store and strictly ordered; Tasks 9–14 build the UI bottom-up; Task 15 gates completion.





