package engine

import "strings"

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
// for any draw-pile top reachable by drawing/recycling, simulated on a clone.
// Drawing without moving cards is deterministic, but this engine's recycle
// rebuilds the deck in reverse 3-chunks, which PERMUTES card order whenever
// the drained count is not a multiple of 3 — later cycles can surface
// different tops. Termination therefore tracks exact (deck, draw) state
// identity, not pile lengths: the walk stops when a state repeats. The state
// space is finite (~thousands at worst); the cap is a pure safety net.
func (g *Game) DeadEnd() bool {
	if g.status != Playing {
		return false
	}
	if len(g.ProductiveMoves()) > 0 {
		return false
	}
	sim := g.Clone()
	seen := map[string]bool{deckKey(sim): true}
	for range 10000 {
		if !sim.Draw() {
			return true // nothing left to draw or recycle
		}
		if len(sim.ProductiveMoves()) > 0 {
			return false
		}
		key := deckKey(sim)
		if seen[key] {
			return true // deck/draw state repeated: every reachable top was seen
		}
		seen[key] = true
	}
	return true
}

// deckKey encodes the exact stock and waste contents; tableau and foundations
// never change during a draw-only walk, so this identifies the full state.
func deckKey(g *Game) string {
	var b strings.Builder
	for _, c := range g.stock {
		b.WriteByte(byte(c.Suit)<<4 | byte(c.Rank))
	}
	b.WriteByte(0xFF)
	for _, c := range g.waste {
		b.WriteByte(byte(c.Suit)<<4 | byte(c.Rank))
	}
	return b.String()
}
