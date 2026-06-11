package ui

import (
	"strings"
	"testing"
)

func TestViewGoldenInitialBoard(t *testing.T) {
	m := testGameModel(t) // 4x4, 2 mines, nothing revealed
	got := m.View()
	want := strings.Join([]string{
		" 🚩 002   🙂   ⏱ 000",
		"",
		"[░░]░░ ░░ ░░",
		" ░░ ░░ ░░ ░░",
		" ░░ ░░ ░░ ░░",
		" ░░ ░░ ░░ ░░",
		" hjkl move · spc reveal · f flag · c chord · n new · ? help · m menu · q quit",
	}, "\n")
	if got != want {
		t.Errorf("View() mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestViewShowsRevealedNumbersAndEmpty(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" ")) // reveal at (0,0)
	v := m.View()
	if strings.Count(v, "░") == 16*2 {
		t.Error("view should show fewer hidden cells after a reveal")
	}
}

func TestViewLostShowsMinesAndStatus(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" ")) // start: places mines
	// Move the cursor onto a mine and reveal it.
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if c, _ := m.game.CellAt(x, y); c.Mine {
				m.cursorX, m.cursorY = x, y
			}
		}
	}
	m, _ = m.Update(key(" "))
	v := m.View()
	if !strings.Contains(v, "✱") {
		t.Error("lost view must expose mines")
	}
	if !strings.Contains(v, "💀") {
		t.Error("lost view must show the dead face")
	}
	if !strings.Contains(v, "n new") {
		t.Error("lost view must show restart hint")
	}
}

func TestViewHelpOverlay(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key("?"))
	v := m.View()
	if !strings.Contains(v, "chord") || !strings.Contains(v, "flag") {
		t.Error("help overlay should list key bindings")
	}
}
