# Terminal Games + Classic 2048 — Design

**Date:** 2026-07-15  
**Status:** Ready for review  
**Goal:** Add classic 4×4 2048 as a third launcher game alongside Minesweeper and Klondike Solitaire, preserving the suite feel: compact terminal rendering, keyboard-only controls, pure game engines, and simple persistent records.

## Decisions made during brainstorming

| Question | Decision |
|---|---|
| Third game | Classic 2048 (not Sokoban, Nonogram, or other candidates). |
| Board | 4×4 only in v1. |
| Win behavior | **Continue after first 2048 tile** (classic): show a win banner once, allow further play for higher score / larger tiles. |
| Modes | Single classic mode only. No 5×5, no undo, no custom goals. |
| Launcher label | `2048`. |
| Package name | `internal/g2048` (Go package identifiers cannot start with a digit). |
| Visual direction | Match suite style: compact terminal text, Lip Gloss styling, bordered tile cells, no mouse, no slide animations. |
| Architecture | Explicit third game packages under `g2048/engine` + `g2048/ui`. No generic game plugin framework. |
| Records | Best score (live updates) and best tile ever reached for classic mode. |
| Store version | Bump multi-game records to version 3; migrate v2 files in memory. |
| Config path | Keep existing `os.UserConfigDir()/mines/scores.json`. |
| First-version exclusions | Undo, 5×5, themes, animations, seed picker, timer records, fewest-moves-to-2048, AI/autoplay, multiplayer. |

## Product shape

The launcher becomes a three-game menu:

```text
TERMINAL GAMES

  Minesweeper
  Solitaire
> 2048
  Quit

↑↓ move · enter select · q quit
```

Selecting **2048** starts a new classic game immediately. There is no difficulty submenu in v1.

Flow:

```text
Launcher
  ├── Minesweeper
  │     └── (unchanged)
  ├── Solitaire
  │     └── (unchanged)
  └── 2048
        └── Classic 4×4 game
```

Navigation:

| Context | Key | Behavior |
|---|---|---|
| Launcher | `enter` / `space` | Select launcher item. |
| Launcher | `q` / `ctrl+c` | Quit. |
| 2048 game | `m` / `esc` | Back to launcher (unless help is open). |
| 2048 game | `?` | Help overlay. |
| 2048 game | `n` | New run (same classic rules). |
| 2048 game | `q` / `ctrl+c` | Quit app. |
| 2048 game | `←↑↓→` / `hjkl` | Move board. |

Help overlay precedence (match Solitaire):

1. If help is open, **any key** closes help.
2. Else `esc` / `m` returns to launcher.

## Architecture

Extend the existing explicit multi-game layout. Do not introduce a plugin `Game` interface.

Target additions:

```text
terminal-game/
├── main.go
└── internal/
    ├── app/
    │   ├── app.go           // + view2048 routing
    │   └── launcher.go      // + 2048 item
    ├── g2048/
    │   ├── engine/
    │   │   ├── board.go     // 4×4 grid helpers
    │   │   ├── game.go      // status, score, won flag
    │   │   ├── actions.go   // Move(dir), spawn, merge
    │   │   └── *_test.go
    │   └── ui/
    │       ├── game.go      // Bubble Tea model, keys, HUD
    │       ├── styles.go    // tile colors by value
    │       └── *_test.go
    ├── mines/               // unchanged behavior
    ├── solitaire/           // unchanged behavior
    ├── uiutil/              // shared key/text helpers only
    └── store/
        └── store.go         // + G2048 records, version 3
```

Package responsibilities:

- `internal/app`: root model routes launcher ↔ mines ↔ solitaire ↔ 2048.
- `internal/g2048/engine`: pure classic 2048 rules; zero TUI imports.
- `internal/g2048/ui`: rendering, keys, help, best-score display, `BackMsg`.
- `internal/store`: multi-game records including 2048 bests; backward-compatible load.

Conceptual root app state (names may vary):

```go
type view int

const (
    viewLauncher view = iota
    viewMines
    viewSolitaire
    view2048
)

type App struct {
    view      view
    launcher  launcherModel
    mines     tea.Model
    solitaire solitaireui.Model
    g2048     g2048ui.Model
    scores    *store.Store
    termW     int
    termH     int
}
```

On launcher select for 2048: construct a fresh UI model with a new RNG seed (same pattern as Solitaire). On `n`, reset the engine with a new game; keep the same UI model.

## Classic 2048 rules

### Board

- Fixed **4×4** grid of non-negative integers.
- `0` means empty.
- No other sizes in v1.

### Start

- Empty board, then spawn **two** tiles using the spawn rules below.

### Spawn

After each move that changes the board, and twice at game start:

- Choose a uniformly random empty cell.
- Place `2` with probability **90%**, `4` with probability **10%**.
- RNG is injected (`*rand.Rand`) so tests are deterministic.

### Move

A move is one of four directions: Up, Down, Left, Right.

For each line in the move direction (row or column):

1. Slide all non-zero tiles as far as possible toward the direction.
2. Merge: adjacent equal values combine **once** into their sum, left-to-right (or top-to-bottom) in the direction of motion. A tile created by a merge cannot merge again in the same move.
3. Slide again so there are no gaps between tiles after merges.

Classic double-merge examples:

| Before (left move) | After |
|---|---|
| `[2, 2, 2, 2]` | `[4, 4, 0, 0]` |
| `[2, 2, 4, 0]` | `[4, 4, 0, 0]` |
| `[4, 4, 8, 8]` | `[8, 16, 0, 0]` |
| `[2, 0, 2, 0]` | `[4, 0, 0, 0]` |

### No-op

If a direction would leave the board unchanged, the move is a **no-op**:

- Do not spawn a tile.
- Do not change score.
- Do not change status.

### Score

- Start at 0.
- When two tiles of value `v` merge into `2v`, add **`2v`** to the score.
- Score never decreases during a run.

### Win (continue)

- The first time any tile value is **≥ 2048**, the run is marked **won**.
- Show a one-time win indication in the UI (status line / banner).
- **Play continues** after win for higher score and larger tiles (4096, etc.).
- Further 2048+ tiles do not re-trigger the win banner.

### Lose (game over)

The game is over when:

- There is no empty cell, **and**
- No two adjacent (orthogonal) cells share the same positive value.

Game over freezes moves: direction keys are no-ops (or show a brief status). `n` starts a new run; `m`/`esc` returns to launcher.

### New game

`n` replaces the engine state with a fresh `New(rng)` (or reseeded equivalent). Clear the per-run win-banner-already-shown flag.

## Engine model

The engine has no Bubble Tea, Lip Gloss, timer, or terminal-size dependency.

Core shape:

```go
type Dir int

const (
    DirUp Dir = iota
    DirDown
    DirLeft
    DirRight
)

type Status int

const (
    StatusPlaying Status = iota
    StatusOver
)

type Game struct {
    // unexported board, score, won, over, rng
}

type MoveResult struct {
    Changed bool
    Score   int // total score after move
    Won     bool // true if this move first reached 2048
    Over    bool // true if board is dead after this move
}

func New(rng *rand.Rand) *Game
func (g *Game) Move(d Dir) MoveResult
func (g *Game) Board() [4][4]int
func (g *Game) Score() int
func (g *Game) HighestTile() int
func (g *Game) Won() bool   // ever reached 2048 this run
func (g *Game) Over() bool
```

Notes:

- After win-with-continue, `Won() == true` and `Over() == false` until a dead board.
- When over, `Over() == true`; `Move` returns `Changed: false` without mutating state.
- Out-of-contract calls must not panic; invalid states are programmer errors caught by tests.

## UI / UX

### Layout

```text
 ┌─ 2048 ───────────────────────────┐
 │  SCORE  1234     BEST  5678      │
 ├──────────────────────────────────┤
 │    ┌────┬────┬────┬────┐         │
 │    │  2 │  4 │    │    │         │
 │    ├────┼────┼────┼────┤         │
 │    │  8 │ 16 │  2 │    │         │
 │    ├────┼────┼────┼────┤         │
 │    │ 32 │    │  4 │  2 │         │
 │    ├────┼────┼────┼────┤         │
 │    │    │ 64 │128 │  2 │         │
 │    └────┴────┴────┴────┘         │
 ├──────────────────────────────────┤
 │  status / win / game over        │
 │  ←↑↓→ / hjkl move · n new        │
 │  m menu · ? help · q quit        │
 └──────────────────────────────────┘
```

Exact box-drawing vs padded cells is an implementation detail; requirements:

- Fixed cell width so values through at least **65536** fit without breaking alignment.
- Empty cells clearly empty (blank or dim placeholder), not Mines-style hidden glyphs.
- Tile values colored by magnitude via Lip Gloss (2/4 muted; mid values warmer; high values bright). Exact palette in `styles.go`.
- HUD shows current score and persisted best score.
- When a new best score is saved, indicate it (e.g. status or BEST highlight) consistent with Solitaire’s “new best” feedback.
- Optional HUD detail: show best tile from records when useful; not required if it crowds the compact layout.

### Status lines

| State | Message (wording may be tightened in UI) |
|---|---|
| Playing | Empty or last action hint |
| Just won | `YOU WIN — keep going` (once) |
| Playing after win | Normal; no repeated win spam |
| Game over | `GAME OVER` |
| New best | Include in status when a record is saved |

### No board cursor

Unlike Mines/Solitaire, 2048 has no cell cursor. Every move is a whole-board direction.

### Terminal size

If the frame does not fit, show a too-small message (same idea as Solitaire) rather than a broken grid.

### Help overlay

`?` opens a short help listing keys and one-line rules (merge equals, reach 2048, continue, game over). Closing help matches Solitaire behavior.

## Records and store

### Schema

Bump `recordsVersion` to **3**.

```json
{
  "version": 3,
  "mines": { "...": {} },
  "solitaire": { "...": {} },
  "g2048": {
    "classic": {
      "bestScore": 12345,
      "bestTile": 2048
    }
  }
}
```

Types (illustrative):

```go
type G2048Record struct {
    BestScore int `json:"bestScore"`
    BestTile  int `json:"bestTile"`
}

// Records gains:
// G2048 map[string]G2048Record `json:"g2048,omitempty"`
// key: "classic"
```

### When to write

- **Best score:** whenever current score exceeds stored best — **live during play** after a scoring merge (classic 2048 feel), not only on game over.
- **Best tile:** whenever `HighestTile()` exceeds stored best tile (including mid-run).

Both may update on the same move. Use a dedicated store API, e.g. `RecordG2048Classic(score, highestTile) (newBestScore, newBestTile bool, err error)`, that preserves mines and solitaire data.

### Migration

- Load version 3 records as-is (sanitize non-negative scores/tiles).
- Load version 2 records: keep mines/solitaire; empty `g2048`.
- Load legacy mines-only map: existing path unchanged; empty solitaire and g2048.
- Missing or corrupt file: empty records; never crash the app.
- Reject nonsense values (negative bests) on sanitize.

### Config path

Unchanged: `os.UserConfigDir()/mines/scores.json`.

## App integration

| Item | Behavior |
|---|---|
| Launcher item | `2048` between Solitaire and Quit (or after Solitaire; order: Minesweeper, Solitaire, 2048, Quit). |
| Enter 2048 | New model + fresh RNG; forward window size if known. |
| Leave 2048 | `BackMsg` → launcher (do not auto-save partial state beyond records already written). |
| Preserve | Mines and Solitaire behavior unchanged except launcher list growth. |

## Testing

### Engine (required)

- Move left/right/up/down golden boards.
- Double-merge: `[2,2,2,2]` → `[4,4,0,0]` (and directional equivalents).
- No-op direction does not spawn and does not change score.
- Spawn only into empty cells; 90/10 distribution testable via fixed RNG sequence or mocked draws if exposed for tests.
- Win flag set exactly once when first ≥2048 appears; further moves allowed.
- Game over when full and no merges; `Move` is inert when over.
- Score increments by merged tile value only.
- Deterministic full game script with fixed seed.

### Store

- v2 → v3 migration preserves mines/solitaire.
- Recording 2048 does not wipe other games.
- Best score only updates when strictly greater.
- Best tile only updates when strictly greater.

### UI (lightweight)

- Direction keys and `hjkl` call the engine.
- `n` resets; `m`/`esc` emit `BackMsg`; help toggles.
- Too-small path if implemented.

## Implementation order (guidance for plan)

1. Engine + unit tests (rules complete before UI).
2. Store v3 + 2048 record API + tests.
3. UI game view + styles + help.
4. Launcher + app routing.
5. Manual play pass: win-continue, game over, records, resize.

## Non-goals (v1)

- Undo / redo
- Board sizes other than 4×4
- Animations or multi-frame slides
- Mouse input
- Themes / alternate palettes as settings
- Seed entry or daily challenge mode
- Timer or moves-to-2048 records
- “End run on 2048” mode
- Online leaderboards

## Success criteria

- Classic 2048 feels correct (merge rules, spawn, continue after win, game over).
- Keyboard ergonomics match the suite (`hjkl` + arrows, `n`/`m`/`?`/`q`).
- Best score (and best tile) persist across sessions without corrupting mines/solitaire records.
- Code structure mirrors mines/solitaire: pure engine, thin UI, explicit app routing.
- Mines and Solitaire remain behaviorally unchanged aside from the new launcher entry.
