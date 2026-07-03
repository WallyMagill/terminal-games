package uiutil

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FitLine returns full if it fits termW columns, otherwise compact.
func FitLine(termW int, full, compact string) string {
	if termW <= 0 || lipgloss.Width(full) <= termW {
		return full
	}
	return compact
}

// AllFit reports whether every line fits termW columns.
func AllFit(lines []string, termW int) bool {
	for _, l := range lines {
		if lipgloss.Width(l) > termW {
			return false
		}
	}
	return true
}

// WrapText greedily word-wraps s for terminals termW columns wide.
func WrapText(s string, termW int) []string {
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
			head := ClampLine(w, budget)
			if head == "" {
				head = string([]rune(w)[:1])
			}
			lines = append(lines, head)
			w = string([]rune(w)[len([]rune(head)):])
		}
		switch {
		case w == "":
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

// ClampLine hard-truncates a plain-text line to termW display columns.
func ClampLine(s string, termW int) string {
	if termW <= 0 || lipgloss.Width(s) <= termW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > termW {
		r = r[:len(r)-1]
	}
	return string(r)
}
