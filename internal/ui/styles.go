package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Cell glyphs are two columns wide so the board reads roughly square.
const (
	glyphHidden   = "░░"
	glyphEmpty    = "· "
	glyphFlag     = "⚑ "
	glyphQuestion = "? "
	glyphMine     = "✱ "
	glyphWrong    = "✕ " // a flag that was not a mine, shown on loss
)

// boardChromeRows is the game view's chrome height in rows: HUD (1) + the
// blank line under it (1) + status line (1) + the App's margins above and
// below (2). Pinned against a rendered View in tests.
const boardChromeRows = 5

// chromeCols is the game frame's minimum width in columns: the HUD, the
// compact status variants, and the help overlay all fit within it. Wider
// full-length hints are swapped in by fitLine only when the terminal can
// show them whole.
const chromeCols = 30

// boardCols is the rendered width of a board: 3 columns per cell (separator
// + 2-char glyph) plus the cursor's trailing bracket on the last column.
func boardCols(w int) int { return 3*w + 1 }

// frameCols and frameRows are the terminal size a w×h board's full game
// view needs — board plus chrome. Shared by the App's too-small guard and
// the custom-form fit check so the two can never disagree.
func frameCols(w int) int { return max(boardCols(w), chromeCols) }
func frameRows(h int) int { return h + boardChromeRows }

// fitLine returns full if it fits a terminal termW columns wide (or if the
// width is not yet known), otherwise the compact variant.
func fitLine(termW int, full, compact string) string {
	if termW <= 0 || lipgloss.Width(full) <= termW {
		return full
	}
	return compact
}

// allFit reports whether every line fits termW columns.
func allFit(lines []string, termW int) bool {
	for _, l := range lines {
		if lipgloss.Width(l) > termW {
			return false
		}
	}
	return true
}

// wrapText greedily word-wraps s for terminals termW columns wide, leaving
// one column for the leading margin its callers prepend. A word wider than
// the budget hard-breaks into budget-sized chunks across consecutive lines
// — no characters are lost. Width 0 means the size is unknown: no
// wrapping. An empty s wraps to no lines at all.
func wrapText(s string, termW int) []string {
	if s == "" {
		return nil
	}
	if termW <= 0 {
		return []string{s}
	}
	budget := max(termW-1, 1)
	var lines []string
	line := ""
	for _, w := range strings.Fields(s) {
		for lipgloss.Width(w) > budget {
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			head := clampLine(w, budget)
			if head == "" { // a single rune wider than the budget
				head = string([]rune(w)[:1])
			}
			lines = append(lines, head)
			w = string([]rune(w)[len([]rune(head)):])
		}
		switch {
		case w == "": // word consumed whole by the chunk loop
		case line == "":
			line = w
		case lipgloss.Width(line)+1+lipgloss.Width(w) <= budget:
			line += " " + w
		default:
			lines = append(lines, line)
			line = w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// clampLine hard-truncates a plain-text line to termW display columns. The
// degradation ladders pick variants that fit whole; this is the last-resort
// floor for terminals narrower than any variant.
func clampLine(s string, termW int) string {
	if termW <= 0 || lipgloss.Width(s) <= termW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > termW {
		r = r[:len(r)-1]
	}
	return string(r)
}

// Classic Windows minesweeper number palette.
var numberStyles = map[int]lipgloss.Style{
	1: lipgloss.NewStyle().Foreground(lipgloss.Color("12")), // blue
	2: lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // green
	3: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // red
	4: lipgloss.NewStyle().Foreground(lipgloss.Color("4")),  // navy
	5: lipgloss.NewStyle().Foreground(lipgloss.Color("1")),  // maroon
	6: lipgloss.NewStyle().Foreground(lipgloss.Color("6")),  // teal
	7: lipgloss.NewStyle().Foreground(lipgloss.Color("15")), // white
	8: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),  // gray
}

var (
	styleHidden = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleFlag   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleMine   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleHUD    = lipgloss.NewStyle().Bold(true)
	styleFaint  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)
