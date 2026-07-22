package ui

import (
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
