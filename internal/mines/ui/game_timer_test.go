package ui

import (
	"strings"
	"testing"

	"github.com/wallymagill/terminal-games/internal/store"
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
	p := Preset{Name: "Tiny", Key: "beginner", Width: 2, Height: 2, Mines: 1}
	m := winSeed3Board(t, s, p, 7) // pretend 7 seconds passed before the win
	if got := s.Load()["beginner"]; got != 7 {
		t.Errorf("recorded best = %d, want 7", got)
	}
	if !m.newBest {
		t.Error("first win should be flagged as a new best")
	}
}

// An accidental keypress after a new-best win must not re-run the record
// path: a second RecordIfBest call reports isBest=false (best <= secs) and
// would erase the NEW BEST! celebration. recordIfWon's m.recorded guard
// prevents that.
func TestNewBestSurvivesExtraKeypressAfterWin(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	p := Preset{Name: "Tiny", Key: "beginner", Width: 2, Height: 2, Mines: 1}
	m := winSeed3Board(t, s, p, 0)
	if !m.newBest {
		t.Fatal("setup: first win should be a new best")
	}
	m, _ = m.Update(key(" ")) // accidental double-tap after the win
	if !m.newBest || !strings.Contains(m.View(), "NEW BEST!") {
		t.Errorf("extra keypress erased the NEW BEST celebration (newBest = %v)", m.newBest)
	}
}

func TestCustomGamesDoNotRecordTimes(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	winSeed3Board(t, s, Preset{Name: "Custom", Key: "", Width: 2, Height: 2, Mines: 1}, 0)
	if got := s.Load(); len(got) != 0 {
		t.Errorf("custom game recorded a time: %v", got)
	}
}
