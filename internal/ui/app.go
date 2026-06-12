package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

type view int

const (
	viewMenu view = iota
	viewGame
)

// App is the Bubble Tea root model: routes between menu and game, owns
// global keys (quit, back-to-menu) and terminal size.
type App struct {
	view   view
	menu   menuModel
	game   gameModel
	scores *store.Store
	termW  int
	termH  int
}

func NewApp(scores *store.Store) App {
	return App{
		menu:   newMenuModel(scores.Load(), 0, 0),
		scores: scores,
	}
}

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.termW, a.termH = msg.Width, msg.Height
		a.menu.termW, a.menu.termH = msg.Width, msg.Height
		a.game.termW = msg.Width
		if a.view == viewGame && a.gameTooSmall() {
			// The too-small screen replaces everything, including an open
			// help overlay; clear it so its catch-all key handling cannot
			// swallow the advertised "press m for menu".
			a.game.showHelp = false
		}
		return a, nil
	case startGameMsg:
		gm, err := newGameModel(msg.preset, a.scores,
			rand.New(rand.NewSource(time.Now().UnixNano())))
		if err != nil {
			a.menu.err = err.Error()
			return a, nil
		}
		// Thread the tick generation across rounds: a tick loop from the
		// previous game may still be in flight, and a fresh model at gen 0
		// would mistake it for its own.
		gm.tickGen = a.game.tickGen + 1
		gm.termW = a.termW
		a.game = gm
		a.view = viewGame
		return a, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		if a.view == viewGame {
			if !a.game.showHelp {
				switch msg.String() {
				case "m", "esc":
					a.view = viewMenu
					a.menu.scores = a.scores.Load() // refresh best times
					a.menu.err = ""
					return a, nil
				case "q":
					return a, tea.Quit
				}
			}
			if a.gameTooSmall() {
				// Only the menu/quit keys above act while the too-small
				// screen is up: other keys must not mutate the invisible
				// game (e.g. ? re-opening the overlay it just replaced).
				return a, nil
			}
		}
	}
	var cmd tea.Cmd
	if a.view == viewGame {
		a.game, cmd = a.game.Update(msg)
	} else {
		a.menu, cmd = a.menu.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	if a.view != viewGame {
		return "\n" + a.menu.View() + "\n"
	}
	if a.gameTooSmall() {
		return a.tooSmallView()
	}
	return "\n" + a.game.View() + "\n"
}

// tooSmallView renders the guard screen with progressive degradation so the
// fallback itself fits every terminal that can trigger it: the full stacked
// form when it fits, the payload lines without margins under height
// pressure, compact variants under width pressure, and a two-line floor
// below that. A final clamp covers terminals narrower than any variant.
func (a App) tooSmallView() string {
	needW, needH := frameCols(a.game.game.Width()), frameRows(a.game.game.Height())
	full := []string{
		"",
		" Terminal too small.",
		fmt.Sprintf(" need %d×%d", needW, needH),
		fmt.Sprintf(" have %d×%d", a.termW, a.termH),
		" press m for menu.",
		"",
	}
	compact := []string{
		"too small",
		fmt.Sprintf("need %d×%d", needW, needH),
		fmt.Sprintf("have %d×%d", a.termW, a.termH),
		"m: menu",
	}
	var lines []string
	switch {
	case allFit(full, a.termW) && len(full) <= a.termH:
		lines = full
	case allFit(full[1:5], a.termW) && len(full)-2 <= a.termH:
		lines = full[1:5]
	case allFit(compact, a.termW) && len(compact) <= a.termH:
		lines = compact
	case a.termH >= 2:
		lines = []string{compact[0], compact[3]}
	default:
		lines = compact[:1]
	}
	for i, l := range lines {
		lines[i] = clampLine(l, a.termW)
	}
	return strings.Join(lines, "\n")
}

// gameTooSmall reports whether the active game's full frame — board plus
// chrome — cannot fit the terminal. A zero width means the size is not yet
// known; render optimistically. Only valid while a game is active.
func (a App) gameTooSmall() bool {
	return a.termW > 0 && (a.termW < frameCols(a.game.game.Width()) ||
		a.termH < frameRows(a.game.game.Height()))
}
