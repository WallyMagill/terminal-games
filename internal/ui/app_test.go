package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/engine"
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

func TestAppTooSmallExpertAt80Cols(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[2]}) // Expert: 30 wide → 91 cols
	a = model.(App)
	model, _ = a.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	a = model.(App)
	if !strings.Contains(a.View(), "too small") {
		t.Error("Expert on an 80-column terminal should show the too-small message")
	}
}

func TestHelpOverlaySwallowsMenuKeys(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	model, _ = a.Update(key("?"))
	a = model.(App)
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = model.(App)
	if a.view != viewGame {
		t.Error("esc while help is open must close help, not abandon the game")
	}
	if a.game.showHelp {
		t.Error("esc should have closed the help overlay")
	}
}

func TestAppQKeyQuitsFromGame(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	_, cmd := a.Update(key("q"))
	if cmd == nil {
		t.Fatal("q in game view should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("q should produce tea.QuitMsg")
	}
}

func TestAppWinShowsBestTimeOnMenu(t *testing.T) {
	a := testApp(t)
	tiny := Preset{Name: "Tiny", Key: "beginner", Width: 2, Height: 2, Mines: 1}
	model, _ := a.Update(startGameMsg{preset: tiny})
	a = model.(App)
	// First reveal at (0,0): first-click safety places the mine elsewhere.
	model, _ = a.Update(key(" "))
	a = model.(App)
	// Reveal the remaining safe cells, steering the cursor directly but
	// routing every reveal keystroke through App.Update.
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			cell, _ := a.game.game.CellAt(x, y)
			if cell.Mine || cell.State == engine.Revealed {
				continue
			}
			a.game.cursorX, a.game.cursorY = x, y
			model, _ = a.Update(key(" "))
			a = model.(App)
		}
	}
	if a.game.game.Status() != engine.Won {
		t.Fatalf("status = %v, want Won", a.game.game.Status())
	}
	model, _ = a.Update(key("m"))
	a = model.(App)
	if !strings.Contains(a.View(), "best: 000") {
		t.Errorf("menu should show the new beginner best 000:\n%s", a.View())
	}
}

func TestAppResizeWhileFormOpenRevalidates(t *testing.T) {
	a := testApp(t)
	// Navigate to Custom… and open the form through the App.
	for i := 0; i < 3; i++ {
		model, _ := a.Update(key("j"))
		a = model.(App)
	}
	model, _ := a.Update(key("enter"))
	a = model.(App)
	if !a.menu.showForm {
		t.Fatal("enter on Custom should open the form")
	}
	// Shrink the terminal while the form is open.
	model, _ = a.Update(tea.WindowSizeMsg{Width: 30, Height: 24})
	a = model.(App)
	// 12×8/20 passes the engine rules, but 12 wide needs 37 cols > 30.
	for _, k := range []string{"1", "2", "enter", "8", "enter", "2", "0", "enter"} {
		model, _ = a.Update(key(k))
		a = model.(App)
	}
	if !strings.Contains(a.menu.err, "too large") {
		t.Errorf("err = %q, want containing %q", a.menu.err, "too large")
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
