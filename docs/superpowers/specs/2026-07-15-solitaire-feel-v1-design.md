# Solitaire Feel v1 — Design

**Date:** 2026-07-15  
**Status:** Approved; implementation plan written  
**Goal:** Make Klondike Draw 3 feel clean and intentional on keyboard: vim navigation that prefers useful cards, soft stuck detection when a deal stalls, and hints that suggest good plays instead of shallow or wrong-ranked legal moves — without turning the game into a phone clone or a hard loss screen.

**Related:** builds on the existing Solitaire engine/UI and the usability-assist layer (legal moves, smart sole-destination move, safe auto-foundation, undo, basic hints, no-progress stock-pass tracking). Does not rewrite Mines.

## Decisions made during brainstorming

| Question | Decision |
|---|---|
| Stuck / uncompletable deals | **Soft v1:** clear “no more productive moves · press n for a new deal” message; deal stays playable; no hard Lost status, no auto-quit. |
| Navigation model | **Keep vim** (`h j k l` + arrows). Do not replace with action-list-only UI. |
| What to fix under vim | Cursor prefers playable content; while selected, vim moves among legal destinations; refuse dead selection; highlight destinations; status compass. |
| Hint key vs left | Restore **`h` = left** (match Mines). Remap hint to **`i`**. |
| Hint quality | Fix ranking holes; deprioritize empty tableau shuffles; focus source **and** destination; stuck-aware (“new deal” not endless draw). |
| Hint apply | Not required for v1; focus + highlight + message is enough. Optional second-press apply can wait. |
| Solvability proof | No full hidden-state solver in v1. Soft stuck is **practical**, not mathematical unsolvability. |
| Records / assisted | Soft-stuck message is **not** assisted. Improved hints still mark **assisted** when used (same policy as today). Undo remains assisted. Safe auto-foundation remains normal-record eligible. |
| Out of scope for v1 | Mouse, animations, Draw 1 mode, hard Lost screen, full solver, endgame autocomplete, legal-vs-safe foundation key split (`f`), glyph/width overhaul beyond what destination highlight needs. |

## Problem summary

Solitaire is rules-correct and already has assist helpers, but play still feels rough because:

1. **Navigation tax** — the cursor walks board geometry (including face-down cards and dead piles) instead of useful cards; multi-destination moves lack destination affordances; dead selection traps the player in cancel mode.
2. **Silent stall** — when a deal is practically stuck, the only signal is a mild stock-pass warning; there is no clear “restart” guidance, so the deal feels unclean compared to other solitaire apps.
3. **Weak / wrong-ranked hints** — ranking is a flat list with holes (e.g. legal-but-unsafe foundation moves fall to worst rank and lose to draw); pointless tableau rearrangements outrank many good plays; hints only focus the source; when stalled, hints still push draw/recycle.

Mines feels better because it is a one-step grid action model with consistent `hjkl` and clear terminal states. Feel v1 closes the biggest Solitaire gaps without abandoning keyboard vim play.

## Product principles

1. **Never trap the player in a mode that cannot succeed** (no dead selection).
2. **Vim is the language; the map can shrink** (destinations while selected).
3. **Always know what mode you are in** — playing, selected, soft-stuck, won.
4. **Mechanical clarity ≠ strategy oracle** — soft stuck and destination highlights are normal UX; hints/undo stay assisted.
5. **Same app, same navigation alphabet** — `h j k l` mean the same thing in Mines and Solitaire.
6. **Practical stuck, not false certainty** — never claim a deal is mathematically unsolvable in v1.

---

## 1. Keymap changes

| Key | Current | v1 |
|---|---|---|
| `h` | Hint (assisted) | **Move cursor left** |
| `j` / `k` / `l` | Down / up / right | Unchanged (with smarter clamping) |
| arrows | Move cursor | Unchanged (same clamping rules as vim keys) |
| `i` | Unused | **Hint** (assisted) |
| `space` / `enter` | Select / smart-move / place | Unchanged intent; dead selection removed |
| `tab` / shift-tab | Cycle playable sources; or destinations when selected | Keep; still power shortcuts |
| `1`–`7` | Jump tableau columns | Keep; clamp to playable row |
| `d` | Draw / recycle | Keep |
| `a` / `A` | Safe foundation (focused / all) | Keep |
| `u` | Undo (assisted) | Keep |
| `n` | New deal | Keep; primary recovery from soft stuck |
| `esc` | Cancel selection / menu | Keep |
| `?` | Help | Keep; document new keys |

Help overlay and default status chrome must list `i` for hint and `hjkl` for move. Remove any help text that says `h` is hint.

---

## 2. Vim navigation improvements

### 2.1 Tableau vertical motion

When moving up/down within a tableau column:

- Land only on **face-up** cards (valid stack positions) or on **empty column** (row 0).
- Never leave the cursor on a face-down card after a navigation key.
- Number-key jumps (`1`–`7`) already try to clamp playable; keep and align with the same face-up rule.

### 2.2 Horizontal column changes

When moving left/right between tableau columns:

- Preserve “same relative height” only when that row is face-up and in range.
- Otherwise clamp to the **topmost face-up card** of the destination column, or empty row 0.

Do not invent skip-empty-columns-by-default behavior in v1; empty columns remain reachable (required for kings).

### 2.3 Waste and foundations

- Waste cursor always addresses the **playable top** waste card (single logical slot).
- Foundations address pile tops only (existing model).

### 2.4 Stock

- Stock remains a single slot; space/draw behavior unchanged.

### 2.5 Free roam vs power cycle

- Arrows/`hjkl` = board navigation with the clamps above.
- `tab` / shift-tab = cycle **sources that currently have ≥1 legal move** (existing assist intent); when a source is selected, cycle **legal destinations** (existing).

---

## 3. Selection and destination flow

### 3.1 Refuse dead selection

When no source is selected and the player presses `space`/`enter` on a focused location:

| Focus | Behavior |
|---|---|
| Stock | Draw / recycle (unchanged). |
| Card/stack with **exactly one** legal move | Apply immediately (unchanged smart move). |
| Card/stack with **multiple** legal moves | Enter selection; store `selectedMoves`; focus first destination; show destination chrome. |
| Card/stack with **zero** legal moves | **Do not select.** Message: `no moves for <card>` (or location if no card). Stay free. |
| Empty / non-selectable | Message: `No card selected` (or clearer empty-slot text). Stay free. |

### 3.2 Destination-scoped vim while selected

When `selected != nil` and `selectedMoves` is non-empty:

- `h j k l` and arrows move the cursor among **legal destinations only** (cycle in a stable order: foundations by suit index, then tableau columns 1–7).
- `space` / `enter` applies the move to the focused destination if it matches a selected move.
- `tab` / shift-tab continue to cycle destinations (same set).
- `esc` clears selection.
- Invalid free targets are not required; destination-only navigation is the v1 selected mode.

If implementation needs a simple ordered list: use `selectedMoves` order from the engine (already enumerated) as the cycle order for both tab and vim keys while selected.

### 3.3 Destination highlighting

While a source is selected (or while a hint is showing a concrete card move):

- Style **legal destination** piles/cards differently from normal and from cursor/selection.
- Selected source keeps the stronger selected style.
- Cursor style remains distinct from “legal destination” style (e.g. reverse cursor, bold/colored dest, selected source reverse+bold — exact Lip Gloss choices left to implementation, but three states must be distinguishable: normal, destination, focused).

### 3.4 Status compass

Idle (playing, not selected, not soft-stuck), prefer a dynamic status when terminal width allows:

```text
 focus <card> [stack N] · M moves · spc act · tab next · i hint
```

Compact fallback may drop fields. If focus has zero moves, show `0 moves` so the player is not surprised by refuse-select.

When selected:

```text
 selected <card> from <src> · dest <dst> · spc place · esc cancel
```

When soft-stuck (see §4), soft-stuck status **wins** over idle chrome until the player makes a card-progress move or starts a new deal.

---

## 4. Soft stuck detection

### 4.1 Goal

Tell the player when the deal has **practically stalled** and that a new deal is the intended recovery — without locking input or claiming mathematical unsolvability.

### 4.2 Definition (v1)

A deal is **soft-stuck** when both are true:

1. There is **no legal non-draw card move** on the current visible position  
   (no waste/tableau/foundation moves in `LegalMoves()` except `MoveDraw` / `MoveRecycle`), **and**
2. The player has completed a **full stock pass with no card progress** since the last card-progress move  
   (reuse/extend existing `stockPassProgress` / recycle boundary tracking).

Notes:

- “Card progress” means any successful non-draw move (tableau/waste/foundation placement), including safe auto-foundation batches.
- Draw and recycle alone never clear soft-stuck once latched for this pass; only card progress or a new deal clears it.
- If stock and waste are both empty and no card moves exist, soft-stuck applies immediately (nothing left to cycle).

### 4.3 Player-facing behavior

When soft-stuck becomes true:

- Set a durable UI flag `softStuck` (name flexible).
- Status (full):  
  `No more productive moves · press n for a new deal`
- Compact:  
  `No more moves · n new`
- Do **not** change engine `Status` (still `Playing` unless won).
- Do **not** mark assisted.
- Do **not** block keys: player may still draw, undo, fiddle, or press `n`.
- Clear `softStuck` when any card-progress move succeeds, or on `n` new deal.

### 4.4 Interaction with existing no-progress warning

The current one-shot stock-pass message is absorbed into soft-stuck:

- Prefer the soft-stuck copy when the stronger condition holds.
- Avoid stacking two simultaneous messages about the same situation.

### 4.5 Explicit non-goals

- No `Lost` engine status.
- No automatic new deal.
- No solver-based “this deal was never winnable.”
- No penalty to records for becoming soft-stuck.

---

## 5. Hints

### 5.1 Trigger and assisted policy

- Key: **`i`**
- Using a hint sets `assisted = true` and `assistReason = "hint"` (unchanged policy).
- Hint does not mutate game state (unchanged).

### 5.2 Ranking (replace flat holes)

Compute a total order over `LegalMoves()`. Lower rank number = better. Suggested tiers:

| Rank | Move class |
|---|---|
| 1 | Tableau move that **reveals** a face-down card |
| 2 | **Any legal** foundation move (not only `Safe`) — fix the current bug where unsafe-but-legal foundations fall to worst rank |
| 3 | Waste → tableau |
| 4 | Tableau → tableau that **reveals** (if not already rank 1) or moves onto a pile creating progress; prefer revealing |
| 5 | King (or king stack) onto **empty** tableau |
| 6 | Other tableau → tableau (non-revealing rearrangements) — **deprioritized** vs today |
| 7 | Draw |
| 8 | Recycle |
| 9 | Foundation → tableau (usually last resort) |

Within the same rank, stable tie-breaks:

1. Prefer moves that reveal.
2. Prefer waste involvement over pure tableau churn when ranks collide after the table above.
3. Otherwise keep engine enumeration order for determinism.

Exact secondary heuristics may be refined in implementation tests; the **required** fixes are: foundation legal moves must not rank below draw, and empty non-revealing tableau shuffles must rank below revealing/waste/foundation plays.

### 5.3 Presentation

When a hint finds a card move (not draw/recycle):

1. Focus **source**.
2. Also mark/focus **destination** for highlight (cursor may sit on destination or source; status must name both — prefer focusing destination after naming source in the message, or focus source and paint dest highlights; pick one in implementation and test it).
3. Message:  
   `Hint: move <card> from <src> to <dst>[ · reveals]`  
   matching improved `describeHintMove`.

When the best move is draw/recycle and the deal is **not** soft-stuck:

- `Hint: draw from stock` / `Hint: recycle waste into stock`
- Focus stock.

When the deal **is** soft-stuck (or would only hint draw/recycle after the soft-stuck condition is true):

- Do **not** recommend endless draw as strategy help.
- Message:  
  `Hint: no more productive moves · press n for a new deal`
- Do not pretend a productive card play exists.

When no legal moves at all (empty stock/waste, no card moves):

- Same restart-oriented message.

### 5.4 Out of scope for v1

- Auto-applying the hinted move on a second keypress.
- Multi-step look-ahead search / solver.
- Hint that peeks face-down identities.

---

## 6. Feedback and messaging polish (minimal, required for cleanliness)

| Event | Style / copy |
|---|---|
| Success (moved, drew, undid, auto-foundation) | **Info** style (faint or normal), not error red |
| Soft stuck / no moves for card | **Warn** style (distinct from error if easy; else faint + clear words) |
| True failure (save failed) | **Error** style (red) |
| Dead selection refusal | `no moves for <card>` |
| Soft stuck | `No more productive moves · press n for a new deal` |

Implementation may use a small message severity field or separate render paths; success must not use `styleErr`.

---

## 7. Architecture

Keep the existing split:

| Package | Responsibility in v1 |
|---|---|
| `internal/solitaire/engine` | Rules, `LegalMoves`, safe foundation classification, clone. Prefer putting pure ranking helpers here only if unit-tested without UI; otherwise ranking can stay in UI if it only consumes `Move` metadata. **Soft-stuck policy may live in UI** (it depends on pass progress), with engine only answering “any non-draw legal moves?” if helpful. |
| `internal/solitaire/ui` | Cursor clamps, selection flow, destination vim mode, highlights, soft-stuck flag, hint key/ranking presentation, status/help, message severity. |
| `internal/mines/**` | Unchanged. |
| `internal/store` | Unchanged assisted vs normal record split. |

No new packages required.

Suggested engine helper (optional but clean):

```go
func (g *Game) HasCardMoves() bool // true if any LegalMoves kind is not Draw/Recycle
```

Soft-stuck latching and stock-pass progress remain UI state.

---

## 8. Rendering notes

- Destination highlight must work with existing bracketed tokens (`[A♠ ]`, `[##]`, etc.).
- No full layout redesign of stock/waste/foundations in this v1 scope (called out as future polish in the audit). If a tiny spacing tweak is needed for highlight readability, allowed; do not expand into a visual redesign project.
- Help and status must stay width-adaptive via existing `fitLine` patterns.

---

## 9. Testing plan

### Engine (if helpers added)

- `HasCardMoves` true/false cases: only draw left; waste playable; empty everything.

### UI — navigation

- Down/up skip face-down cards.
- Horizontal clamp to face-up top when leaving a tall column for a short one.
- `h` moves left; does not mark assisted.
- `i` marks assisted and sets hint message.

### UI — selection

- Zero legal moves: no `selected`, message contains card name.
- Multiple destinations: vim keys cycle only destinations; enter applies.
- Destination tokens appear styled differently in view (string/style assertion as feasible).

### UI — soft stuck

- After full no-progress stock pass with only draw/recycle left: status contains “No more productive moves” and “n”.
- Card-progress move clears soft stuck.
- Soft stuck does not set `assisted`.
- Soft stuck does not change engine status away from Playing.

### UI — hints

- Legal foundation move ranks above draw when both exist.
- Non-revealing tableau shuffle ranks below foundation/waste/reveal when both exist.
- When soft-stuck, hint message recommends new deal, not draw.
- Hint still does not mutate stock/moves counters.

### Regression

- Full `go test ./...`
- Mines behavior untouched.
- Existing undo/auto-foundation/smart sole-destination tests remain green (update only where keymap or message contracts change).

---

## 10. Manual smoke checklist

1. Launch Solitaire; `hjkl` move; `h` does not trigger hint.
2. `i` shows hint and marks assisted in HUD.
3. Land only on face-up tableau cards when walking a column.
4. Select a multi-destination card; `hjkl` jumps destinations; highlights visible; space places.
5. Focus a stuck card; space refuses select with clear message.
6. Burn a stock pass with no plays; soft-stuck status appears; `n` deals fresh.
7. Win path still records normal/assisted correctly.
8. Help lists new keys; Mines `hjkl` still works.

---

## 11. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Soft stuck false positives while a clever foundation pull-back still exists | Foundation→tableau remains a card move; if any exists, not soft-stuck. Accept that rare deep lines may still exist when only draw remains — copy stays “productive moves,” not “unsolvable.” |
| Destination-only vim feels constraining | `esc` cancels; free roam returns. Tab still available. |
| Hint remaps break muscle memory of `h`=hint from usability pass | Help + status teach `i`; `h`=left matches Mines and vim, higher priority. |
| Ranking still imperfect | v1 requires fixing known holes and stuck-awareness; not perfect strategy. |

---

## 12. Success criteria

Feel v1 is done when:

1. Vim navigation no longer regularly lands on useless face-down cells.
2. Multi-destination plays are doable with vim + highlights without feeling like a mode trap.
3. Dead selection is gone.
4. Soft-stuck deals tell you to press `n` instead of leaving you in limbo.
5. Hints no longer prefer draw over obvious legal foundations, and when soft-stuck they say restart.
6. `h`/`j`/`k`/`l` match Mines; hint is `i`.
7. Success messages no longer render as errors.
8. All automated tests pass; manual smoke checklist is clean.

---

## 13. Implementation sequencing (for the later plan)

1. Keymap: `h` left, `i` hint; help/status/tests.
2. Message severity (success ≠ error).
3. Cursor clamps (face-up / column entry).
4. Refuse dead selection + destination-only vim + highlights + selected status.
5. Soft stuck detection + status priority.
6. Hint ranking + presentation + stuck-aware copy.
7. Full test suite + manual smoke.

---

## 14. Follow-ups (explicitly not v1)

- Endgame autocomplete
- Top-row layout separation (stock / waste / foundations)
- Draw 1 mode
- Quick legal foundation key separate from safe `a`/`A`
- Hard Lost screen
- Solver-grade stuck / win prediction
- Hint apply-on-second-press
- Terminal-safe fixed-width suit glyphs overhaul
