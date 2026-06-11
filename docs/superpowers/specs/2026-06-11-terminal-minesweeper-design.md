# Terminal Minesweeper — Design

**Date:** 2026-06-11
**Status:** Approved
**Goal:** A polished, keyboard-driven terminal minesweeper for personal daily use. Full classic (Windows-style) minesweeper functionality, nothing more.

## Decisions made during brainstorming

| Question | Decision |
|---|---|
| Purpose | Polished personal tool — feel, keyboard ergonomics, visual polish |
| Language/stack | Go + Bubble Tea + Lip Gloss |
| Scope | Classic feature set only; keyboard-only; one default color scheme |
| Architecture | Pure engine package + thin Bubble Tea TUI layer |
| Binary name | `mines` |

## Feature set (classic minesweeper, complete)

- Difficulty presets: Beginner 9×9/10 mines, Intermediate 16×16/40, Expert 30×16/99, plus custom width/height/mines.
- Reveal with numbers 1–8; zero cells flood-fill (BFS) their connected region plus numbered border.
- First-click safety: first reveal is never a mine. Mines are placed lazily on first reveal with the clicked cell excluded (classic rule — first click may still be a number; no guaranteed-opening variant).
- Flag cycle on hidden cells: hidden → flagged → question mark → hidden. Flagged cells cannot be revealed; question-marked cells can (classic rule — only flags block reveals).
- Chording: activating a revealed number whose adjacent flag count equals its value reveals all remaining unflagged neighbors (question-marked neighbors count as hidden and are revealed). Wrong flags cause a loss, faithful to the original.
- Win: all non-mine cells revealed (flags irrelevant); remaining unflagged mines are auto-flagged in the display. Loss: reveal a mine — all mines exposed, incorrect flags marked.
- Mine counter (mines − flags placed; may go negative), timer starting at first reveal (caps at 999), status face (playing/won/lost).
- Persistent best times per preset (custom boards do not record times).

## Architecture

```
terminal-game/
├── go.mod                  module github.com/wallymagill/mines
├── main.go                 entry point (~15 lines)
└── internal/
    ├── engine/             pure game logic — zero TUI imports
    │   ├── board.go        Board/Cell types, lazy mine placement, adjacency counts
    │   ├── actions.go      reveal + flood fill, flag cycle, chord
    │   └── game.go         status (NotStarted/Playing/Won/Lost), win/loss detection
    ├── ui/                 Bubble Tea + Lip Gloss presentation
    │   ├── app.go          root model, routes menu ↔ game
    │   ├── menu.go         difficulty select, custom board form, best times display
    │   ├── game.go         board rendering, cursor, HUD, timer tick
    │   └── styles.go       colors and cell glyphs
    └── store/
        └── store.go        best-times persistence (JSON)
```

### Engine contract

- `Cell` = mine flag + state (`Hidden | Revealed | Flagged | Questioned`) + adjacent-mine count.
- Public API: `New(w, h, mines, rng) (*Game, error)`, `Reveal(x, y)`, `CycleFlag(x, y)`, `Chord(x, y)`, plus read-only accessors for status, board snapshot, mine counter.
- RNG injected (seeded `*rand.Rand`) so generation is deterministic in tests.
- Go-idiomatic internal mutation behind the method API (per ECC language-override rule); UI only reads snapshots — no shared mutable state crosses the package boundary.
- The engine has no clock. The timer is UI state: ticks once per second while status is `Playing`; elapsed seconds passed to the store on a win.
- Out-of-range coordinates and wrong-state actions (e.g., reveal a flagged cell) are safe no-ops with a typed result. The engine never panics.

### UI/UX

Two views under one root model: **Menu** (presets with inline best times, custom form, quit) and **Game**. Win/loss shown in-place in the game view with a status line — no separate screen.

Game screen layout:

```
 ┌─ MINES ──────────────────────────┐
 │  🚩 032          🙂        ⏱ 047 │
 ├──────────────────────────────────┤
 │  ░░ ░░ 1  ·  ·  1  ░░ ░░         │
 │  ░░ 2  1  ·  ·  1  2  ░░         │
 │  1  1  ·  ·  ·  ·  1 [░]         │
 │  ·  ·  ·  1  1  1  1  ░░         │
 ├──────────────────────────────────┤
 │ hjkl/←↓↑→ move · spc reveal      │
 │ f flag · c chord · n new · q quit│
 └──────────────────────────────────┘
```

- Cells render 2 chars wide. `░░` hidden, `·` revealed empty, `⚑` flag, `?` question, `✱` mine, cursor as bracket highlight.
- Glyphs target modern macOS terminals (Unicode + emoji assumed; ⚑/░/✱ in the grid, 🚩/🙂/⏱ in the HUD). No ASCII fallback — out of scope.
- Numbers in the classic Windows palette: 1 blue, 2 green, 3 red, 4 navy, 5 maroon, 6 teal, 7 white, 8 gray.
- Keys: `hjkl`/arrows move; `space`/`enter` reveal (chord when on a revealed number); `f` flag cycle; `c` chord explicitly; `n` new game same settings; `m`/`esc` menu; `q`/`ctrl+c` quit; `?` help overlay.
- Resize: on `WindowSizeMsg`, if the board doesn't fit, show "terminal too small (need W×H)" instead of clipping. Custom form validates against current terminal size.

### Persistence & error handling

- Best times JSON at `os.UserConfigDir()/mines/scores.json`, written atomically (temp file + rename), only when a record is beaten.
- Missing/corrupted scores file → start empty, overwrite on next save. Score I/O never crashes the game; save failure shows a one-line HUD warning.
- Custom board validated at the form (numeric; min 2×2; 1 ≤ mines ≤ w×h−2 so first-click safety always has a free cell; must fit terminal) and again in the engine constructor — both boundaries.

## Testing (TDD throughout)

- **Engine** (≥90%): table-driven tests — first-click exclusion, flood fill (corners, edges, full cascade), flag cycle, chording (correct/wrong/insufficient flags), win/loss, constructor validation. Deterministic via seeded RNG.
- **Store**: round-trip, corrupted-file recovery, atomic write — against `t.TempDir()`.
- **UI**: `Update` tested as message-in/model-out (cursor movement, key routing, game-over keys); one `teatest` golden-file smoke test of a small rendered board. Visual polish verified by playing.
- Overall ≥80% via `go test -cover`.

## Build & run

- Dev: `go build -o mines . && ./mines`
- Install: `go install` → `mines` on PATH.

## Out of scope

Mouse support, themes, guaranteed-opening first click, sounds, online/shared leaderboards, replays, hints/solver.
