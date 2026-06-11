package engine

import (
	"errors"
	"math/rand"
	"testing"
)

func testRNG() *rand.Rand { return rand.New(rand.NewSource(1)) }

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name    string
		w, h    int
		mines   int
		wantErr error
	}{
		{"valid beginner", 9, 9, 10, nil},
		{"valid minimum board", 2, 2, 1, nil},
		{"valid max mines", 4, 4, 14, nil},
		{"width too small", 1, 5, 1, ErrBoardTooSmall},
		{"height too small", 5, 1, 1, ErrBoardTooSmall},
		{"zero mines", 4, 4, 0, ErrMineCount},
		{"negative mines", 4, 4, -1, ErrMineCount},
		{"too many mines", 4, 4, 15, ErrMineCount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := New(tt.w, tt.h, tt.mines, testRNG())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("New(%d,%d,%d) error = %v, want %v", tt.w, tt.h, tt.mines, err, tt.wantErr)
			}
			if tt.wantErr == nil && g == nil {
				t.Fatal("expected non-nil game")
			}
		})
	}
}

func TestNewInitialState(t *testing.T) {
	g, err := New(9, 9, 10, testRNG())
	if err != nil {
		t.Fatal(err)
	}
	if g.Status() != NotStarted {
		t.Errorf("status = %v, want NotStarted", g.Status())
	}
	if g.Width() != 9 || g.Height() != 9 || g.Mines() != 10 {
		t.Errorf("dims = %dx%d/%d, want 9x9/10", g.Width(), g.Height(), g.Mines())
	}
	if g.MinesRemaining() != 10 {
		t.Errorf("MinesRemaining = %d, want 10", g.MinesRemaining())
	}
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			c, ok := g.CellAt(x, y)
			if !ok {
				t.Fatalf("CellAt(%d,%d) out of bounds", x, y)
			}
			if c.State != Hidden || c.Mine {
				t.Fatalf("cell (%d,%d) = %+v, want hidden non-mine", x, y, c)
			}
		}
	}
}

func TestCellAtOutOfBounds(t *testing.T) {
	g, _ := New(4, 4, 2, testRNG())
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {4, 0}, {0, 4}} {
		if _, ok := g.CellAt(p[0], p[1]); ok {
			t.Errorf("CellAt(%d,%d) ok = true, want false", p[0], p[1])
		}
	}
}
