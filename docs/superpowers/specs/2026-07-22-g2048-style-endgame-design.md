# 2048 Style Pass + Game-Over Screen — Design

**Date:** 2026-07-22
**Status:** Approved
**Goal:** Polish the 2048 board's color ramp and replace the inline `GAME OVER` status with a dedicated end-game summary screen. No rule, key, store, or layout-size changes.

## Decisions made during brainstorming

| Question | Decision |
|---|---|
| End-game presentation | Summary screen **replaces the board** on game over. |
| Win presentation | Unchanged — one-time inline `YOU WIN — keep going` banner; play continues. |
| Board style | Foreground polish only (Style B). No background fills, no box-drawing grid — matches the plain-text aesthetic of Mines and Solitaire. |
| Scope | `internal/g2048/ui` only (`styles.go`, `game.go`, view tests). |

## Style pass (styles.go)

New tile foreground ramp (256-color codes), replacing the current one:

| Value | Color | Bold |
|---|---|---|
| empty (0) | `238` dim `·` placeholder | no |
| 2–4 | `250` light gray | no |
| 8–16 | `222` warm yellow | no |
| 32–64 | `214` orange | no |
| 128–256 | `208` deep orange | no |
| 512–1024 | `203` red-orange | **yes** |
| 2048+ | `227` bright gold | **yes** |

Rationale: the current ramp goes bold only past 1024 and gives 512/1024 no distinction; the new ramp makes high tiles visibly "heat up" and crowns 2048+ in gold. Empty cells drop from `240` to `238` so tiles pop more.

Cell width, alignment, HUD, hints, and help styling are unchanged.

## Game-over screen (game.go View)

When `game.Over()` is true (and not too-small, and help not open), `View()` renders a summary **instead of** the board:

```text
 2048
 SCORE 1464    BEST 1464

      G A M E   O V E R

      score   1464  ★ new best
      tile     128  ★ new best
      best    1464

 n new game · m menu · q quit
```

Rules:

- Heading: spaced `G A M E   O V E R`, styled with the error/emphasis style.
- `score` = final run score; shows `★ new best` when this run set a new best score (`newBestScore`).
- `tile` = `HighestTile()` for this run; shows `★ new best` when this run set a new best tile (`newBestTile`).
- `best` = persisted best score.
- Key hints line replaces the normal footer: `n new game · m menu · q quit`.
- Save-error status (`could not save record`) still surfaces on this screen when `saveErr` is set.

Behavior is untouched: this is purely a `View()` branch. `n` starts a new run (board returns), `m`/`esc` emit `BackMsg`, `q` quits, direction keys remain inert when over. The win banner path is unchanged.

The summary renders fewer lines than the board view, so `g2048MinCols`/`g2048MinRows` and `TooSmall()` are unchanged.

## Testing

- View test: dead-board model renders `G A M E   O V E R`, final score, highest tile, and hides the board grid (fewer than 4 `·` occurrences — the hint line's `·` separators remain, but the 14+ cell placeholders must be gone).
- View test: `★ new best` markers appear exactly when `newBestScore` / `newBestTile` are set.
- Style test: ramp thresholds return distinct styles at 4→8, 16→32, 64→128, 256→512, 1024→2048 boundaries (bold at 512+, gold at 2048+).
- All existing engine/UI/app/store tests stay green; Mines and Solitaire untouched.

## Non-goals

- Background-fill tiles, box-drawing grid, animations.
- Any change to win-continue behavior, keys, store schema, or records timing.
- End screen for wins.
