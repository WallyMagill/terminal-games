package engine

func (g *Game) Clone() *Game {
	clone := &Game{
		stock:  append([]Card(nil), g.stock...),
		waste:  append([]Card(nil), g.waste...),
		moves:  g.moves,
		status: g.status,
	}
	for col := range g.tableau {
		clone.tableau[col] = append([]TableauCard(nil), g.tableau[col]...)
	}
	for suit := range g.foundations {
		clone.foundations[suit] = append([]Card(nil), g.foundations[suit]...)
	}
	return clone
}
