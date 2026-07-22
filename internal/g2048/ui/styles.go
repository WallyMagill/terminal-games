package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/wallymagill/mines/internal/uiutil"
)

const (
	cellInnerWidth = 5 // fits 65536
	g2048MinCols   = 34
	g2048MinRows   = 16
)

var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleFaint = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleBest  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
)

func tileStyle(v int) lipgloss.Style {
	s := lipgloss.NewStyle().Width(cellInnerWidth).Align(lipgloss.Right)
	switch {
	case v == 0:
		return s.Foreground(lipgloss.Color("240"))
	case v <= 4:
		return s.Foreground(lipgloss.Color("252"))
	case v <= 16:
		return s.Foreground(lipgloss.Color("228"))
	case v <= 64:
		return s.Foreground(lipgloss.Color("214"))
	case v <= 256:
		return s.Foreground(lipgloss.Color("208"))
	case v <= 1024:
		return s.Foreground(lipgloss.Color("202"))
	default:
		return s.Bold(true).Foreground(lipgloss.Color("196"))
	}
}

func formatCell(v int) string {
	if v == 0 {
		return tileStyle(0).Render("·")
	}
	return tileStyle(v).Render(fmt.Sprintf("%d", v))
}

func fitLine(termW int, full, compact string) string { return uiutil.FitLine(termW, full, compact) }
func clampLine(s string, termW int) string           { return uiutil.ClampLine(s, termW) }
