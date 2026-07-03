package engine

type MoveKind uint8

const (
	MoveDraw MoveKind = iota
	MoveRecycle
	MoveWasteToFoundation
	MoveWasteToTableau
	MoveTableauToFoundation
	MoveTableauToTableau
	MoveFoundationToTableau
)

type Move struct {
	Kind        MoveKind
	Source      Location
	Destination Location
	Cards       []Card
	RevealsCard bool
	Safe        bool
}

func (g *Game) LegalMoves() []Move {
	if g.status == Won {
		return nil
	}

	var moves []Move
	for _, src := range g.PlayableSources() {
		moves = append(moves, g.LegalMovesFrom(src)...)
	}
	return moves
}

func (g *Game) LegalMovesFrom(src Location) []Move {
	if g.status == Won {
		return nil
	}

	switch src.Kind {
	case Stock:
		return g.stockMoves()
	case Waste:
		return g.cardMovesFrom(src)
	case Tableau:
		return g.cardMovesFrom(src)
	case Foundation:
		return g.cardMovesFrom(src)
	default:
		return nil
	}
}

func (g *Game) PlayableSources() []Location {
	if g.status == Won {
		return nil
	}

	var sources []Location
	if len(g.stock) > 0 || len(g.waste) > 0 {
		sources = append(sources, Location{Kind: Stock})
	}
	if len(g.waste) > 0 {
		sources = append(sources, Location{Kind: Waste})
	}
	for pile, cards := range g.tableau {
		for index := range cards {
			if validTableauStack(cards, index) {
				sources = append(sources, Location{Kind: Tableau, Pile: pile, Index: index})
			}
		}
	}
	for pile, cards := range g.foundations {
		if len(cards) > 0 {
			sources = append(sources, Location{Kind: Foundation, Pile: pile})
		}
	}
	return sources
}

func (g *Game) ApplyMove(move Move) bool {
	switch move.Kind {
	case MoveDraw, MoveRecycle:
		return g.Draw()
	case MoveWasteToFoundation, MoveWasteToTableau, MoveTableauToFoundation, MoveTableauToTableau, MoveFoundationToTableau:
		return g.Move(move.Source, move.Destination)
	default:
		return false
	}
}

func (g *Game) stockMoves() []Move {
	if len(g.stock) > 0 {
		start := len(g.stock) - 3
		if start < 0 {
			start = 0
		}
		return []Move{{
			Kind:        MoveDraw,
			Source:      Location{Kind: Stock},
			Destination: Location{Kind: Waste},
			Cards:       cloneCards(g.stock[start:]),
		}}
	}
	if len(g.waste) > 0 {
		return []Move{{
			Kind:        MoveRecycle,
			Source:      Location{Kind: Stock},
			Destination: Location{Kind: Stock},
			Cards:       cloneCards(g.waste),
		}}
	}
	return nil
}

func (g *Game) cardMovesFrom(src Location) []Move {
	cards, ok := g.peek(src)
	if !ok {
		return nil
	}
	cards = cloneCards(cards)

	var moves []Move
	if src.Kind != Foundation {
		for pile := range g.foundations {
			dst := Location{Kind: Foundation, Pile: pile}
			if g.canPlace(cards, dst) {
				moves = append(moves, Move{
					Kind:        moveKind(src.Kind, dst.Kind),
					Source:      src,
					Destination: dst,
					Cards:       cloneCards(cards),
					RevealsCard: g.revealsCard(src),
				})
			}
		}
	}
	for pile := range g.tableau {
		dst := Location{Kind: Tableau, Pile: pile}
		if sameMoveLocation(src, dst) || !g.canPlace(cards, dst) {
			continue
		}
		moves = append(moves, Move{
			Kind:        moveKind(src.Kind, dst.Kind),
			Source:      src,
			Destination: dst,
			Cards:       cloneCards(cards),
			RevealsCard: g.revealsCard(src),
		})
	}
	return moves
}

func moveKind(src, dst PileKind) MoveKind {
	switch {
	case src == Waste && dst == Foundation:
		return MoveWasteToFoundation
	case src == Waste && dst == Tableau:
		return MoveWasteToTableau
	case src == Tableau && dst == Foundation:
		return MoveTableauToFoundation
	case src == Tableau && dst == Tableau:
		return MoveTableauToTableau
	case src == Foundation && dst == Tableau:
		return MoveFoundationToTableau
	default:
		return MoveDraw
	}
}

func (g *Game) revealsCard(src Location) bool {
	if src.Kind != Tableau || src.Pile < 0 || src.Pile >= len(g.tableau) || src.Index <= 0 || src.Index > len(g.tableau[src.Pile]) {
		return false
	}
	return !g.tableau[src.Pile][src.Index-1].FaceUp
}

func cloneCards(cards []Card) []Card {
	return append([]Card(nil), cards...)
}
