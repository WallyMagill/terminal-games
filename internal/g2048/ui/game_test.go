package ui

import (
	"math/rand"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/terminal-games/internal/store"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestEscRequestsBack(t *testing.T) {
	m := testModel(t)
	_, cmd := m.Update(key("esc"))
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("esc should produce BackMsg")
	}
}

func TestHelpDismissesOnAnyKey(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("?"))
	if !m.showHelp {
		t.Fatal("help should open")
	}
	m, _ = m.Update(key("x"))
	if m.showHelp {
		t.Fatal("help should close")
	}
}

func TestMoveKeysChangeScoreOrBoard(t *testing.T) {
	m := testModel(t)
	before := m.game.Board()
	changed := false
	for _, k := range []string{"left", "right", "up", "down", "h", "j", "k", "l"} {
		m, _ = m.Update(key(k))
		if m.game.Board() != before || m.game.Score() > 0 {
			changed = true
			break
		}
	}
	if !changed {
		for i := 0; i < 20; i++ {
			m, _ = m.Update(key("left"))
			m, _ = m.Update(key("up"))
			if m.game.Score() > 0 || m.game.Board() != before {
				changed = true
				break
			}
		}
	}
	if !changed && !m.game.Over() {
		t.Fatalf("expected some move to affect game; board=%v", m.game.Board())
	}
}

func TestNewGameResets(t *testing.T) {
	m := testModel(t)
	for i := 0; i < 30; i++ {
		m, _ = m.Update(key("left"))
		m, _ = m.Update(key("up"))
	}
	m, _ = m.Update(key("n"))
	if m.game.Score() != 0 {
		t.Fatalf("after n score = %d, want 0", m.game.Score())
	}
	if m.showWin {
		t.Fatal("showWin should clear on new game")
	}
}

func TestMoveRecordsBestScore(t *testing.T) {
	path := t.TempDir() + "/scores.json"
	s := store.New(path)
	var m Model
	var err error
	// Search seeds until a merge occurs quickly.
	for seed := int64(1); seed < 200; seed++ {
		m, err = NewModel(s, rand.New(rand.NewSource(seed)))
		if err != nil {
			t.Fatal(err)
		}
		m.termW, m.termH = 80, 24
		for i := 0; i < 40; i++ {
			m, _ = m.Update(key("left"))
			m, _ = m.Update(key("down"))
			m, _ = m.Update(key("right"))
			m, _ = m.Update(key("up"))
			if m.game.Score() > 0 {
				break
			}
		}
		if m.game.Score() > 0 {
			break
		}
	}
	if m.game.Score() == 0 {
		t.Fatal("could not produce a scoring run for record test")
	}
	rec := s.LoadG2048Classic()
	if rec.BestScore == 0 && !m.saveErr {
		t.Fatalf("score %d not recorded; rec=%#v saveErr=%v", m.game.Score(), rec, m.saveErr)
	}
	if rec.BestScore > 0 && m.bestScore != rec.BestScore {
		t.Fatalf("model bestScore = %d, store = %d", m.bestScore, rec.BestScore)
	}
}

func TestTooSmallIgnoresMoveKeys(t *testing.T) {
	m := testModel(t)
	m.termW, m.termH = 10, 5
	before := m.game.Board()
	m, _ = m.Update(key("left"))
	if m.game.Board() != before {
		t.Fatal("moves ignored when too small")
	}
}
