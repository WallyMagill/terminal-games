package engine

func (g *Game) Move(d Dir) MoveResult {
	if g.over {
		return MoveResult{Score: g.score, Over: true}
	}
	if g.isDead() {
		g.over = true
		return MoveResult{Score: g.score, Over: true}
	}

	next, delta := applyMove(g.board, d)
	if next == g.board {
		return MoveResult{Score: g.score, Over: false}
	}
	g.board = next
	g.score += delta

	firstWin := false
	if !g.won {
		for r := 0; r < Size; r++ {
			for c := 0; c < Size; c++ {
				if g.board[r][c] >= 2048 {
					g.won = true
					firstWin = true
					break
				}
			}
			if firstWin {
				break
			}
		}
	}

	g.spawn()
	g.over = g.isDead()
	return MoveResult{
		Changed: true,
		Score:   g.score,
		Won:     firstWin,
		Over:    g.over,
	}
}

func applyMove(board [Size][Size]int, d Dir) ([Size][Size]int, int) {
	var out [Size][Size]int
	score := 0
	switch d {
	case DirLeft:
		for r := 0; r < Size; r++ {
			line, dlt := slideLine(board[r])
			out[r] = line
			score += dlt
		}
	case DirRight:
		for r := 0; r < Size; r++ {
			var rev [Size]int
			for c := 0; c < Size; c++ {
				rev[c] = board[r][Size-1-c]
			}
			line, dlt := slideLine(rev)
			score += dlt
			for c := 0; c < Size; c++ {
				out[r][Size-1-c] = line[c]
			}
		}
	case DirUp:
		for c := 0; c < Size; c++ {
			var col [Size]int
			for r := 0; r < Size; r++ {
				col[r] = board[r][c]
			}
			line, dlt := slideLine(col)
			score += dlt
			for r := 0; r < Size; r++ {
				out[r][c] = line[r]
			}
		}
	case DirDown:
		for c := 0; c < Size; c++ {
			var col [Size]int
			for r := 0; r < Size; r++ {
				col[r] = board[Size-1-r][c]
			}
			line, dlt := slideLine(col)
			score += dlt
			for r := 0; r < Size; r++ {
				out[Size-1-r][c] = line[r]
			}
		}
	}
	return out, score
}

func (g *Game) spawn() {
	type cell struct{ r, c int }
	var empties []cell
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			if g.board[r][c] == 0 {
				empties = append(empties, cell{r, c})
			}
		}
	}
	if len(empties) == 0 {
		return
	}
	pick := empties[g.rng.Intn(len(empties))]
	v := 2
	if g.rng.Intn(10) == 0 {
		v = 4
	}
	g.board[pick.r][pick.c] = v
}
