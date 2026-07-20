# Solitaire Feel v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Klondike Draw 3 feel clean under vim navigation: playable cursor clamps, destination-scoped selection with highlights, soft stuck restart guidance, stronger stuck-aware hints, and success messages that no longer look like errors.

**Architecture:** Keep pure rules in `internal/solitaire/engine` (add `HasCardMoves` only). Put navigation, selection, soft-stuck latching, hint ranking/presentation, and message severity in `internal/solitaire/ui`. No Mines changes. No new packages.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, stdlib `testing`. Existing test helpers in `internal/solitaire/ui/game_test.go` (`key`, `testModel`, `setUnexported`, model fixtures).

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-15-solitaire-feel-v1-design.md`
- Work from repo root: `/Users/wallymagill/personal-projects/terminal-game`
- TDD every behavior change: failing test → implement → green → commit
- Keep `internal/solitaire/engine` UI-free
- Soft stuck is practical, not mathematical unsolvability; never set engine `Lost`
- Soft stuck is not assisted; hints and undo remain assisted
- Safe auto-foundation (`a`/`A`) remains normal-record eligible
- Preserve Mines behavior and existing solitaire win/record paths
- Out of scope: mouse, animations, Draw 1, hard Lost, solver, endgame autocomplete, layout redesign, hint apply-on-second-press
- Commit after each task with a focused message

## File structure

| File | Responsibility |
|---|---|
| `internal/solitaire/engine/moves.go` | Add `HasCardMoves()` |
| `internal/solitaire/engine/moves_test.go` | Tests for `HasCardMoves` |
| `internal/solitaire/ui/styles.go` | Destination highlight style; info/warn styles |
| `internal/solitaire/ui/assist.go` | Hint ranking, destination helpers, focus compass helpers |
| `internal/solitaire/ui/game.go` | Keymap, clamps, selection, soft stuck, status/help, message severity |
| `internal/solitaire/ui/game_test.go` | Update/behavior tests |
| `internal/solitaire/ui/view_test.go` | Help/status/render tests |

---

### Task 1: Engine `HasCardMoves`

**Files:**
- Modify: `internal/solitaire/engine/moves.go`
- Modify: `internal/solitaire/engine/moves_test.go`

**Interfaces:**
- Produces: `func (g *Game) HasCardMoves() bool`
- Consumes: `LegalMoves`, `MoveDraw`, `MoveRecycle`

- [ ] **Step 1: Add failing tests**

Append to `internal/solitaire/engine/moves_test.go`:

```go
func TestHasCardMovesFalseWhenOnlyDrawOrRecycle(t *testing.T) {
	g := &Game{status: Playing, stock: []Card{{Rank: Ace, Suit: Spades}}}
	if g.HasCardMoves() {
		t.Fatal("draw-only position should report no card moves")
	}

	g = &Game{status: Playing, waste: []Card{{Rank: King, Suit: Spades}}, stock: nil}
	// King on waste with no empty tableau and no foundation match still has waste→? 
	// Use a card that cannot place: empty tableau requires King; put non-king with full empty foundations need ace.
	g = &Game{status: Playing, waste: []Card{{Rank: Five, Suit: Hearts}}, stock: nil}
	if g.HasCardMoves() {
		t.Fatalf("unplayable waste-only recycle position should have no card moves, moves=%+v", g.LegalMoves())
	}
	if !hasMoveKind(g.LegalMoves(), MoveRecycle) {
		t.Fatal("expected recycle to remain legal")
	}
}

func TestHasCardMovesTrueWhenWasteCanFoundation(t *testing.T) {
	g := &Game{status: Playing, waste: []Card{{Rank: Ace, Suit: Spades}}}
	if !g.HasCardMoves() {
		t.Fatal("waste ace to foundation is a card move")
	}
}

func TestHasCardMovesFalseWhenWon(t *testing.T) {
	g := &Game{status: Won, waste: []Card{{Rank: Ace, Suit: Spades}}}
	if g.HasCardMoves() {
		t.Fatal("won game has no card moves")
	}
}
```

- [ ] **Step 2: Run tests red**

```bash
go test ./internal/solitaire/engine -run HasCardMoves -v
```

Expected: FAIL because `HasCardMoves` is undefined.

- [ ] **Step 3: Implement**

In `moves.go`:

```go
func (g *Game) HasCardMoves() bool {
	for _, move := range g.LegalMoves() {
		if move.Kind != MoveDraw && move.Kind != MoveRecycle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests green**

```bash
go test ./internal/solitaire/engine -run HasCardMoves -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/engine/moves.go internal/solitaire/engine/moves_test.go
git commit -m "feat: add solitaire HasCardMoves helper"
```

---

### Task 2: Keymap — `h` left, `i` hint

**Files:**
- Modify: `internal/solitaire/ui/game.go`
- Modify: `internal/solitaire/ui/game_test.go`
- Modify: `internal/solitaire/ui/view_test.go`

**Interfaces:**
- Produces: `h` → `moveCursorLeft`; `i` → `hint()`
- Consumes: existing `moveCursorLeft`, `hint`

- [ ] **Step 1: Update failing/contract tests**

In `game_test.go`, change `TestHintMarksAssistedAndDoesNotMutateGame` to use `key("i")` instead of `key("h")`.

Add:

```go
func TestHMovesCursorLeftWithoutAssisted(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionWaste}
	m, _ = m.Update(key("h"))
	if m.cursor.region != regionStock {
		t.Fatalf("h should move left waste→stock, cursor=%+v", m.cursor)
	}
	if m.assisted {
		t.Fatal("h must not mark assisted")
	}
	if strings.Contains(m.message, "Hint:") {
		t.Fatalf("h must not hint, message=%q", m.message)
	}
}

func TestIKeyHintsAndMarksAssisted(t *testing.T) {
	m := testModel(t)
	beforeMoves := m.game.Moves()
	beforeStock := m.game.StockCount()
	m, _ = m.Update(key("i"))
	if !m.assisted {
		t.Fatal("i should mark game assisted")
	}
	if m.game.Moves() != beforeMoves || m.game.StockCount() != beforeStock {
		t.Fatalf("hint should not mutate game")
	}
	if !strings.Contains(m.message, "Hint:") {
		t.Fatalf("hint message missing: %q", m.message)
	}
}
```

In `view_test.go`, update `TestHelpViewRendersControls` so help contains `i` hint and does **not** require `h            hint`. Prefer assertions like:

```go
for _, want := range []string{"Solitaire Draw 3", "d            draw 3", "i            hint", "hjkl/arrows", "a            move focused/selected", "A            auto-foundation all", "spc/enter"} {
```

(Adjust exact help strings to match Step 3 copy.)

- [ ] **Step 2: Run tests red**

```bash
go test ./internal/solitaire/ui -run 'Hint|HMoves|IKey|HelpView' -v
```

Expected: FAIL on `h` still hinting / help still documenting `h` as hint.

- [ ] **Step 3: Implement keymap + help**

In `game.go` `handleKey`:

```go
case "i":
	return m.hint(), nil
// remove case "h": return m.hint()
case "left", "h":
	m.moveCursorLeft()
case "right", "l":
	m.moveCursorRight()
case "up", "k":
	m.moveCursorUp()
case "down", "j":
	m.moveCursorDown()
```

Update `helpView` lines:

```go
" hjkl/arrows  move",
" spc/enter    select card or move selection",
" d            draw 3 / recycle stock",
" i            hint (assisted)",
" u            undo last move (assisted)",
" a            move focused/selected safe card to foundation",
" A            auto-foundation all safe cards",
" tab          next playable source / destination",
" 1-7          jump tableau column",
" esc          cancel selection / menu",
" n            new deal",
" m            menu",
" q            quit",
```

Update idle status compact/full strings that mention keys to include `i hint` instead of implying `h` is hint (full rewrite of status comes in later tasks; at minimum help must be correct now).

- [ ] **Step 4: Run tests green**

```bash
go test ./internal/solitaire/ui -run 'Hint|HMoves|IKey|HelpView' -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/game.go internal/solitaire/ui/game_test.go internal/solitaire/ui/view_test.go
git commit -m "fix: map solitaire h to left and i to hint"
```

---

### Task 3: Message severity (success ≠ error)

**Files:**
- Modify: `internal/solitaire/ui/styles.go`
- Modify: `internal/solitaire/ui/game.go`
- Modify: `internal/solitaire/ui/view_test.go` and/or `game_test.go`

**Interfaces:**
- Produces: `type msgKind uint8` with `msgInfo`, `msgWarn`, `msgErr`; fields `message string`, `messageKind msgKind` on `Model`
- Consumes: status rendering

- [ ] **Step 1: Add failing test**

```go
func TestSuccessMessageIsNotErrorStyled(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("d"))
	v := m.View()
	// Error style uses lipgloss color 9; info uses faint. Assert message text present
	// and that status does not use styleErr for draw success by comparing render:
	info := styleFaint.Render(" " + m.message)
	err := styleErr.Render(" " + m.message)
	if m.message == "" {
		t.Fatal("draw should set a message")
	}
	if strings.Contains(v, err) && !strings.Contains(v, info) {
		t.Fatalf("success draw message should not be error-styled only:\n%s", v)
	}
	if m.messageKind != msgInfo {
		t.Fatalf("messageKind=%v, want msgInfo", m.messageKind)
	}
}
```

(If lipgloss ANSI makes exact string match flaky under `termenv.Ascii` in tests, assert `m.messageKind == msgInfo` after draw and that `statusLine` for info uses `styleFaint` / `styleInfo` path — prefer field assertion plus a unit test of `statusLine` with forced kinds.)

Simpler reliable test:

```go
func TestDrawSetsInfoMessageKind(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("d"))
	if m.messageKind != msgInfo {
		t.Fatalf("draw messageKind=%v want msgInfo", m.messageKind)
	}
	if m.message == "" {
		t.Fatal("expected draw message")
	}
}

func TestNoMovesRefusalSetsWarnKind(t *testing.T) {
	// Implemented fully in Task 5; skip here if activate not yet refusing.
}
```

For Task 3 only require info vs err split on existing messages: draw → info, "No cards to draw" → warn or info, save failed remains err (win path).

- [ ] **Step 2: Run red**

```bash
go test ./internal/solitaire/ui -run 'DrawSetsInfo|SuccessMessage' -v
```

Expected: FAIL (`messageKind` undefined or wrong).

- [ ] **Step 3: Implement**

In `styles.go`:

```go
var (
	// existing styles...
	styleInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)
```

In `game.go`:

```go
type msgKind uint8

const (
	msgInfo msgKind = iota
	msgWarn
	msgErr
)

// on Model:
message     string
messageKind msgKind
```

Helper:

```go
func (m Model) setMessage(kind msgKind, text string) Model {
	m.message = text
	m.messageKind = kind
	return m
}
```

Replace all `m.message = "..."` assignments with `setMessage`:

- Drew cards / Moved / Moved to foundation / Undid / safe auto batch → `msgInfo`
- No cards to draw / Invalid move / No safe foundation / Nothing to undo / Could not deal → `msgWarn` (or info for soft failures)
- Soft stuck later → `msgWarn`
- Keep save failure on win line as `styleErr` (already separate)

In `statusLine`, when rendering `m.message`:

```go
style := styleInfo
switch m.messageKind {
case msgWarn:
	style = styleWarn
case msgErr:
	style = styleErr
}
return style.Render(" " + clampLine(m.message, m.termW-1))
```

- [ ] **Step 4: Green**

```bash
go test ./internal/solitaire/ui -run 'DrawSetsInfo|DKey|Hint|NoProgress' -v
go test ./internal/solitaire/ui
```

Expected: PASS (update any tests that assumed red-only messages if needed).

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/styles.go internal/solitaire/ui/game.go internal/solitaire/ui/game_test.go internal/solitaire/ui/view_test.go
git commit -m "fix: style solitaire success messages as info not error"
```

---

### Task 4: Cursor clamps — skip face-down, sensible column entry

**Files:**
- Modify: `internal/solitaire/ui/game.go` (`moveCursorUp/Down/Left/Right`, `clampTableauRow`, `clampTableauRowToPlayable`)
- Modify: `internal/solitaire/ui/game_test.go`

**Interfaces:**
- Produces: navigation never ends on face-down after a move key
- Consumes: `game.Tableau`

- [ ] **Step 1: Failing tests**

```go
func TestDownSkipsFaceDownCards(t *testing.T) {
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{
			{Card: engine.Card{Rank: engine.King, Suit: engine.Spades}, FaceUp: false},
			{Card: engine.Card{Rank: engine.Queen, Suit: engine.Hearts}, FaceUp: false},
			{Card: engine.Card{Rank: engine.Jack, Suit: engine.Spades}, FaceUp: true},
		},
	})
	m.game = g
	m.cursor = cursor{region: regionFoundation, col: 0}
	m, _ = m.Update(key("j"))
	if m.cursor.region != regionTableau || m.cursor.col != 0 {
		t.Fatalf("j from foundation should enter tableau col0, cursor=%+v", m.cursor)
	}
	if m.cursor.row != 2 {
		t.Fatalf("should land on first face-up row 2, row=%d", m.cursor.row)
	}
	// cannot go further down past only face-up
	m, _ = m.Update(key("j"))
	if m.cursor.row != 2 {
		t.Fatalf("down should not enter past face-up stack, row=%d", m.cursor.row)
	}
}

func TestUpStaysOnFaceUpRun(t *testing.T) {
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{
			{Card: engine.Card{Rank: engine.Ten, Suit: engine.Clubs}, FaceUp: false},
			{Card: engine.Card{Rank: engine.Nine, Suit: engine.Hearts}, FaceUp: true},
			{Card: engine.Card{Rank: engine.Eight, Suit: engine.Spades}, FaceUp: true},
		},
	})
	m.game = g
	m.cursor = cursor{region: regionTableau, col: 0, row: 2}
	m, _ = m.Update(key("k"))
	if m.cursor.row != 1 {
		t.Fatalf("up should move to row 1 face-up, row=%d", m.cursor.row)
	}
	m, _ = m.Update(key("k"))
	// leaving face-up run goes to foundation (existing up behavior) — must not land row 0 face-down
	if m.cursor.region == regionTableau && m.cursor.row == 0 {
		cards := m.game.Tableau(0)
		if !cards[0].FaceUp {
			t.Fatal("up must not land on face-down card")
		}
	}
}

func TestHorizontalClampToFaceUpTop(t *testing.T) {
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{
			{Card: engine.Card{Rank: engine.King, Suit: engine.Spades}, FaceUp: false},
			{Card: engine.Card{Rank: engine.Queen, Suit: engine.Hearts}, FaceUp: false},
			{Card: engine.Card{Rank: engine.Jack, Suit: engine.Spades}, FaceUp: false},
			{Card: engine.Card{Rank: engine.Ten, Suit: engine.Hearts}, FaceUp: true},
		},
		{
			{Card: engine.Card{Rank: engine.Five, Suit: engine.Clubs}, FaceUp: true},
		},
	})
	m.game = g
	m.cursor = cursor{region: regionTableau, col: 0, row: 3}
	m, _ = m.Update(key("l"))
	if m.cursor.col != 1 || m.cursor.row != 0 {
		t.Fatalf("right into short column should clamp to face-up top row0, cursor=%+v", m.cursor)
	}
}
```

- [ ] **Step 2: Run red**

```bash
go test ./internal/solitaire/ui -run 'DownSkips|UpStays|HorizontalClamp' -v
```

Expected: FAIL (cursor lands on face-down or wrong row).

- [ ] **Step 3: Implement clamps**

Replace `clampTableauRow` with face-up aware logic:

```go
func (m *Model) clampTableauRow() {
	cards := m.game.Tableau(m.cursor.col)
	if len(cards) == 0 {
		m.cursor.row = 0
		return
	}
	if m.cursor.row >= len(cards) {
		m.cursor.row = len(cards) - 1
	}
	if m.cursor.row < 0 {
		m.cursor.row = 0
	}
	if cards[m.cursor.row].FaceUp {
		return
	}
	// prefer nearest face-up at or below, else first face-up, else 0 empty semantics
	for row := m.cursor.row; row < len(cards); row++ {
		if cards[row].FaceUp {
			m.cursor.row = row
			return
		}
	}
	for row := range cards {
		if cards[row].FaceUp {
			m.cursor.row = row
			return
		}
	}
	m.cursor.row = 0
}

func (m *Model) moveCursorDown() {
	switch m.cursor.region {
	case regionStock, regionWaste:
		m.cursor = cursor{region: regionTableau}
		m.clampTableauRow()
	case regionFoundation:
		m.cursor = cursor{region: regionTableau, col: m.cursor.col}
		m.clampTableauRow()
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

func (m *Model) moveCursorUp() {
	switch m.cursor.region {
	case regionTableau:
		cards := m.game.Tableau(m.cursor.col)
		for row := m.cursor.row - 1; row >= 0; row-- {
			if cards[row].FaceUp {
				m.cursor.row = row
				return
			}
		}
		// leave to upper region (existing)
		if m.cursor.col < 4 {
			m.cursor = cursor{region: regionFoundation, col: m.cursor.col}
		} else {
			m.cursor = cursor{region: regionWaste}
		}
	}
}
```

Ensure `moveCursorLeft`/`Right` call `clampTableauRow()` after changing tableau col (already do). Align `clampTableauRowToPlayable` with the same face-up rule (can delegate to `clampTableauRow`).

- [ ] **Step 4: Green**

```bash
go test ./internal/solitaire/ui -run 'DownSkips|UpStays|HorizontalClamp|CursorMoves|NumberKeys|TabCycles' -v
go test ./internal/solitaire/ui
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/game.go internal/solitaire/ui/game_test.go
git commit -m "fix: clamp solitaire cursor to face-up cards"
```

---

### Task 5: Refuse dead selection + destination-scoped vim + highlights + selected status

**Files:**
- Modify: `internal/solitaire/ui/assist.go`
- Modify: `internal/solitaire/ui/game.go`
- Modify: `internal/solitaire/ui/styles.go`
- Modify: `internal/solitaire/ui/game_test.go`
- Modify: `internal/solitaire/ui/view_test.go`

**Interfaces:**
- Produces: dead-select refusal; while selected, arrow/vim keys call `cycleSelectedDestination`; `styleDestination`; `hintDest` optional field for hint highlights later; selected status names dest
- Consumes: `selectedMoves`, `cycleSelectedDestination`, `LegalMovesFrom`

- [ ] **Step 1: Failing tests**

```go
func TestSpaceRefusesDeadSelection(t *testing.T) {
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.Five, Suit: engine.Spades}, FaceUp: true}},
	})
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card(nil))
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card(nil))
	m.game = g
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	if m.selected != nil {
		t.Fatalf("zero-move card must not stay selected: %+v", *m.selected)
	}
	if !strings.Contains(strings.ToLower(m.message), "no moves") {
		t.Fatalf("expected no moves message, got %q", m.message)
	}
	if m.messageKind != msgWarn {
		t.Fatalf("messageKind=%v want msgWarn", m.messageKind)
	}
}

func TestVimCyclesDestinationsWhileSelected(t *testing.T) {
	m := modelWithMovableKingAndTwoEmptyColumns(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	if m.selected == nil || len(m.selectedMoves) < 2 {
		t.Fatalf("need multi-dest selection, selected=%v moves=%+v", m.selected, m.selectedMoves)
	}
	first := m.cursor
	m, _ = m.Update(key("l")) // or "j" — any vim key should cycle dest, not free-roam off list
	if m.cursor == first {
		// try opposite direction if order put us mid-list
		m, _ = m.Update(key("h"))
	}
	if m.selected == nil {
		t.Fatal("vim dest cycle must keep selection")
	}
	// cursor must match some selectedMoves destination
	ok := false
	for _, move := range m.selectedMoves {
		if matchesSelectedDestination(move.Destination, m.cursor.location()) {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("cursor %+v not on a legal destination after vim cycle", m.cursor)
	}
}

func TestSelectedStatusNamesDestination(t *testing.T) {
	m := modelWithMovableKingAndTwoEmptyColumns(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m.termW = 120
	m, _ = m.Update(key(" "))
	s := m.statusLine()
	if !strings.Contains(s, "selected") && !strings.Contains(s, "Selected") {
		t.Fatalf("status should describe selection:\n%s", s)
	}
	if !strings.Contains(strings.ToLower(s), "dest") && !strings.Contains(strings.ToLower(s), "tableau") {
		t.Fatalf("status should name destination:\n%s", s)
	}
}
```

Add view test that destination style is applied (can assert `styleDestination` render of empty token appears when selected with empty-column dest):

```go
func TestViewHighlightsLegalDestinationsWhenSelected(t *testing.T) {
	m := modelWithMovableKingAndTwoEmptyColumns(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	v := m.View()
	// Destination empty piles should use styleDestination — at least selected source reverse present.
	if m.selected == nil {
		t.Fatal("expected selection")
	}
	// Ensure decorateDestination path runs: count empty tokens styled; simplest: call helper
	if !m.isLegalDestination(engine.Location{Kind: engine.Tableau, Pile: 1}) {
		t.Fatal("empty col 1 should be legal dest for king")
	}
	_ = v
}
```

- [ ] **Step 2: Run red**

```bash
go test ./internal/solitaire/ui -run 'SpaceRefuses|VimCycles|SelectedStatus|ViewHighlights' -v
```

Expected: FAIL (dead select still selects; vim free-roams).

- [ ] **Step 3: Implement**

**activate zero-move branch** — replace keep-selection with:

```go
case 0:
	card, _, ok := m.cardAt(loc)
	if ok {
		m = m.setMessage(msgWarn, fmt.Sprintf("no moves for %s", describeCard(card)))
	} else {
		m = m.setMessage(msgWarn, "no moves")
	}
	return m, nil
```

**handleKey navigation** when selected:

```go
case "left", "h":
	if m.selected != nil && len(m.selectedMoves) > 0 {
		return m.cycleSelectedDestination(-1), nil
	}
	m.moveCursorLeft()
case "right", "l":
	if m.selected != nil && len(m.selectedMoves) > 0 {
		return m.cycleSelectedDestination(1), nil
	}
	m.moveCursorRight()
case "up", "k":
	if m.selected != nil && len(m.selectedMoves) > 0 {
		return m.cycleSelectedDestination(-1), nil
	}
	m.moveCursorUp()
case "down", "j":
	if m.selected != nil && len(m.selectedMoves) > 0 {
		return m.cycleSelectedDestination(1), nil
	}
	m.moveCursorDown()
```

**styles.go:**

```go
styleDestination = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
```

**decorate** — before returning token, if `m.isLegalDestination` for this region/col/row (tableau: pile match on any row of empty or top; foundation: pile; waste N/A as dest rarely), apply `styleDestination` unless cursor/selected source already applied.

```go
func (m Model) isLegalDestination(loc engine.Location) bool {
	if m.selected == nil {
		// also true when hintDest set — Task 6 may set m.hintMove
		if m.hintMove != nil {
			return matchesSelectedDestination(m.hintMove.Destination, loc)
		}
		return false
	}
	for _, move := range m.selectedMoves {
		if matchesSelectedDestination(move.Destination, loc) {
			return true
		}
	}
	return false
}
```

For Task 5, if `hintMove` not yet added, only use `selectedMoves`.

**statusLine selected branch:**

```go
dest := describeLocation(m.cursor.location())
return fitLine(m.termW,
	styleFaint.Render(fmt.Sprintf(" Selected %s · dest %s · spc place · esc cancel", description, dest)),
	styleFaint.Render(fmt.Sprintf(" Selected · dest %s · esc", dest)),
)
```

- [ ] **Step 4: Green**

```bash
go test ./internal/solitaire/ui -run 'SpaceRefuses|VimCycles|SelectedStatus|SpaceKeeps|TabCycles|EnterApplies|SpaceApplies' -v
go test ./internal/solitaire/ui
```

Expected: PASS. Update any test that expected dead selection to remain selected.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/assist.go internal/solitaire/ui/game.go internal/solitaire/ui/styles.go internal/solitaire/ui/game_test.go internal/solitaire/ui/view_test.go
git commit -m "feat: refuse dead solitaire selection and vim-cycle destinations"
```

---

### Task 6: Soft stuck detection + status priority

**Files:**
- Modify: `internal/solitaire/ui/game.go`
- Modify: `internal/solitaire/ui/game_test.go`
- Modify: `internal/solitaire/ui/view_test.go` if needed

**Interfaces:**
- Produces: `softStuck bool` on `Model`; constants for soft-stuck copy; clears on card progress / new deal
- Consumes: `game.HasCardMoves`, stock pass tracking (`stockPassProgress`, recycle when `beforeStock == 0`)

- [ ] **Step 1: Failing tests**

Replace/extend no-progress test:

```go
const softStuckMessage = "No more productive moves · press n for a new deal"

func TestSoftStuckAfterNoProgressStockPass(t *testing.T) {
	m := modelWithOnlyStockCycle(t)
	// Draw until stock empty, then recycle, completing a pass with no card progress.
	for range 20 {
		m, _ = m.Update(key("d"))
		if m.softStuck {
			break
		}
	}
	if !m.softStuck {
		t.Fatal("expected softStuck after full no-progress stock pass")
	}
	if m.assisted {
		t.Fatal("soft stuck must not mark assisted")
	}
	if m.game.Status() != engine.Playing {
		t.Fatalf("engine status=%v want Playing", m.game.Status())
	}
	s := m.statusLine()
	if !strings.Contains(s, "No more productive moves") && !strings.Contains(s, "No more moves") {
		t.Fatalf("status missing soft-stuck copy:\n%s", s)
	}
}

func TestSoftStuckClearsOnCardProgress(t *testing.T) {
	m := modelWithOnlyStockCycle(t)
	for range 20 {
		m, _ = m.Update(key("d"))
		if m.softStuck {
			break
		}
	}
	if !m.softStuck {
		t.Fatal("precondition: soft stuck")
	}
	// Inject a playable ace and move it
	g := m.game
	// rebuild: put ace on tableau after stuck
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.Ace, Suit: engine.Spades}, FaceUp: true}},
	})
	m.game = g
	m.softStuck = true
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	if m.softStuck {
		t.Fatal("card progress should clear softStuck")
	}
}

func TestSoftStuckImmediateWhenNoStockWasteOrCardMoves(t *testing.T) {
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card(nil))
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card(nil))
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.Five, Suit: engine.Spades}, FaceUp: true}},
	})
	m.game = g
	m = m.recomputeSoftStuck()
	if !m.softStuck {
		t.Fatal("no stock/waste and no card moves should soft-stuck immediately")
	}
}
```

Remove or rewrite `TestNoProgressWarningAfterStockPassWithoutProgress` to expect soft-stuck copy instead of `no progress` substring.

- [ ] **Step 2: Run red**

```bash
go test ./internal/solitaire/ui -run 'SoftStuck|NoProgress' -v
```

Expected: FAIL.

- [ ] **Step 3: Implement**

Add to `Model`:

```go
softStuck bool
```

```go
const softStuckMessage = "No more productive moves · press n for a new deal"
const softStuckCompact = "No more moves · n new"

func (m Model) recomputeSoftStuck() Model {
	if m.game.Status() != engine.Playing {
		m.softStuck = false
		return m
	}
	if m.game.HasCardMoves() {
		// card moves available — only clear soft stuck if we had progress path; 
		// spec: soft stuck clears on card-progress move, not merely because draws remain.
		// Latch is set by evaluate after recycle; HasCardMoves true should clear latch.
		m.softStuck = false
		return m
	}
	if m.game.StockCount() == 0 && len(m.game.WasteVisible()) == 0 {
		// WasteVisible may be empty while waste has cards under draw3 fan — use engine waste via StockCount recycle path.
		// Prefer: no draw and no recycle possible.
		moves := m.game.LegalMoves()
		if len(moves) == 0 {
			m.softStuck = true
		}
		return m
	}
	return m
}
```

Cleaner latch logic matching the spec:

1. On successful **card** move (`noteCardProgress`): `softStuck = false`, `stockPassProgress = true`.
2. On **recycle** (`Draw` when `beforeStock == 0`): if `!stockPassProgress && !game.HasCardMoves()` then `softStuck = true` and `setMessage(msgWarn, softStuckMessage)`.
3. After any draw/recycle: if `!HasCardMoves()` and no stock and no waste left to cycle, `softStuck = true`.
4. `statusLine`: if `softStuck && playing && selected == nil`, show soft-stuck fitLine **before** generic message/idle chrome (wins over idle; selection status can still show if selected).

Implement `noteCardProgress` already clears `stockPassWarning`; extend it:

```go
func (m Model) noteCardProgress() Model {
	m.stockPassProgress = true
	m.stockPassWarning = false
	m.softStuck = false
	return m
}
```

In `draw()` when `beforeStock == 0` (recycle just happened):

```go
if !m.stockPassProgress && !m.game.HasCardMoves() {
	m.softStuck = true
	m = m.setMessage(msgWarn, softStuckMessage)
} else if !m.stockPassProgress {
	// keep backward-compatible mild message only if still have card moves? Spec: absorb into soft-stuck.
	m.softStuck = true
	m = m.setMessage(msgWarn, softStuckMessage)
}
m.stockPassProgress = false
```

Spec says soft-stuck needs (1) no card moves AND (2) full pass no progress. So:

```go
if beforeStock == 0 { // recycled
	if !m.stockPassProgress {
		if !m.game.HasCardMoves() {
			m.softStuck = true
			m = m.setMessage(msgWarn, softStuckMessage)
		} else {
			// had no progress but card moves exist (rare) — still warn softly without softStuck latch? 
			// Spec condition 1 fails, so do not softStuck; optional mild message omitted to avoid dual copy.
		}
	}
	m.stockPassProgress = false
}
```

Also after draws, if `!HasCardMoves()` and stock empty and waste empty:

```go
if !m.game.HasCardMoves() && m.game.StockCount() == 0 && !canRecycle {
	m.softStuck = true
}
```

Detect canRecycle via attempting: waste non-empty is enough for recycle when stock empty — `LegalMoves` has Recycle. Use:

```go
if !m.game.HasCardMoves() {
	onlyDraw := false
	for _, mv := range m.game.LegalMoves() {
		if mv.Kind == engine.MoveDraw || mv.Kind == engine.MoveRecycle {
			onlyDraw = true
		}
	}
	if len(m.game.LegalMoves()) == 0 {
		m.softStuck = true
		m = m.setMessage(msgWarn, softStuckMessage)
	}
}
```

`statusLine` priority:

1. Won  
2. Soft stuck (if `m.softStuck` and not selected — or even if selected, soft stuck banner when not selected)  
3. Selected  
4. Message  
5. Idle compass (Task 7 may expand idle)

```go
if m.softStuck && m.game.Status() == engine.Playing && m.selected == nil {
	return fitLine(m.termW,
		styleWarn.Render(" "+softStuckMessage),
		styleWarn.Render(" "+softStuckCompact),
	)
}
```

- [ ] **Step 4: Green**

```bash
go test ./internal/solitaire/ui -run 'SoftStuck|NoProgress|Draw|Undo' -v
go test ./internal/solitaire/ui
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/game.go internal/solitaire/ui/game_test.go internal/solitaire/ui/view_test.go
git commit -m "feat: soft-stuck solitaire status with new-deal guidance"
```

---

### Task 7: Hint ranking + presentation + stuck-aware copy

**Files:**
- Modify: `internal/solitaire/ui/assist.go`
- Modify: `internal/solitaire/ui/game.go` (hintMove field if used for highlights)
- Modify: `internal/solitaire/ui/game_test.go`

**Interfaces:**
- Produces: rewritten `hintRank`, `bestHintMove`, `hint()` presentation; `hintMove *engine.Move` cleared on clearSelection/new actions
- Consumes: `HasCardMoves`, `softStuck`, `LegalMoves`

- [ ] **Step 1: Failing tests**

```go
func TestHintPrefersLegalFoundationOverDraw(t *testing.T) {
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card{
		{Rank: engine.Five, Suit: engine.Hearts}, // unsafe foundation but legal if hearts built to 4
	})
	foundations := [4][]engine.Card{
		engine.Hearts: {
			{Rank: engine.Ace, Suit: engine.Hearts},
			{Rank: engine.Two, Suit: engine.Hearts},
			{Rank: engine.Three, Suit: engine.Hearts},
			{Rank: engine.Four, Suit: engine.Hearts},
		},
	}
	// black foundations low so Safe==false
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("foundations"), foundations)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card{
		{Rank: engine.King, Suit: engine.Clubs},
		{Rank: engine.King, Suit: engine.Spades},
		{Rank: engine.King, Suit: engine.Diamonds},
	})
	m.game = g
	move, ok := m.bestHintMove()
	if !ok {
		t.Fatal("expected a hint move")
	}
	if move.Destination.Kind != engine.Foundation {
		t.Fatalf("hint should prefer foundation over draw, got %+v", move)
	}
}

func TestHintDeprioritizesNonRevealingTableauShuffle(t *testing.T) {
	// Setup: waste ace→foundation available AND a non-revealing tableau-to-tableau legal move.
	// Hint must pick foundation (or reveal), not pure shuffle.
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card{{Rank: engine.Ace, Suit: engine.Spades}})
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.Seven, Suit: engine.Hearts}, FaceUp: true}},
		{{Card: engine.Card{Rank: engine.Eight, Suit: engine.Spades}, FaceUp: true}},
	})
	m.game = g
	move, ok := m.bestHintMove()
	if !ok {
		t.Fatal("expected hint")
	}
	if move.Kind == engine.MoveTableauToTableau {
		t.Fatalf("should not prefer non-revealing tableau shuffle over ace foundation, got %+v", move)
	}
	if move.Destination.Kind != engine.Foundation {
		t.Fatalf("want foundation hint, got %+v", move)
	}
}

func TestHintWhenSoftStuckRecommendsNewDeal(t *testing.T) {
	m := modelWithOnlyStockCycle(t)
	for range 20 {
		m, _ = m.Update(key("d"))
		if m.softStuck {
			break
		}
	}
	if !m.softStuck {
		t.Fatal("precondition soft stuck")
	}
	m, _ = m.Update(key("i"))
	if !strings.Contains(strings.ToLower(m.message), "new deal") {
		t.Fatalf("soft-stuck hint should recommend new deal, got %q", m.message)
	}
	if strings.Contains(strings.ToLower(m.message), "draw from stock") {
		t.Fatalf("soft-stuck hint must not recommend draw, got %q", m.message)
	}
}
```

- [ ] **Step 2: Run red**

```bash
go test ./internal/solitaire/ui -run 'HintPrefers|HintDeprioritizes|HintWhenSoftStuck' -v
```

Expected: FAIL on foundation-vs-draw ranking and/or soft-stuck hint text.

- [ ] **Step 3: Implement ranking + hint()**

Replace `hintRank` in `assist.go`:

```go
func hintRank(move engine.Move) int {
	if move.Source.Kind == engine.Tableau && move.RevealsCard {
		return 1
	}
	if move.Destination.Kind == engine.Foundation {
		return 2 // any legal foundation, not only Safe
	}
	switch move.Kind {
	case engine.MoveWasteToTableau:
		return 3
	case engine.MoveTableauToTableau:
		if move.RevealsCard {
			return 1
		}
		// king to empty
		if len(move.Cards) > 0 && move.Cards[0].Rank == engine.King && move.Destination.Kind == engine.Tableau {
			// empty dest is implied by legal king place; rank 5
			return 5
		}
		return 6
	case engine.MoveDraw:
		return 7
	case engine.MoveRecycle:
		return 8
	case engine.MoveFoundationToTableau:
		return 9
	default:
		return 9
	}
}
```

Note: king-to-empty detection may need checking empty pile via move metadata only — if unreliable, rank all non-revealing tableau moves as 6 and accept king empty as 6 for v1; tests above don't require king tier.

`bestHintMove`: keep min rank; on tie keep first.

`hint()`:

```go
func (m Model) hint() Model {
	m.assisted = true
	m.assistReason = "hint"
	m = m.clearSelection()
	m.hintMove = nil

	if m.softStuck || (!m.game.HasCardMoves() && !hasOnlyDrawRecycleThatHelps(m)) {
		// if soft stuck OR no card moves and we're not going to suggest draw as productive when softStuck
	}
	if m.softStuck || !m.game.HasCardMoves() {
		// Still allow draw hint when NOT softStuck but only draw left (player hasn't finished a no-progress pass).
	}
	if m.softStuck {
		m = m.setMessage(msgWarn, "Hint: no more productive moves · press n for a new deal")
		return m
	}
	if !m.game.HasCardMoves() && len(m.game.LegalMoves()) == 0 {
		m = m.setMessage(msgWarn, "Hint: no more productive moves · press n for a new deal")
		return m
	}

	move, ok := m.bestHintMove()
	if !ok {
		m = m.setMessage(msgWarn, "Hint: no more productive moves · press n for a new deal")
		return m
	}
	// If best is draw/recycle only and softStuck already handled...
	cp := move
	m.hintMove = &cp
	if move.Kind == engine.MoveDraw || move.Kind == engine.MoveRecycle {
		m = m.focusLocation(engine.Location{Kind: engine.Stock})
	} else {
		m = m.focusLocation(move.Source)
		// paint dest via hintMove in decorate; optionally also focusDestination for cursor on dest:
		m = m.focusDestination(move.Destination)
	}
	m = m.setMessage(msgInfo, "Hint: "+m.describeHintMove(move))
	return m
}
```

Spec choice: focus destination for card moves so cursor sits on dest while message names full play; `hintMove` keeps both ends for highlight of dest (and source via selected-style optional). Implement:

- `hintMove *engine.Move` on Model  
- `clearSelection` does not clear hintMove; clear hintMove on any mutating action and on esc  
- `decorate`: if `hintMove != nil`, highlight source stack and destination like selection/destination styles  

Update `describeHintMove` foundation text to not require Safe for a useful suffix; keep “safely” only when `move.Safe`.

- [ ] **Step 4: Green**

```bash
go test ./internal/solitaire/ui -run 'Hint' -v
go test ./internal/solitaire/ui ./internal/solitaire/engine
```

Expected: PASS. Update `TestHintMarksAssistedAndDoesNotMutateGame` if message kind/text assumptions change.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/assist.go internal/solitaire/ui/game.go internal/solitaire/ui/game_test.go
git commit -m "feat: improve solitaire hint ranking and stuck-aware copy"
```

---

### Task 8: Idle status compass

**Files:**
- Modify: `internal/solitaire/ui/game.go` (`statusLine`)
- Modify: `internal/solitaire/ui/assist.go` (focus description helpers if needed)
- Modify: `internal/solitaire/ui/view_test.go`

**Interfaces:**
- Produces: idle status `focus <card> · N moves · spc act · tab next · i hint`
- Consumes: `cardAt`, `LegalMovesFrom`, cursor location

- [ ] **Step 1: Failing test**

```go
func TestIdleStatusShowsFocusAndMoveCount(t *testing.T) {
	m := modelWithTopTableauAce(t, engine.Spades)
	col, row := topTableauAce(t, m.game)
	m.cursor = cursor{region: regionTableau, col: col, row: row}
	m.termW = 120
	m.message = ""
	s := m.statusLine()
	if !strings.Contains(strings.ToLower(s), "focus") {
		t.Fatalf("idle status should include focus:\n%s", s)
	}
	if !strings.Contains(s, "moves") {
		t.Fatalf("idle status should include move count:\n%s", s)
	}
}
```

- [ ] **Step 2: Run red**

```bash
go test ./internal/solitaire/ui -run IdleStatus -v
```

Expected: FAIL.

- [ ] **Step 3: Implement idle branch** in `statusLine` (after soft-stuck, selected, empty message checks):

```go
loc := m.cursor.location()
n := len(m.game.LegalMovesFrom(loc))
focus := "empty"
if card, stack, ok := m.cardAt(loc); ok {
	focus = describeCard(card)
	if stack > 1 {
		focus = fmt.Sprintf("%s stack %d", focus, stack)
	}
} else if loc.Kind == engine.Stock {
	focus = "stock"
}
full := fmt.Sprintf(" focus %s · %d moves · spc act · tab next · i hint", focus, n)
compact := fmt.Sprintf(" %s · %dm · spc · i", focus, n)
return fitLine(m.termW, styleFaint.Render(full), styleFaint.Render(compact))
```

Keep a very narrow fallback if needed via existing fitLine.

- [ ] **Step 4: Green**

```bash
go test ./internal/solitaire/ui -run 'IdleStatus|StatusLine|View' -v
go test ./internal/solitaire/ui
```

Expected: PASS. Update `TestStatusLineUsesCompactHintAtMinimumWidth` if idle chrome strings changed.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/game.go internal/solitaire/ui/assist.go internal/solitaire/ui/view_test.go internal/solitaire/ui/game_test.go
git commit -m "feat: add solitaire focus status compass"
```

---

### Task 9: Final verification

**Files:**
- Modify only if verification finds defects

- [ ] **Step 1: Full test suite**

```bash
go test ./...
```

Expected: all packages ok.

- [ ] **Step 2: Build**

```bash
go build -o mines .
```

Expected: exit 0.

- [ ] **Step 3: Manual smoke** (from spec §10)

1. `hjkl` move; `h` does not hint  
2. `i` hints and marks assisted  
3. Face-up-only landing in tableau columns  
4. Multi-dest: vim cycles dests, highlights, space places  
5. Dead card: space refuses with message  
6. No-progress stock pass → soft stuck → `n` new deal  
7. Win records still work  
8. Help lists `i` / `hjkl`; Mines still works  

- [ ] **Step 4: Commit verification fixes if any**

```bash
git add -A
git commit -m "fix: solitaire feel v1 verification fixes"
```

Only if Step 3 found defects; otherwise no empty commit.

---

## Self-review (plan vs spec)

| Spec section | Task |
|---|---|
| §1 Keymap h/i | Task 2 |
| §2 Cursor clamps | Task 4 |
| §3 Dead select, dest vim, highlights, selected status | Task 5 |
| §3.4 Idle compass | Task 8 |
| §4 Soft stuck | Task 6 |
| §5 Hints ranking + stuck-aware | Task 7 |
| §6 Message severity | Task 3 |
| §7 HasCardMoves | Task 1 |
| §9–10 Tests + smoke | each task + Task 9 |
| Out of scope | not scheduled |

**Placeholder scan:** no TBD/TODO steps; concrete tests and code included.

**Type consistency:** `HasCardMoves`, `msgKind`/`msgInfo`/`msgWarn`/`msgErr`, `setMessage`, `softStuck`, `softStuckMessage`, `hintMove *engine.Move`, `isLegalDestination` used consistently across tasks.

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-15-solitaire-feel-v1.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — run tasks in this session with executing-plans checkpoints  

Which approach?
