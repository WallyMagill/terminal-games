package app

import (
	legacyrand "math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	minesui "github.com/wallymagill/mines/internal/mines/ui"
	solitaireui "github.com/wallymagill/mines/internal/solitaire/ui"
	"github.com/wallymagill/mines/internal/store"
	"github.com/wallymagill/mines/internal/uiutil"
)

type view int

const (
	viewLauncher view = iota
	viewMines
	viewSolitaire
)

type App struct {
	view      view
	launcher  launcherModel
	mines     tea.Model
	solitaire solitaireui.Model
	scores    *store.Store
	termW     int
	termH     int
}

func New(scores *store.Store) App {
	rng := legacyrand.New(legacyrand.NewSource(time.Now().UnixNano()))
	solitaire, _ := solitaireui.NewModel(scores, rng)
	return App{
		view:      viewLauncher,
		launcher:  newLauncherModel(),
		mines:     minesui.NewApp(scores),
		solitaire: solitaire,
		scores:    scores,
	}
}

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.termW, a.termH = msg.Width, msg.Height
		a.launcher.termW = msg.Width
		if a.view == viewMines {
			model, cmd := a.mines.Update(msg)
			a.mines = model
			return a, cmd
		}
		if a.view == viewSolitaire {
			model, cmd := a.solitaire.Update(msg)
			a.solitaire = model
			return a, cmd
		}
		return a, nil
	case launcherSelectMsg:
		switch msg.item {
		case launcherMines:
			a.view = viewMines
			if a.termW > 0 || a.termH > 0 {
				model, cmd := a.mines.Update(tea.WindowSizeMsg{Width: a.termW, Height: a.termH})
				a.mines = model
				return a, cmd
			}
			return a, nil
		case launcherSolitaire:
			solitaire, err := solitaireui.NewModel(a.scores, legacyrand.New(legacyrand.NewSource(time.Now().UnixNano())))
			if err == nil {
				if a.termW > 0 || a.termH > 0 {
					model, _ := solitaire.Update(tea.WindowSizeMsg{Width: a.termW, Height: a.termH})
					solitaire = model
				}
				a.solitaire = solitaire
				a.view = viewSolitaire
			}
			return a, nil
		case launcherQuit:
			return a, tea.Quit
		}
	case minesui.BackMsg:
		a.view = viewLauncher
		return a, nil
	case solitaireui.BackMsg:
		a.view = viewLauncher
		return a, nil
	case tea.KeyMsg:
		if !uiutil.IsRuneBatch(msg) && msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	}

	switch a.view {
	case viewLauncher:
		var cmd tea.Cmd
		a.launcher, cmd = a.launcher.Update(msg)
		if cmd == nil {
			return a, nil
		}
		if selectMsg, ok := cmd().(launcherSelectMsg); ok {
			return a.Update(selectMsg)
		}
		return a, cmd
	case viewMines:
		model, cmd := a.mines.Update(msg)
		a.mines = model
		return a, cmd
	case viewSolitaire:
		model, cmd := a.solitaire.Update(msg)
		a.solitaire = model
		return a, cmd
	default:
		return a, nil
	}
}

func (a App) View() string {
	switch a.view {
	case viewLauncher:
		return "\n" + a.launcher.View() + "\n"
	case viewMines:
		return a.mines.View()
	case viewSolitaire:
		return "\n" + a.solitaire.View() + "\n"
	default:
		return ""
	}
}
