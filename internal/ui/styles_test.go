package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/wallymagill/mines/internal/store"
)

// fitLine keeps the full variant when it fits exactly: the swap is for lines
// the terminal can't show whole, and an exactly-fitting line can be.
func TestFitLineExactWidthShowsFull(t *testing.T) {
	full, compact := "abcdef", "x"
	if got := fitLine(6, full, compact); got != full {
		t.Errorf("fitLine(6) = %q, want the full line: it fits exactly", got)
	}
	if got := fitLine(5, full, compact); got != compact {
		t.Errorf("fitLine(5) = %q, want the compact line", got)
	}
}

// clampLine truncates to exactly termW columns, not one fewer: the
// degradation floors rely on every clamped column they are given. Kills
// the mutant relaxing the truncation loop bound from > termW to >= termW.
func TestClampLineKeepsExactWidth(t *testing.T) {
	if got := clampLine("abcdef", 4); got != "abcd" {
		t.Errorf("clampLine(\"abcdef\", 4) = %q, want \"abcd\"", got)
	}
	if got := clampLine("too small", 5); got != "too s" {
		t.Errorf("clampLine(\"too small\", 5) = %q, want \"too s\"", got)
	}
}

// Width 0 means the terminal size is not yet known: clampLine must pass
// the line through untouched, so the menu rendered before the first
// WindowSizeMsg keeps its key hint. Kills the mutant narrowing the
// sentinel from termW <= 0 to termW < 0.
func TestClampLineWidthZeroMeansUnknown(t *testing.T) {
	if got := clampLine("abc", 0); got != "abc" {
		t.Errorf("clampLine(\"abc\", 0) = %q, want \"abc\" (0 = unknown)", got)
	}
	m := newMenuModel(store.Scores{}, 0, 0)
	if !strings.Contains(m.View(), "enter select") {
		t.Error("menu at unknown size lost its key hint")
	}
}

// wrapText budgets termW-1 columns, reserving one for the leading margin
// its callers prepend: a line wrapped to exactly termW would render at
// termW+1 and be truncated mid-word. "ab cd" is 5 cols, so at termW=5 it
// must split. Kills the mutant widening the budget to termW. (Compared
// with slices.Equal: fmt.Sprint flattens ["ab cd"] and ["ab","cd"] to
// the same string.)
func TestWrapTextReservesMarginColumn(t *testing.T) {
	got := wrapText("ab cd", 5)
	want := []string{"ab", "cd"}
	if !slices.Equal(got, want) {
		t.Errorf("wrapText(\"ab cd\", 5) = %q, want %q", got, want)
	}
}

// Audit round 3 (finding 6): wrapText rune-truncated words wider than the
// line budget, so the engine message word "width*height-2" lost its "-2"
// tail on terminals 14 columns or narrower. Over-budget words must
// hard-break into budget-sized chunks across consecutive lines instead:
// every line fits, no line is empty, and joining the lines reconstructs
// every original word with no characters lost.
func TestWrapTextHardBreaksOverlongWords(t *testing.T) {
	const msg = "use abcdefghijklmnopqrstuvwxyz then stop"
	for _, termW := range []int{10, 12} {
		lines := wrapText(msg, termW)
		for i, l := range lines {
			if w := lipgloss.Width(l); w > termW-1 {
				t.Errorf("termW=%d: line %d is %d cols, budget %d: %q",
					termW, i, w, termW-1, l)
			}
			if l == "" {
				t.Errorf("termW=%d: line %d is empty", termW, i)
			}
		}
		joined := strings.Join(lines, "")
		for _, w := range strings.Fields(msg) {
			if !strings.Contains(joined, w) {
				t.Errorf("termW=%d: word %q not reconstructible from %q",
					termW, w, lines)
			}
		}
	}
}

// A double-width rune at budget 1 can never fit a chunk; it must still be
// emitted whole (one rune per line) rather than looping or vanishing.
func TestWrapTextWideRuneAtUnitBudget(t *testing.T) {
	got := wrapText("中中", 2)
	want := []string{"中", "中"}
	if !slices.Equal(got, want) {
		t.Errorf("wrapText(\"中中\", 2) = %q, want %q", got, want)
	}
}
