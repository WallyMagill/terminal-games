package engine

type PileKind uint8

const (
	Stock PileKind = iota
	Waste
	Foundation
	Tableau
)

type Location struct {
	Kind  PileKind
	Pile  int
	Index int
}

func (g *Game) Move(src, dst Location) bool {
	if g.status == Won || sameMoveLocation(src, dst) {
		return false
	}
	moving, ok := g.take(src)
	if !ok || len(moving) == 0 {
		return false
	}
	if !g.canPlace(moving, dst) {
		g.put(src, moving)
		return false
	}
	g.place(dst, moving)
	g.flipExposed(src)
	g.moves++
	g.checkWin()
	return true
}

func (g *Game) AutoFoundation(src Location) bool {
	moving, ok := g.peek(src)
	if !ok || len(moving) != 1 {
		return false
	}
	return g.Move(src, Location{Kind: Foundation, Pile: int(moving[0].Suit)})
}

func (g *Game) peek(loc Location) ([]Card, bool) {
	switch loc.Kind {
	case Waste:
		if len(g.waste) == 0 {
			return nil, false
		}
		return []Card{g.waste[len(g.waste)-1]}, true
	case Foundation:
		if loc.Pile < 0 || loc.Pile >= len(g.foundations) || len(g.foundations[loc.Pile]) == 0 {
			return nil, false
		}
		pile := g.foundations[loc.Pile]
		return []Card{pile[len(pile)-1]}, true
	case Tableau:
		if loc.Pile < 0 || loc.Pile >= len(g.tableau) || !validTableauStack(g.tableau[loc.Pile], loc.Index) {
			return nil, false
		}
		stack := g.tableau[loc.Pile][loc.Index:]
		cards := make([]Card, len(stack))
		for i, c := range stack {
			cards[i] = c.Card
		}
		return cards, true
	default:
		return nil, false
	}
}

func (g *Game) take(loc Location) ([]Card, bool) {
	cards, ok := g.peek(loc)
	if !ok {
		return nil, false
	}
	switch loc.Kind {
	case Waste:
		g.waste = g.waste[:len(g.waste)-1]
	case Foundation:
		g.foundations[loc.Pile] = g.foundations[loc.Pile][:len(g.foundations[loc.Pile])-1]
	case Tableau:
		g.tableau[loc.Pile] = g.tableau[loc.Pile][:loc.Index]
	}
	return cards, true
}

func (g *Game) put(loc Location, cards []Card) {
	switch loc.Kind {
	case Waste:
		g.waste = append(g.waste, cards...)
	case Foundation:
		g.foundations[loc.Pile] = append(g.foundations[loc.Pile], cards...)
	case Tableau:
		for _, card := range cards {
			g.tableau[loc.Pile] = append(g.tableau[loc.Pile], TableauCard{Card: card, FaceUp: true})
		}
	}
}

func (g *Game) canPlace(cards []Card, dst Location) bool {
	if len(cards) == 0 {
		return false
	}
	switch dst.Kind {
	case Foundation:
		if len(cards) != 1 || dst.Pile < 0 || dst.Pile >= len(g.foundations) || cards[0].Suit != Suit(dst.Pile) {
			return false
		}
		top, ok := g.FoundationTop(Suit(dst.Pile))
		return canBuildFoundation(cards[0], top, ok)
	case Tableau:
		if dst.Pile < 0 || dst.Pile >= len(g.tableau) {
			return false
		}
		pile := g.tableau[dst.Pile]
		if len(pile) == 0 {
			return cards[0].Rank == King
		}
		dest := pile[len(pile)-1]
		return dest.FaceUp && canStackOn(cards[0], dest.Card)
	default:
		return false
	}
}

func (g *Game) place(dst Location, cards []Card) {
	switch dst.Kind {
	case Foundation:
		g.foundations[dst.Pile] = append(g.foundations[dst.Pile], cards[0])
	case Tableau:
		for _, card := range cards {
			g.tableau[dst.Pile] = append(g.tableau[dst.Pile], TableauCard{Card: card, FaceUp: true})
		}
	}
}

func (g *Game) flipExposed(src Location) {
	if src.Kind != Tableau || src.Pile < 0 || src.Pile >= len(g.tableau) {
		return
	}
	pile := g.tableau[src.Pile]
	if len(pile) == 0 {
		return
	}
	top := len(pile) - 1
	if !pile[top].FaceUp {
		g.tableau[src.Pile][top].FaceUp = true
	}
}

func validTableauStack(cards []TableauCard, index int) bool {
	if index < 0 || index >= len(cards) || !cards[index].FaceUp {
		return false
	}
	for i := index + 1; i < len(cards); i++ {
		if !cards[i].FaceUp || !canStackOn(cards[i].Card, cards[i-1].Card) {
			return false
		}
	}
	return true
}

func canStackOn(card Card, dest Card) bool {
	return dest.Rank == card.Rank+1 && dest.Suit.Red() != card.Suit.Red()
}

func canBuildFoundation(card Card, top Card, ok bool) bool {
	if !ok {
		return card.Rank == Ace
	}
	return card.Suit == top.Suit && card.Rank == top.Rank+1
}

func sameMoveLocation(src, dst Location) bool {
	if src.Kind != dst.Kind || src.Pile != dst.Pile {
		return false
	}
	switch src.Kind {
	case Waste, Foundation:
		return true
	case Tableau:
		return true
	default:
		return false
	}
}

func (g *Game) checkWin() {
	total := 0
	for _, pile := range g.foundations {
		total += len(pile)
	}
	if total == 52 {
		g.status = Won
	}
}
