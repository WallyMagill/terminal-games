package ui

import "github.com/wallymagill/terminal-games/internal/solitaire/engine"

type region uint8

const (
	regionStock region = iota
	regionWaste
	regionFoundation
	regionTableau
)

type cursor struct {
	region region
	col    int
	row    int
}

func (c cursor) location() engine.Location {
	switch c.region {
	case regionWaste:
		return engine.Location{Kind: engine.Waste}
	case regionFoundation:
		return engine.Location{Kind: engine.Foundation, Pile: c.col}
	case regionTableau:
		return engine.Location{Kind: engine.Tableau, Pile: c.col, Index: c.row}
	default:
		return engine.Location{Kind: engine.Stock}
	}
}
