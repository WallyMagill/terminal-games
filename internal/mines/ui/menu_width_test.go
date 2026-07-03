package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/wallymagill/mines/internal/store"
)

func checkLinesFit(t *testing.T, ctx, view string, termW int) {
	t.Helper()
	for i, line := range strings.Split(view, "\n") {
		if w := lipgloss.Width(line); w > termW {
			t.Errorf("%s: line %d is %d cols (limit %d): %q", ctx, i, w, termW, line)
		}
	}
}

// Audit round 2 (finding 8): the menu and custom form emitted fixed
// 42-51-column lines regardless of terminal width, wrapping/garbling on
// the 30-43-column terminals where the game itself (chromeCols = 30)
// plays fine. Menu and form lines must fit any terminal at or above the
// game's 30-column floor.
func TestMenuAndFormLinesFitTerminalWidth(t *testing.T) {
	scores := store.Scores{"beginner": 3, "intermediate": 123, "expert": 999}
	for _, termW := range []int{30, 35, 43, 50, 80} {
		for cursor := 0; cursor <= itemQuit; cursor++ {
			m := newMenuModel(scores, termW, 24)
			m.cursor = cursor
			checkLinesFit(t, fmt.Sprintf("menu w=%d cursor=%d", termW, cursor), m.View(), termW)
		}
		f := newMenuModel(scores, termW, 24)
		f.showForm = true
		f.form.fields = [3]string{"99", "99", "10"}
		if _, err := f.form.preset(termW, 24); err != nil {
			f.err = err.Error()
		} else {
			t.Fatalf("expected a validation error for 99×99 at width %d", termW)
		}
		checkLinesFit(t, fmt.Sprintf("form w=%d", termW), f.View(), termW)
	}
}

// Audit round 3 (finding 5): a row's full width depends on its best time
// ("best: 003" is two columns wider than "best: —"), and menuRow used to
// pick full vs compact per row — so at widths 42-43 with a partial score
// set, recorded rows rendered compact directly above unrecorded full ones,
// misaligning the best value by 16 columns. The variant must be decided
// once for the whole preset block: if any full row overflows the budget,
// every row renders compact, and the best value column stays aligned.
func TestMenuPresetRowsShareOneVariant(t *testing.T) {
	for termW, wantFull := range map[int]bool{42: false, 43: false, 44: true} {
		m := newMenuModel(store.Scores{"beginner": 3}, termW, 24)
		m.cursor = itemQuit // keep preset rows unstyled
		v := m.View()
		checkLinesFit(t, fmt.Sprintf("menu w=%d", termW), v, termW)
		rows := strings.Split(v, "\n")[2 : 2+len(presets)]
		var bestCols []int
		for i, row := range rows {
			if got := strings.Contains(row, "best:"); got != wantFull {
				t.Errorf("w=%d: preset row %d full=%v, want full=%v: %q",
					termW, i, got, wantFull, row)
			}
			bestCols = append(bestCols, lipgloss.Width(row[:strings.LastIndexByte(row, ' ')+1]))
		}
		for i, c := range bestCols[1:] {
			if c != bestCols[0] {
				t.Errorf("w=%d: best value column ragged: row 0 starts it at col %d, row %d at col %d\n%s",
					termW, bestCols[0], i+1, c, v)
			}
		}
	}
}

// Wide terminals (and unknown size, width 0) keep the spacious full row
// format and the full form hint — adaptation must not regress them.
func TestMenuKeepsFullFormatWhenWide(t *testing.T) {
	for _, termW := range []int{0, 80, 120} {
		m := newMenuModel(store.Scores{"beginner": 42}, termW, 40)
		if v := m.View(); !strings.Contains(v, "· 10 mines  best: 042") {
			t.Errorf("w=%d: menu lost the full row format:\n%s", termW, v)
		}
		m.showForm = true
		if !strings.Contains(m.View(), "tab/enter next · enter on mines starts · esc back") {
			t.Errorf("w=%d: form lost the full hint", termW)
		}
	}
}

// Audit round 3 (finding 6): the real engine.ErrMineCount text contains
// the 14-column word "width*height-2"; on a 10-column terminal the old
// per-word truncation rendered it "width*hei", silently dropping the "-2"
// rule content. Long words must hard-break across lines, so joining the
// rendered fields without separators reconstructs every original word.
func TestFormErrorWordsSurviveHardWrapAtWidthTen(t *testing.T) {
	m := newMenuModel(store.Scores{}, 10, 24)
	m.showForm = true
	m.form.fields = [3]string{"5", "5", "99"}
	_, err := m.form.preset(10, 24)
	if err == nil {
		t.Fatal("expected ErrMineCount for a 5×5 board with 99 mines")
	}
	m.err = err.Error()
	v := m.View()
	joined := strings.Join(strings.Fields(v), "")
	for _, w := range strings.Fields(err.Error()) {
		if !strings.Contains(joined, w) {
			t.Errorf("error word %q lost at width 10:\n%s", w, v)
		}
	}
}

// A validation message longer than the terminal is word-wrapped, not
// clipped: every word survives somewhere on screen.
func TestFormErrorWrapsVisiblyAtNarrowWidth(t *testing.T) {
	m := newMenuModel(store.Scores{}, 30, 24)
	m.showForm = true
	m.form.fields = [3]string{"99", "99", "10"}
	_, err := m.form.preset(30, 24)
	if err == nil {
		t.Fatal("expected a too-large validation error")
	}
	m.err = err.Error()
	v := m.View()
	checkLinesFit(t, "form err w=30", v, 30)
	joined := strings.Join(strings.Fields(v), " ")
	for _, w := range strings.Fields(err.Error()) {
		if !strings.Contains(joined, w) {
			t.Errorf("error word %q lost in wrapping:\n%s", w, v)
		}
	}
}
