package ui

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

// wonGameModel returns a freshly won 2×2 game (mirrors the seed-3 walk in
// TestViewWonShowsCelebrationAndAutoFlags).
func wonGameModel(t *testing.T) gameModel {
	t.Helper()
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
	if m.game.Status() != engine.Won {
		t.Fatal("seed-3 walk should win the 2×2 game")
	}
	return m
}

// Every status-line variant the game can show on the narrowest accepted
// terminal (chromeCols wide) must actually fit it.
func TestStatusLineCompactVariantsFit(t *testing.T) {
	t.Run("playing", func(t *testing.T) {
		m := testGameModel(t)
		m.termW = chromeCols
		if w := lipgloss.Width(m.statusLine()); w > chromeCols {
			t.Errorf("playing hint is %d cols, max %d", w, chromeCols)
		}
	})
	t.Run("won new best", func(t *testing.T) {
		m := wonGameModel(t)
		m.termW, m.elapsed, m.newBest = chromeCols, 999, true
		s := m.statusLine()
		if w := lipgloss.Width(s); w > chromeCols {
			t.Errorf("won line %q is %d cols, max %d", s, w, chromeCols)
		}
		if !strings.Contains(s, "NEW BEST") {
			t.Errorf("compact won line must keep the new-best callout: %q", s)
		}
	})
	t.Run("won save error", func(t *testing.T) {
		m := wonGameModel(t)
		m.termW, m.elapsed, m.newBest, m.saveErr = chromeCols, 999, false, true
		s := m.statusLine()
		if w := lipgloss.Width(s); w > chromeCols {
			t.Errorf("won line %q is %d cols, max %d", s, w, chromeCols)
		}
		if !strings.Contains(s, "save failed") {
			t.Errorf("compact won line must keep the save warning: %q", s)
		}
	})
	t.Run("lost", func(t *testing.T) {
		m := testGameModel(t)
		m, _ = m.Update(key(" ")) // place mines
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				if c, _ := m.game.CellAt(x, y); c.Mine {
					m.cursorX, m.cursorY = x, y
				}
			}
		}
		m, _ = m.Update(key(" ")) // boom
		if m.game.Status() != engine.Lost {
			t.Fatal("revealing a mine should lose")
		}
		m.termW = chromeCols
		if w := lipgloss.Width(m.statusLine()); w > chromeCols {
			t.Errorf("lost line is %d cols, max %d", w, chromeCols)
		}
	})
}

// The help overlay holds the full key list; it must not clip in width on
// any terminal the guard accepts.
func TestHelpOverlayLinesFitChromeCols(t *testing.T) {
	m := testGameModel(t)
	m.termW = chromeCols
	m, _ = m.Update(key("?"))
	for _, line := range strings.Split(m.View(), "\n") {
		if w := lipgloss.Width(line); w > chromeCols {
			t.Errorf("overlay line %q is %d cols, max %d", line, w, chromeCols)
		}
	}
}

// The HUD must keep a constant width as the mine counter moves through its
// full range, or the face and timer shift mid-game. The displayed counter
// clamps at -99 (classic minesweeper) so the three-character field never
// grows, even though the real count keeps going below.
func TestHUDWidthStableAcrossCounterValues(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	m, err := newGameModel(Preset{Name: "Dense", Width: 40, Height: 40, Mines: 999},
		s, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	// flagTo flags distinct hidden cells until the counter reads target.
	next := 0
	flagTo := func(target int) {
		t.Helper()
		for m.game.MinesRemaining() > target {
			if next >= 40*40 {
				t.Fatalf("ran out of cells flagging down to %d", target)
			}
			m.game.CycleFlag(next%40, next/40)
			next++
		}
	}
	baseW := lipgloss.Width(m.hudView()) // counter 999, the widest positive
	tests := []struct {
		counter int
		want    string
	}{
		{999, "🚩 999"},
		{42, "🚩 042"},
		{7, "🚩 007"},
		{0, "🚩 000"},
		{-5, "🚩 -05"},
		{-99, "🚩 -99"},
		{-100, "🚩 -99"}, // display clamps; MinesRemaining itself is -100
	}
	for _, tt := range tests {
		flagTo(tt.counter)
		hud := m.hudView()
		if w := lipgloss.Width(hud); w != baseW {
			t.Errorf("counter %d: HUD %q is %d cols, want %d (face/timer shift)",
				tt.counter, hud, w, baseW)
		}
		if !strings.Contains(hud, tt.want) {
			t.Errorf("counter %d: HUD %q missing %q", tt.counter, hud, tt.want)
		}
	}
	if got := m.game.MinesRemaining(); got != -100 {
		t.Fatalf("MinesRemaining = %d, want -100 (clamp must be display-only)", got)
	}
}
