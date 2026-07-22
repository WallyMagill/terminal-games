package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/terminal-games/internal/mines/engine"
	"github.com/wallymagill/terminal-games/internal/store"
)

// winTiny drives a 2×2/1 game under presetKey to a win, routing every
// keystroke through App.Update so the real win path (timer, record, HUD
// state) is exercised end to end.
func winTiny(t *testing.T, a App, presetKey string) App {
	t.Helper()
	tiny := Preset{Name: "Tiny", Key: presetKey, Width: 2, Height: 2, Mines: 1}
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
	return a
}

func testApp(t *testing.T) App {
	t.Helper()
	a := NewApp(store.New(t.TempDir() + "/scores.json"))
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return model.(App)
}

// updApp routes msg through App.Update and re-asserts the concrete type.
func updApp(t *testing.T, a App, msg tea.Msg) (App, tea.Cmd) {
	t.Helper()
	model, cmd := a.Update(msg)
	return model.(App), cmd
}

// boardLines extracts the rendered board rows from an App game view: the
// leading margin (1), HUD (1) and blank line (1) precede the board.
func boardLines(t *testing.T, v string, boardH int) []string {
	t.Helper()
	lines := strings.Split(v, "\n")
	if len(lines) < 3+boardH {
		t.Fatalf("view has %d lines, need at least %d:\n%s", len(lines), 3+boardH, v)
	}
	return lines[3 : 3+boardH]
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

// Spec key map binds both m and esc to "menu" from the game view; the
// help-overlay test only covers esc while help is OPEN, so pin the
// help-closed path separately.
func TestAppEscReturnsToMenuFromGame(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = model.(App)
	if a.view != viewMenu {
		t.Error("esc in the game view (help closed) must return to the menu")
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

func TestMenuEscRequestsLauncherBack(t *testing.T) {
	a := testApp(t)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc on Mines menu should request launcher back")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("esc on Mines menu should produce BackMsg")
	}
}

func TestAppTooSmallTerminal(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[2]}) // Expert: 30 wide → needs 91 cols
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

// A terminal exactly as tall as the rendered view must be accepted: the
// guard's chrome-rows count has to match the real layout.
func TestAppExactFitHeightPlays(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: Preset{Name: "T", Width: 4, Height: 4, Mines: 2}})
	a = model.(App)
	full := len(strings.Split(a.View(), "\n"))
	model, _ = a.Update(tea.WindowSizeMsg{Width: 80, Height: full})
	a = model.(App)
	if strings.Contains(a.View(), "too small") {
		t.Errorf("view occupies %d lines; a %d-row terminal must be accepted:\n%s",
			full, full, a.View())
	}
}

// When the guard accepts a terminal, no rendered line may be wider than it:
// bubbletea hard-truncates over-wide lines, silently cutting off key hints.
func TestAppNarrowAcceptedTerminalChromeFits(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(tea.WindowSizeMsg{Width: 50, Height: 20})
	a = model.(App)
	model, _ = a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	v := a.View()
	if strings.Contains(v, "too small") {
		t.Fatal("Beginner must be accepted on a 50×20 terminal")
	}
	checkLinesFit(t, "accepted 50-col terminal", v, 50)
}

// The too-small screen must itself fit the terminal that triggered it —
// otherwise the "need W×H" payload it exists to convey is truncated away.
func TestAppTooSmallMessageFitsTrigger(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(tea.WindowSizeMsg{Width: 27, Height: 15})
	a = model.(App)
	model, _ = a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	v := a.View()
	if !strings.Contains(v, "too small") {
		t.Fatal("27×15 must trigger the too-small screen for Beginner")
	}
	lines := strings.Split(v, "\n")
	if len(lines) > 15 {
		t.Errorf("too-small screen has %d lines on a 15-row terminal", len(lines))
	}
	checkLinesFit(t, "too-small screen on its 27-col trigger", v, 27)
}

// Before the first WindowSizeMsg the terminal size is unknown (0×0); the
// game must render optimistically rather than claim "too small … have 0×0".
func TestAppZeroSizeRendersGameOptimistically(t *testing.T) {
	a := NewApp(store.New(t.TempDir() + "/scores.json"))
	a, _ = updApp(t, a, startGameMsg{preset: presets[2]}) // Expert, before any size is known
	v := a.View()
	if strings.Contains(v, "too small") {
		t.Error("unknown terminal size must render the game, not the too-small screen")
	}
	if !strings.Contains(v, "🚩") {
		t.Error("zero-size render must still show the game HUD")
	}
}

// The need-dims line is the too-small screen's sole actionable payload; it
// must report the frame as width×height, matching the adjacent "have" line —
// swapped numbers would send the user resizing to the wrong shape.
func TestAppTooSmallNeedDimsAccurate(t *testing.T) {
	a := testApp(t)
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]}) // Beginner: frame 30×14
	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 10, Height: 5})
	v := a.View()
	if !strings.Contains(v, "need 30×14") {
		t.Errorf("too-small screen must state the needed frame 30×14:\n%s", v)
	}
	if !strings.Contains(v, "have 10×5") {
		t.Errorf("too-small screen must state the current size 10×5:\n%s", v)
	}
}

// The too-small screen advertises the m key; the first m must obey even
// if the help overlay was open (and hidden) when the terminal shrank. On a
// 10-column terminal the full "press m for menu." line cannot fit, so the
// screen degrades it to the compact "m: menu" hint (audit round 2: the
// fallback must itself fit any terminal that triggers it).
func TestAppTooSmallFirstMOpensMenuDespiteHelp(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	model, _ = a.Update(key("?"))
	a = model.(App)
	model, _ = a.Update(tea.WindowSizeMsg{Width: 10, Height: 5})
	a = model.(App)
	if !strings.Contains(a.View(), "m: menu") {
		t.Fatal("expected the too-small m hint")
	}
	model, _ = a.Update(key("m"))
	a = model.(App)
	if a.view != viewMenu {
		t.Errorf("first m on the too-small screen must open the menu, view = %v", a.view)
	}
}

// While the too-small screen is up, game keys must not mutate the invisible
// game — in particular ? must not re-open the overlay and swallow the next m.
func TestAppTooSmallIgnoresGameKeys(t *testing.T) {
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	model, _ = a.Update(tea.WindowSizeMsg{Width: 10, Height: 5})
	a = model.(App)
	model, _ = a.Update(key("?"))
	a = model.(App)
	if a.game.showHelp {
		t.Error("? while too small must not open the help overlay")
	}
	model, _ = a.Update(key("m"))
	a = model.(App)
	if a.view != viewMenu {
		t.Error("m while too small must return to the menu")
	}
}

// A mid-game WindowSizeMsg must thread the new width into the game model:
// compact-chrome switching depends on it, and a stale width keeps rendering
// the 77-col full key hint, which bubbletea hard-truncates on a narrow
// terminal. The startGameMsg path threads termW separately and is covered
// above; this pins the mid-game resize path.
func TestAppResizeMidGameAdaptsChrome(t *testing.T) {
	a := testApp(t) // 120×40
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]})
	if !strings.Contains(a.View(), "q quit") {
		t.Fatal("wide terminal must show the full key hint")
	}
	wideBoard := boardLines(t, a.View(), presets[0].Height)

	// Shrink mid-game to the narrowest accepted width (Beginner needs 30×14).
	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 30, Height: 20})
	if a.game.termW != 30 {
		t.Errorf("mid-game resize: game.termW = %d, want 30 (stale width keeps full chrome)", a.game.termW)
	}
	v := a.View()
	if strings.Contains(v, "too small") {
		t.Fatal("30×20 must be accepted for Beginner")
	}
	if strings.Contains(v, "q quit") {
		t.Error("30-col terminal must swap to the compact hint")
	}
	checkLinesFit(t, "accepted 30-col terminal after mid-game shrink", v, 30)
	if got := boardLines(t, v, presets[0].Height); strings.Join(got, "\n") != strings.Join(wideBoard, "\n") {
		t.Error("board content changed across a resize; model state must be size-independent")
	}

	// Grow back: full chrome returns, board still identical.
	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 120, Height: 40})
	if !strings.Contains(a.View(), "q quit") {
		t.Error("growing back must restore the full key hint")
	}
	if got := boardLines(t, a.View(), presets[0].Height); strings.Join(got, "\n") != strings.Join(wideBoard, "\n") {
		t.Error("board content changed across a resize round trip")
	}
}

// The help-clear on resize is scoped to entering the too-small state; an
// ordinary resize that still fits the game must leave an open overlay alone
// (the overlay's contract is "any key to close help", and a WindowSizeMsg is
// not a key).
func TestAppResizeWhileHelpOpenKeepsOverlay(t *testing.T) {
	a := testApp(t)
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]})
	a, _ = updApp(t, a, key("?"))
	if !a.game.showHelp {
		t.Fatal("? must open help")
	}
	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 100, Height: 35})
	if !a.game.showHelp {
		t.Error("a resize that still fits the game must not close the help overlay")
	}
	if !strings.Contains(a.View(), "any key to close help") {
		t.Error("help overlay must still be rendered after a still-fitting resize")
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

// Pins the view-scoping of the App's m/esc/q interception (a.view ==
// viewGame): q in the custom form is a non-digit no-op (not quit), and
// esc in the form closes it without losing the typed input.
func TestAppMenuKeysNotInterceptedByGameSwitch(t *testing.T) {
	a := testApp(t)
	for i := 0; i < 3; i++ {
		a, _ = updApp(t, a, key("j"))
	}
	a, _ = updApp(t, a, key("enter"))
	if !a.menu.showForm {
		t.Fatal("enter on Custom should open the form")
	}
	a, _ = updApp(t, a, key("9"))
	_, cmd := a.Update(key("q"))
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("q while the custom form is open must be ignored as a non-digit, not quit")
		}
	}
	a, _ = updApp(t, a, tea.KeyMsg{Type: tea.KeyEsc})
	if a.menu.showForm {
		t.Error("esc through the App must close the form")
	}
	if a.view != viewMenu {
		t.Errorf("view = %v, want viewMenu", a.view)
	}
	if got := a.menu.form.fields[0]; got != "9" {
		t.Errorf("form width field = %q after esc, want the typed %q retained", got, "9")
	}
}

func TestAppWinShowsBestTimeOnMenu(t *testing.T) {
	a := winTiny(t, testApp(t), "beginner")
	model, _ := a.Update(key("m"))
	a = model.(App)
	if !strings.Contains(a.View(), "best: 000") {
		t.Errorf("menu should show the new beginner best 000:\n%s", a.View())
	}
}

// A real persistence failure on the win path must set the HUD warning and
// must not celebrate NEW BEST: the spec requires "save failure shows a
// one-line HUD warning", and an unsaved record is not a record.
func TestSaveFailureShowsHUDWarning(t *testing.T) {
	// Block the store path with a regular file where a directory is needed
	// so RecordIfBest fails for real — no injection, just the filesystem.
	blocker := filepath.Join(t.TempDir(), "mines")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp(store.New(filepath.Join(blocker, "scores.json")))
	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = winTiny(t, model.(App), "beginner")

	if !a.game.saveErr {
		t.Error("saveErr not set after a real save failure")
	}
	if a.game.newBest {
		t.Error("an unsaved record must not be celebrated as NEW BEST")
	}
	if v := a.View(); !strings.Contains(v, "could not save score") {
		t.Errorf("win view missing save warning:\n%s", v)
	}
}

// A tampered scores file with a negative best must not leak into the menu
// as "best: -03" — out-of-range values are dropped on load.
func TestMenuDropsTamperedNegativeBest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scores.json")
	if err := os.WriteFile(path, []byte(`{"beginner": -3}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp(store.New(path))
	if v := a.View(); strings.Contains(v, "-03") {
		t.Errorf("menu renders tampered negative best verbatim:\n%s", v)
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
