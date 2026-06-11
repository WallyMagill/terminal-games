package engine

// checkWin promotes the game to Won when every safe cell is revealed.
// Full behavior (auto-flagging remaining mines) is completed in Task 7.
func (g *Game) checkWin() {
	if g.revealed == g.width*g.height-g.mines {
		g.status = Won
	}
}
