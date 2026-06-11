package engine

// Reveal uncovers the cell at (x, y). Returns true if the board changed.
func (g *Game) Reveal(x, y int) bool {
	if g.status == Won || g.status == Lost || !g.inBounds(x, y) {
		return false
	}
	c := g.at(x, y)
	if c.State == Revealed || c.State == Flagged {
		return false
	}
	if !g.minesPlaced {
		g.placeMines(x, y)
		g.status = Playing
	}
	c.State = Revealed
	g.revealed++
	return true
}
