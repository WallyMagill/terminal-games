package ui

import (
	"fmt"
	"math/rand"
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
		a.game = gm
		a.view = viewGame
		return a, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		if a.view == viewGame && !a.game.showHelp {
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
	needW := boardCols(a.game.game.Width())
	needH := a.game.game.Height() + boardChromeRows
	if a.termW > 0 && (a.termW < needW || a.termH < needH) {
		return fmt.Sprintf(
			"\n Terminal too small — need %d×%d, have %d×%d.\n Resize, or press m for menu.\n",
			needW, needH, a.termW, a.termH)
	}
	return "\n" + a.game.View() + "\n"
}
