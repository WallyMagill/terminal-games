package engine

// productive reports whether a move makes progress: its destination is a
// foundation, its source is the draw pile, or it reveals a face-down card.
// Draw and recycle are never productive by themselves, and tableau reshuffles
// that reveal nothing keep no game alive.
func productive(move Move) bool {
	switch move.Kind {
	case MoveDraw, MoveRecycle:
		return false
	}
	return move.Destination.Kind == Foundation || move.Source.Kind == Waste || move.RevealsCard
}

func (g *Game) ProductiveMoves() []Move {
	var moves []Move
	for _, move := range g.LegalMoves() {
		if productive(move) {
			moves = append(moves, move)
		}
	}
	return moves
}

// DeadEnd reports whether no productive move is reachable: none now, and none
// for any draw-pile top reachable by drawing/recycling through one full deck
// cycle, simulated on a clone. Drawing without moving cards is deterministic
// and returns to the starting deck/draw configuration after one full cycle,
// so one cycle is sufficient. The iteration cap is a safety net only: deck
// plus draw pile hold at most 24 cards, needing at most 9 draws per cycle.
func (g *Game) DeadEnd() bool {
	if g.status != Playing {
		return false
	}
	if len(g.ProductiveMoves()) > 0 {
		return false
	}
	sim := g.Clone()
	startStock, startWaste := len(sim.stock), len(sim.waste)
	for range 60 {
		if !sim.Draw() {
			return true // nothing left to draw or recycle
		}
		if len(sim.ProductiveMoves()) > 0 {
			return false
		}
		if len(sim.stock) == startStock && len(sim.waste) == startWaste {
			return true // full cycle completed without a productive move
		}
	}
	return true
}
