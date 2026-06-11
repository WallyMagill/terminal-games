package engine

// Reveal uncovers the cell at (x, y). Zero cells flood-fill their region.
// Returns true if the board changed.
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
	if c.Mine {
		c.State = Revealed
		g.hitX, g.hitY, g.mineHit = x, y, true
		g.status = Lost
		return true
	}
	g.floodReveal(x, y)
	g.checkWin()
	return true
}

// floodReveal opens (x,y); if it has zero adjacent mines it BFS-expands
// through the connected zero region and its numbered border. Flags block.
func (g *Game) floodReveal(x, y int) {
	type pt struct{ x, y int }
	queue := []pt{{x, y}}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		c := g.at(p.x, p.y)
		if c.State == Revealed || c.State == Flagged || c.Mine {
			continue
		}
		c.State = Revealed
		g.revealed++
		if c.Adjacent == 0 {
			g.eachNeighbor(p.x, p.y, func(nx, ny int) {
				queue = append(queue, pt{nx, ny})
			})
		}
	}
}

// CycleFlag cycles a covered cell hidden → flagged → questioned → hidden.
// Flags block reveals; question marks do not.
func (g *Game) CycleFlag(x, y int) bool {
	if g.status == Won || g.status == Lost || !g.inBounds(x, y) {
		return false
	}
	c := g.at(x, y)
	switch c.State {
	case Hidden:
		c.State = Flagged
		g.flags++
		return true
	case Flagged:
		c.State = Questioned
		g.flags--
		return true
	case Questioned:
		c.State = Hidden
		return true
	}
	return false
}

// Chord is implemented in a later task; only the game-over guard exists now.
func (g *Game) Chord(x, y int) bool {
	if g.status == Won || g.status == Lost || !g.inBounds(x, y) {
		return false
	}
	return false
}
