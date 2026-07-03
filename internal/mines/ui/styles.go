package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/wallymagill/mines/internal/uiutil"
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

func fitLine(termW int, full, compact string) string { return uiutil.FitLine(termW, full, compact) }
func allFit(lines []string, termW int) bool          { return uiutil.AllFit(lines, termW) }
func wrapText(s string, termW int) []string          { return uiutil.WrapText(s, termW) }
func clampLine(s string, termW int) string           { return uiutil.ClampLine(s, termW) }

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
