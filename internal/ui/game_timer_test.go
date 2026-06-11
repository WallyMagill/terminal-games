package ui

import (
	"math/rand"
	"testing"
	"time"

	"github.com/wallymagill/mines/internal/store"
)

func TestTickIncrementsOnlyWhilePlaying(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(tickMsg(time.Time{})) // before first reveal: no-op
	if m.elapsed != 0 {
		t.Errorf("elapsed before start = %d, want 0", m.elapsed)
	}
	m, _ = m.Update(key(" ")) // start
	m, cmd := m.Update(tickMsg(time.Time{}))
	if m.elapsed != 1 {
		t.Errorf("elapsed after one tick = %d, want 1", m.elapsed)
	}
	if cmd == nil {
		t.Error("tick while playing must schedule the next tick")
	}
}

func TestTimerCapsAt999(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" "))
	m.elapsed = 999
	m, _ = m.Update(tickMsg(time.Time{}))
	if m.elapsed != 999 {
		t.Errorf("elapsed = %d, want capped at 999", m.elapsed)
	}
}

func TestWinRecordsBestTimeForPreset(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	// 2x2 board, 1 mine: revealing the three safe cells wins. The first
	// reveal triggers placement excluding (0,0); later iterations read the
	// real board.
	p := Preset{Name: "Tiny", Key: "beginner", Width: 2, Height: 2, Mines: 1}
	m, err := newGameModel(p, s, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatal(err)
	}
	m.elapsed = 7 // pretend 7 seconds passed
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			m.cursorX, m.cursorY = x, y
			if c, _ := m.game.CellAt(x, y); !c.Mine {
				m, _ = m.Update(key(" "))
			}
		}
	}
	if got := s.Load()["beginner"]; got != 7 {
		t.Errorf("recorded best = %d, want 7", got)
	}
	if !m.newBest {
		t.Error("first win should be flagged as a new best")
	}
}

func TestCustomGamesDoNotRecordTimes(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	p := Preset{Name: "Custom", Key: "", Width: 2, Height: 2, Mines: 1}
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
	if got := s.Load(); len(got) != 0 {
		t.Errorf("custom game recorded a time: %v", got)
	}
}
