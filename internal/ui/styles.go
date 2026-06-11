package ui

import "github.com/charmbracelet/lipgloss"

// Cell glyphs are two columns wide so the board reads roughly square.
const (
	glyphHidden   = "░░"
	glyphEmpty    = "· "
	glyphFlag     = "⚑ "
	glyphQuestion = "? "
	glyphMine     = "✱ "
	glyphWrong    = "✕ " // a flag that was not a mine, shown on loss
)

// boardChromeRows: HUD (1) + blank (1) + blank (1) + status (1) + margins (2).
const boardChromeRows = 6

// boardCols is the rendered width of a board: 3 columns per cell (separator
// + 2-char glyph) plus the cursor's trailing bracket on the last column.
func boardCols(w int) int { return 3*w + 1 }

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
