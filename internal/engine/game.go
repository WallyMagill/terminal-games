package engine

// checkWin promotes the game to Won when every safe cell is revealed and
// auto-flags any remaining unflagged mines (classic behavior: the counter
// reads zero on a win).
func (g *Game) checkWin() {
	if g.revealed != g.width*g.height-g.mines {
		return
	}
	g.status = Won
	g.autoFlagMines()
}

func (g *Game) autoFlagMines() {
	for i := range g.cells {
		c := &g.cells[i]
		if c.Mine && c.State != Flagged {
			c.State = Flagged
			g.flags++
		}
	}
}
