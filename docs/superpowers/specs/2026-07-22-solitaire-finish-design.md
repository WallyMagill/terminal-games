# Solitaire Finishing Pass — Design

**Date:** 2026-07-22
**Status:** Approved
**Goal:** Finish the Solitaire game: remove the hint feature, rename the top piles, show a single draw card, make navigation snappier, and add real no-moves-left detection with a game-over screen.

## Decisions made during brainstorming

| Question | Decision |
|---|---|
| Hints | Remove entirely. Undo remains the only "assisted" trigger. |
| Pile labels | Face-down pile = **Deck**, face-up pile = **Draw**. |
| Draw display | Show only the top draw card (one slot, not three). |
| Navigation | All four upgrades: `h` = left (full `hjkl`), region jump keys, face-down skipping, productivity-sorted destinations. |
| Game over | Summary screen replaces the board; `u` undo escapes back into play. |
| Detection | "No productive moves" across the full reachable deck cycle — shuffling tableau cards back and forth does not keep a game alive. |
| Records | Losses are not recorded. Win recording unchanged. |

## Scope

`internal/solitaire/engine` (productive-move + dead-end detection) and `internal/solitaire/ui` (everything else). No store, app, mines, or g2048 changes.

## 1. Hint removal (ui)

Delete from `assist.go`: `hint()`, `bestHintMove()`, `hintRank()`, `describeHintMove()`. Remove the `case "h"` binding in `handleKey` (replaced by cursor-left, §4) and the `h  hint` help line. `assistReason` keeps only `"undo"`; the assisted record mode and its store schema are unchanged.

The productivity ordering that `hintRank` encoded is reborn in two places: destination sorting (§4) and productive-move detection (§5). It lives in one function used by both UI sorting and engine filtering — no duplicated ranking tables.

## 2. Labels (ui)

- `topView` header: ` Stock        Waste             Foundations` → ` Deck   Draw      Foundations` (exact spacing set by the new single-slot layout, §3).
- `describeLocation`: `"waste"` → `"draw pile"`, `"stock"` → `"deck"`. Foundation/tableau strings unchanged.
- `noProgressMessage`: `"No progress this pass — move a card before recycling again"`.
- Help text updated to match (`d  draw 3 / recycle deck`).

## 3. Single-card draw display (ui)

`topView` renders exactly one slot for the draw pile: the top card of `WasteVisible()`, or the empty token when none. The under-cards are never shown. Selection, moves, and cursor semantics are untouched — only the top card was ever selectable. `solitaireMinCols` and `TooSmall()` stay as they are (the layout only narrows).

## 4. Navigation (ui)

| Key | Behavior |
|---|---|
| `h` / `left` | Cursor left (completes `hjkl`; freed by hint removal). |
| `w` | Jump to draw pile. |
| `s` | Jump to deck. |
| `f` | Jump to foundations; pressing `f` again cycles foundation slots 1→2→3→4→1. |
| `1`–`7` | Jump to tableau column (existing). |
| `up`/`k`, `down`/`j` in tableau | Skip face-down cards: land only on face-up rows. From the top face-up row, `up` leaves the column as today; `down` below the last row stays put as today. |

**Smarter enter:** in `activate()`, when a card has multiple legal destinations, sort `selectedMoves` by productivity before storing: (1) reveals a face-down card, (2) safe foundation move, (3) tableau destination, (4) foundation destination. First `enter` selects and pre-focuses the best destination; second `enter` confirms. `tab` still cycles. The existing single-destination instant move is unchanged.

## 5. Game-over detection (engine) and screen (ui)

### Engine

```go
// productive reports whether a move makes progress:
// destination is a foundation, source is the draw pile, or it reveals a face-down card.
func productive(move Move) bool

// ProductiveMoves filters LegalMoves() by productive().
func (g *Game) ProductiveMoves() []Move

// DeadEnd reports whether no productive move is reachable: none now, and none
// after any number of draws/recycles through one full deck cycle (simulated on
// a clone; drawing without moving cards is deterministic and repeats after one
// full pass, so one cycle is sufficient).
func (g *Game) DeadEnd() bool
```

`DeadEnd` returns false immediately when `Status() != Playing`. The simulation bound is the number of draws needed to return to the starting deck/draw configuration, capped at 60 (deck+draw hold at most 24 cards, so a full draw-3 cycle needs at most 9 draws + recycle; the cap is a safety net, never expected to bind).

`productive` deliberately excludes: tableau→tableau moves that reveal nothing (reshuffles, king swaps between empty columns) and draw/recycle themselves.

### UI

- New model flag `gameOver bool`. After every board-changing action (draw, activate move, auto-foundation, auto-all, undo), recompute: `m.gameOver = m.game.Status() == engine.Playing && m.game.DeadEnd()`.
- When `gameOver` is set: stop the timer (`running = false`), and `View()` renders the summary instead of the board (after the too-small and help checks):

```text
 SOLITAIRE DRAW 3

      G A M E   O V E R
      no moves left

      time    145s
      moves     87
      best    120s / 95m

 n new game · u undo · m menu · q quit
```

- `best` line reuses `recordSummary()`.
- Keys while `gameOver`: `n`, `u`, `m`/`esc`, `q`, `?` only; all other keys inert. `u` undoes one move (assisted, as today), clears `gameOver` if the restored position has productive moves, and restarts the timer via the existing undo path.
- Losses write nothing to the store.
- Win path (`Status() == Won`) is checked before `gameOver` and unchanged.

## Testing

**Engine** (constructed positions):
- `productive`: foundation move ✓, draw-pile source ✓, reveal ✓; non-revealing tableau shuffle ✗, king-to-empty from row 0 ✗.
- `DeadEnd` true: full-cycle-dead position (no productive move for any reachable draw top).
- `DeadEnd` false: playable card buried mid-deck (only reachable after several draws); false when a productive move exists now; false when `Status() == Won`.
- `DeadEnd` does not mutate the game (board identical before/after).

**UI:**
- `h` moves cursor left; `w`/`s`/`f` jump; repeated `f` cycles foundations.
- Up/down skip face-down tableau cards.
- Multi-destination selection orders best-first (constructed position where a revealing move exists alongside a plain one).
- Game-over screen renders (heading, time/moves/best, hints) and hides the board; movement keys inert; `u` returns to the board; `n`/`m` work.
- Renamed labels appear (`Deck`, `Draw`, `draw pile` in messages); no `Waste`/`waste` remains in UI output.
- Hint removal: `h` no longer produces hint messages; help has no hint line.

Full suite green; Mines, 2048, store, and app untouched.

## Non-goals

- Unwinnable-game detection (solver). `DeadEnd` detects *no productive moves*, not *cannot win*.
- Loss records or streak tracking.
- Draw-1 mode, scoring modes, or any rule changes.
- Changing assisted-record semantics beyond hints no longer existing.
