package ui

import "testing"

func TestTileRampBoundaries(t *testing.T) {
	// Each adjacent pair crosses a ramp boundary and must render distinctly.
	pairs := [][2]int{{4, 8}, {16, 32}, {64, 128}, {256, 512}, {1024, 2048}}
	for _, p := range pairs {
		a, b := tileStyle(p[0]), tileStyle(p[1])
		if a.GetForeground() == b.GetForeground() {
			t.Fatalf("values %d and %d share foreground %v", p[0], p[1], a.GetForeground())
		}
	}
}

func TestTileRampBold(t *testing.T) {
	for _, v := range []int{2, 4, 8, 16, 32, 64, 128, 256} {
		if tileStyle(v).GetBold() {
			t.Fatalf("%d must not be bold", v)
		}
	}
	for _, v := range []int{512, 1024, 2048, 4096, 65536} {
		if !tileStyle(v).GetBold() {
			t.Fatalf("%d must be bold", v)
		}
	}
	if tileStyle(1024).GetForeground() == tileStyle(2048).GetForeground() {
		t.Fatal("1024 and 2048 must differ in color")
	}
}
