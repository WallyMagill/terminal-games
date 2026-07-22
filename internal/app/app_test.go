package app

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	g2048ui "github.com/wallymagill/mines/internal/g2048/ui"
	solitaireui "github.com/wallymagill/mines/internal/solitaire/ui"
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
	if !strings.Contains(v, "TERMINAL GAMES") || !strings.Contains(v, "Minesweeper") || !strings.Contains(v, "Solitaire") || !strings.Contains(v, "2048") {
		t.Fatalf("launcher missing expected labels:\n%s", v)
	}
}

func TestLauncherSelects2048(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("enter"))
	a = model.(App)
	if a.view != view2048 {
		t.Fatalf("view = %v, want view2048", a.view)
	}
	v := a.View()
	if !strings.Contains(v, "2048") || !strings.Contains(v, "SCORE") {
		t.Fatalf("2048 view missing pieces:\n%s", v)
	}
}

func Test2048EscReturnsToLauncher(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(launcherSelectMsg{item: launcher2048})
	a = model.(App)
	if a.view != view2048 {
		t.Fatalf("view = %v, want view2048", a.view)
	}
	_, cmd := a.Update(key("esc"))
	if cmd == nil {
		t.Fatal("esc in 2048 should request launcher back")
	}
	if _, ok := cmd().(g2048ui.BackMsg); !ok {
		t.Fatal("esc in 2048 should produce g2048 BackMsg")
	}
	model, _ = a.Update(g2048ui.BackMsg{})
	a = model.(App)
	if a.view != viewLauncher {
		t.Fatalf("view = %v, want viewLauncher", a.view)
	}
	if !strings.Contains(a.View(), "TERMINAL GAMES") {
		t.Fatalf("launcher did not render after 2048 back:\n%s", a.View())
	}
}

func TestLauncherShows2048TooSmall(t *testing.T) {
	a := New(store.New(t.TempDir() + "/scores.json"))
	model, _ := a.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	a = model.(App)
	model, _ = a.Update(launcherSelectMsg{item: launcher2048})
	a = model.(App)
	v := a.View()
	if !strings.Contains(v, "too small") {
		t.Fatalf("2048 should show too-small view at 20x8:\n%s", v)
	}
}

func TestReentered2048StartsFreshGame(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(launcherSelectMsg{item: launcher2048})
	a = model.(App)
	if a.view != view2048 {
		t.Fatal("expected view2048")
	}
	model, _ = a.Update(g2048ui.BackMsg{})
	a = model.(App)
	model, _ = a.Update(launcherSelectMsg{item: launcher2048})
	a = model.(App)
	if a.view != view2048 {
		t.Fatal("expected view2048 on re-entry")
	}
	if !strings.Contains(a.View(), "SCORE 0") {
		t.Fatalf("re-entered 2048 should start at score 0:\n%s", a.View())
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

func TestLauncherSelectsSolitaireModel(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("enter"))
	a = model.(App)
	if a.view != viewSolitaire {
		t.Fatalf("view = %v, want viewSolitaire", a.view)
	}
	view := a.View()
	if !strings.Contains(view, "SOLITAIRE DRAW 3") || !strings.Contains(view, "Stock") {
		t.Fatalf("Solitaire model did not render:\n%s", view)
	}
	if strings.Contains(view, "Draw 3 coming next") {
		t.Fatalf("Solitaire temporary screen still rendered:\n%s", view)
	}
}

func TestLauncherShowsSolitaireTooSmallWhenAppChromeWouldOverflow(t *testing.T) {
	a := New(store.New(t.TempDir() + "/scores.json"))
	model, _ := a.Update(tea.WindowSizeMsg{Width: 48, Height: 16})
	a = model.(App)
	model, _ = a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("enter"))
	a = model.(App)
	v := a.View()
	if !strings.Contains(v, "too small") {
		t.Fatalf("Solitaire should reject terminal height that cannot fit app chrome:\n%s", v)
	}
}

func TestSolitaireTooSmallViewFitsUltraShortAppHeight(t *testing.T) {
	a := New(store.New(t.TempDir() + "/scores.json"))
	model, _ := a.Update(tea.WindowSizeMsg{Width: 48, Height: 4})
	a = model.(App)
	model, _ = a.Update(launcherSelectMsg{item: launcherSolitaire})
	a = model.(App)

	view := a.View()
	if !strings.Contains(view, "press m for menu") {
		t.Fatalf("Solitaire too-small view should keep menu hint visible:\n%s", view)
	}
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	if len(lines) > 4 {
		t.Fatalf("Solitaire too-small view rendered %d rows, want <= 4:\n%s", len(lines), view)
	}
	if lines[0] == "" || lines[len(lines)-1] == "" {
		t.Fatalf("Solitaire too-small view should not be wrapped in blank app chrome:\n%q", view)
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
	_, cmd := a.Update(key("esc"))
	if cmd == nil {
		t.Fatal("esc in Solitaire should request launcher back")
	}
	if _, ok := cmd().(solitaireui.BackMsg); !ok {
		t.Fatal("esc in Solitaire should produce solitaire BackMsg")
	}
	model, _ = a.Update(solitaireui.BackMsg{})
	a = model.(App)
	if a.view != viewLauncher {
		t.Fatalf("view = %v, want viewLauncher", a.view)
	}
}

func TestReenteredSolitaireIgnoresStaleTickFromPreviousSession(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(launcherSelectMsg{item: launcherSolitaire})
	a = model.(App)
	model, oldTickCmd := a.Update(key("d"))
	a = model.(App)
	if oldTickCmd == nil {
		t.Fatal("first Solitaire session draw should start timer")
	}
	staleTick := oldTickCmd()

	model, _ = a.Update(solitaireui.BackMsg{})
	a = model.(App)
	model, _ = a.Update(launcherSelectMsg{item: launcherSolitaire})
	a = model.(App)
	model, newTickCmd := a.Update(key("d"))
	a = model.(App)
	if newTickCmd == nil {
		t.Fatal("second Solitaire session draw should start timer")
	}

	model, cmd := a.Update(staleTick)
	a = model.(App)
	if cmd != nil {
		t.Fatal("stale tick from previous Solitaire session should not schedule another tick")
	}
	if elapsed := solitaireIntField(t, a, "elapsed"); elapsed != 0 {
		t.Fatalf("elapsed = %d, want 0 after stale tick", elapsed)
	}
}

func solitaireIntField(t *testing.T, a App, name string) int {
	t.Helper()
	field := reflect.ValueOf(a.solitaire).FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("Solitaire field %q not found", name)
	}
	return int(field.Int())
}

func TestLauncherQuit(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(key("down"))
	a = model.(App)
	model, _ = a.Update(key("down"))
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
