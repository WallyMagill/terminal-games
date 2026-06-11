package ui

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/wallymagill/mines/internal/engine"
	"github.com/wallymagill/mines/internal/store"
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
	// Wrong-flag one safe hidden cell, question-mark another, then find a mine.
	var mineX, mineY int
	marked := 0
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			c, _ := m.game.CellAt(x, y)
			if c.Mine {
				mineX, mineY = x, y
				continue
			}
			if c.State == engine.Hidden && marked < 2 {
				m.cursorX, m.cursorY = x, y
				m, _ = m.Update(key("f")) // → Flagged
				if marked == 1 {
					m, _ = m.Update(key("f")) // second cell → Questioned
				}
				marked++
			}
		}
	}
	m.cursorX, m.cursorY = mineX, mineY
	m, _ = m.Update(key(" "))
	v := m.View()
	if !strings.Contains(v, "✱") {
		t.Error("lost view must expose mines")
	}
	if !strings.Contains(v, "✕") {
		t.Error("lost view must mark the wrong flag with ✕")
	}
	if !strings.Contains(v, "?") {
		t.Error("lost view must keep the question mark on safe cells")
	}
	if !strings.Contains(v, "💀") {
		t.Error("lost view must show the dead face")
	}
	if !strings.Contains(v, "n new") {
		t.Error("lost view must show restart hint")
	}
}

func TestViewCursorAtRightEdgeClosesBracket(t *testing.T) {
	m := testGameModel(t)
	m.cursorX, m.cursorY = m.game.Width()-1, 0
	rows := strings.Split(m.View(), "\n")
	if got := rows[2]; !strings.HasSuffix(got, "]") {
		t.Errorf("cursor on last column must close its bracket, got %q", got)
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

func TestViewWonShowsCelebrationAndAutoFlags(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	p := Preset{Name: "Tiny", Key: "beginner", Width: 2, Height: 2, Mines: 1}
	m, err := newGameModel(p, s, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			m.cursorX, m.cursorY = x, y
			if c, _ := m.game.CellAt(x, y); !c.Mine {
				m, _ = m.Update(key(" "))
			}
		}
	}
	v := m.View()
	for _, want := range []string{"😎", "Cleared in", "NEW BEST!", "⚑"} {
		if !strings.Contains(v, want) {
			t.Errorf("won view missing %q:\n%s", want, v)
		}
	}
	m.saveErr = true
	if !strings.Contains(m.View(), "could not save score") {
		t.Error("won view with saveErr must show the save warning")
	}
}
