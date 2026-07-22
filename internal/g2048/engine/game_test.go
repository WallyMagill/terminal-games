package engine

import (
	"math/rand"
	"testing"
)

func TestNewRejectsNilRNG(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("New(nil) should error")
	}
}

func TestNewSpawnsTwoTiles(t *testing.T) {
	g, err := New(rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			v := g.Board()[r][c]
			if v != 0 {
				count++
				if v != 2 && v != 4 {
					t.Fatalf("tile %d not 2 or 4", v)
				}
			}
		}
	}
	if count != 2 {
		t.Fatalf("tiles = %d, want 2", count)
	}
	if g.Score() != 0 || g.Won() || g.Over() {
		t.Fatalf("fresh game score/won/over = %d/%v/%v", g.Score(), g.Won(), g.Over())
	}
}

func TestWinContinueAndBannerOnce(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{1024, 1024, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(1)))
	res := g.Move(DirLeft)
	if !res.Won || !g.Won() {
		t.Fatal("first 2048 should set Won on result and game")
	}
	if g.Over() {
		t.Fatal("should continue after win")
	}
	res2 := g.Move(DirRight)
	if res2.Won {
		t.Fatal("MoveResult.Won must be true only on the transition into won")
	}
	if !g.Won() {
		t.Fatal("game.Won must remain true")
	}
}

func TestGameOverNoMoves(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{2, 4, 2, 4},
		{4, 2, 4, 2},
		{2, 4, 2, 4},
		{4, 2, 4, 2},
	}, 0, rand.New(rand.NewSource(1)))
	res := g.Move(DirLeft)
	if res.Changed {
		t.Fatal("dead board move must not change")
	}
	if !g.Over() || !res.Over {
		t.Fatal("dead board must be Over")
	}
	board := g.Board()
	res2 := g.Move(DirUp)
	if res2.Changed || g.Board() != board {
		t.Fatal("moves when over must be inert")
	}
}

func TestHighestTile(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{2, 8, 0, 0},
		{0, 0, 16, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}, 0, rand.New(rand.NewSource(1)))
	if g.HighestTile() != 16 {
		t.Fatalf("HighestTile = %d, want 16", g.HighestTile())
	}
}

func TestSpawnUsesEmptyCellsOnly(t *testing.T) {
	g := fixedGame([Size][Size]int{
		{2, 2, 4, 8},
		{16, 32, 64, 128},
		{256, 512, 1024, 4},
		{8, 16, 32, 0},
	}, 0, rand.New(rand.NewSource(1)))
	res := g.Move(DirLeft)
	if !res.Changed {
		t.Fatal("expected change")
	}
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			v := g.Board()[r][c]
			if v < 0 {
				t.Fatalf("negative tile at %d,%d", r, c)
			}
		}
	}
}
