# Solitaire Usability Round 2 — Design

**Date:** 2026-07-22
**Status:** Approved
**Goal:** Add a show-possible-moves key, fix the disappearing cursor, and rename the built binary to `games`.

## Decisions made during brainstorming

| Question | Decision |
|---|---|
| Show moves | Press `v`: all sources with a **productive** move highlight at once; any key dismisses. |
| Which moves | Productive only (same `engine.ProductiveMoves()` definition as dead-end detection). |
| Records | Using `v` marks the run **assisted** (reason `"moves"`), like undo. |
| Cursor styling | **Unchanged** — plain reverse video for the cursor, bold reverse for selection. |
| Cursor problem | It *disappears* on empty columns and after moves; fix persistence, not style. |
| Binary name | `./games` (and `~/go/bin/games`); stale `mines` binaries deleted. Module path unchanged. |

## 1. Show possible moves (`v`)

New key `v` (board view only; inert on game over, win, help, and too-small):

- Compute `m.game.ProductiveMoves()` and collect the distinct source locations.
- If at least one: set a `showMoves` display flag; every source cell renders with a highlight style (bold + green background `lipgloss.Color("28")`, white text) — distinct from cursor (reverse) and selection (bold reverse). Status line: `N moves · any key to dismiss` (N = number of productive moves).
- If none on the board (game not over, so a draw card must be buried in the cycle): no highlights; status line: `no moves on board — keep drawing`. This state auto-clears like any status message.
- Pressing `v` sets `assisted = true`, `assistReason = "moves"` (only when the game is `Playing`).
- Dismissal: any subsequent key clears the highlights AND still performs its normal action (unlike help, which swallows the key). Cursor movement, selection, draws, etc. work immediately.
- Highlighting draws from source locations: draw-pile top, tableau cells, deck (`MoveDraw` is never productive, so the deck itself never highlights; a productive waste source highlights the draw slot).
- Selection interaction: pressing `v` clears any active selection first (like `esc`), so highlight semantics are unambiguous.

## 2. Cursor persistence (no style change)

Two rendering/state fixes; reverse-video styling stays exactly as is.

**Empty-column placeholder:** `tableauView` renders a dim `[  ]` slot (the existing `emptyToken` look) at row 0 of every empty tableau column, and that slot passes through `decorate`, so a cursor on an empty column shows as an inverted empty slot. (The cursor can already legally sit there — kings are placed onto empty columns.)

**Post-mutation re-clamp:** after every board mutation — successful move via `activate`, `autoFoundation`, `autoFoundationAll`, `undo`, and new deal — the cursor immediately re-clamps to a valid position in its current column via the existing `clampTableauRowToPlayable` (tableau region only; other regions are always valid). The cursor is therefore always on a rendered cell on the very next frame.

## 3. Binary rename to `./games`

- The repo-root build artifact becomes `games`: build with `go build -o games .`.
- `~/go/bin`: after `go install .` (which produces `mines` from the module path), copy to `~/go/bin/games` and delete `~/go/bin/mines`.
- Delete the stale `./mines` binary from the repo root.
- Update `.gitignore` if it lists `mines` (add `games`).
- The Go module path stays `github.com/wallymagill/mines` (locked decision from the original suite design).

## Help text

Add to the Solitaire help overlay: `v            show possible moves (assisted)`.

## Testing

**UI:**
- `v` on a position with productive moves: exactly the productive source cells render highlighted (constructed position via `setUnexported`); status shows the count; `assisted` set with reason `"moves"`.
- `v` with no board moves but a live deck (buried playable card): no highlights, status `no moves on board — keep drawing`, run marked assisted.
- Any key after `v` both clears highlights and performs its action (e.g. cursor moves).
- `v` is inert on the game-over screen.
- Cursor on an empty column renders an inverted placeholder (view contains a decorated empty slot).
- After a move that shortens the cursor's pile (or empties it), the immediate `View()` contains a cursor-styled cell without any cursor keypress.
- Help lists the `v` key.

**Build:** final verification builds `./games` and confirms the stale binaries are gone.

## Non-goals

- Best-move ranking or hint text (removed feature stays removed; `v` shows *what* is legal-and-productive, never *which* to prefer).
- Highlighting non-productive shuffle moves.
- Any cursor/selection restyling.
- Module rename.
