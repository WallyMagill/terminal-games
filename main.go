package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
	"github.com/wallymagill/mines/internal/ui"
)

func main() {
	path, err := store.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mines: cannot determine config dir:", err)
		os.Exit(1)
	}
	app := ui.NewApp(store.New(path))
	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "mines:", err)
		os.Exit(1)
	}
}
