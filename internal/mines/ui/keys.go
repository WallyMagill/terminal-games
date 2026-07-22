package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/terminal-games/internal/uiutil"
)

func typedKeys(msg tea.KeyMsg) []string { return uiutil.TypedKeys(msg) }
func isRuneBatch(msg tea.KeyMsg) bool   { return uiutil.IsRuneBatch(msg) }
