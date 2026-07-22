# Solitaire Usability Round 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `v` show-possible-moves key, make the cursor persist on empty columns and after moves, and rename the built binary to `games`.

**Architecture:** All game changes live in `internal/solitaire/ui` (`game.go`, `assist.go`, `styles.go`, tests) on top of the existing `engine.ProductiveMoves()`. The rename is a build-artifact change only — the module path is untouched.

**Tech Stack:** Go 1.26.4, Bubble Tea v1.3.10, Lip Gloss v1.1.0, `go test`.

**Spec:** `docs/superpowers/specs/2026-07-22-solitaire-usability-2-design.md`

## Global Constraints

- No new dependencies; module stays `github.com/wallymagill/mines`.
- `v` highlights **productive** sources only (`engine.ProductiveMoves()`); it marks the run assisted with reason `"moves"` (only when `Status() == engine.Playing`); any key dismisses AND performs its normal action; `v` is inert on game over / win / help / too-small.
- Status strings exactly: `%d moves · any key to dismiss` and `no moves on board — keep drawing`.
- Highlight style: bold, background `28`, foreground `231`. Cursor stays plain reverse video; selection stays bold reverse. No other restyling.
- Cursor persistence: empty tableau columns render a decorated `[  ]` slot at row 0; cursor re-clamps immediately after every board mutation.
- Binary: `./games` in the repo root and `~/go/bin/games`; stale `./mines` and `~/go/bin/mines` deleted; `.gitignore` gains `/games`.
- Mines, 2048, store, app, and solitaire engine packages untouched.
- TDD: failing test → implement → pass → commit. Commit messages end with the trailer `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

**Existing helpers in `internal/solitaire/ui` (do not redefine):** `testModel(t)`, `key(s)`, `setUnexported(field, value)` in `game_test.go`; `cursor{region, col, row}` with `.location()` in `cursor.go`; `emptyToken`, `decorate` in `styles.go`/`game.go`.

---

## File Structure

```text
internal/solitaire/ui/
  styles.go      // Modify: + styleMoveHint                     (Task 1)
  game.go        // Modify: v key, decorate branch, help (T1); tableauView placeholder, re-clamp (T2)
  assist.go      // Modify: showPossibleMoves helper            (Task 1)
  game_test.go   // Modify: T1, T2
  view_test.go   // Modify: T1 (help), T2 (placeholder)

.gitignore       // Modify: + /games                            (Task 3)
```

---

### Task 1: Show possible moves (`v`)

**Files:**
- Modify: `internal/solitaire/ui/styles.go`
- Modify: `internal/solitaire/ui/game.go`
- Modify: `internal/solitaire/ui/assist.go`
- Modify: `internal/solitaire/ui/game_test.go`
- Modify: `internal/solitaire/ui/view_test.go`

**Interfaces:**
- Consumes: `engine.ProductiveMoves()`, `cursor{...}.location()`, model fields `assisted`/`assistReason`/`message`/`gameOver`, `clearSelection()`.
- Produces: model fields `showMoves bool`, `moveSources map[engine.Location]bool`; method `showPossibleMoves() Model`; style `styleMoveHint`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/solitaire/ui/game_test.go`:

```go
// movesModel builds a position with exactly two productive sources:
// A♠ on tableau 0 (foundation move) and the draw-pile 8♠ (plays on 9♦).
func movesModel(t *testing.T) Model {
	t.Helper()
	m := testModel(t)
	rv := reflect.ValueOf(m.game).Elem()
	setUnexported(rv.FieldByName("stock"), []engine.Card(nil))
	setUnexported(rv.FieldByName("waste"), []engine.Card{{Suit: engine.Spades, Rank: engine.Eight}})
	var tableau [7][]engine.TableauCard
	tableau[0] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Spades, Rank: engine.Ace}, FaceUp: true}}
	tableau[1] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Diamonds, Rank: engine.Nine}, FaceUp: true}}
	setUnexported(rv.FieldByName("tableau"), tableau)
	return m
}

func TestShowMovesHighlightsProductiveSources(t *testing.T) {
	m := movesModel(t)
	m, _ = m.Update(key("v"))
	if !m.showMoves {
		t.Fatal("v must enable showMoves")
	}
	if !m.assisted || m.assistReason != "moves" {
		t.Fatalf("v must mark assisted/moves, got %v/%q", m.assisted, m.assistReason)
	}
	wantSources := []engine.Location{
		{Kind: engine.Tableau, Pile: 0, Index: 0},
		{Kind: engine.Waste},
	}
	for _, want := range wantSources {
		if !m.moveSources[want] {
			t.Fatalf("source %+v missing from moveSources %v", want, m.moveSources)
		}
	}
	if m.moveSources[engine.Location{Kind: engine.Tableau, Pile: 1, Index: 0}] {
		t.Fatal("9♦ has no productive move and must not highlight")
	}
	if !strings.Contains(m.message, "moves · any key to dismiss") {
		t.Fatalf("status = %q", m.message)
	}
}

func TestShowMovesKeepDrawingMessage(t *testing.T) {
	m := testModel(t)
	rv := reflect.ValueOf(m.game).Elem()
	setUnexported(rv.FieldByName("waste"), []engine.Card(nil))
	setUnexported(rv.FieldByName("stock"), []engine.Card{
		{Suit: engine.Clubs, Rank: engine.Five},
		{Suit: engine.Clubs, Rank: engine.Ace},
	})
	var tableau [7][]engine.TableauCard
	tableau[0] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Hearts, Rank: engine.Nine}, FaceUp: true}}
	setUnexported(rv.FieldByName("tableau"), tableau)
	if m.game.DeadEnd() {
		t.Fatal("fixture must not be a dead end (ace surfaces on draw)")
	}
	m, _ = m.Update(key("v"))
	if m.showMoves {
		t.Fatal("no board moves: must not enable highlights")
	}
	if m.message != "no moves on board — keep drawing" {
		t.Fatalf("status = %q", m.message)
	}
	if !m.assisted || m.assistReason != "moves" {
		t.Fatal("v marks assisted even in keep-drawing case")
	}
}

func TestShowMovesDismissedByNextKeyWhichStillActs(t *testing.T) {
	m := movesModel(t)
	m, _ = m.Update(key("v"))
	if !m.showMoves {
		t.Fatal("expected showMoves on")
	}
	m.cursor = cursor{region: regionTableau, col: 1}
	m, _ = m.Update(key("h"))
	if m.showMoves || m.moveSources != nil {
		t.Fatal("any key must clear highlights")
	}
	if m.cursor.col != 0 {
		t.Fatalf("the dismissing key must still act (h moves left), cursor %+v", m.cursor)
	}
}

func TestShowMovesInertOnGameOver(t *testing.T) {
	m := forceDeadEnd(t, testModel(t))
	m, _ = m.Update(key("d"))
	if !m.gameOver {
		t.Fatal("expected game over")
	}
	m, _ = m.Update(key("v"))
	if m.showMoves {
		t.Fatal("v must be inert on game over")
	}
}
```

In `internal/solitaire/ui/view_test.go` (~line 84), add `"v            show possible moves"` to the expected help strings slice.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/solitaire/ui/ -run 'TestShowMoves' -count=1`
Expected: FAIL — no `showMoves` field (build error). The help test fails after the build is fixed.

- [ ] **Step 3: Implement**

In `internal/solitaire/ui/styles.go` add to the `var` block:

```go
	styleMoveHint = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("28")).Foreground(lipgloss.Color("231"))
```

In `internal/solitaire/ui/game.go`, add to `Model` (after `gameOver`):

```go
	showMoves   bool
	moveSources map[engine.Location]bool
```

In `handleKey`, at the very top (before the `gameOver` gate):

```go
	if m.showMoves {
		m.showMoves = false
		m.moveSources = nil
		m.message = ""
	}
```

Add the key case (alongside the other letters):

```go
	case "v":
		return m.showPossibleMoves(), nil
```

In `decorate`, insert between the cursor check and the selection checks:

```go
	if m.showMoves && m.moveSources[cursor{region: r, col: col, row: row}.location()] {
		return styleMoveHint.Render(token)
	}
```

In `helpView`, after the `" w / s / f    jump to draw pile / deck / foundations",` line:

```go
		" v            show possible moves (assisted)",
```

In `internal/solitaire/ui/assist.go` add:

```go
// showPossibleMoves highlights every source with a productive move. Any key
// dismisses. Using it marks the run assisted — it is help, like undo.
func (m Model) showPossibleMoves() Model {
	m = m.clearSelection()
	if m.game.Status() == engine.Playing {
		m.assisted = true
		m.assistReason = "moves"
	}
	moves := m.game.ProductiveMoves()
	if len(moves) == 0 {
		m.message = "no moves on board — keep drawing"
		return m
	}
	sources := map[engine.Location]bool{}
	for _, move := range moves {
		sources[move.Source] = true
	}
	m.showMoves = true
	m.moveSources = sources
	m.message = fmt.Sprintf("%d moves · any key to dismiss", len(moves))
	return m
}
```

(The `gameOver` gate in `handleKey` does not list `v`, so it is already inert on game over. Help/too-small swallow keys before `handleKey`. A won game never enters `showPossibleMoves` with `Playing`, so `assisted` is not set after a win.)

- [ ] **Step 4: Run the ui package tests**

Run: `go test ./internal/solitaire/ui/ -count=1`
Expected: PASS, `gofmt -l internal/solitaire/ui/` empty.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/
git commit -m "feat(solitaire): v shows productive moves

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: Cursor persistence

**Files:**
- Modify: `internal/solitaire/ui/game.go` (`tableauView`, mutation paths)
- Modify: `internal/solitaire/ui/game_test.go`
- Modify: `internal/solitaire/ui/view_test.go`

**Interfaces:**
- Consumes: `emptyToken`, `decorate`, `clampTableauRowToPlayable` (exists in assist.go), the mutation paths (`activate`, `autoFoundation`, `autoFoundationAll`, `undo`).
- Produces: method `clampCursorAfterBoardChange()` (pointer receiver); empty columns render a decorated `[  ]` slot at row 0.

- [ ] **Step 1: Write the failing tests**

Add to `internal/solitaire/ui/view_test.go`:

```go
func TestEmptyColumnRendersCursorSlot(t *testing.T) {
	m := testModel(t)
	rv := reflect.ValueOf(m.game).Elem()
	var tableau [7][]engine.TableauCard
	tableau[1] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Clubs, Rank: engine.King}, FaceUp: true}}
	setUnexported(rv.FieldByName("tableau"), tableau)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	view := m.tableauView()
	if !strings.Contains(view, "[  ]") {
		t.Fatalf("empty column must render a placeholder slot:\n%s", view)
	}
}
```

(`view_test.go` needs `"reflect"` and the `engine` import if not already present.)

Add to `internal/solitaire/ui/game_test.go`:

```go
func TestCursorReclampsAfterMoveEmptiesPile(t *testing.T) {
	m := testModel(t)
	rv := reflect.ValueOf(m.game).Elem()
	// Ace alone on pile 2; drawable card keeps the game live after the move.
	setUnexported(rv.FieldByName("waste"), []engine.Card(nil))
	setUnexported(rv.FieldByName("stock"), []engine.Card{{Suit: engine.Diamonds, Rank: engine.Ace}})
	var tableau [7][]engine.TableauCard
	tableau[2] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Spades, Rank: engine.Ace}, FaceUp: true}}
	tableau[3] = []engine.TableauCard{
		{Card: engine.Card{Suit: engine.Hearts, Rank: engine.Three}, FaceUp: false},
		{Card: engine.Card{Suit: engine.Hearts, Rank: engine.Two}, FaceUp: true},
	}
	setUnexported(rv.FieldByName("tableau"), tableau)
	m.cursor = cursor{region: regionTableau, col: 2, row: 0}
	m, _ = m.Update(key("a")) // safe auto-foundation moves the ace, emptying pile 2
	if got := m.game.TableauLen(2); got != 0 {
		t.Fatalf("expected pile 2 emptied, len %d", got)
	}
	if m.cursor.region != regionTableau || m.cursor.col != 2 || m.cursor.row != 0 {
		t.Fatalf("cursor must re-clamp to the empty pile slot, got %+v", m.cursor)
	}
	// The very next frame must show the cursor: placeholder slot exists at col 2.
	if !strings.Contains(m.tableauView(), "[  ]") {
		t.Fatal("post-move view must contain the empty-column slot the cursor sits on")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/solitaire/ui/ -run 'TestEmptyColumnRenders|TestCursorReclamps' -count=1`
Expected: FAIL — no placeholder in `tableauView`. (The re-clamp test may pass its cursor assertions coincidentally — `row` stays 0 — but must fail on the placeholder assertion.)

- [ ] **Step 3: Implement**

In `internal/solitaire/ui/game.go` `tableauView`, replace the row-rendering loop:

```go
	if maxRows == 0 {
		maxRows = 1
	}
	for row := range maxRows {
		for col := range 7 {
			token := "     "
			if row < len(cols[col]) {
				card := cols[col][row]
				if card.FaceUp {
					token = styledCard(card.Card)
				} else {
					token = hiddenToken()
				}
				token = m.decorate(regionTableau, col, row, token)
			} else if row == 0 && len(cols[col]) == 0 {
				token = m.decorate(regionTableau, col, 0, emptyToken(""))
			}
			b.WriteString(" ")
			b.WriteString(token)
		}
		if row < maxRows-1 {
			b.WriteString("\n")
		}
	}
```

Add the method (near `clampTableauRow`):

```go
// clampCursorAfterBoardChange snaps the cursor back onto a rendered cell the
// moment the board mutates, so it can never point past a shortened pile.
func (m *Model) clampCursorAfterBoardChange() {
	if m.cursor.region == regionTableau {
		m.clampTableauRowToPlayable()
	}
}
```

Insert `m.clampCursorAfterBoardChange()` immediately after each successful mutation, before the existing `m = m.checkDeadEnd()` lines, in: both `activate` apply branches, `autoFoundation`, `autoFoundationAll`, and `undo`. (Five call sites; `draw` does not move tableau cards. `checkDeadEnd` and `clampCursorAfterBoardChange` have different receivers — keep the pointer call on its own line before the value reassignment.)

- [ ] **Step 4: Run the ui package tests**

Run: `go test ./internal/solitaire/ui/ -count=1`
Expected: PASS, `gofmt -l internal/solitaire/ui/` empty. Note `clampTableauRowToPlayable` on an empty pile already sets `row = 0` — exactly the placeholder slot.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/
git commit -m "fix(solitaire): cursor persists on empty columns and after moves

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Binary rename and full verification

**Files:**
- Modify: `.gitignore`
- Delete (untracked artifacts): `./mines`, `~/go/bin/mines`

- [ ] **Step 1: Update .gitignore**

Change the first line `/mines` block to:

```text
/mines
/games
```

(Keep `/mines` so a stale artifact never shows up as untracked on other checkouts.)

- [ ] **Step 2: Build the renamed binaries and remove stale ones**

```sh
go build -o games . \
  && go install . \
  && /bin/cp -f "$HOME/go/bin/mines" "$HOME/go/bin/games" \
  && /bin/rm -f "$HOME/go/bin/mines" ./mines \
  && ls -l ./games "$HOME/go/bin/games"
```

Expected: both `games` binaries listed; `./mines` and `~/go/bin/mines` gone.

- [ ] **Step 3: Run all gates**

```sh
go build ./... && go vet ./... && gofmt -l . && go test ./... -count=1
```

Expected: clean, all packages PASS.

- [ ] **Step 4: Manual smoke test**

Launch `./games` (or drive with `expect` and a sandboxed `HOME`): in Solitaire, press `v` — productive sources highlight and the status shows the count; press an arrow — highlights clear and the cursor moves; move a lone card off a pile — the cursor stays visible on the emptied column's `[  ]` slot; check the help lists `v`. Mines and 2048 unchanged.

- [ ] **Step 5: Commit**

```bash
git add .gitignore
git commit -m "chore: rename built binary to games

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| `v` highlights productive sources; exact status strings | 1 |
| Assisted with reason `"moves"`, only while Playing | 1 |
| Any-key dismissal that still acts; inert on game over | 1 |
| Highlight style bold/bg 28/fg 231; cursor & selection untouched | 1 |
| Help line for `v` | 1 |
| Empty-column placeholder slot, decorated | 2 |
| Post-mutation cursor re-clamp (5 call sites) | 2 |
| `./games` + `~/go/bin/games`; stale `mines` deleted; `.gitignore` | 3 |
| Module path unchanged; other packages untouched | Global |
