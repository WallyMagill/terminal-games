package ui

import "testing"

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
