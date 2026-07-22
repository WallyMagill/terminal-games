# Solitaire Draw-3 Fidelity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix recycle to the classic waste flip (identical triples every pass) and restore the three-card draw display.

**Architecture:** One engine change (the recycle branch of `Draw()` in `internal/solitaire/engine/game.go`) with a reworked dead-end test whose premise the fix invalidates, and one UI change (`topView` in `internal/solitaire/ui/game.go`) restoring the three-slot fan. Nothing else moves.

**Tech Stack:** Go 1.26.4, Bubble Tea v1.3.10, Lip Gloss v1.1.0, `go test`.

**Spec:** `docs/superpowers/specs/2026-07-22-solitaire-draw3-fidelity-design.md`

## Global Constraints

- **Binding recycle invariant (from spec):** after a recycle with no cards played, redrawing exposes exactly the same sequence of waste tops, in the same groups, as the previous pass. Naive slice reversal is WRONG (reverses in-group order); the correct rebuild walks the waste bottom-up in chunks of 3 and appends chunks in reverse chunk order.
- Three-card fan: up to 3 waste cards, oldest → newest left to right, only the rightmost decorated/selectable; `Deck`/`Draw`/`Foundations` labels stay; `solitaireMinCols` (48) stays.
- Dead-end scan keeps its state-identity termination (`deadend.go`) — comment updated to "redundant defense," logic unchanged.
- Recycle still increments the move counter (accepted as-is per spec).
- No store, app, mines, g2048, or records changes.
- TDD: failing test → implement → pass → commit. Commit messages end with the trailer `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

**Existing helpers (do not redefine):** engine tests are same-package and construct `Game` literals directly; ui tests have `testModel(t)`, `key(s)`, `setUnexported(field, value)` in `game_test.go`; `cardToken` is in `styles.go` (note: it pads by BYTES, so use `cardToken(card)` in assertions, never hand-written literals).

---

## File Structure

```text
internal/solitaire/engine/
  game.go           // Modify: recycle branch of Draw()          (Task 1)
  draw_test.go      // Modify: + misaligned replay test          (Task 1)
  deadend.go        // Modify: comment only                      (Task 1)
  deadend_test.go   // Modify: rework permuted-recycle test      (Task 1)

internal/solitaire/ui/
  game.go           // Modify: topView three-card fan            (Task 2)
  view_test.go      // Modify: replace single-card test          (Task 2)
```

---

### Task 1: Classic recycle in the engine

**Files:**
- Modify: `internal/solitaire/engine/game.go` (recycle branch inside `Draw()`)
- Modify: `internal/solitaire/engine/draw_test.go`
- Modify: `internal/solitaire/engine/deadend.go` (comment only)
- Modify: `internal/solitaire/engine/deadend_test.go`

**Interfaces:**
- Consumes: existing `Draw()`, `WasteVisible()`, `StockCount()`, `DeadEnd()`.
- Produces: no signature changes — behavior only. Task 2 does not depend on this task.

- [ ] **Step 1: Write the failing misaligned-replay test**

Add to `internal/solitaire/engine/draw_test.go`:

```go
func TestRecycleReplaysMisalignedPass(t *testing.T) {
	// A 7-card waste drawn as groups [a,b,c],[d,e,f],[g] has pass tops c, f, g.
	// After a classic flip recycle, redrawing must expose the same tops in the
	// same order — and again on every subsequent pass.
	g := &Game{status: Playing}
	g.waste = []Card{
		{Suit: Clubs, Rank: Two}, {Suit: Diamonds, Rank: Two}, {Suit: Hearts, Rank: Two},
		{Suit: Spades, Rank: Two}, {Suit: Clubs, Rank: Three}, {Suit: Diamonds, Rank: Three},
		{Suit: Hearts, Rank: Three},
	}
	wantTops := []Card{
		{Suit: Hearts, Rank: Two},     // top of group [a,b,c]
		{Suit: Diamonds, Rank: Three}, // top of group [d,e,f]
		{Suit: Hearts, Rank: Three},   // the remainder card g
	}
	for pass := 1; pass <= 2; pass++ {
		if !g.Draw() { // recycle
			t.Fatalf("pass %d: recycle should succeed", pass)
		}
		if g.StockCount() != 7 || len(g.waste) != 0 {
			t.Fatalf("pass %d: after recycle stock %d waste %d, want 7/0", pass, g.StockCount(), len(g.waste))
		}
		for i, want := range wantTops {
			if !g.Draw() {
				t.Fatalf("pass %d draw %d failed", pass, i+1)
			}
			visible := g.WasteVisible()
			if got := visible[len(visible)-1]; got != want {
				t.Fatalf("pass %d diverges at draw %d: top %v, want %v", pass, i+1, got, want)
			}
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/solitaire/engine/ -run TestRecycleReplaysMisalignedPass -count=1`
Expected: FAIL — the current top-down chunk rebuild produces different tops on the recycled pass.

- [ ] **Step 3: Fix the recycle branch**

In `internal/solitaire/engine/game.go` `Draw()`, replace the recycle loop:

```go
	if len(g.stock) == 0 {
		if len(g.waste) == 0 {
			return false
		}
		// Classic flip: rebuild the stock so every pass re-deals the same
		// triples with the same tops. Walk the waste bottom-up in chunks of 3
		// (the original draw groups) and stack the chunks in reverse order —
		// the first-drawn group ends up back on top.
		n := len(g.waste)
		for start := ((n - 1) / 3) * 3; start >= 0; start -= 3 {
			end := start + 3
			if end > n {
				end = n
			}
			g.stock = append(g.stock, g.waste[start:end]...)
		}
		g.waste = nil
		g.moves++
		return true
	}
```

- [ ] **Step 4: Rework the dead-end test built on the old permuting recycle**

In `internal/solitaire/engine/deadend_test.go`, DELETE `TestDeadEndFalseWhenCardSurfacesOnlyAfterPermutedRecycle` and add these two (the fix inverts the first fixture's expectation — under classic recycle, pass tops are fixed while nothing is played):

```go
func TestDeadEndTrueWhenBuriedCardNeverSurfaces(t *testing.T) {
	// Classic recycle makes pass tops FIXED during a draw-only walk: the ace
	// sits at a non-top position, never surfaces, and the position is a dead
	// end even though the ace exists in the deck.
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Hearts, Rank: Five}, FaceUp: true}}
	g.stock = []Card{
		{Suit: Clubs, Rank: Ten},
		{Suit: Diamonds, Rank: Ten},
		{Suit: Hearts, Rank: Ten},
		{Suit: Spades, Rank: Ten},
		{Suit: Clubs, Rank: Ace},
		{Suit: Diamonds, Rank: Nine},
		{Suit: Clubs, Rank: Queen},
	}
	if !g.DeadEnd() {
		t.Fatal("buried ace unreachable by draws alone must be a dead end under classic recycle")
	}
}

func TestDeadEndFalseWhenCardSurfacesMidPass(t *testing.T) {
	// Ace at a top position of a later draw group: surfaces on the second
	// draw of the pass, so the position is not dead.
	g := &Game{status: Playing}
	g.tableau[0] = []TableauCard{{Card: Card{Suit: Hearts, Rank: Five}, FaceUp: true}}
	g.stock = []Card{
		{Suit: Clubs, Rank: Ten},
		{Suit: Diamonds, Rank: Ten},
		{Suit: Clubs, Rank: Ace},
		{Suit: Hearts, Rank: Ten},
		{Suit: Spades, Rank: Ten},
		{Suit: Clubs, Rank: Queen},
	}
	if g.DeadEnd() {
		t.Fatal("ace that tops the second draw group must not be a dead end")
	}
}
```

(Verification of the second fixture: 6 cards, tops are index 5 then index 2 — the ace at index 2 tops the second draw. `TestDeadEndFalseWhenCardBuriedInDeck` — ace topping the first draw — already exists and stays.)

- [ ] **Step 5: Update the deadend.go comment**

In `internal/solitaire/engine/deadend.go`, replace the paragraph of the `DeadEnd` doc comment that describes the permuting recycle with:

```go
// DeadEnd reports whether no productive move is reachable: none now, and none
// for any draw-pile top reachable by drawing/recycling, simulated on a clone.
// Recycle preserves pass order (classic flip), so one cycle visits every
// reachable top; the state-identity termination below is kept as redundant
// defense — it stays exact even if draw mechanics ever change again.
```

(Keep the `deckKey` mechanism and the cap exactly as they are.)

- [ ] **Step 6: Run the engine suite**

Run: `go test ./internal/solitaire/engine/ -count=1`
Expected: PASS — including `TestRecyclePreservesDrawThreeOrderAcrossPasses` (aligned case, unchanged behavior) and the conservation tests. `gofmt -l internal/solitaire/engine/` empty.

- [ ] **Step 7: Commit**

```bash
git add internal/solitaire/engine/
git commit -m "fix(solitaire): classic flip recycle preserves pass order

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: Three-card draw fan in the UI

**Files:**
- Modify: `internal/solitaire/ui/game.go` (`topView` only)
- Modify: `internal/solitaire/ui/view_test.go`

**Interfaces:**
- Consumes: `WasteVisible()`, `styledCard`, `emptyToken`, `decorate`, `cardToken` (tests).
- Produces: display-only change. Cursor/selection/show-moves still target only the top card (`decorate(regionWaste, 0, 0, ...)` applied to the last card only).

- [ ] **Step 1: Replace the single-card test with fan tests**

In `internal/solitaire/ui/view_test.go`, DELETE `TestTopViewShowsSingleDrawCard` and add (file already imports `reflect` and `engine` from the cursor-persistence work — verify, add if missing):

```go
func TestTopViewShowsThreeDrawCards(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("d"))
	visible := m.game.WasteVisible()
	if len(visible) != 3 {
		t.Fatalf("expected 3 drawn cards, got %d", len(visible))
	}
	top := m.topView()
	for _, card := range visible {
		if !strings.Contains(top, cardToken(card)) {
			t.Fatalf("card %s missing from fan:\n%s", cardToken(card), top)
		}
	}
}

func TestTopViewFanShowsRemainingCardsAfterTopPlayed(t *testing.T) {
	m := testModel(t)
	rv := reflect.ValueOf(m.game).Elem()
	remaining := []engine.Card{
		{Suit: engine.Clubs, Rank: engine.Four},
		{Suit: engine.Diamonds, Rank: engine.Seven},
	}
	setUnexported(rv.FieldByName("waste"), remaining)
	top := m.topView()
	for _, card := range remaining {
		if !strings.Contains(top, cardToken(card)) {
			t.Fatalf("under-card %s must be visible after the top is played:\n%s", cardToken(card), top)
		}
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/solitaire/ui/ -run 'TestTopView' -count=1`
Expected: FAIL — the single-slot view renders only the last card. (`TestTopViewShowsSingleDrawCard` must be gone or it would now fail for the opposite reason.)

- [ ] **Step 3: Restore the fan in topView**

In `internal/solitaire/ui/game.go`, replace `topView`'s draw-slot section (currently one slot between the deck and foundations) with the original three-slot fan, keeping the new labels:

```go
func (m Model) topView() string {
	var b strings.Builder
	b.WriteString(" Deck   Draw              Foundations\n ")
	stock := emptyToken(" · ")
	if m.game.StockCount() > 0 {
		stock = hiddenToken()
	}
	b.WriteString(m.decorate(regionStock, 0, 0, stock))
	b.WriteString(" ")
	waste := m.game.WasteVisible()
	for i := range 3 - len(waste) {
		b.WriteString(emptyToken(""))
		if i < 2 || len(waste) > 0 {
			b.WriteString(" ")
		}
	}
	for i, card := range waste {
		token := styledCard(card)
		if i == len(waste)-1 {
			token = m.decorate(regionWaste, 0, 0, token)
		}
		b.WriteString(token)
		if i < len(waste)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString(" ")
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

(This is the pre-rename fan layout with `Deck`/`Draw` labels. Header spacing: eyeball in Task 3's smoke test; adjust only spaces in the header string if `Foundations` sits visibly off its slots.)

- [ ] **Step 4: Run the ui suite**

Run: `go test ./internal/solitaire/ui/ -count=1`
Expected: PASS — including the show-moves tests (waste highlight still lands on the top card only) and `gofmt -l internal/solitaire/ui/` empty.

- [ ] **Step 5: Commit**

```bash
git add internal/solitaire/ui/
git commit -m "feat(solitaire): restore three-card draw fan

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Full verification and rebuild

**Files:**
- None expected; fix only what verification reveals.

- [ ] **Step 1: Run all gates**

```sh
go build ./... && go vet ./... && gofmt -l . && go test ./... -count=1
```

Expected: clean, all packages PASS.

- [ ] **Step 2: Manual smoke test**

Rebuild: `go build -o games . && go install . && /bin/cp -f "$HOME/go/bin/mines" "$HOME/go/bin/games" 2>/dev/null; /bin/rm -f "$HOME/go/bin/mines"` (go install re-creates `mines`; keep only `games`). Launch `./games` (or drive with `expect` and a sandboxed `HOME`): in Solitaire, press `d` — three cards fan out with only the rightmost selectable; play the top waste card — the card beneath becomes the new top and stays visible; draw through the deck twice without playing — the second pass shows the same triples; check header alignment. Mines and 2048 unchanged.

- [ ] **Step 3: Commit any fixes**

Only if Steps 1–2 revealed gaps; otherwise nothing to commit.

---

## Spec coverage checklist

| Spec requirement | Task |
|---|---|
| Recycle invariant: identical tops/groups every pass, misaligned included | 1 |
| Aligned behavior unchanged (existing pass-order test stays green) | 1 |
| Dead-end test rework (buried→true, mid-pass→false, first-draw kept) | 1 |
| deadend.go comment: state-identity now redundant defense | 1 |
| Three-card fan, oldest→newest, top-only decorated | 2 |
| Labels/min-cols unchanged | 2 |
| Conservation suite green; full suite green | 1, 3 |
