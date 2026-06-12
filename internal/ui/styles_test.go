package ui

import (
	"strings"
	"testing"

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
