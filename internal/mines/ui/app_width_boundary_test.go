package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

// TestAppWidthBoundaryCoversRightEdgeCursor pins the +1 bracket column in
// boardCols with a board wide enough that chromeCols does not mask it: a
// 10-wide board renders 31 columns when the cursor sits on the right edge
// (3 per cell + the trailing "]"), so a 30-column terminal must be
// rejected. The guard sweep derives needW from frameCols itself and never
// moves the cursor, so a guard that forgets the bracket column accepts 30
// and the right-edge row overflows by one column. Kills boardCols = 3*w.
func TestAppWidthBoundaryCoversRightEdgeCursor(t *testing.T) {
	p := Preset{Name: "10x3", Width: 10, Height: 3, Mines: 2}
	a := NewApp(store.New(t.TempDir() + "/scores.json"))
	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 31, Height: 24})
	a, _ = updApp(t, a, startGameMsg{preset: p})
	if strings.Contains(a.View(), "too small") {
		t.Fatal("31 cols fits a 10-wide board's frame exactly; the game must play")
	}
	for i := 0; i < p.Width-1; i++ { // walk the cursor to the right edge
		a, _ = updApp(t, a, key("l"))
	}
	checkLinesFit(t, "right-edge cursor at exact-fit 31 cols", a.View(), 31)

	// One column narrower: the right-edge row would be 31 columns on a
	// 30-column terminal, so the guard must reject.
	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 30, Height: 24})
	v := a.View()
	if !strings.Contains(v, "too small") {
		t.Errorf("30 cols is one short of the 10-wide frame (31); must show the too-small screen:\n%s", v)
	}
	checkLinesFit(t, "frame at one-under-boundary 30 cols", v, 30)
}
