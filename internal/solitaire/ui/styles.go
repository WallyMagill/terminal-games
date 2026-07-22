package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/wallymagill/terminal-games/internal/solitaire/engine"
	"github.com/wallymagill/terminal-games/internal/uiutil"
)

const (
	solitaireMinCols = 48
	solitaireMinRows = 18
)

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleFaint    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleRed      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleBlack    = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	styleCursor   = lipgloss.NewStyle().Reverse(true)
	styleSelected = lipgloss.NewStyle().Bold(true).Reverse(true)
	styleErr      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleMoveHint = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("28")).Foreground(lipgloss.Color("231"))
)

func fitLine(termW int, full, compact string) string { return uiutil.FitLine(termW, full, compact) }
func allFit(lines []string, termW int) bool          { return uiutil.AllFit(lines, termW) }
func clampLine(s string, termW int) string           { return uiutil.ClampLine(s, termW) }

func cardToken(card engine.Card) string {
	rank := map[engine.Rank]string{
		engine.Ace: "A", engine.Two: "2", engine.Three: "3", engine.Four: "4",
		engine.Five: "5", engine.Six: "6", engine.Seven: "7", engine.Eight: "8",
		engine.Nine: "9", engine.Ten: "10", engine.Jack: "J", engine.Queen: "Q", engine.King: "K",
	}[card.Rank]
	suit := map[engine.Suit]string{
		engine.Spades: "♠", engine.Hearts: "♥", engine.Diamonds: "♦", engine.Clubs: "♣",
	}[card.Suit]
	return fmt.Sprintf("[%-3s]", rank+suit)
}

func styledCard(card engine.Card) string {
	token := cardToken(card)
	if card.Suit.Red() {
		return styleRed.Render(token)
	}
	return styleBlack.Render(token)
}

func hiddenToken() string      { return styleFaint.Render("[##] ") }
func emptyToken(string) string { return styleFaint.Render("[  ] ") }
