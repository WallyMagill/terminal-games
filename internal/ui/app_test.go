package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

func testApp(t *testing.T) App {
	t.Helper()
	a := NewApp(store.New(t.TempDir() + "/scores.json"))
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return model.(App)
}

func TestAppStartsOnMenu(t *testing.T) {
	a := testApp(t)
	if !strings.Contains(a.View(), "Beginner") {
		t.Error("initial view should be the menu")
	}
}

func TestAppRoutesStartGameToGameView(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	if a.view != viewGame {
		t.Error("startGameMsg should switch to the game view")
	}
	if !strings.Contains(a.View(), "🚩") {
		t.Error("game view should render the HUD")
	}
}

func TestAppMenuKeyReturnsToMenu(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	model, _ = a.Update(key("m"))
	a = model.(App)
	if a.view != viewMenu {
		t.Error("m should return to the menu")
	}
}

func TestAppCtrlCQuits(t *testing.T) {
	a := testApp(t)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c should produce tea.QuitMsg")
	}
}

func TestAppTooSmallTerminal(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[2]}) // Expert: needs 62 cols
	a = model.(App)
	model, _ = a.Update(tea.WindowSizeMsg{Width: 40, Height: 40})
	a = model.(App)
	if !strings.Contains(a.View(), "too small") {
		t.Error("undersized terminal should show the too-small message")
	}
}

func TestAppTickGenerationThreadsAcrossRounds(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	firstGen := a.game.tickGen
	model, _ = a.Update(key("m")) // back to menu (game tick may be in flight)
	a = model.(App)
	model, _ = a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	if a.game.tickGen <= firstGen {
		t.Errorf("new round tickGen = %d, want > %d (stale ticks from the old round must be dropped)", a.game.tickGen, firstGen)
	}
}
