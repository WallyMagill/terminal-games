# 2048 Style Pass + Game-Over Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refine the 2048 tile color ramp and replace the inline `GAME OVER` status with a dedicated end-game summary screen.

**Architecture:** Purely presentational changes inside `internal/g2048/ui`: a new foreground ramp in `styles.go`, and a `gameOverView()` branch in `game.go`'s `View()`. No engine, store, app, or key-handling changes.

**Tech Stack:** Go 1.26.4, Bubble Tea v1.3.10, Lip Gloss v1.1.0, `go test`.

**Spec:** `docs/superpowers/specs/2026-07-22-g2048-style-endgame-design.md`

## Global Constraints

- No new dependencies; module stays `github.com/wallymagill/mines`.
- Foreground colors only — no background fills, no box-drawing grid.
- No behavior changes: keys, win banner, records timing, `TooSmall()` constants (`g2048MinCols = 34`, `g2048MinRows = 16`), and cell width (`cellInnerWidth = 5`) all stay as-is.
- Mines, Solitaire, engine, store, and app packages untouched.
- TDD: failing test → implement → pass → commit. Commit messages end with the `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` trailer.

---

## File Structure

```text
internal/g2048/ui/
  styles.go       // Modify: new tile ramp (Task 1)
  styles_test.go  // Create: ramp boundary tests (Task 1)
  game.go         // Modify: game-over View branch (Task 2)
  view_test.go    // Modify: game-over screen tests (Task 2)
```

---

### Task 1: Tile color ramp

**Files:**
- Create: `internal/g2048/ui/styles_test.go`
- Modify: `internal/g2048/ui/styles.go` (the `tileStyle` function only)

**Interfaces:**
- Consumes: existing `tileStyle(v int) lipgloss.Style` and `cellInnerWidth`.
- Produces: same signature, new ramp — empty `238`; 2–4 `250`; 8–16 `222`; 32–64 `214`; 128–256 `208`; 512–1024 bold `203`; 2048+ bold `227`. Task 2 does not depend on specific colors.

- [ ] **Step 1: Write the failing ramp test**

Create `internal/g2048/ui/styles_test.go`:

```go
package ui

import "testing"

func TestTileRampBoundaries(t *testing.T) {
	// Each adjacent pair crosses a ramp boundary and must render distinctly.
	pairs := [][2]int{{4, 8}, {16, 32}, {64, 128}, {256, 512}, {1024, 2048}}
	for _, p := range pairs {
		a, b := tileStyle(p[0]), tileStyle(p[1])
		if a.GetForeground() == b.GetForeground() {
			t.Fatalf("values %d and %d share foreground %v", p[0], p[1], a.GetForeground())
		}
	}
}

func TestTileRampBold(t *testing.T) {
	for _, v := range []int{2, 4, 8, 16, 32, 64, 128, 256} {
		if tileStyle(v).GetBold() {
			t.Fatalf("%d must not be bold", v)
		}
	}
	for _, v := range []int{512, 1024, 2048, 4096, 65536} {
		if !tileStyle(v).GetBold() {
			t.Fatalf("%d must be bold", v)
		}
	}
	if tileStyle(1024).GetForeground() == tileStyle(2048).GetForeground() {
		t.Fatal("1024 and 2048 must differ in color")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/g2048/ui/ -run TestTileRamp -count=1`
Expected: FAIL — `TestTileRampBold` fails because 512/1024 are not bold in the current ramp.

- [ ] **Step 3: Replace the ramp in tileStyle**

In `internal/g2048/ui/styles.go`, replace the body of `tileStyle` with:

```go
func tileStyle(v int) lipgloss.Style {
	s := lipgloss.NewStyle().Width(cellInnerWidth).Align(lipgloss.Right)
	switch {
	case v == 0:
		return s.Foreground(lipgloss.Color("238"))
	case v <= 4:
		return s.Foreground(lipgloss.Color("250"))
	case v <= 16:
		return s.Foreground(lipgloss.Color("222"))
	case v <= 64:
		return s.Foreground(lipgloss.Color("214"))
	case v <= 256:
		return s.Foreground(lipgloss.Color("208"))
	case v <= 1024:
		return s.Bold(true).Foreground(lipgloss.Color("203"))
	default:
		return s.Bold(true).Foreground(lipgloss.Color("227"))
	}
}
```

- [ ] **Step 4: Run the full ui package tests**

Run: `go test ./internal/g2048/ui/ -count=1`
Expected: PASS (existing view/game tests do not assert colors).

- [ ] **Step 5: Commit**

```bash
git add internal/g2048/ui/styles.go internal/g2048/ui/styles_test.go
git commit -m "feat(g2048): refine tile color ramp

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: Game-over summary screen

**Files:**
- Modify: `internal/g2048/ui/game.go` (`View()` plus a new `gameOverView()` method)
- Modify: `internal/g2048/ui/view_test.go`

**Interfaces:**
- Consumes: `m.game.Over()`, `m.game.Score()`, `m.game.HighestTile()`, model fields `bestScore`, `newBestScore`, `newBestTile`, `saveErr`, styles `styleTitle`, `styleErr`, `styleBest`, `styleFaint`, helper `fitLine`. The `key(...)` and `testModel(...)` test helpers already exist in this package (`game_test.go` / `view_test.go`).
- Produces: `func (m Model) gameOverView() string`, rendered by `View()` whenever `m.game.Over()` (after the too-small and help checks). No exported API changes.

- [ ] **Step 1: Write the failing view tests**

Append to `internal/g2048/ui/view_test.go` (add `"fmt"` to its imports):

```go
// playUntilOver drives the seeded game with cycling moves until it dies.
// Deterministic for a fixed model seed.
func playUntilOver(t *testing.T, m Model) Model {
	t.Helper()
	keys := []string{"left", "down", "right", "up"}
	for i := 0; i < 20000 && !m.game.Over(); i++ {
		m, _ = m.Update(key(keys[i%4]))
	}
	if !m.game.Over() {
		t.Fatal("game did not reach game over within 20000 moves")
	}
	return m
}

func TestGameOverScreenReplacesBoard(t *testing.T) {
	m := playUntilOver(t, testModel(t))
	v := m.View()
	if !strings.Contains(v, "G A M E   O V E R") {
		t.Fatalf("missing game over heading:\n%s", v)
	}
	if !strings.Contains(v, fmt.Sprintf("score  %6d", m.game.Score())) {
		t.Fatalf("missing final score %d:\n%s", m.game.Score(), v)
	}
	if !strings.Contains(v, fmt.Sprintf("tile   %6d", m.game.HighestTile())) {
		t.Fatalf("missing highest tile %d:\n%s", m.game.HighestTile(), v)
	}
	// Board must be hidden: the hint line keeps a few "·" separators,
	// but the 14+ cell placeholders must be gone.
	if n := strings.Count(v, "·"); n >= 4 {
		t.Fatalf("board placeholders still visible (%d dots):\n%s", n, v)
	}
	// Fresh store: this run is the record, so both stars show.
	if !strings.Contains(v, "new best") {
		t.Fatalf("expected new best markers on fresh store:\n%s", v)
	}
	if !strings.Contains(v, "n new game") {
		t.Fatalf("missing key hints:\n%s", v)
	}
}

func TestGameOverScreenNoStarsWhenNotBest(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	if _, _, err := s.RecordG2048Classic(999999, 65536); err != nil {
		t.Fatal(err)
	}
	m, err := NewModel(s, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	m.termW, m.termH = 80, 24
	m = playUntilOver(t, m)
	v := m.View()
	if !strings.Contains(v, "G A M E   O V E R") {
		t.Fatalf("missing game over heading:\n%s", v)
	}
	if strings.Contains(v, "new best") {
		t.Fatalf("no stars expected when records were not beaten:\n%s", v)
	}
	if !strings.Contains(v, "best   999999") {
		t.Fatalf("persisted best missing:\n%s", v)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/g2048/ui/ -run TestGameOver -count=1`
Expected: FAIL — the current `View()` still renders the board with `·` placeholders and has no `G A M E   O V E R` heading.

- [ ] **Step 3: Add the gameOverView branch**

In `internal/g2048/ui/game.go`, inside `View()`, insert the branch after the help check:

```go
	if m.showHelp {
		return m.helpView()
	}
	if m.game.Over() {
		return m.gameOverView()
	}
```

Then add the method (styling mirrors the normal view; save errors still surface):

```go
func (m Model) gameOverView() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(" 2048") + "\n")
	bestLabel := fmt.Sprintf("BEST %d", m.bestScore)
	if m.newBestScore {
		bestLabel = styleBest.Render(bestLabel)
	} else {
		bestLabel = styleFaint.Render(bestLabel)
	}
	b.WriteString(fmt.Sprintf(" SCORE %-6d  %s", m.game.Score(), bestLabel) + "\n\n")

	b.WriteString(styleErr.Render("      G A M E   O V E R") + "\n\n")

	star := "  " + styleBest.Render("★ new best")
	scoreLine := fmt.Sprintf("      score  %6d", m.game.Score())
	if m.newBestScore {
		scoreLine += star
	}
	tileLine := fmt.Sprintf("      tile   %6d", m.game.HighestTile())
	if m.newBestTile {
		tileLine += star
	}
	b.WriteString(scoreLine + "\n")
	b.WriteString(tileLine + "\n")
	b.WriteString(fmt.Sprintf("      best   %6d", m.bestScore) + "\n\n")

	if m.saveErr {
		b.WriteString(styleErr.Render(" could not save record") + "\n")
	}
	b.WriteString(styleFaint.Render(fitLine(m.termW,
		" n new game · m menu · q quit",
		" n new · m menu",
	)) + "\n")
	return b.String()
}
```

Do not touch `statusLine()`, `applyDir`, or any key handling — the over branches there stay as harmless fallbacks and keys remain inert when over.

- [ ] **Step 4: Run the ui package tests**

Run: `go test ./internal/g2048/ui/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/g2048/ui/game.go internal/g2048/ui/view_test.go
git commit -m "feat(g2048): dedicated game-over summary screen

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Full verification and rebuild

**Files:**
- None expected; fix only what verification reveals.

**Interfaces:**
- Consumes: everything above.
- Produces: green gates and refreshed binaries (`./mines`, `~/go/bin/mines`).

- [ ] **Step 1: Run all gates**

```sh
go build ./... && go vet ./... && gofmt -l . && go test ./... -count=1
```

Expected: build/vet clean, `gofmt -l` prints nothing, all packages PASS.

- [ ] **Step 2: Manual smoke test**

Rebuild and play: `go build -o mines . && go install .`, then launch `./mines` (or drive with `expect` and a sandboxed `HOME`): enter 2048, confirm the new ramp renders, force moves until game over, confirm the summary screen shows score/tile/best with ★ markers when appropriate, `n` restarts, `m` returns to launcher. Mines and Solitaire unchanged.

- [ ] **Step 3: Commit any fixes**

Only if Step 1–2 revealed gaps; otherwise nothing to commit.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| New foreground ramp incl. bold 512+, gold 2048+, empty `238` | 1 |
| Game-over summary replaces board | 2 |
| ★ markers driven by `newBestScore`/`newBestTile` | 2 |
| Save-error surfaces on summary | 2 |
| Key hints `n new game · m menu · q quit` | 2 |
| Win banner / keys / TooSmall unchanged | Global constraints, 2 Step 3 note |
| All suites green, Mines/Solitaire untouched | 3 |
