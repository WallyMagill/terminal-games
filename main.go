package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/terminal-games/internal/app"
	"github.com/wallymagill/terminal-games/internal/store"
)

func main() {
	path, err := store.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "terminal-games: cannot determine config dir:", err)
		os.Exit(1)
	}
	program := app.New(store.New(path))
	if _, err := tea.NewProgram(program, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "terminal-games:", err)
		os.Exit(1)
	}
}
