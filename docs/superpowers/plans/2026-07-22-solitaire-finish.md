# Solitaire Finishing Pass Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove hints, rename the top piles to Deck/Draw, show a single draw card, complete vim-style navigation with jump keys, and add productive-move dead-end detection with a game-over screen.

**Architecture:** Dead-end detection is pure engine code (`internal/solitaire/engine/deadend.go`) built on the existing `LegalMoves()`/`Clone()`/`Draw()`. Everything else is UI (`internal/solitaire/ui`): deletions in `assist.go`, label/layout edits in `game.go`'s views, key handling in `handleKey`, and a new `gameOverView()` branch. No store, app, mines, or g2048 changes.

**Tech Stack:** Go 1.26.4, Bubble Tea v1.3.10, Lip Gloss v1.1.0, `go test`.

**Spec:** `docs/superpowers/specs/2026-07-22-solitaire-finish-design.md`

## Global Constraints

- No new dependencies; module stays `github.com/wallymagill/mines`.
- Labels: face-down pile = `Deck`, face-up pile = `Draw`; location strings `"deck"` / `"draw pile"`.
- Draw pile renders only its top card (one slot). Selection semantics unchanged.
- `solitaireMinCols` (48), `solitaireMinRows` (18), and `TooSmall()` unchanged.
- Losses are never recorded to the store; win recording unchanged.
- Assisted records: `"undo"` remains the only assist reason after hint removal.
- Mines, 2048, store, and app packages untouched.
- TDD: failing test → implement → pass → commit. Commit messages end with the trailer `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

**Existing test helpers in `internal/solitaire/ui` (do not redefine):** `testModel(t *testing.T) Model`, `testModelWithStore(t, scores)`, `key(s string) tea.KeyMsg`, `setUnexported(field reflect.Value, value any)` — all in `game_test.go`.

---

## File Structure

```text
internal/solitaire/engine/
  deadend.go         // Create: productive(), ProductiveMoves(), DeadEnd()   (Task 1)
  deadend_test.go    // Create                                              (Task 1)

internal/solitaire/ui/
  assist.go          // Modify: delete hint funcs (T2); add destination sort (T4)
  game.go            // Modify: keys (T2, T4, T5), labels+topView (T3), game-over view (T5)
  game_test.go       // Modify: T2, T4, T5
  view_test.go       // Modify: T2, T3, T5
```

---

### Task 1: Engine dead-end detection

**Files:**
- Create: `internal/solitaire/engine/deadend.go`
- Create: `internal/solitaire/engine/deadend_test.go`

**Interfaces:**
- Consumes: `Game` unexported fields (same-package), `LegalMoves()`, `Clone()`, `Draw()`, `Move`, `Location`, `Card`, `TableauCard`.
- Produces (Task 5 depends on these exact names):
  - `func (g *Game) ProductiveMoves() []Move`
  - `func (g *Game) DeadEnd() bool`

- [ ] **Step 1: Write the failing tests**

Create `internal/solitaire/engine/deadend_test.go`:

```go
package engine

import "testing"

// shuffleOnlyGame has exactly one legal card move (8♠ from 9♥ onto 9♦) which
// reveals nothing — a pure reshuffle. Empty piles accept only kings; there are
// no kings, no aces, and no stock or waste.
func shuffleOnlyGame() *Game {
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{
		{Card: Card{Suit: Hearts, Rank: Nine}, FaceUp: true},
		{Card: Card{Suit: Spades, Rank: Eight}, FaceUp: true},
	}
	g.tableau[1] = []TableauCard{
		{Card: Card{Suit: Diamonds, Rank: Nine}, FaceUp: true},
	}
	return g
}

func TestProductiveMovesExcludesShuffles(t *testing.T) {
	g := shuffleOnlyGame()
	if len(g.LegalMoves()) == 0 {
		t.Fatal("expected at least the 8♠→9♦ shuffle to be legal")
	}
	if got := g.ProductiveMoves(); len(got) != 0 {
		t.Fatalf("shuffle-only position must have no productive moves, got %d", len(got))
	}
}

func TestProductiveMovesIncludesFoundationRevealAndDraw(t *testing.T) {
	// Ace on a tableau pile: foundation move is productive.
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Clubs, Rank: Ace}, FaceUp: true}}
	if len(g.ProductiveMoves()) == 0 {
		t.Fatal("ace to foundation must be productive")
	}

	// Revealing move: face-down card under a movable card.
	g = &Game{status: Playing}
	g.tableau[0] = []TableauCard{
		{Card: Card{Suit: Clubs, Rank: Five}, FaceUp: false},
		{Card: Card{Suit: Spades, Rank: Eight}, FaceUp: true},
	}
	g.tableau[1] = []TableauCard{{Card: Card{Suit: Diamonds, Rank: Nine}, FaceUp: true}}
	moves := g.ProductiveMoves()
	if len(moves) == 0 {
		t.Fatal("revealing 8♠→9♦ must be productive")
	}

	// Waste-source move is productive even to a tableau pile.
	g = &Game{status: Playing}
	g.waste = []Card{{Suit: Spades, Rank: Eight}}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Diamonds, Rank: Nine}, FaceUp: true}}
	if len(g.ProductiveMoves()) == 0 {
		t.Fatal("draw-pile card onto tableau must be productive")
	}
}

func TestDeadEndTrueWhenNothingReachable(t *testing.T) {
	// No stock, no waste, only a shuffle available.
	if g := shuffleOnlyGame(); !g.DeadEnd() {
		t.Fatal("shuffle-only position with empty deck must be a dead end")
	}

	// Full deck cycle with no productive tops: three unplayable cards cycle
	// forever with Q♦ as the only reachable top.
	g := shuffleOnlyGame()
	g.stock = []Card{
		{Suit: Clubs, Rank: Five},
		{Suit: Hearts, Rank: Queen},
		{Suit: Diamonds, Rank: Queen},
	}
	if !g.DeadEnd() {
		t.Fatal("cycling deck with no playable top must be a dead end")
	}
}

func TestDeadEndFalseWhenCardBuriedInDeck(t *testing.T) {
	// Ace is not the current waste top but surfaces after one draw.
	g := shuffleOnlyGame()
	g.stock = []Card{
		{Suit: Clubs, Rank: Five},
		{Suit: Clubs, Rank: Ace},
	}
	if g.DeadEnd() {
		t.Fatal("ace reachable via draw must not be a dead end")
	}
}

func TestDeadEndFalseWhenProductiveNowOrWon(t *testing.T) {
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Clubs, Rank: Ace}, FaceUp: true}}
	if g.DeadEnd() {
		t.Fatal("position with immediate productive move is not dead")
	}
	won := &Game{status: Won}
	if won.DeadEnd() {
		t.Fatal("won game is never a dead end")
	}
}

func TestDeadEndDoesNotMutate(t *testing.T) {
	g := shuffleOnlyGame()
	g.stock = []Card{
		{Suit: Clubs, Rank: Five},
		{Suit: Hearts, Rank: Queen},
		{Suit: Diamonds, Rank: Queen},
	}
	g.DeadEnd()
	if g.Moves() != 0 || g.StockCount() != 3 || len(g.waste) != 0 {
		t.Fatalf("DeadEnd mutated game: moves=%d stock=%d waste=%d", g.Moves(), g.StockCount(), len(g.waste))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/solitaire/engine/ -run 'Productive|DeadEnd' -count=1`
Expected: FAIL — `ProductiveMoves` and `DeadEnd` undefined.

- [ ] **Step 3: Implement deadend.go**

Create `internal/solitaire/engine/deadend.go`:

```go
package engine

// productive reports whether a move makes progress: its destination is a
// foundation, its source is the draw pile, or it reveals a face-down card.
// Draw and recycle are never productive by themselves, and tableau reshuffles
// that reveal nothing keep no game alive.
func productive(move Move) bool {
	switch move.Kind {
	case MoveDraw, MoveRecycle:
		return false
	}
	return move.Destination.Kind == Foundation || move.Source.Kind == Waste || move.RevealsCard
}

func (g *Game) ProductiveMoves() []Move {
	var moves []Move
	for _, move := range g.LegalMoves() {
		if productive(move) {
			moves = append(moves, move)
		}
	}
	return moves
}

// DeadEnd reports whether no productive move is reachable: none now, and none
// for any draw-pile top reachable by drawing/recycling through one full deck
// cycle, simulated on a clone. Drawing without moving cards is deterministic
// and returns to the starting deck/draw configuration after one full cycle,
// so one cycle is sufficient. The iteration cap is a safety net only: deck
// plus draw pile hold at most 24 cards, needing at most 9 draws per cycle.
func (g *Game) DeadEnd() bool {
	if g.status != Playing {
		return false
	}
	if len(g.ProductiveMoves()) > 0 {
		return false
	}
	sim := g.Clone()
	startStock, startWaste := len(sim.stock), len(sim.waste)
	for range 60 {
		if !sim.Draw() {
			return true // nothing left to draw or recycle
		}
		if len(sim.ProductiveMoves()) > 0 {
			return false
		}
		if len(sim.stock) == startStock && len(sim.waste) == startWaste {
			return true // full cycle completed without a productive move
		}
	}
	return true
}
```

- [ ] **Step 4: Run engine tests**

Run: `go test ./internal/solitaire/engine/ -count=1`
Expected: PASS (new and existing tests).

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/engine/deadend.go internal/solitaire/engine/deadend_test.go
git commit -m "feat(solitaire): productive-move dead-end detection

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: Remove hints

**Files:**
- Modify: `internal/solitaire/ui/assist.go` (delete `hint`, `bestHintMove`, `hintRank`, `describeHintMove`)
- Modify: `internal/solitaire/ui/game.go` (`handleKey` case `"h"`, help text)
- Modify: `internal/solitaire/ui/game_test.go` (delete `TestHintMarksAssistedAndDoesNotMutateGame`, lines ~431–445)
- Modify: `internal/solitaire/ui/view_test.go` (help expectation, line ~84)

**Interfaces:**
- Consumes: nothing new.
- Produces: `h` key unbound (Task 4 rebinds it to cursor-left). `describeCard`, `describeLocation`, `cardAt`, `focusLocation` and all other `assist.go` functions remain — only the four hint functions go.

- [ ] **Step 1: Update tests to the hint-free contract**

In `internal/solitaire/ui/game_test.go`, delete the whole function `TestHintMarksAssistedAndDoesNotMutateGame`.

In `internal/solitaire/ui/view_test.go` (~line 84), remove `"h            hint"` from the expected help strings slice, leaving:

```go
	for _, want := range []string{"Solitaire Draw 3", "d            draw 3", "a            move focused/selected", "A            auto-foundation all", "spc/enter"} {
```

Add a regression test to `internal/solitaire/ui/game_test.go`:

```go
func TestHintKeyRemoved(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("h"))
	if strings.Contains(m.message, "Hint") {
		t.Fatalf("h must no longer produce hints, got %q", m.message)
	}
	if m.assisted {
		t.Fatal("h must no longer mark the game assisted")
	}
}
```

- [ ] **Step 2: Run tests to verify the new one fails**

Run: `go test ./internal/solitaire/ui/ -run TestHintKeyRemoved -count=1`
Expected: FAIL — `h` still calls `m.hint()`, which sets `assisted` and a `Hint:` message.

- [ ] **Step 3: Delete the hint code**

- In `internal/solitaire/ui/assist.go`: delete the functions `hint`, `bestHintMove`, `hintRank`, and `describeHintMove` entirely. Keep everything else.
- In `internal/solitaire/ui/game.go` `handleKey`: delete the case
  ```go
  	case "h":
  		return m.hint(), nil
  ```
- In `internal/solitaire/ui/game.go` `helpView`: delete the line `" h            hint",`.

- [ ] **Step 4: Run the ui package tests**

Run: `go test ./internal/solitaire/ui/ -count=1`
Expected: PASS. If anything else references the deleted functions, the compiler will point at it — delete those references too (they are hint-only).

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/
git commit -m "feat(solitaire): remove hint feature

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Deck/Draw labels and single-card draw display

**Files:**
- Modify: `internal/solitaire/ui/game.go` (`topView`, `noProgressMessage`, `helpView` draw line)
- Modify: `internal/solitaire/ui/assist.go` (`describeLocation`)
- Modify: `internal/solitaire/ui/view_test.go`

**Interfaces:**
- Consumes: existing `emptyToken`, `hiddenToken`, `styledCard`, `decorate`, `WasteVisible()`.
- Produces: renamed labels and one-slot draw display. Internal identifiers (`regionWaste`, `engine.Waste`, model fields) are NOT renamed — display strings only.

- [ ] **Step 1: Update view tests**

In `internal/solitaire/ui/view_test.go` (~line 18), change the expected labels:

```go
	for _, want := range []string{"SOLITAIRE DRAW 3", "Deck", "Draw", "Foundations", "Tableau"} {
```

Add:

```go
func TestTopViewShowsSingleDrawCard(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("d")) // draw 3: only the top card may render
	visible := m.game.WasteVisible()
	if len(visible) != 3 {
		t.Fatalf("expected 3 drawn cards, got %d", len(visible))
	}
	top := m.topView()
	for i, card := range visible {
		token := cardToken(card)
		if i == len(visible)-1 {
			if !strings.Contains(top, token) {
				t.Fatalf("top card %s missing from view:\n%s", token, top)
			}
		} else if strings.Contains(top, token) {
			t.Fatalf("under-card %s must not render:\n%s", token, top)
		}
	}
}

func TestNoWasteWordingRemains(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionWaste}
	if got := describeLocation(m.cursor.location()); got != "draw pile" {
		t.Fatalf("describeLocation(waste) = %q, want \"draw pile\"", got)
	}
	if got := describeLocation(cursor{region: regionStock}.location()); got != "deck" {
		t.Fatalf("describeLocation(stock) = %q, want \"deck\"", got)
	}
}
```

(If two drawn cards share a rank+suit token that would falsely match, they cannot — a deck has unique cards and tokens encode rank+suit.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/solitaire/ui/ -run 'TestTopView|TestNoWaste|TestViewRendersBoard' -count=1`
Expected: FAIL — labels still `Stock`/`Waste`, three cards render, `describeLocation` returns old names. (Whatever the ~line 18 test is named, it must run and fail; adjust the `-run` pattern to include it.)

- [ ] **Step 3: Implement**

In `internal/solitaire/ui/game.go`, replace `topView` with:

```go
func (m Model) topView() string {
	var b strings.Builder
	b.WriteString(" Deck   Draw    Foundations\n ")
	stock := emptyToken(" · ")
	if m.game.StockCount() > 0 {
		stock = hiddenToken()
	}
	b.WriteString(m.decorate(regionStock, 0, 0, stock))
	b.WriteString("  ")
	waste := m.game.WasteVisible()
	draw := emptyToken("")
	if len(waste) > 0 {
		draw = styledCard(waste[len(waste)-1])
	}
	b.WriteString(m.decorate(regionWaste, 0, 0, draw))
	b.WriteString("   ")
	for suit := engine.Spades; suit <= engine.Clubs; suit++ {
		token := emptyToken("")
		if card, ok := m.game.FoundationTop(suit); ok {
			token = styledCard(card)
		}
		b.WriteString(m.decorate(regionFoundation, int(suit), 0, token))
		if suit < engine.Clubs {
			b.WriteString(" ")
		}
	}
	return b.String()
}
```

(Header spacing note: tokens are 5 columns wide; `hiddenToken`/`emptyToken` carry a trailing space and `styledCard` does not, exactly as before. If `Foundations` sits visibly off its slots when eyeballed in Task 6, adjust only the spaces in the header string and the two separator strings — nothing else.)

In `internal/solitaire/ui/assist.go` `describeLocation`: `"waste"` → `"draw pile"`, `"stock"` → `"deck"`.

In `internal/solitaire/ui/game.go`:
- `noProgressMessage` → `"No progress this pass — move a card before recycling again"`
- `helpView` draw line → `" d            draw 3 / recycle deck",`

- [ ] **Step 4: Run the ui package tests**

Run: `go test ./internal/solitaire/ui/ -count=1`
Expected: PASS. Any test asserting the old `noProgressMessage` text (see `TestNoProgressWarningAfterStockPassWithoutProgress`) must be updated to the new string, not deleted.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/
git commit -m "feat(solitaire): deck/draw labels and single-card draw display

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: Navigation — hjkl, jump keys, face-down skip, sorted destinations

**Files:**
- Modify: `internal/solitaire/ui/game.go` (`handleKey`, `moveCursorUp`, `moveCursorDown`)
- Modify: `internal/solitaire/ui/assist.go` (destination sorting)
- Modify: `internal/solitaire/ui/game_test.go`

**Interfaces:**
- Consumes: `cursor`/regions from `cursor.go`, `engine.Move.RevealsCard/.Safe/.Destination`, existing `activate()` flow.
- Produces: `func sortMovesByPriority(moves []engine.Move) []engine.Move` and `func destinationRank(move engine.Move) int` in `assist.go`; keys `h`, `w`, `s`, `f`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/solitaire/ui/game_test.go`:

```go
func TestHMovesCursorLeft(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionTableau, col: 3}
	m, _ = m.Update(key("h"))
	if m.cursor.region != regionTableau || m.cursor.col != 2 {
		t.Fatalf("h from tableau col 3 = %+v, want col 2", m.cursor)
	}
}

func TestJumpKeysReachRegions(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("w"))
	if m.cursor.region != regionWaste {
		t.Fatalf("w = %+v, want draw pile", m.cursor)
	}
	m, _ = m.Update(key("s"))
	if m.cursor.region != regionStock {
		t.Fatalf("s = %+v, want deck", m.cursor)
	}
	m, _ = m.Update(key("f"))
	if m.cursor.region != regionFoundation || m.cursor.col != 0 {
		t.Fatalf("f = %+v, want foundation 0", m.cursor)
	}
	m, _ = m.Update(key("f"))
	if m.cursor.col != 1 {
		t.Fatalf("second f = %+v, want foundation 1", m.cursor)
	}
	for range 3 {
		m, _ = m.Update(key("f"))
	}
	if m.cursor.col != 0 {
		t.Fatalf("f must wrap 4→1, got %+v", m.cursor)
	}
}

func TestVerticalNavigationSkipsFaceDownCards(t *testing.T) {
	m := testModel(t)
	// Column 3 deals rows 0-2 face-down, row 3 face-up.
	m.cursor = cursor{region: regionTableau, col: 3, row: 3}
	m, _ = m.Update(key("k"))
	if m.cursor.region == regionTableau && m.cursor.row < 3 && !m.game.Tableau(3)[m.cursor.row].FaceUp {
		t.Fatalf("up landed on face-down card: %+v", m.cursor)
	}
	if m.cursor.region != regionFoundation {
		t.Fatalf("up from only face-up row must exit to foundations, got %+v", m.cursor)
	}
	m, _ = m.Update(key("j"))
	if m.cursor.region != regionTableau || !m.game.Tableau(m.cursor.col)[m.cursor.row].FaceUp {
		t.Fatalf("down must land on a face-up card, got %+v", m.cursor)
	}
}

func TestSelectedDestinationsSortedByPriority(t *testing.T) {
	moves := []engine.Move{
		{Destination: engine.Location{Kind: engine.Foundation, Pile: 0}},
		{Destination: engine.Location{Kind: engine.Tableau, Pile: 2}},
		{Destination: engine.Location{Kind: engine.Foundation, Pile: 1}, Safe: true},
	}
	sorted := sortMovesByPriority(moves)
	if !(sorted[0].Safe && sorted[0].Destination.Kind == engine.Foundation) {
		t.Fatalf("safe foundation must sort first: %+v", sorted[0])
	}
	if sorted[1].Destination.Kind != engine.Tableau {
		t.Fatalf("tableau must sort before unsafe foundation: %+v", sorted[1])
	}
	if len(sorted) != 3 || len(moves) != 3 {
		t.Fatal("sort must not add or drop moves")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/solitaire/ui/ -run 'TestHMoves|TestJumpKeys|TestVertical|TestSelectedDestinationsSorted' -count=1`
Expected: FAIL — `h` unbound, `w`/`s`/`f` unbound, no skipping, `sortMovesByPriority` undefined.

- [ ] **Step 3: Implement**

In `internal/solitaire/ui/game.go` `handleKey`:

```go
	case "left", "h":
		m.moveCursorLeft()
	case "w":
		m.cursor = cursor{region: regionWaste}
	case "s":
		m.cursor = cursor{region: regionStock}
	case "f":
		if m.cursor.region == regionFoundation {
			m.cursor.col = (m.cursor.col + 1) % 4
		} else {
			m.cursor = cursor{region: regionFoundation, col: 0}
		}
```

(Replace the existing `case "left":`; keep every other case as is.)

Replace the tableau arms of `moveCursorUp` / `moveCursorDown`:

```go
func (m *Model) moveCursorUp() {
	switch m.cursor.region {
	case regionTableau:
		cards := m.game.Tableau(m.cursor.col)
		for row := m.cursor.row - 1; row >= 0; row-- {
			if row < len(cards) && cards[row].FaceUp {
				m.cursor.row = row
				return
			}
		}
		if m.cursor.col < 4 {
			m.cursor = cursor{region: regionFoundation, col: m.cursor.col}
		} else {
			m.cursor = cursor{region: regionWaste}
		}
	}
}

func (m *Model) moveCursorDown() {
	switch m.cursor.region {
	case regionStock, regionWaste:
		m.cursor = cursor{region: regionTableau}
		m.clampTableauRowToPlayable()
	case regionFoundation:
		m.cursor = cursor{region: regionTableau, col: m.cursor.col}
		m.clampTableauRowToPlayable()
	case regionTableau:
		cards := m.game.Tableau(m.cursor.col)
		for row := m.cursor.row + 1; row < len(cards); row++ {
			if cards[row].FaceUp {
				m.cursor.row = row
				return
			}
		}
	}
}
```

(Entering a column from above now uses the existing `clampTableauRowToPlayable`, which already snaps to the first face-up card.)

In `internal/solitaire/ui/assist.go` add (import `"sort"`):

```go
// destinationRank orders a card's candidate destinations: reveals first, then
// safe foundation, then tableau, then unsafe foundation. All moves from one
// source share RevealsCard, so within a selection the first clause is a tie.
func destinationRank(move engine.Move) int {
	switch {
	case move.RevealsCard:
		return 0
	case move.Destination.Kind == engine.Foundation && move.Safe:
		return 1
	case move.Destination.Kind == engine.Tableau:
		return 2
	default:
		return 3
	}
}

func sortMovesByPriority(moves []engine.Move) []engine.Move {
	sorted := append([]engine.Move(nil), moves...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return destinationRank(sorted[i]) < destinationRank(sorted[j])
	})
	return sorted
}
```

In `internal/solitaire/ui/game.go` `helpView`, replace the line `" arrows/jkl   move",` with:

```go
		" arrows/hjkl  move",
		" w / s / f    jump to draw pile / deck / foundations",
```

(The `1-7` column jumps predate this task and are already undocumented; leave that as is.)

In `internal/solitaire/ui/game.go` `activate()`, in the multi-destination branch, replace

```go
			m.selected = &loc
			m.selectedMoves = append([]engine.Move(nil), moves...)
```

with

```go
			m.selected = &loc
			m.selectedMoves = sortMovesByPriority(moves)
```

- [ ] **Step 4: Run the ui package tests**

Run: `go test ./internal/solitaire/ui/ -count=1`
Expected: PASS. `TestSpaceKeepsSelectionWhenMultipleDestinationsExist` and the tab-cycling tests must still pass — they assert behavior, not engine ordering; if one pinned the old first destination, update it to expect the sorted order.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/
git commit -m "feat(solitaire): hjkl, jump keys, face-down skip, sorted destinations

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 5: Game-over screen

**Files:**
- Modify: `internal/solitaire/ui/game.go` (model flag, dead-end checks, key gating, view branch)
- Modify: `internal/solitaire/ui/game_test.go`
- Modify: `internal/solitaire/ui/view_test.go`

**Interfaces:**
- Consumes: `engine.DeadEnd()` and `engine.ProductiveMoves()` (Task 1), existing `undo()`, `recordSummary()`, `fitLine`, `setUnexported` test helper.
- Produces: model field `gameOver bool`, methods `checkDeadEnd()`, `gameOverView()`, `newDeal()`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/solitaire/ui/game_test.go`:

```go
// forceDeadEnd rewrites the model's game into a two-queen position with no
// deck, draw pile, kings, or aces: no legal moves at all.
func forceDeadEnd(t *testing.T, m Model) Model {
	t.Helper()
	rv := reflect.ValueOf(m.game).Elem()
	setUnexported(rv.FieldByName("stock"), []engine.Card(nil))
	setUnexported(rv.FieldByName("waste"), []engine.Card(nil))
	var tableau [7][]engine.TableauCard
	tableau[0] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Spades, Rank: engine.Queen}, FaceUp: true}}
	tableau[1] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Hearts, Rank: engine.Queen}, FaceUp: true}}
	setUnexported(rv.FieldByName("tableau"), tableau)
	if !m.game.DeadEnd() {
		t.Fatal("constructed position must be a dead end")
	}
	return m
}

func TestDeadEndShowsGameOverScreen(t *testing.T) {
	m := forceDeadEnd(t, testModel(t))
	m, _ = m.Update(key("d")) // any action recomputes dead-end state
	if !m.gameOver {
		t.Fatal("gameOver must be set after action on dead board")
	}
	if m.running {
		t.Fatal("timer must stop on game over")
	}
	v := m.View()
	for _, want := range []string{"G A M E   O V E R", "no moves left", "n new game", "u undo"} {
		if !strings.Contains(v, want) {
			t.Fatalf("game-over view missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "Tableau") {
		t.Fatalf("board must be hidden on game over:\n%s", v)
	}
}

func TestGameOverGatesKeys(t *testing.T) {
	m := forceDeadEnd(t, testModel(t))
	m, _ = m.Update(key("d"))
	before := m.cursor
	m, _ = m.Update(key("1"))
	m, _ = m.Update(key("j"))
	if m.cursor != before {
		t.Fatal("cursor keys must be inert on game over")
	}
	_, cmd := m.Update(key("m"))
	if cmd == nil {
		t.Fatal("m must still return to launcher")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("m must produce BackMsg")
	}
}

func TestGameOverUndoReturnsToBoard(t *testing.T) {
	m := testModel(t)
	m.undoStack = append(m.undoStack, m.game.Clone()) // live position to restore
	m = forceDeadEnd(t, m)
	m, _ = m.Update(key("d"))
	if !m.gameOver {
		t.Fatal("expected game over")
	}
	m, _ = m.Update(key("u"))
	if m.gameOver {
		t.Fatal("undo into a live position must clear gameOver")
	}
	if !m.assisted {
		t.Fatal("undo must still mark the run assisted")
	}
	if !strings.Contains(m.View(), "Tableau") {
		t.Fatal("board must render again after undo")
	}
}

func TestGameOverNewDealResets(t *testing.T) {
	m := forceDeadEnd(t, testModel(t))
	m, _ = m.Update(key("d"))
	m, _ = m.Update(key("n"))
	if m.gameOver {
		t.Fatal("new deal must clear gameOver")
	}
	if m.game.Moves() != 0 {
		t.Fatalf("new deal must reset moves, got %d", m.game.Moves())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/solitaire/ui/ -run 'TestDeadEnd|TestGameOver' -count=1`
Expected: FAIL — no `gameOver` field.

- [ ] **Step 3: Implement**

In `internal/solitaire/ui/game.go`:

1. Add `gameOver bool` to `Model` (after `recordedWin`).

2. Add:

```go
func (m Model) checkDeadEnd() Model {
	m.gameOver = m.game.Status() == engine.Playing && m.game.DeadEnd()
	if m.gameOver {
		m.running = false
	}
	return m
}
```

3. Guard the timer — in `startTimerIfNeeded`, extend the condition:

```go
	if !changed || m.running || m.gameOver || m.game.Status() != engine.Playing {
		return m, nil
	}
```

4. Recompute after every board-changing path. In `draw`, `activate` (both apply branches), `autoFoundation`, `autoFoundationAll`, and `undo`, insert `m = m.checkDeadEnd()` immediately before each `return m.startTimerIfNeeded(...)`. (In `undo` this both clears a stale `gameOver` and re-detects a still-dead restored position.)

5. Extract the new-deal body (currently inline under `case "n"`) into a method and call it from the case:

```go
func (m Model) newDeal() (Model, tea.Cmd) {
	fresh, err := NewModel(m.scores, m.rng)
	if err != nil {
		m.message = "Could not deal a new game"
		return m, nil
	}
	fresh.tickGen = m.tickGen + 1
	fresh.termW = m.termW
	fresh.termH = m.termH
	return fresh, nil
}
```

```go
	case "n":
		return m.newDeal()
```

6. Gate keys — at the top of `handleKey`:

```go
	if m.gameOver {
		switch k {
		case "n":
			return m.newDeal()
		case "u":
			return m.undo()
		case "m", "esc":
			return m, back()
		case "q":
			return m, tea.Quit
		case "?":
			m.showHelp = true
		}
		return m, nil
	}
```

7. View branch — in `View()`, after the help check:

```go
	if m.showHelp {
		return m.helpView()
	}
	if m.gameOver {
		return m.gameOverView()
	}
```

8. Add:

```go
func (m Model) gameOverView() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(" SOLITAIRE DRAW 3") + "\n\n")
	b.WriteString(styleErr.Render("      G A M E   O V E R") + "\n")
	b.WriteString(styleFaint.Render("      no moves left") + "\n\n")
	b.WriteString(fmt.Sprintf("      time   %4ds\n", m.elapsed))
	b.WriteString(fmt.Sprintf("      moves  %5d\n", m.game.Moves()))
	b.WriteString(styleFaint.Render("      "+m.recordSummary()) + "\n\n")
	b.WriteString(styleFaint.Render(fitLine(m.termW,
		" n new game · u undo · m menu · q quit",
		" n new · u undo · m menu",
	)) + "\n")
	return b.String()
}
```

- [ ] **Step 4: Run the ui package tests**

Run: `go test ./internal/solitaire/ui/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/
git commit -m "feat(solitaire): game-over screen on dead-end detection

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 6: Full verification and rebuild

**Files:**
- None expected; fix only what verification reveals.

- [ ] **Step 1: Run all gates**

```sh
go build ./... && go vet ./... && gofmt -l . && go test ./... -count=1
```

Expected: build/vet clean, `gofmt -l` prints nothing, all packages PASS.

- [ ] **Step 2: Grep for leftover wording**

```sh
grep -rn "Waste\|waste\|Stock\b\|hint" internal/solitaire/ui/ --include='*.go' | grep -v _test.go | grep -vi "regionWaste\|engine.Waste\|WasteVisible\|stockPass\|StockCount"
```

Expected: no user-visible `Waste`/`Stock`/`hint` strings remain (internal identifiers are fine and are excluded above; judge any remaining hits by whether the string reaches the screen).

- [ ] **Step 3: Manual smoke test**

Rebuild: `go build -o mines . && go install .`. Launch `./mines` (or drive with `expect` and a sandboxed `HOME`): enter Solitaire, confirm `Deck`/`Draw` labels and single draw card; `h` moves left; `w`/`s`/`f` jump; up/down skip face-down cards; select a card with multiple destinations and confirm the best is pre-focused; play or construct toward a dead end if feasible and confirm the screen, `u` escape, and `n` reset. Confirm Mines and 2048 are unchanged.

- [ ] **Step 4: Commit any fixes**

Only if Steps 1–3 revealed gaps; otherwise nothing to commit.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `productive` / `ProductiveMoves` / `DeadEnd` with full-cycle simulation | 1 |
| Dead-end excludes reshuffles and king swaps | 1 |
| Hint removal; undo remains sole assist reason | 2 |
| Deck/Draw labels; `deck` / `draw pile` strings; no-progress rewording | 3 |
| Single-card draw display; min sizes unchanged | 3 |
| `h` = left; `w`/`s`/`f` jumps; `f` cycles foundations | 4 |
| Up/down skip face-down cards | 4 |
| Productivity-sorted destinations; enter-enter = best move | 4 |
| Game-over screen: heading, time/moves/best, gated keys, undo escape | 5 |
| Timer stops on game over; losses unrecorded | 5 |
| Full suite green; other games untouched | 6 |
