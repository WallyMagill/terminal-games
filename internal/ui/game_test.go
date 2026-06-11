package ui

import (
	"math/rand"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/engine"
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

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	if s == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
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
