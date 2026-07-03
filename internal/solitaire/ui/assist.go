package ui

import (
	"fmt"

	"github.com/wallymagill/mines/internal/solitaire/engine"
)

func (m Model) playableSources() []engine.Location {
	sources := m.game.PlayableSources()
	playable := sources[:0]
	for _, source := range sources {
		if len(m.game.LegalMovesFrom(source)) > 0 {
			playable = append(playable, source)
		}
	}
	return playable
}

func (m Model) focusLocation(loc engine.Location) Model {
	switch loc.Kind {
	case engine.Waste:
		m.cursor = cursor{region: regionWaste}
	case engine.Foundation:
		m.cursor = cursor{region: regionFoundation, col: loc.Pile}
	case engine.Tableau:
		m.cursor = cursor{region: regionTableau, col: loc.Pile, row: loc.Index}
		m.clampTableauRow()
	default:
		m.cursor = cursor{region: regionStock}
	}
	return m
}

func (m Model) cyclePlayableSource(direction int) Model {
	sources := m.playableSources()
	if len(sources) == 0 {
		return m
	}
	current := m.cursor.location()
	index := -1
	for i, source := range sources {
		if source == current {
			index = i
			break
		}
	}
	if index == -1 {
		if direction < 0 {
			return m.focusLocation(sources[len(sources)-1])
		}
		return m.focusLocation(sources[0])
	}
	next := (index + direction + len(sources)) % len(sources)
	return m.focusLocation(sources[next])
}

func (m Model) cycleSelectedDestination(direction int) Model {
	if len(m.selectedMoves) == 0 {
		return m
	}
	current := m.cursor.location()
	index := -1
	for i, move := range m.selectedMoves {
		if move.Destination == current {
			index = i
			break
		}
	}
	if index == -1 {
		if direction < 0 {
			return m.focusLocation(m.selectedMoves[len(m.selectedMoves)-1].Destination)
		}
		return m.focusLocation(m.selectedMoves[0].Destination)
	}
	next := (index + direction + len(m.selectedMoves)) % len(m.selectedMoves)
	return m.focusLocation(m.selectedMoves[next].Destination)
}

func (m Model) selectedMoveTo(dst engine.Location) (engine.Move, bool) {
	for _, move := range m.selectedMoves {
		if move.Destination == dst {
			return move, true
		}
	}
	return engine.Move{}, false
}

func (m Model) clearSelection() Model {
	m.selected = nil
	m.selectedMoves = nil
	return m
}

func (m *Model) clampTableauRowToPlayable() {
	cards := m.game.Tableau(m.cursor.col)
	if len(cards) == 0 {
		m.cursor.row = 0
		return
	}
	if m.cursor.row >= len(cards) {
		m.cursor.row = len(cards) - 1
	}
	if m.cursor.row < 0 {
		m.cursor.row = 0
	}
	if cards[m.cursor.row].FaceUp {
		return
	}
	for row, card := range cards {
		if card.FaceUp {
			m.cursor.row = row
			return
		}
	}
}

func (m Model) selectedDescription() string {
	if m.selected == nil {
		return ""
	}
	loc := *m.selected
	card, stackLen, ok := m.cardAt(loc)
	if !ok {
		return describeLocation(loc)
	}
	description := describeCard(card)
	if stackLen > 1 {
		description = fmt.Sprintf("%s stack (%d cards)", description, stackLen)
	}
	return fmt.Sprintf("%s from %s", description, describeLocation(loc))
}

func (m Model) cardAt(loc engine.Location) (engine.Card, int, bool) {
	switch loc.Kind {
	case engine.Waste:
		waste := m.game.WasteVisible()
		if len(waste) == 0 {
			return engine.Card{}, 0, false
		}
		return waste[len(waste)-1], 1, true
	case engine.Foundation:
		card, ok := m.game.FoundationTop(engine.Suit(loc.Pile))
		return card, 1, ok
	case engine.Tableau:
		cards := m.game.Tableau(loc.Pile)
		if loc.Index < 0 || loc.Index >= len(cards) || !cards[loc.Index].FaceUp {
			return engine.Card{}, 0, false
		}
		return cards[loc.Index].Card, len(cards) - loc.Index, true
	default:
		return engine.Card{}, 0, false
	}
}

func describeCard(card engine.Card) string {
	rank := ""
	switch card.Rank {
	case engine.Ace:
		rank = "A"
	case engine.Jack:
		rank = "J"
	case engine.Queen:
		rank = "Q"
	case engine.King:
		rank = "K"
	default:
		rank = fmt.Sprintf("%d", card.Rank)
	}
	return rank + suitGlyph(card.Suit)
}

func describeLocation(loc engine.Location) string {
	switch loc.Kind {
	case engine.Waste:
		return "waste"
	case engine.Foundation:
		return fmt.Sprintf("%s foundation", suitName(engine.Suit(loc.Pile)))
	case engine.Tableau:
		return fmt.Sprintf("tableau %d", loc.Pile+1)
	default:
		return "stock"
	}
}

func suitName(suit engine.Suit) string {
	switch suit {
	case engine.Spades:
		return "spades"
	case engine.Hearts:
		return "hearts"
	case engine.Diamonds:
		return "diamonds"
	default:
		return "clubs"
	}
}
