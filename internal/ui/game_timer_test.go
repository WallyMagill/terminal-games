package ui

import (
	"math/rand"
	"testing"

	"github.com/wallymagill/mines/internal/store"
)

func TestTickIncrementsOnlyWhilePlaying(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(tickMsg{}) // before first reveal: no-op
	if m.elapsed != 0 {
		t.Errorf("elapsed before start = %d, want 0", m.elapsed)
	}
	m, _ = m.Update(key(" ")) // start
	m, cmd := m.Update(tickMsg{})
	if m.elapsed != 1 {
		t.Errorf("elapsed after one tick = %d, want 1", m.elapsed)
	}
	if cmd == nil {
		t.Error("tick while playing must schedule the next tick")
	}
}

// The rescheduled tick must carry the model's current generation: a mismatch
// would be dropped as stale on the next Update and the timer would freeze at
// one second forever. Pin it by executing the rescheduled cmd for real and
// feeding the message it produces back through Update.
func TestTickRescheduleCarriesCurrentGeneration(t *testing.T) {
	m := testGameModel(t)
	m, cmd := m.Update(key(" ")) // first reveal starts the timer loop
	if cmd == nil {
		t.Fatal("first reveal must schedule a tick")
	}
	m, cmd = m.Update(tickMsg{gen: m.tickGen}) // tick 1 lands
	if m.elapsed != 1 || cmd == nil {
		t.Fatalf("after tick 1: elapsed = %d (want 1), cmd nil = %v", m.elapsed, cmd == nil)
	}
	m, cmd = m.Update(cmd()) // run the reschedule (~1s) and land tick 2
	if m.elapsed != 2 || cmd == nil {
		t.Fatalf("after tick 2: elapsed = %d, want 2 (rescheduled tick was dropped as stale)", m.elapsed)
	}
}

func TestTimerCapsAt999(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" "))
	m.elapsed = 999
	m, _ = m.Update(tickMsg{})
	if m.elapsed != 999 {
		t.Errorf("elapsed = %d, want capped at 999", m.elapsed)
	}
}

func TestStaleTickFromPreviousRoundIsIgnored(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(key(" ")) // start round 1; tick T1 in flight
	staleGen := m.tickGen
	m, _ = m.Update(key("n"))                  // restart
	m, _ = m.Update(key(" "))                  // start round 2; tick T2 in flight
	m, cmd := m.Update(tickMsg{gen: staleGen}) // T1 lands late
	if m.elapsed != 0 {
		t.Errorf("stale tick incremented elapsed to %d, want 0", m.elapsed)
	}
	if cmd != nil {
		t.Error("stale tick must not reschedule (would create a second tick loop)")
	}
	m, cmd = m.Update(tickMsg{gen: m.tickGen}) // T2 lands
	if m.elapsed != 1 || cmd == nil {
		t.Errorf("current-generation tick: elapsed = %d (want 1), cmd nil = %v (want false)", m.elapsed, cmd == nil)
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
