# Solitaire Draw-3 Fidelity — Design

**Date:** 2026-07-22
**Status:** Approved
**Goal:** Restore the classic three-card draw display and fix recycle to a true waste flip, based on a full engine correctness audit.

## Audit summary (context)

A read-only audit (`.superpowers/sdd/engine-audit.md`, randomized 120k+-step invariant harness) validated the engine against classic Klondike draw-3 rules:

| Area | Verdict |
|---|---|
| 52-card conservation through draws/recycles/moves/undo | PASS |
| Deal shape (7 piles 1..7, last face-up, 24 stock) | PASS |
| Draw-3 semantics; only top waste card playable | PASS |
| Recycle order fidelity | **FAIL — permutes when waste % 3 ≠ 0** |
| Move legality (colors, runs, kings, foundations, face-down) | PASS |
| Win detection at exactly 52 on foundations | PASS |

Accepted as-is (deliberate): recycle increments the move counter; the unused `Move.Cards` snapshot field.

## 1. Three-card draw display (ui)

Revert the single-card draw slot to the classic three-card fan:

- `topView` renders up to three cards from `WasteVisible()`, oldest → newest left to right; the rightmost is the top.
- Only the top (rightmost) card is decorated for cursor/selection/show-moves; under-cards render plainly. Selection and engine semantics are unchanged — only the top card was ever playable.
- Empty slots render nothing (blank space), as the original three-slot layout did; the `Deck` / `Draw` / `Foundations` labels stay.
- `solitaireMinCols` (48) already accommodated the three-slot layout; it stays.

## 2. Classic recycle (engine)

Fix `Draw()`'s recycle branch (`internal/solitaire/engine/game.go`) so that recycling replays the deck faithfully. **Binding invariant:** after a recycle with no cards played, redrawing through the deck exposes exactly the same sequence of waste tops, in the same groups, as the previous pass.

Concretely (slice convention: top at the end; `Draw` appends `stock[start:]` preserving slice order): the waste slice is the concatenation of the drawn groups in draw order. Rebuild the stock by walking the waste from the **bottom up** in chunks of 3 and appending those chunks to the new stock in **reverse chunk order** (last chunk lands first / at the bottom, first chunk lands last / on top). Redraw then peels the chunks off in original draw order with original in-group order — identical triples, identical tops. For a waste count divisible by 3 this reproduces what the current code already does (restores the stock exactly); the fix changes only the misaligned case, where the current top-down chunking cuts groups across the old boundaries and permutes the pass. Note: naive slice reversal is NOT correct — it preserves the flip metaphor but reverses in-group order under this engine's draw convention.

- The dead-end scan keeps its state-identity termination (`deadend.go`) — still correct, now redundant defense rather than a necessity. Its comment is updated to say so.
- Any existing engine test that pinned the permuted rebuild order is updated to the flip order.
- New regression test: with a misaligned waste (count % 3 ≠ 0, e.g. 7 cards), a full recycle + redraw cycle exposes exactly the same sequence of waste tops as the previous pass.
- Conservation tests unchanged and must stay green.

## Interaction notes

- `TestDeadEndFalseWhenCardSurfacesOnlyAfterPermutedRecycle` (engine) was built on the permuting recycle; after the fix its premise changes. Replace it with an equivalent misaligned-waste test that matches flip semantics: a card that is never a waste top in the current pass alignment but becomes one after cards are played is out of reach of a draw-only scan — so the correct expectation flips to `DeadEnd() == true` for a pure-draw cycle where no reachable top is productive. Keep a companion test where the productive card IS reachable mid-pass (`DeadEnd() == false`).
- The show-moves (`v`) and game-over features consume `ProductiveMoves()`/`DeadEnd()` unchanged.

## Testing

- Engine: true-flip regression (identical tops across passes, misaligned count); updated dead-end tests per above; full conservation suite green.
- UI: three-card fan renders (all visible waste cards shown, only top decorated); playing the top reveals the card beneath in the next render; single-card-display test replaced.
- Full suite green; Mines, 2048, store, app untouched.

## Non-goals

- Pass limits, scoring modes, draw-1 mode.
- Changing the move counter or `Move.Cards`.
- Any records/store change.
