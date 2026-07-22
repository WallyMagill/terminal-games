package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/wallymagill/mines/internal/store"
)

func testModel(t *testing.T) Model {
	t.Helper()
	m, err := NewModel(store.New(t.TempDir()+"/scores.json"), rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	m.termW, m.termH = 80, 24
	return m
}

func TestViewRendersTitleScoreAndBoard(t *testing.T) {
	m := testModel(t)
	v := m.View()
	if !strings.Contains(v, "2048") {
		t.Fatalf("missing title:\n%s", v)
	}
	if !strings.Contains(v, "SCORE") || !strings.Contains(v, "BEST") {
		t.Fatalf("missing HUD:\n%s", v)
	}
	if !strings.Contains(v, "2") && !strings.Contains(v, "4") {
		t.Fatalf("board tiles missing:\n%s", v)
	}
}

func TestViewMarksEmptyCells(t *testing.T) {
	m := testModel(t)
	v := m.View()
	// Fresh board has 14 empty cells; each shows a dim placeholder dot
	// so the 4x4 grid extent is visible without box-drawing borders.
	if got := strings.Count(v, "·"); got < 14 {
		t.Fatalf("empty-cell placeholders = %d, want >= 14:\n%s", got, v)
	}
}

func TestViewShowsSavedBest(t *testing.T) {
	path := t.TempDir() + "/scores.json"
	s := store.New(path)
	if _, _, err := s.RecordG2048Classic(9999, 2048); err != nil {
		t.Fatal(err)
	}
	m, err := NewModel(s, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	m.termW, m.termH = 80, 24
	v := m.View()
	if !strings.Contains(v, "9999") {
		t.Fatalf("best score not shown:\n%s", v)
	}
}

func TestTooSmallGuard(t *testing.T) {
	m := testModel(t)
	m.termW, m.termH = 10, 5
	if !m.TooSmall() {
		t.Fatal("expected too small")
	}
	v := m.View()
	if !strings.Contains(strings.ToLower(v), "too small") {
		t.Fatalf("want too small message:\n%s", v)
	}
}

// playUntilOver drives the seeded game with cycling moves until it dies.
// Deterministic for a fixed model seed.
func playUntilOver(t *testing.T, m Model) Model {
	t.Helper()
	keys := []string{"left", "down", "right", "up"}
	for i := 0; i < 20000 && !m.game.Over(); i++ {
		m, _ = m.Update(key(keys[i%4]))
	}
	if !m.game.Over() {
		t.Fatal("game did not reach game over within 20000 moves")
	}
	return m
}

func TestGameOverScreenReplacesBoard(t *testing.T) {
	m := playUntilOver(t, testModel(t))
	v := m.View()
	if !strings.Contains(v, "G A M E   O V E R") {
		t.Fatalf("missing game over heading:\n%s", v)
	}
	if !strings.Contains(v, fmt.Sprintf("score  %6d", m.game.Score())) {
		t.Fatalf("missing final score %d:\n%s", m.game.Score(), v)
	}
	if !strings.Contains(v, fmt.Sprintf("tile   %6d", m.game.HighestTile())) {
		t.Fatalf("missing highest tile %d:\n%s", m.game.HighestTile(), v)
	}
	// Board must be hidden: the hint line keeps a few "·" separators,
	// but the 14+ cell placeholders must be gone.
	if n := strings.Count(v, "·"); n >= 4 {
		t.Fatalf("board placeholders still visible (%d dots):\n%s", n, v)
	}
	// Fresh store: this run is the record, so both stars show.
	if !strings.Contains(v, "new best") {
		t.Fatalf("expected new best markers on fresh store:\n%s", v)
	}
	if !strings.Contains(v, "n new game") {
		t.Fatalf("missing key hints:\n%s", v)
	}
}

func TestGameOverScreenNoStarsWhenNotBest(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	if _, _, err := s.RecordG2048Classic(999999, 65536); err != nil {
		t.Fatal(err)
	}
	m, err := NewModel(s, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	m.termW, m.termH = 80, 24
	m = playUntilOver(t, m)
	v := m.View()
	if !strings.Contains(v, "G A M E   O V E R") {
		t.Fatalf("missing game over heading:\n%s", v)
	}
	if strings.Contains(v, "new best") {
		t.Fatalf("no stars expected when records were not beaten:\n%s", v)
	}
	if !strings.Contains(v, "best   999999") {
		t.Fatalf("persisted best missing:\n%s", v)
	}
}
