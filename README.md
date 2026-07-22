# Terminal Games

A small collection of classic games that run right in your terminal — **Minesweeper**, **Klondike Solitaire (Draw 3)**, and **2048** — bundled behind a single launcher. Built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

```
TERMINAL GAMES

> Minesweeper
  Solitaire
  2048
  Quit

↑↓ move · enter select · q quit
```

## Games

- **Minesweeper** — beginner / intermediate / expert presets, flagging, chording, a live timer, and best-time records per difficulty.
- **Solitaire** — Klondike Draw 3, with cursor navigation, undo, one-key sends to the foundations, a move assist, and dead-end detection. Tracks best time and fewest moves.
- **2048** — slide-to-merge on a 4×4 grid; tracks your best score and highest tile.

Scores and best times persist between sessions in a `scores.json` file inside your OS user-config directory.

## Install

### With `go install`

Requires [Go](https://go.dev/dl/) 1.26 or newer.

```sh
go install github.com/wallymagill/terminal-games@latest
```

This drops a `terminal-games` binary in your `$GOPATH/bin` (usually `~/go/bin`). Make sure that directory is on your `PATH`, then run:

```sh
terminal-games
```

### Build from source

```sh
git clone https://github.com/WallyMagill/terminal-games.git
cd terminal-games
go build -o games .
./games
```

## Controls

Global: `q` / `esc` quits or backs out · `m` returns to the launcher from a game · `ctrl+c` quits anywhere · `?` toggles in-game help.

**Minesweeper**

| Key | Action |
| --- | --- |
| `hjkl` / arrows | move cursor |
| `space` / `enter` | reveal (or chord on a satisfied number) |
| `f` | cycle flag → `?` → clear |
| `c` | chord |
| `n` | new game |
| `m` | menu (change difficulty) |

**Solitaire**

| Key | Action |
| --- | --- |
| `hjkl` / arrows | move cursor |
| `1`–`7` | jump to a tableau column |
| `space` / `enter` | select / move |
| `tab` / `shift+tab` | cycle destinations |
| `d` | draw from the stock |
| `a` | send card to a foundation |
| `A` | auto-send everything available to the foundations |
| `u` | undo |
| `v` | show available moves |
| `n` | new game |

**2048**

| Key | Action |
| --- | --- |
| `←↑↓→` / `hjkl` | slide tiles |
| `n` | new game |
| `m` | menu |

## Development

The code is organized under `internal/`, with each game split into a pure `engine` package (game logic, fully unit-tested) and a `ui` package (the Bubble Tea view/update layer). Shared helpers live in `internal/uiutil`, and score persistence in `internal/store`.

```sh
go build ./...
go vet ./...
go test ./...
```

## License

[MIT](LICENSE) © Wally Magill
