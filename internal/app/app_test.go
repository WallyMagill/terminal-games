package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

func testApp(t *testing.T) App {
	t.Helper()
	a := New(store.New(t.TempDir() + "/scores.json"))
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return model.(App)
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestStartsOnLauncher(t *testing.T) {
	a := testApp(t)
	v := a.View()
	if !strings.Contains(v, "TERMINAL GAMES") || !strings.Contains(v, "Minesweeper") || !strings.Contains(v, "Solitaire") {
		t.Fatalf("launcher missing expected labels:\n%s", v)
	}
}

func TestLauncherSelectsMinesweeper(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(key("enter"))
	a = model.(App)
	if a.view != viewMines {
		t.Fatalf("view = %v, want viewMines", a.view)
	}
	if !strings.Contains(a.View(), "Beginner") {
		t.Fatalf("Mines menu did not render:\n%s", a.View())
	}
}

func TestLauncherSelectsSolitaireTemporaryScreen(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("enter"))
	a = model.(App)
	if a.view != viewSolitaire {
		t.Fatalf("view = %v, want viewSolitaire", a.view)
	}
	if !strings.Contains(a.View(), "SOLITAIRE") {
		t.Fatalf("Solitaire temporary screen did not render:\n%s", a.View())
	}
}

func TestMinesMenuEscReturnsToLauncher(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(key("enter"))
	a = model.(App)
	_, cmd := a.Update(key("esc"))
	if cmd == nil {
		t.Fatal("esc in Mines menu should request launcher back")
	}
	model, _ = a.Update(cmd())
	a = model.(App)
	if a.view != viewLauncher {
		t.Fatalf("view = %v, want viewLauncher", a.view)
	}
	if !strings.Contains(a.View(), "TERMINAL GAMES") {
		t.Fatalf("launcher did not render after Mines back:\n%s", a.View())
	}
}

func TestSolitaireEscReturnsToLauncher(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("enter"))
	a = model.(App)
	model, _ = a.Update(key("esc"))
	a = model.(App)
	if a.view != viewLauncher {
		t.Fatalf("view = %v, want viewLauncher", a.view)
	}
}

func TestLauncherQuit(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("down"))
	a = model.(App)
	_, cmd := a.Update(key("enter"))
	if cmd == nil {
		t.Fatal("selecting Quit should return tea.Quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("selecting Quit should produce tea.QuitMsg")
	}
}
