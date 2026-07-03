package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/uiutil"
)

type launcherItem int

const (
	launcherMines launcherItem = iota
	launcherSolitaire
	launcherQuit
	launcherItemCount
)

type launcherModel struct {
	cursor int
	termW  int
}

func newLauncherModel() launcherModel { return launcherModel{} }

func (m launcherModel) Update(msg tea.Msg) (launcherModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	for _, k := range uiutil.TypedKeys(keyMsg) {
		switch k {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < int(launcherItemCount)-1 {
				m.cursor++
			}
		case "enter", " ":
			return m, selectLauncherItem(launcherItem(m.cursor))
		case "q", "esc":
			return m, selectLauncherItem(launcherQuit)
		}
	}
	return m, nil
}

type launcherSelectMsg struct{ item launcherItem }

func selectLauncherItem(item launcherItem) tea.Cmd {
	return func() tea.Msg { return launcherSelectMsg{item: item} }
}

func (m launcherModel) View() string {
	labels := []string{"Minesweeper", "Solitaire", "Quit"}
	var b strings.Builder
	b.WriteString("TERMINAL GAMES\n\n")
	for i, label := range labels {
		prefix := "  "
		if m.cursor == i {
			prefix = "> "
		}
		b.WriteString(prefix + label + "\n")
	}
	b.WriteString("\n↑↓ move · enter select · q quit")
	return b.String()
}
