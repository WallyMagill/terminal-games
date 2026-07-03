package ui

import (
	"math/rand"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/mines/engine"
	"github.com/wallymagill/mines/internal/store"
)

var testPreset = Preset{Name: "Test", Key: "", Width: 4, Height: 4, Mines: 2}

func testGameModel(t *testing.T) gameModel {
	t.Helper()
	s := store.New(t.TempDir() + "/scores.json")
	m, err := newGameModel(testPreset, s, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// winSeed3Board builds a 2×2/1-mine game for preset p over store s and wins
// it via the deterministic seed-3 walk: the first reveal triggers placement
// excluding (0,0), later iterations read the real board, and revealing the
// three safe cells wins. elapsed is set before the walk so the winning
// reveal records that time.
func winSeed3Board(t *testing.T, s *store.Store, p Preset, elapsed int) gameModel {
	t.Helper()
	m, err := newGameModel(p, s, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	m.elapsed = elapsed
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			m.cursorX, m.cursorY = x, y
			if c, _ := m.game.CellAt(x, y); !c.Mine {
				m, _ = m.Update(key(" "))
			}
		}
	}
	if m.game.Status() != engine.Won {
		t.Fatal("seed-3 walk should win the 2×2 game")
	}
	return m
}

func key(s string) tea.KeyMsg {
	switch s {
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestCursorMovementClamps(t *testing.T) {
	tests := []struct {
		name  string
		keys  []string
		wantX int
		wantY int
	}{
		{"starts at origin, h/k clamp", []string{"h", "k"}, 0, 0},
		{"l and j move", []string{"l", "l", "j"}, 2, 1},
		{"clamps at right/bottom edge", []string{"l", "l", "l", "l", "l", "j", "j", "j", "j"}, 3, 3},
		{"moves back up and left", []string{"l", "l", "j", "j", "k", "h"}, 1, 1},
		{"arrow keys move too", []string{"right", "down"}, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testGameModel(t)
			for _, k := range tt.keys {
				m, _ = m.Update(key(k))
			}
			if m.cursorX != tt.wantX || m.cursorY != tt.wantY {
				t.Errorf("cursor = (%d,%d), want (%d,%d)", m.cursorX, m.cursorY, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestSpaceRevealsAndStartsGame(t *testing.T) {
	m := testGameModel(t)
	m, cmd := m.Update(key(" "))
	c, _ := m.game.CellAt(0, 0)
	if c.State != engine.Revealed {
		t.Error("space should reveal the cursor cell")
	}
	if m.game.Status() == engine.NotStarted {
		t.Error("game should have started")
	}
	if !m.running || cmd == nil {
		t.Error("first reveal must start the timer (running=true, tick cmd returned)")
	}
}

func TestSpaceOnRevealedNumberChords(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" ")) // first reveal places mines and opens (0,0) area
	// Find a revealed numbered cell with at least one hidden neighbor, flag
	// its mines, then chord it via space and verify neighbors opened.
	g := m.game
	for y := 0; y < g.Height(); y++ {
		for x := 0; x < g.Width(); x++ {
			c, _ := g.CellAt(x, y)
			if c.State != engine.Revealed || c.Adjacent == 0 {
				continue
			}
			// flag all mine neighbors
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if n, ok := g.CellAt(nx, ny); ok && n.Mine && n.State == engine.Hidden {
						m.cursorX, m.cursorY = nx, ny
						m, _ = m.Update(key("f"))
					}
				}
			}
			// count hidden non-flagged neighbors before chording
			hiddenBefore := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if n, ok := g.CellAt(x+dx, y+dy); ok && n.State == engine.Hidden {
						hiddenBefore++
					}
				}
			}
			if hiddenBefore == 0 {
				continue
			}
			m.cursorX, m.cursorY = x, y
			m, _ = m.Update(key(" ")) // space on revealed number must chord
			hiddenAfter := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if n, ok := g.CellAt(x+dx, y+dy); ok && n.State == engine.Hidden {
						hiddenAfter++
					}
				}
			}
			if hiddenAfter >= hiddenBefore {
				t.Errorf("space on revealed (%d,%d) did not chord: hidden %d -> %d", x, y, hiddenBefore, hiddenAfter)
			}
			return
		}
	}
	t.Skip("seed produced no chordable cell; adjust seed if this ever happens")
}

func TestCKeyOnHiddenCellDoesNotReveal(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key("c"))
	c, _ := m.game.CellAt(0, 0)
	if c.State != engine.Hidden {
		t.Errorf("c on hidden cell changed state to %v, want Hidden", c.State)
	}
}

func TestFKeyCyclesFlag(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key("f"))
	c, _ := m.game.CellAt(0, 0)
	if c.State != engine.Flagged {
		t.Errorf("state = %v, want Flagged", c.State)
	}
	if m.game.MinesRemaining() != 1 {
		t.Errorf("MinesRemaining = %d, want 1", m.game.MinesRemaining())
	}
}

func TestNRestartsGame(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" "))
	m, _ = m.Update(key("n"))
	if m.game.Status() != engine.NotStarted {
		t.Error("n should produce a fresh, unstarted game")
	}
	if m.elapsed != 0 || m.running {
		t.Error("n should reset the timer")
	}
}

func TestNRestartPreservesTermW(t *testing.T) {
	m := testGameModel(t)
	m.termW = chromeCols
	m, _ = m.Update(key(" "))
	m, _ = m.Update(key("n"))
	if m.termW != chromeCols {
		t.Errorf("termW after restart = %d, want %d (chrome must keep adapting)", m.termW, chromeCols)
	}
}

func TestHelpOverlayToggles(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key("?"))
	if !m.showHelp {
		t.Error("? should open help")
	}
	m, _ = m.Update(key("j"))
	if m.showHelp {
		t.Error("any key should close help")
	}
	if m.cursorY != 0 {
		t.Error("the help-dismissing key must not also act on the board")
	}
}
