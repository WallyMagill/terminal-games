package engine

import (
	"math/rand"
	"testing"
)

// fixedGame builds a game with a known board and a deterministic rng.
func fixedGame(board [Size][Size]int, score int, rng *rand.Rand) *Game {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	return &Game{board: board, score: score, rng: rng}
}

func TestMoveLeftMergesRow(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{2, 2, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(1)))
	res := g.Move(DirLeft)
	if !res.Changed {
		t.Fatal("expected change")
	}
	if g.board[0][0] != 4 {
		t.Fatalf("board[0][0] = %d, want 4", g.board[0][0])
	}
	if res.Score != 4 || g.Score() != 4 {
		t.Fatalf("score = %d/%d, want 4", res.Score, g.Score())
	}
	empties := 0
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			if g.board[r][c] == 0 {
				empties++
			}
		}
	}
	if empties != 14 { // 16 - 1 merged tile - 1 spawn
		t.Fatalf("empties = %d, want 14", empties)
	}
}

func TestMoveRightUpDown(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{2, 2, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(2)))
	g.Move(DirRight)
	if g.board[0][3] != 4 {
		t.Fatalf("right: [0][3]=%d want 4; board=%v", g.board[0][3], g.board)
	}

	g = fixedGame([Size][Size]int{
		{2, 0, 0, 0},
		{2, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(3)))
	g.Move(DirUp)
	if g.board[0][0] != 4 {
		t.Fatalf("up: [0][0]=%d want 4", g.board[0][0])
	}

	g = fixedGame([Size][Size]int{
		{2, 0, 0, 0},
		{2, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(4)))
	g.Move(DirDown)
	if g.board[3][0] != 4 {
		t.Fatalf("down: [3][0]=%d want 4", g.board[3][0])
	}
}

func TestNoOpDoesNotSpawnOrScore(t *testing.T) {
	board := [Size][Size]int{
		{2, 4, 8, 16},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	g := fixedGame(board, 10, rand.New(rand.NewSource(1)))
	res := g.Move(DirLeft)
	if res.Changed {
		t.Fatal("left on already-packed distinct row must be no-op")
	}
	if g.Score() != 10 {
		t.Fatalf("score = %d, want 10", g.Score())
	}
	if g.board != board {
		t.Fatalf("board mutated on no-op: %v", g.board)
	}
}

func TestDoubleMergeAllDirectionsShape(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{2, 2, 2, 2},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(99)))
	res := g.Move(DirLeft)
	if g.board[0][0] != 4 || g.board[0][1] != 4 {
		t.Fatalf("row0 after left = %v, want 4,4,...", g.board[0])
	}
	if res.Score != 8 {
		t.Fatalf("score = %d, want 8", res.Score)
	}
}
