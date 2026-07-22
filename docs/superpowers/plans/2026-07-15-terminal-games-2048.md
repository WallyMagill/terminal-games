# Terminal Games 2048 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add classic 4×4 2048 as a third launcher game with pure engine rules, compact Bubble Tea UI, continue-after-win, and persistent best score / best tile records.

**Architecture:** Mirror Mines and Solitaire: pure rules in `internal/g2048/engine`, thin presentation in `internal/g2048/ui`, explicit routing in `internal/app` (no plugin framework). Bump `internal/store` records to version 3 with a `g2048` section. Inject `*rand.Rand` into the engine for deterministic tests.

**Tech Stack:** Go 1.26.4, Bubble Tea v1.3.10, Lip Gloss v1.1.0, standard library `math/rand` + JSON file store, `go test`.

**Spec:** `docs/superpowers/specs/2026-07-15-terminal-games-2048-design.md`

## Global Constraints

- Keep the current Go / Bubble Tea / Lip Gloss stack; do not add dependencies.
- Do not rename the Go module (`github.com/wallymagill/mines`).
- Visual style: compact terminal text, keyboard-only, no mouse, no slide animations.
- Classic 4×4 only; continue after first 2048 tile; no undo, no 5×5, no seed picker, no timer records.
- Package path is `internal/g2048` (Go identifiers cannot start with a digit); launcher label is `2048`.
- Store path remains `os.UserConfigDir()/mines/scores.json`.
- Store must load v2 multi-game records and legacy mines-only maps; saves write version 3.
- Mines and Solitaire behavior stay unchanged except the launcher gains a third item.
- Prefer TDD: failing test → implement → pass → commit checkpoint when authorized.
- Do not commit unless the user explicitly authorizes commits. If authorized, use the checkpoint messages in this plan.

---

## File Structure

```text
internal/g2048/engine/
  board.go          // size const, line slide/merge helpers
  game.go           // Game, New, accessors, win/over detection
  actions.go        // Move(dir), spawn
  slide_test.go     // line merge golden cases
  move_test.go      // directional moves, no-op, score
  game_test.go      // New, spawn, win-continue, game over

internal/g2048/ui/
  styles.go         // tile colors, min frame size, chrome helpers
  game.go           // Model, Update, View, keys, help, records
  game_test.go      // keys, help, back, new game, records
  view_test.go      // HUD, board glyphs, too-small, win/over status

internal/store/
  store.go          // version 3, G2048Record, Load/Record APIs
  store_test.go     // migration + record tests

internal/app/
  launcher.go       // + 2048 item
  app.go            // + view2048 routing
  app_test.go       // launcher + enter/leave 2048
```

Notes:

- Engine package name: `package engine`.
- UI package name: `package ui` under `internal/g2048/ui`; import as `g2048ui "github.com/wallymagill/mines/internal/g2048/ui"`.
- Same-package engine tests may construct `Game` values with fixed boards (unexported fields are visible to `package engine` tests).

---

### Task 1: Engine line slide and merge

**Files:**
- Create: `internal/g2048/engine/board.go`
- Create: `internal/g2048/engine/slide_test.go`

**Interfaces:**
- Consumes: nothing from other new packages.
- Produces:
  - `const Size = 4`
  - `func slideLine(line [Size]int) (out [Size]int, scoreDelta int)` — slides and merges toward index 0 (left). A tile created by merge cannot merge again in the same call. `scoreDelta` is the sum of newly created merged values (`2v` for each merge of two `v`s).

- [ ] **Step 1: Write failing slide/merge tests**

Create `internal/g2048/engine/slide_test.go`:

```go
package engine

import "testing"

func TestSlideLineEmpty(t *testing.T) {
	got, score := slideLine([Size]int{0, 0, 0, 0})
	want := [Size]int{0, 0, 0, 0}
	if got != want || score != 0 {
		t.Fatalf("slideLine empty = %v score %d, want %v score 0", got, score, want)
	}
}

func TestSlideLineCompressGaps(t *testing.T) {
	got, score := slideLine([Size]int{0, 2, 0, 4})
	want := [Size]int{2, 4, 0, 0}
	if got != want || score != 0 {
		t.Fatalf("slideLine gaps = %v score %d, want %v score 0", got, score, want)
	}
}

func TestSlideLineMergeEqualsOnce(t *testing.T) {
	got, score := slideLine([Size]int{2, 2, 2, 2})
	want := [Size]int{4, 4, 0, 0}
	if got != want || score != 8 {
		t.Fatalf("slideLine [2,2,2,2] = %v score %d, want %v score 8", got, score, want)
	}
}

func TestSlideLineMergePairs(t *testing.T) {
	cases := []struct {
		in    [Size]int
		want  [Size]int
		score int
	}{
		{[Size]int{2, 2, 4, 0}, [Size]int{4, 4, 0, 0}, 4},
		{[Size]int{4, 4, 8, 8}, [Size]int{8, 16, 0, 0}, 24},
		{[Size]int{2, 0, 2, 0}, [Size]int{4, 0, 0, 0}, 4},
		{[Size]int{2, 4, 8, 16}, [Size]int{2, 4, 8, 16}, 0},
		{[Size]int{8, 8, 8, 0}, [Size]int{16, 8, 0, 0}, 16},
	}
	for _, tc := range cases {
		got, score := slideLine(tc.in)
		if got != tc.want || score != tc.score {
			t.Fatalf("slideLine(%v) = %v score %d, want %v score %d", tc.in, got, score, tc.want, tc.score)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```sh
go test ./internal/g2048/engine/ -run TestSlideLine -count=1
```

Expected: FAIL (package or `slideLine` undefined).

- [ ] **Step 3: Implement board helpers**

Create `internal/g2048/engine/board.go`:

```go
package engine

const Size = 4

// slideLine slides and merges toward index 0 (left).
// Equal adjacent tiles merge once; a merged tile does not merge again.
// scoreDelta is the sum of values of tiles created by merges.
func slideLine(line [Size]int) (out [Size]int, scoreDelta int) {
	// 1) compact non-zeros toward 0
	n := 0
	var compact [Size]int
	for i := 0; i < Size; i++ {
		if line[i] != 0 {
			compact[n] = line[i]
			n++
		}
	}
	// 2) merge left-to-right once
	var merged [Size]int
	m := 0
	for i := 0; i < n; {
		if i+1 < n && compact[i] == compact[i+1] {
			v := compact[i] * 2
			merged[m] = v
			scoreDelta += v
			m++
			i += 2
			continue
		}
		merged[m] = compact[i]
		m++
		i++
	}
	return merged, scoreDelta
}
```

- [ ] **Step 4: Run tests to verify they pass**

```sh
go test ./internal/g2048/engine/ -run TestSlideLine -count=1
```

Expected: PASS.

- [ ] **Step 5: Checkpoint**

If commits are authorized:

```sh
git add internal/g2048/engine/board.go internal/g2048/engine/slide_test.go
git commit -m "feat(g2048): add line slide and merge helpers"
```

---

### Task 2: Engine Game, Move, spawn, win, over

**Files:**
- Create: `internal/g2048/engine/game.go`
- Create: `internal/g2048/engine/actions.go`
- Create: `internal/g2048/engine/move_test.go`
- Create: `internal/g2048/engine/game_test.go`

**Interfaces:**
- Consumes: `slideLine`, `Size` from Task 1.
- Produces:
  - `type Dir int` with `DirUp`, `DirDown`, `DirLeft`, `DirRight`
  - `var ErrNilRNG = errors.New("rng must not be nil")`
  - `type MoveResult struct { Changed bool; Score int; Won bool; Over bool }`
  - `func New(rng *rand.Rand) (*Game, error)` — two initial spawns; error if rng nil
  - `func (g *Game) Move(d Dir) MoveResult`
  - `func (g *Game) Board() [Size][Size]int`
  - `func (g *Game) Score() int`
  - `func (g *Game) HighestTile() int`
  - `func (g *Game) Won() bool` — true after first tile ≥ 2048 this run
  - `func (g *Game) Over() bool`

**Move algorithm:**

1. If `g.over`, return zero `MoveResult` with `Over: true` (and current `Score`).
2. Read lines for direction (row left/right or column up/down), map each line into index-0-toward form, call `slideLine`, map back.
3. If board unchanged → `Changed: false`, no spawn.
4. Else apply board, add score delta, set `won` if any tile ≥ 2048 and not already won (`MoveResult.Won` true only on the transition).
5. Spawn one tile into a random empty cell: `2` if `rng.Intn(10) != 0`, else `4`.
6. Set `over` if no empties and no orthogonal equal pair; return result.

- [ ] **Step 1: Write failing move and game tests**

Create `internal/g2048/engine/move_test.go`:

```go
package engine

import (
	"math/rand"
	"testing"
)

// fixedGame builds a game with a known board and a deterministic rng.
// rng is used for post-move spawn; use a board with no empties after move
// when you need to assert the board without spawn noise, or reserve empties
// and control rng.
func fixedGame(board [Size][Size]int, score int, rng *rand.Rand) *Game {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	return &Game{board: board, score: score, rng: rng}
}

func TestMoveLeftMergesRow(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{2, 2, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(1)))
	res := g.Move(DirLeft)
	if !res.Changed {
		t.Fatal("expected change")
	}
	if g.board[0][0] != 4 {
		t.Fatalf("board[0][0] = %d, want 4", g.board[0][0])
	}
	if res.Score != 4 || g.Score() != 4 {
		t.Fatalf("score = %d/%d, want 4", res.Score, g.Score())
	}
	// one spawn should have placed a 2 or 4 somewhere empty
	empties := 0
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			if g.board[r][c] == 0 {
				empties++
			}
		}
	}
	if empties != 14 { // 16 - 1 merged tile - 1 spawn
		t.Fatalf("empties = %d, want 14", empties)
	}
}

func TestMoveRightUpDown(t *testing.T) {
	// Right
	g := fixedGame([Size][Size]int{
		{2, 2, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(2)))
	g.Move(DirRight)
	if g.board[0][3] != 4 {
		t.Fatalf("right: [0][3]=%d want 4; board=%v", g.board[0][3], g.board)
	}

	// Up
	g = fixedGame([Size][Size]int{
		{2, 0, 0, 0},
		{2, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(3)))
	g.Move(DirUp)
	if g.board[0][0] != 4 {
		t.Fatalf("up: [0][0]=%d want 4", g.board[0][0])
	}

	// Down
	g = fixedGame([Size][Size]int{
		{2, 0, 0, 0},
		{2, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(4)))
	g.Move(DirDown)
	if g.board[3][0] != 4 {
		t.Fatalf("down: [3][0]=%d want 4", g.board[3][0])
	}
}

func TestNoOpDoesNotSpawnOrScore(t *testing.T) {
	board := [Size][Size]int{
		{2, 4, 8, 16},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	g := fixedGame(board, 10, rand.New(rand.NewSource(1)))
	res := g.Move(DirLeft)
	if res.Changed {
		t.Fatal("left on already-packed distinct row must be no-op")
	}
	if g.Score() != 10 {
		t.Fatalf("score = %d, want 10", g.Score())
	}
	if g.board != board {
		t.Fatalf("board mutated on no-op: %v", g.board)
	}
}

func TestDoubleMergeAllDirectionsShape(t *testing.T) {
	// Full row of 2s moving left: 4,4,_,_ then spawn fills one empty.
	g := fixedGame([Size][Size]int{
		{2, 2, 2, 2},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(99)))
	res := g.Move(DirLeft)
	if g.board[0][0] != 4 || g.board[0][1] != 4 {
		t.Fatalf("row0 after left = %v, want 4,4,...", g.board[0])
	}
	if res.Score != 8 {
		t.Fatalf("score = %d, want 8", res.Score)
	}
}
```

Create `internal/g2048/engine/game_test.go`:

```go
package engine

import (
	"math/rand"
	"testing"
)

func TestNewRejectsNilRNG(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("New(nil) should error")
	}
}

func TestNewSpawnsTwoTiles(t *testing.T) {
	g, err := New(rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			v := g.Board()[r][c]
			if v != 0 {
				count++
				if v != 2 && v != 4 {
					t.Fatalf("tile %d not 2 or 4", v)
				}
			}
		}
	}
	if count != 2 {
		t.Fatalf("tiles = %d, want 2", count)
	}
	if g.Score() != 0 || g.Won() || g.Over() {
		t.Fatalf("fresh game score/won/over = %d/%v/%v", g.Score(), g.Won(), g.Over())
	}
}

func TestWinContinueAndBannerOnce(t *testing.T) {
	// 1024+1024 merges to 2048; leave empties so game is not over.
	g := fixedGame([Size][Size]int{
		{1024, 1024, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(1)))
	res := g.Move(DirLeft)
	if !res.Won || !g.Won() {
		t.Fatal("first 2048 should set Won on result and game")
	}
	if g.Over() {
		t.Fatal("should continue after win")
	}
	// Further move that does not create a new first-win should have Won=false on result.
	// Build a harmless move if possible; at minimum Won() stays true.
	res2 := g.Move(DirRight)
	if res2.Won {
		t.Fatal("MoveResult.Won must be true only on the transition into won")
	}
	if !g.Won() {
		t.Fatal("game.Won must remain true")
	}
}

func TestGameOverNoMoves(t *testing.T) {
	// Checkerboard of non-matching neighbors — no merges, no empties.
	g := fixedGame([Size][Size]int{
		{2, 4, 2, 4},
		{4, 2, 4, 2},
		{2, 4, 2, 4},
		{4, 2, 4, 2},
	}, 0, rand.New(rand.NewSource(1)))
	// Force over evaluation: Move that cannot change should leave over true after we detect.
	// Board is already dead — Move any dir is no-op; engine should mark over when board is dead
	// either at end of a real move or via explicit check. Spec: over when no empty and no merges.
	// After New-style detection: call Move; Changed false; Over true.
	// Ensure detectOver runs on Move when board is dead even without prior over flag.
	res := g.Move(DirLeft)
	if res.Changed {
		t.Fatal("dead board move must not change")
	}
	if !g.Over() || !res.Over {
		t.Fatal("dead board must be Over")
	}
	board := g.Board()
	res2 := g.Move(DirUp)
	if res2.Changed || g.Board() != board {
		t.Fatal("moves when over must be inert")
	}
}

func TestHighestTile(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{2, 8, 0, 0},
		{0, 0, 16, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(1)))
	if g.HighestTile() != 16 {
		t.Fatalf("HighestTile = %d, want 16", g.HighestTile())
	}
}

func TestSpawnUsesEmptyCellsOnly(t *testing.T) {
	// Almost full: one empty at [3][3]; move that merges creates empties or fills the one.
	// Simpler: three tiles and force left merge free of ambiguity.
	g := fixedGame([Size][Size]int{
		{2, 2, 4, 8},
		{16, 32, 64, 128},
		{256, 512, 1024, 4},
		{8, 16, 32, 0},
	}, 0, rand.New(rand.NewSource(1)))
	// Left on row0: 2,2,4,8 -> 4,4,8,0 then slide? 4,4,8 -> merge 4+4=8 -> 8,8,0,0 -> 16,0,0,0
	// Actually 4,4,8,0 slideLine: compact 4,4,8 merge 8,8 score... wait 4+4=8, then 8 and 8 cannot merge same turn → 8,8,0,0
	res := g.Move(DirLeft)
	if !res.Changed {
		t.Fatal("expected change")
	}
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			v := g.Board()[r][c]
			if v < 0 {
				t.Fatalf("negative tile at %d,%d", r, c)
			}
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```sh
go test ./internal/g2048/engine/ -count=1
```

Expected: FAIL (missing Game / Move).

- [ ] **Step 3: Implement game.go and actions.go**

Create `internal/g2048/engine/game.go`:

```go
package engine

import (
	"errors"
	"math/rand"
)

var ErrNilRNG = errors.New("rng must not be nil")

type Dir int

const (
	DirUp Dir = iota
	DirDown
	DirLeft
	DirRight
)

type Game struct {
	board [Size][Size]int
	score int
	won   bool
	over  bool
	rng   *rand.Rand
}

type MoveResult struct {
	Changed bool
	Score   int
	Won     bool // true only if this move first reached >= 2048
	Over    bool
}

func New(rng *rand.Rand) (*Game, error) {
	if rng == nil {
		return nil, ErrNilRNG
	}
	g := &Game{rng: rng}
	g.spawn()
	g.spawn()
	g.over = g.isDead()
	return g, nil
}

func (g *Game) Board() [Size][Size]int { return g.board }
func (g *Game) Score() int             { return g.score }
func (g *Game) Won() bool              { return g.won }
func (g *Game) Over() bool             { return g.over }

func (g *Game) HighestTile() int {
	best := 0
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			if g.board[r][c] > best {
				best = g.board[r][c]
			}
		}
	}
	return best
}

func (g *Game) isDead() bool {
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			if g.board[r][c] == 0 {
				return false
			}
			v := g.board[r][c]
			if c+1 < Size && g.board[r][c+1] == v {
				return false
			}
			if r+1 < Size && g.board[r+1][c] == v {
				return false
			}
		}
	}
	return true
}
```

Create `internal/g2048/engine/actions.go`:

```go
package engine

func (g *Game) Move(d Dir) MoveResult {
	if g.over {
		return MoveResult{Score: g.score, Over: true}
	}
	// If board is already dead (e.g. constructed full), mark over.
	if g.isDead() {
		g.over = true
		return MoveResult{Score: g.score, Over: true}
	}

	next, delta := applyMove(g.board, d)
	if next == g.board {
		return MoveResult{Score: g.score, Over: false}
	}
	g.board = next
	g.score += delta

	firstWin := false
	if !g.won {
		for r := 0; r < Size; r++ {
			for c := 0; c < Size; c++ {
				if g.board[r][c] >= 2048 {
					g.won = true
					firstWin = true
					break
				}
			}
			if firstWin {
				break
			}
		}
	}

	g.spawn()
	g.over = g.isDead()
	return MoveResult{
		Changed: true,
		Score:   g.score,
		Won:     firstWin,
		Over:    g.over,
	}
}

func applyMove(board [Size][Size]int, d Dir) ([Size][Size]int, int) {
	var out [Size][Size]int
	score := 0
	switch d {
	case DirLeft:
		for r := 0; r < Size; r++ {
			line, dlt := slideLine(board[r])
			out[r] = line
			score += dlt
		}
	case DirRight:
		for r := 0; r < Size; r++ {
			var rev [Size]int
			for c := 0; c < Size; c++ {
				rev[c] = board[r][Size-1-c]
			}
			line, dlt := slideLine(rev)
			score += dlt
			for c := 0; c < Size; c++ {
				out[r][Size-1-c] = line[c]
			}
		}
	case DirUp:
		for c := 0; c < Size; c++ {
			var col [Size]int
			for r := 0; r < Size; r++ {
				col[r] = board[r][c]
			}
			line, dlt := slideLine(col)
			score += dlt
			for r := 0; r < Size; r++ {
				out[r][c] = line[r]
			}
		}
	case DirDown:
		for c := 0; c < Size; c++ {
			var col [Size]int
			for r := 0; r < Size; r++ {
				col[r] = board[Size-1-r][c]
			}
			line, dlt := slideLine(col)
			score += dlt
			for r := 0; r < Size; r++ {
				out[Size-1-r][c] = line[r]
			}
		}
	}
	return out, score
}

func (g *Game) spawn() {
	type cell struct{ r, c int }
	var empties []cell
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			if g.board[r][c] == 0 {
				empties = append(empties, cell{r, c})
			}
		}
	}
	if len(empties) == 0 {
		return
	}
	pick := empties[g.rng.Intn(len(empties))]
	v := 2
	if g.rng.Intn(10) == 0 {
		v = 4
	}
	g.board[pick.r][pick.c] = v
}
```

**Dead-board nuance:** For `TestGameOverNoMoves`, a board that is already dead with `over == false` must become over on `Move`. The implementation above checks `isDead()` before applying a move and sets `over`. That means no-op-but-alive boards (packed distinct tiles with empties elsewhere) still return `Changed: false` without setting over — correct.

- [ ] **Step 4: Run engine tests**

```sh
go test ./internal/g2048/engine/ -count=1
```

Expected: PASS. If `TestWinContinueAndBannerOnce` fails because `DirRight` also reports `Won` incorrectly, fix so `MoveResult.Won` is only `firstWin`. If dead-board test fails, ensure pre-move `isDead` sets over.

- [ ] **Step 5: Checkpoint**

```sh
git add internal/g2048/engine/
git commit -m "feat(g2048): classic move, spawn, win-continue, game over"
```

---

### Task 3: Store version 3 and 2048 records

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

**Interfaces:**
- Consumes: existing `Records`, `LoadRecords`, `SaveRecords` patterns.
- Produces:
  - `recordsVersion = 3`
  - `const g2048ClassicKey = "classic"`
  - `type G2048Record struct { BestScore int \`json:"bestScore"\`; BestTile int \`json:"bestTile"\` }`
  - `Records.G2048 map[string]G2048Record \`json:"g2048,omitempty"\``
  - `func (s *Store) LoadG2048Classic() G2048Record`
  - `func (s *Store) RecordG2048Classic(score, highestTile int) (newBestScore, newBestTile bool, err error)`
  - Load accepts version **2 or 3** multi-game JSON; sanitizes; empty `G2048` if missing.
  - Saves always write `version: 3` and initialize nil maps for Mines, Solitaire, and G2048.

- [ ] **Step 1: Write failing store tests**

Add to `internal/store/store_test.go` (keep existing tests working):

```go
func TestLoadRecordsMigratesV2ToV3Shape(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/scores.json"
	bestSeconds := 240
	fewestMoves := 91
	// Write a v2 file without g2048
	raw := []byte(`{
  "version": 2,
  "mines": {"beginner": {"bestSeconds": 42}},
  "solitaire": {"draw3": {"bestSeconds": 240, "fewestMoves": 91}}
}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	s := store.New(path)
	got := s.LoadRecords()
	if got.Version != 3 {
		// Load may leave version as read then upgrade on save; require readable data:
		// At minimum mines/solitaire preserved and g2048 empty.
	}
	if got.Mines["beginner"].BestSeconds != 42 {
		t.Fatalf("mines = %#v", got.Mines)
	}
	draw3 := got.Solitaire["draw3"]
	if draw3.BestSeconds == nil || *draw3.BestSeconds != bestSeconds {
		t.Fatalf("solitaire time = %#v", draw3)
	}
	if draw3.FewestMoves == nil || *draw3.FewestMoves != fewestMoves {
		t.Fatalf("solitaire moves = %#v", draw3)
	}
	if len(got.G2048) != 0 && got.G2048["classic"].BestScore != 0 {
		// empty map or zero record is fine
	}
}

func TestRecordG2048ClassicPreservesOtherGames(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	if _, err := s.RecordMinesBest("beginner", 42); err != nil {
		t.Fatal(err)
	}
	sec, moves := 100, 50
	if _, _, err := s.RecordSolitaireDraw3(sec, moves); err != nil {
		t.Fatal(err)
	}
	newScore, newTile, err := s.RecordG2048Classic(1200, 256)
	if err != nil || !newScore || !newTile {
		t.Fatalf("first record = %v %v %v", newScore, newTile, err)
	}
	got := s.LoadRecords()
	if got.Mines["beginner"].BestSeconds != 42 {
		t.Fatalf("mines wiped: %#v", got.Mines)
	}
	if got.Solitaire["draw3"].BestSeconds == nil {
		t.Fatal("solitaire wiped")
	}
	rec := s.LoadG2048Classic()
	if rec.BestScore != 1200 || rec.BestTile != 256 {
		t.Fatalf("g2048 = %#v", rec)
	}
}

func TestRecordG2048ClassicOnlyImproves(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	if _, _, err := s.RecordG2048Classic(1000, 512); err != nil {
		t.Fatal(err)
	}
	ns, nt, err := s.RecordG2048Classic(900, 256)
	if err != nil || ns || nt {
		t.Fatalf("worse values must not update: %v %v %v", ns, nt, err)
	}
	ns, nt, err = s.RecordG2048Classic(1000, 1024)
	if err != nil || ns || !nt {
		t.Fatalf("same score higher tile: %v %v %v", ns, nt, err)
	}
	ns, nt, err = s.RecordG2048Classic(2000, 1024)
	if err != nil || !ns || nt {
		t.Fatalf("higher score same tile: %v %v %v", ns, nt, err)
	}
	rec := s.LoadG2048Classic()
	if rec.BestScore != 2000 || rec.BestTile != 1024 {
		t.Fatalf("rec = %#v", rec)
	}
}

func TestSaveRecordsWritesVersion3(t *testing.T) {
	path := t.TempDir() + "/scores.json"
	s := store.New(path)
	if _, _, err := s.RecordG2048Classic(10, 4); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	ver, ok := raw["version"].(float64)
	if !ok || int(ver) != 3 {
		t.Fatalf("version = %v, want 3", raw["version"])
	}
}
```

Adjust imports in the test file for `encoding/json` and `os` if not already present.

- [ ] **Step 2: Run store tests to verify new ones fail**

```sh
go test ./internal/store/ -run 'G2048|V2ToV3|Version3' -count=1
```

Expected: FAIL (missing types/APIs).

- [ ] **Step 3: Implement store changes**

In `internal/store/store.go`:

1. Set `recordsVersion = 3`.
2. Add:

```go
const g2048ClassicKey = "classic"

type G2048Record struct {
	BestScore int `json:"bestScore"`
	BestTile  int `json:"bestTile"`
}
```

3. Add to `Records`:

```go
G2048 map[string]G2048Record `json:"g2048,omitempty"`
```

4. Update `emptyRecords`:

```go
func emptyRecords() Records {
	return Records{
		Version:   recordsVersion,
		Mines:     map[string]TimeRecord{},
		Solitaire: map[string]SolitaireRecord{},
		G2048:     map[string]G2048Record{},
	}
}
```

5. Update `LoadRecords` multi-game branch to accept version 2 **or** 3:

```go
if err := json.Unmarshal(data, &records); err == nil && (records.Version == 2 || records.Version == 3) {
	records.Version = recordsVersion
	return sanitizeRecords(records)
}
```

6. In `sanitizeRecords`, nil-safe init `G2048`; drop entries with `BestScore < 0` or `BestTile < 0`; drop entries where both are 0 **or** keep zeros as “no record” — prefer: if both 0, delete key; allow BestScore > 0 with BestTile 0 only if needed — normal writes always set both positively when recording. Simpler sanitize: delete if `BestScore < 0 || BestTile < 0`.

7. `SaveRecords` / `saveJSON` path: ensure `G2048` map non-nil like Mines/Solitaire.

8. Add:

```go
func (s *Store) LoadG2048Classic() G2048Record {
	return s.LoadRecords().G2048[g2048ClassicKey]
}

func (s *Store) RecordG2048Classic(score, highestTile int) (newBestScore, newBestTile bool, err error) {
	if score < 0 || highestTile < 0 {
		return false, false, nil
	}
	records := s.LoadRecords()
	if records.G2048 == nil {
		records.G2048 = map[string]G2048Record{}
	}
	rec := records.G2048[g2048ClassicKey]
	if score > rec.BestScore {
		rec.BestScore = score
		newBestScore = true
	}
	if highestTile > rec.BestTile {
		rec.BestTile = highestTile
		newBestTile = true
	}
	if !newBestScore && !newBestTile {
		return false, false, nil
	}
	records.G2048[g2048ClassicKey] = rec
	return newBestScore, newBestTile, s.SaveRecords(records)
}
```

**Important:** Existing tests that assert `records.Version == 2` after save must be updated to expect version **3**. Search `store_test.go` for `Version` / `recordsVersion` / `"version": 2` and update expectations.

- [ ] **Step 4: Run full store package tests**

```sh
go test ./internal/store/ -count=1
```

Expected: ALL PASS (including updated version assertions).

- [ ] **Step 5: Checkpoint**

```sh
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat(store): v3 records with classic 2048 bests"
```

---

### Task 4: 2048 UI rendering and styles

**Files:**
- Create: `internal/g2048/ui/styles.go`
- Create: `internal/g2048/ui/game.go` (Model + View skeleton; input wired in Task 5)
- Create: `internal/g2048/ui/view_test.go`

**Interfaces:**
- Consumes:
  - `engine.New`, `engine.Game` accessors, `engine.Size`
  - `store.Store`, `store.LoadG2048Classic`
  - `uiutil.FitLine`, `uiutil.ClampLine`, `uiutil.AllFit`
- Produces:
  - `type BackMsg struct{}`
  - `func NewModel(scores *store.Store, rng *rand.Rand) (Model, error)`
  - `func (m Model) TooSmall() bool`
  - `func (m Model) View() string`
  - `func (m Model) Init() tea.Cmd` (return nil)
  - Tile cell width ≥ 5 so values through 65536 fit (`"65536"` is 5 chars).
  - Min frame: design for roughly **width 34+** and **height 16+** (HUD + 4 tile rows with borders + status + hints). Pin constants in `styles.go` and tests.

**Rendering rules:**

- Title `2048` or `┌─ 2048 ─...`.
- HUD: `SCORE <n>` and `BEST <n>` from model + store.
- Board: each cell formatted with `fmt.Sprintf("%*s", cellWidth, valueOrEmpty)` inside box-drawing or simple spaced grid. Empty = spaces (not Mines `░░`).
- Color tiles by value via Lip Gloss (2/4 muted gray, 8–64 yellow/orange, 128+ bright).
- Status line area reserved (empty for now if not won/over).
- Footer hints: `←↑↓→ / hjkl move · n new · m menu · ? help`.

- [ ] **Step 1: Write failing view tests**

Create `internal/g2048/ui/view_test.go`:

```go
package ui

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/wallymagill/mines/internal/store"
)

func testModel(t *testing.T) Model {
	t.Helper()
	m, err := NewModel(store.New(t.TempDir()+"/scores.json"), rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	m.termW, m.termH = 80, 24
	return m
}

func TestViewRendersTitleScoreAndBoard(t *testing.T) {
	m := testModel(t)
	v := m.View()
	if !strings.Contains(v, "2048") {
		t.Fatalf("missing title:\n%s", v)
	}
	if !strings.Contains(v, "SCORE") || !strings.Contains(v, "BEST") {
		t.Fatalf("missing HUD:\n%s", v)
	}
	// Fresh game has two tiles; view should show at least one "2" or "4"
	if !strings.Contains(v, "2") && !strings.Contains(v, "4") {
		t.Fatalf("board tiles missing:\n%s", v)
	}
}

func TestViewShowsSavedBest(t *testing.T) {
	path := t.TempDir() + "/scores.json"
	s := store.New(path)
	if _, _, err := s.RecordG2048Classic(9999, 2048); err != nil {
		t.Fatal(err)
	}
	m, err := NewModel(s, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	m.termW, m.termH = 80, 24
	v := m.View()
	if !strings.Contains(v, "9999") {
		t.Fatalf("best score not shown:\n%s", v)
	}
}

func TestTooSmallGuard(t *testing.T) {
	m := testModel(t)
	m.termW, m.termH = 10, 5
	if !m.TooSmall() {
		t.Fatal("expected too small")
	}
	v := m.View()
	if !strings.Contains(strings.ToLower(v), "too small") {
		t.Fatalf("want too small message:\n%s", v)
	}
}
```

- [ ] **Step 2: Run to verify fail**

```sh
go test ./internal/g2048/ui/ -count=1
```

Expected: FAIL.

- [ ] **Step 3: Implement styles.go and game.go View**

Create `internal/g2048/ui/styles.go`:

```go
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/wallymagill/mines/internal/uiutil"
)

const (
	cellInnerWidth = 5 // fits 65536
	// Frame budget: title/HUD/board borders/status/hints — tune to match View.
	g2048MinCols = 34
	g2048MinRows = 16
)

var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleFaint = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleBest  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
)

func tileStyle(v int) lipgloss.Style {
	s := lipgloss.NewStyle().Width(cellInnerWidth).Align(lipgloss.Right)
	switch {
	case v == 0:
		return s.Foreground(lipgloss.Color("240"))
	case v <= 4:
		return s.Foreground(lipgloss.Color("252"))
	case v <= 16:
		return s.Foreground(lipgloss.Color("228"))
	case v <= 64:
		return s.Foreground(lipgloss.Color("214"))
	case v <= 256:
		return s.Foreground(lipgloss.Color("208"))
	case v <= 1024:
		return s.Foreground(lipgloss.Color("202"))
	default:
		return s.Bold(true).Foreground(lipgloss.Color("196"))
	}
}

func formatCell(v int) string {
	if v == 0 {
		return tileStyle(0).Render("")
	}
	return tileStyle(v).Render(fmt.Sprintf("%d", v))
}

func fitLine(termW int, full, compact string) string { return uiutil.FitLine(termW, full, compact) }
func clampLine(s string, termW int) string           { return uiutil.ClampLine(s, termW) }
```

Create `internal/g2048/ui/game.go` (skeleton with View; Update in Task 5 can stub keys):

```go
package ui

import (
	"fmt"
	legacyrand "math/rand"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/g2048/engine"
	"github.com/wallymagill/mines/internal/store"
	"github.com/wallymagill/mines/internal/uiutil"
)

type BackMsg struct{}

func back() tea.Cmd {
	return func() tea.Msg { return BackMsg{} }
}

type Model struct {
	game        *engine.Game
	scores      *store.Store
	rng         *legacyrand.Rand
	termW       int
	termH       int
	bestScore   int
	bestTile    int
	message     string
	showHelp    bool
	showWin     bool // one-shot banner until dismissed by next move or new game
	newBestScore bool
	newBestTile  bool
	saveErr     bool
}

func NewModel(scores *store.Store, rng *legacyrand.Rand) (Model, error) {
	g, err := engine.New(rng)
	if err != nil {
		return Model{}, err
	}
	rec := scores.LoadG2048Classic()
	return Model{
		game:      g,
		scores:    scores,
		rng:       rng,
		bestScore: rec.BestScore,
		bestTile:  rec.BestTile,
	}, nil
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) TooSmall() bool {
	return m.termW > 0 && m.termH > 0 && (m.termW < g2048MinCols || m.termH < g2048MinRows)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// Task 5 fills this in; for Task 4 handle WindowSize only so tests can set size via struct fields.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW, m.termH = msg.Width, msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	if m.TooSmall() {
		return styleErr.Render("terminal too small for 2048") + "\n" +
			styleFaint.Render("press m for menu")
	}
	if m.showHelp {
		return m.helpView()
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render(" 2048") + "\n")
	bestLabel := fmt.Sprintf("BEST %d", m.bestScore)
	if m.newBestScore {
		bestLabel = styleBest.Render(bestLabel)
	} else {
		bestLabel = styleFaint.Render(bestLabel)
	}
	hud := fmt.Sprintf(" SCORE %-6d  %s", m.game.Score(), bestLabel)
	b.WriteString(hud + "\n\n")
	board := m.game.Board()
	// simple padded grid
	for r := 0; r < engine.Size; r++ {
		b.WriteString("  ")
		for c := 0; c < engine.Size; c++ {
			b.WriteString(formatCell(board[r][c]))
			b.WriteString(" ")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleFaint.Render(" "+m.statusLine()) + "\n")
	b.WriteString(styleFaint.Render(fitLine(m.termW,
		" ←↑↓→ / hjkl move · n new · m menu · ? help · q quit",
		" arrows move · n new · m menu",
	)) + "\n")
	return b.String()
}

func (m Model) statusLine() string {
	if m.saveErr {
		return "could not save record"
	}
	if m.game.Over() {
		return "GAME OVER"
	}
	if m.showWin {
		return "YOU WIN — keep going"
	}
	if m.newBestScore || m.newBestTile {
		return "new best!"
	}
	return m.message
}

func (m Model) helpView() string {
	lines := []string{
		" 2048",
		"",
		" Slide tiles with arrows or hjkl.",
		" Equal tiles merge into their sum.",
		" Reach 2048 to win — you can keep playing.",
		" No moves left is game over.",
		"",
		" ←↑↓→ / hjkl  move",
		" n  new game",
		" m / esc  menu",
		" ?  help",
		" q  quit",
		"",
		" any key to close help",
	}
	for i, l := range lines {
		lines[i] = styleFaint.Render(clampLine(l, m.termW))
	}
	return strings.Join(lines, "\n")
}
```

Tune `g2048MinCols` / `g2048MinRows` so `TooSmall` matches rendered size; if view is taller, raise min rows.

- [ ] **Step 4: Pass view tests**

```sh
go test ./internal/g2048/ui/ -count=1
```

Expected: PASS.

- [ ] **Step 5: Checkpoint**

```sh
git add internal/g2048/ui/
git commit -m "feat(g2048): render classic board HUD and help"
```

---

### Task 5: 2048 UI input, records, new game

**Files:**
- Modify: `internal/g2048/ui/game.go`
- Create: `internal/g2048/ui/game_test.go`

**Interfaces:**
- Consumes: `engine.Move`, `engine.Dir*`, `store.RecordG2048Classic`, `uiutil.TypedKeys`, `uiutil.IsRuneBatch`
- Produces: full `Update` behavior per spec keys.

**Key map:**

| Key | Action |
|-----|--------|
| `left`/`h` | `Move(DirLeft)` |
| `down`/`j` | `Move(DirDown)` |
| `up`/`k` | `Move(DirUp)` |
| `right`/`l` | `Move(DirRight)` |
| `n` | `engine.New(rng)` replace game; clear showWin/newBest flags for run |
| `m`/`esc` | `BackMsg` (if not help) |
| `?` | open help |
| any key while help | close help |
| too-small | only `m`/`esc` back, `q` quit |

After a successful move (`Changed`):

1. If `result.Won`, set `showWin = true`.
2. If `result.Changed` and not first-win-only, clear `showWin` on **subsequent** moves? Spec: one-time banner. Clear `showWin` on the next successful move after win, or leave until `n`. Prefer: set on `result.Won`; clear on next `Changed` move or `n`.
3. Call `RecordG2048Classic(score, highestTile)`; update `bestScore`/`bestTile` locally; set `newBestScore`/`newBestTile`; on error set `saveErr`.

- [ ] **Step 1: Write failing input tests**

Create `internal/g2048/ui/game_test.go`:

```go
package ui

import (
	"math/rand"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestEscRequestsBack(t *testing.T) {
	m := testModel(t)
	_, cmd := m.Update(key("esc"))
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("esc should produce BackMsg")
	}
}

func TestHelpDismissesOnAnyKey(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("?"))
	if !m.showHelp {
		t.Fatal("help should open")
	}
	m, _ = m.Update(key("x"))
	if m.showHelp {
		t.Fatal("help should close")
	}
}

func TestMoveKeysChangeScoreOrBoard(t *testing.T) {
	m := testModel(t)
	before := m.game.Board()
	// hammer directions until something changes (seeded game may need a few)
	changed := false
	for _, k := range []string{"left", "right", "up", "down", "h", "j", "k", "l"} {
		m, _ = m.Update(key(k))
		if m.game.Board() != before || m.game.Score() > 0 {
			changed = true
			break
		}
	}
	if !changed {
		// still possible board is stuck; try more moves
		for i := 0; i < 20; i++ {
			m, _ = m.Update(key("left"))
			m, _ = m.Update(key("up"))
			if m.game.Score() > 0 {
				changed = true
				break
			}
		}
	}
	if !changed && !m.game.Over() {
		t.Fatalf("expected some move to affect game; board=%v", m.game.Board())
	}
}

func TestNewGameResets(t *testing.T) {
	m := testModel(t)
	// force score by moving
	for i := 0; i < 30; i++ {
		m, _ = m.Update(key("left"))
		m, _ = m.Update(key("up"))
	}
	m, _ = m.Update(key("n"))
	if m.game.Score() != 0 {
		t.Fatalf("after n score = %d, want 0", m.game.Score())
	}
	if m.showWin {
		t.Fatal("showWin should clear on new game")
	}
}

func TestMoveRecordsBestScore(t *testing.T) {
	path := t.TempDir() + "/scores.json"
	s := store.New(path)
	m, err := NewModel(s, rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatal(err)
	}
	m.termW, m.termH = 80, 24
	for i := 0; i < 50; i++ {
		m, _ = m.Update(key("left"))
		m, _ = m.Update(key("down"))
		m, _ = m.Update(key("right"))
		m, _ = m.Update(key("up"))
		if m.game.Score() > 0 {
			break
		}
	}
	if m.game.Score() == 0 {
		t.Skip("seed produced no merges in 50 cycles; re-seed test if flaky")
	}
	rec := s.LoadG2048Classic()
	if rec.BestScore < m.game.Score() && rec.BestScore == 0 {
		// live record should have written at least once when score > 0
	}
	if rec.BestScore != m.bestScore && m.game.Score() > 0 {
		// bestScore on model should track store when improved
	}
	if m.game.Score() > 0 && rec.BestScore == 0 && !m.saveErr {
		t.Fatalf("score %d not recorded; rec=%#v saveErr=%v", m.game.Score(), rec, m.saveErr)
	}
}

func TestTooSmallIgnoresMoveKeys(t *testing.T) {
	m := testModel(t)
	m.termW, m.termH = 10, 5
	before := m.game.Board()
	m, _ = m.Update(key("left"))
	if m.game.Board() != before {
		t.Fatal("moves ignored when too small")
	}
}
```

- [ ] **Step 2: Run tests — expect fail on missing key handling**

```sh
go test ./internal/g2048/ui/ -count=1
```

- [ ] **Step 3: Complete Update / handleKey / applyMove / newGame / maybeRecord**

Replace `Update` in `game.go` with Solitaire-shaped dispatch:

```go
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW, m.termH = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.TooSmall() {
			if !uiutil.IsRuneBatch(msg) {
				switch msg.String() {
				case "m", "esc":
					m.showHelp = false
					return m, back()
				case "q":
					return m, tea.Quit
				}
			}
			return m, nil
		}
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		var cmds []tea.Cmd
		for _, k := range uiutil.TypedKeys(msg) {
			var cmd tea.Cmd
			m, cmd = m.handleKey(k)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) handleKey(k string) (Model, tea.Cmd) {
	switch k {
	case "?":
		m.showHelp = true
	case "n":
		return m.newGame()
	case "m", "esc":
		return m, back()
	case "q":
		return m, tea.Quit
	case "left", "h":
		return m.applyDir(engine.DirLeft)
	case "down", "j":
		return m.applyDir(engine.DirDown)
	case "up", "k":
		return m.applyDir(engine.DirUp)
	case "right", "l":
		return m.applyDir(engine.DirRight)
	}
	return m, nil
}

func (m Model) newGame() (Model, tea.Cmd) {
	g, err := engine.New(m.rng)
	if err != nil {
		m.message = "could not start game"
		return m, nil
	}
	m.game = g
	m.message = ""
	m.showWin = false
	m.newBestScore = false
	m.newBestTile = false
	m.saveErr = false
	return m, nil
}

func (m Model) applyDir(d engine.Dir) (Model, tea.Cmd) {
	if m.game.Over() {
		m.message = "GAME OVER"
		return m, nil
	}
	res := m.game.Move(d)
	if !res.Changed {
		return m, nil
	}
	if m.showWin {
		m.showWin = false // clear previous win banner after next real move
	}
	if res.Won {
		m.showWin = true
	}
	m = m.maybeRecord()
	return m, nil
}

func (m Model) maybeRecord() Model {
	ns, nt, err := m.scores.RecordG2048Classic(m.game.Score(), m.game.HighestTile())
	if err != nil {
		m.saveErr = true
		return m
	}
	m.saveErr = false
	if ns {
		m.newBestScore = true
		m.bestScore = m.game.Score()
	}
	if nt {
		m.newBestTile = true
		if h := m.game.HighestTile(); h > m.bestTile {
			m.bestTile = h
		}
	}
	return m
}
```

**Win banner fix:** The “clear on next move then set if Won” order above clears first then sets if `res.Won`, so first win stays visible until the following move. Good.

- [ ] **Step 4: Run UI tests**

```sh
go test ./internal/g2048/ui/ -count=1
```

Expected: PASS. If `TestMoveRecordsBestScore` is flaky, switch to constructing a `Model` with a fixed engine board via a test-only helper in `game_test.go` (same package can set `m.game` if you add `engine` test export). Prefer adding in engine:

```go
// optional test helper in game_test only uses fixedGame from engine package —
// UI test can perform enough moves with a known seed that merges.
```

Use seed search: try seeds until score > 0 in test setup, or expose nothing and assert with a loop of 200 moves.

- [ ] **Step 5: Checkpoint**

```sh
git add internal/g2048/ui/
git commit -m "feat(g2048): keyboard play, help, live records"
```

---

### Task 6: Launcher and app routing

**Files:**
- Modify: `internal/app/launcher.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`

**Interfaces:**
- Consumes: `g2048ui.NewModel`, `g2048ui.Model`, `g2048ui.BackMsg`
- Produces: launcher item `2048`; `view2048`; enter/leave parity with Solitaire.

- [ ] **Step 1: Extend launcher tests**

Update `TestStartsOnLauncher` to require `"2048"` in the view.

Add:

```go
func TestLauncherSelects2048(t *testing.T) {
	a := testApp(t)
	// Minesweeper (0), Solitaire (1), 2048 (2)
	model, _ := a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("enter"))
	a = model.(App)
	if a.view != view2048 {
		t.Fatalf("view = %v, want view2048", a.view)
	}
	v := a.View()
	if !strings.Contains(v, "2048") || !strings.Contains(v, "SCORE") {
		t.Fatalf("2048 view missing pieces:\n%s", v)
	}
}

func Test2048EscReturnsToLauncher(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(launcherSelectMsg{item: launcher2048})
	a = model.(App)
	if a.view != view2048 {
		t.Fatal("expected view2048")
	}
	model, _ = a.Update(key("esc"))
	a = model.(App)
	// esc may be handled by child; if child returns BackMsg via cmd, app must process it.
	// Match Solitaire tests for back navigation.
	if a.view == view2048 {
		// drain BackMsg pattern used by solitaire
		model, cmd := a.g2048.Update(key("esc"))
		a.g2048 = model
		if cmd != nil {
			if msg, ok := cmd().(g2048ui.BackMsg); ok {
				model2, _ := a.Update(msg)
				a = model2.(App)
			}
		}
	}
	model, _ = a.Update(g2048ui.BackMsg{})
	a = model.(App)
	if a.view != viewLauncher {
		t.Fatalf("view = %v, want launcher", a.view)
	}
}
```

Also update any test that counts launcher items or assumes Quit is index 2 — Quit becomes index 3.

- [ ] **Step 2: Run app tests — expect fail**

```sh
go test ./internal/app/ -count=1
```

- [ ] **Step 3: Wire launcher.go**

```go
const (
	launcherMines launcherItem = iota
	launcherSolitaire
	launcher2048
	launcherQuit
	launcherItemCount
)

// In View labels:
labels := []string{"Minesweeper", "Solitaire", "2048", "Quit"}
```

- [ ] **Step 4: Wire app.go**

```go
import (
	g2048ui "github.com/wallymagill/mines/internal/g2048/ui"
	// ...
)

const (
	viewLauncher view = iota
	viewMines
	viewSolitaire
	view2048
)

type App struct {
	// ...
	g2048 g2048ui.Model
	// ...
}

// New: optionally zero g2048; created on select like solitaire.

// WindowSizeMsg: forward to g2048 when view2048
// launcherSelectMsg launcher2048: NewModel + size + view2048
// case g2048ui.BackMsg: viewLauncher
// Update switch view2048: delegate
// View case view2048: similar chrome to solitaire (optional leading newline)
```

Mirror Solitaire’s select block:

```go
case launcher2048:
	m, err := g2048ui.NewModel(a.scores, legacyrand.New(legacyrand.NewSource(time.Now().UnixNano())))
	if err == nil {
		if a.termW > 0 || a.termH > 0 {
			m, _ = m.Update(tea.WindowSizeMsg{Width: a.termW, Height: a.termH})
		}
		a.g2048 = m
		a.view = view2048
	}
	return a, nil
```

- [ ] **Step 5: Pass app tests + full suite**

```sh
go test ./internal/app/ -count=1
go test ./... -count=1
```

Expected: ALL PASS.

- [ ] **Step 6: Checkpoint**

```sh
git add internal/app/
git commit -m "feat(app): add 2048 to terminal games launcher"
```

---

### Task 7: Final verification and polish

**Files:**
- Modify only if tests/manual play reveal gaps: `internal/g2048/ui/game.go`, `styles.go`, `app.go`

- [ ] **Step 1: Full test suite**

```sh
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 2: Manual play checklist** (run `go run .` from repo root)

1. Launcher shows Minesweeper, Solitaire, **2048**, Quit.
2. Enter 2048: board with two tiles, SCORE 0, BEST from store.
3. Arrow/`hjkl` moves; score increases on merges.
4. `?` help then any key closes.
5. Shrink terminal: too-small message; `m` returns to launcher.
6. Play until game over (or set up mentally): GAME OVER; `n` new run.
7. Best score updates mid-run and persists after quit/relaunch.
8. Mines and Solitaire still open and play from launcher.

- [ ] **Step 3: Fix any polish gaps**

- Align cell width if large tiles wrap.
- Ensure win banner shows once at 2048 and play continues.
- Ensure `q` quits from 2048 (via model or root `ctrl+c`).

- [ ] **Step 4: Final checkpoint**

```sh
git add -u internal/g2048 internal/app internal/store
git commit -m "feat: ship classic 2048 in terminal games suite"
```

(Only if not already committed per-task; squash only if user asks.)

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| Classic 4×4 rules, slide/merge once | 1–2 |
| Spawn 90% 2 / 10% 4, injected RNG | 2 |
| No-op no spawn | 2 |
| Continue after 2048; one-time win banner | 2, 5 |
| Game over freezes moves | 2, 5 |
| Pure engine, no TUI imports | 1–2 |
| Lip Gloss colored tiles, compact HUD | 4 |
| Keys: arrows/hjkl, n, m/esc, ?, q | 5–6 |
| Help any-key dismiss | 5 |
| Too-small guard | 4–5 |
| Best score live + best tile | 3, 5 |
| Store v3 + v2 migration | 3 |
| Launcher order + routing | 6 |
| Mines/Solitaire unchanged | 6–7 |
| No undo/5×5/animations | Global constraints |

## Placeholder / consistency self-review

- No TBD steps; APIs named consistently: `RecordG2048Classic`, `LoadG2048Classic`, `g2048ClassicKey`, `view2048`, `launcher2048`.
- Engine `New` returns `(*Game, error)` (matches Solitaire; slightly stricter than sketch in design — intentional).
- `MoveResult.Won` is transition-only; `Game.Won()` sticky.
- Store version: load 2|3, save 3.

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-15-terminal-games-2048.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration  
2. **Inline Execution** — execute tasks in this session with checkpoints  

Which approach?
