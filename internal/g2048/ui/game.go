package ui

import (
	"fmt"
	legacyrand "math/rand"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/terminal-games/internal/g2048/engine"
	"github.com/wallymagill/terminal-games/internal/store"
	"github.com/wallymagill/terminal-games/internal/uiutil"
)

type BackMsg struct{}

func back() tea.Cmd {
	return func() tea.Msg { return BackMsg{} }
}

type Model struct {
	game         *engine.Game
	scores       *store.Store
	rng          *legacyrand.Rand
	termW        int
	termH        int
	bestScore    int
	bestTile     int
	message      string
	showHelp     bool
	showWin      bool
	newBestScore bool
	newBestTile  bool
	saveErr      bool
}

func NewModel(scores *store.Store, rng *legacyrand.Rand) (Model, error) {
	g, err := engine.New(rng)
	if err != nil {
		return Model{}, err
	}
	rec := scores.LoadG2048Classic()
	return Model{
		game:      g,
		scores:    scores,
		rng:       rng,
		bestScore: rec.BestScore,
		bestTile:  rec.BestTile,
	}, nil
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) TooSmall() bool {
	return m.termW > 0 && m.termH > 0 && (m.termW < g2048MinCols || m.termH < g2048MinRows)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW, m.termH = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.TooSmall() {
			if !uiutil.IsRuneBatch(msg) {
				switch msg.String() {
				case "m", "esc":
					m.showHelp = false
					return m, back()
				case "q":
					return m, tea.Quit
				}
			}
			return m, nil
		}
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		var cmds []tea.Cmd
		for _, k := range uiutil.TypedKeys(msg) {
			var cmd tea.Cmd
			m, cmd = m.handleKey(k)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) handleKey(k string) (Model, tea.Cmd) {
	switch k {
	case "?":
		m.showHelp = true
	case "n":
		return m.newGame()
	case "m", "esc":
		return m, back()
	case "q":
		return m, tea.Quit
	case "left", "h":
		return m.applyDir(engine.DirLeft)
	case "down", "j":
		return m.applyDir(engine.DirDown)
	case "up", "k":
		return m.applyDir(engine.DirUp)
	case "right", "l":
		return m.applyDir(engine.DirRight)
	}
	return m, nil
}

func (m Model) newGame() (Model, tea.Cmd) {
	g, err := engine.New(m.rng)
	if err != nil {
		m.message = "could not start game"
		return m, nil
	}
	m.game = g
	m.message = ""
	m.showWin = false
	m.newBestScore = false
	m.newBestTile = false
	m.saveErr = false
	return m, nil
}

func (m Model) applyDir(d engine.Dir) (Model, tea.Cmd) {
	if m.game.Over() {
		m.message = "GAME OVER"
		return m, nil
	}
	res := m.game.Move(d)
	if !res.Changed {
		return m, nil
	}
	if m.showWin {
		m.showWin = false
	}
	if res.Won {
		m.showWin = true
	}
	m = m.maybeRecord()
	return m, nil
}

func (m Model) maybeRecord() Model {
	ns, nt, err := m.scores.RecordG2048Classic(m.game.Score(), m.game.HighestTile())
	if err != nil {
		m.saveErr = true
		return m
	}
	m.saveErr = false
	if ns {
		m.newBestScore = true
		m.bestScore = m.game.Score()
	}
	if nt {
		m.newBestTile = true
		if h := m.game.HighestTile(); h > m.bestTile {
			m.bestTile = h
		}
	}
	return m
}

func (m Model) View() string {
	if m.TooSmall() {
		return styleErr.Render("terminal too small for 2048") + "\n" +
			styleFaint.Render("press m for menu")
	}
	if m.showHelp {
		return m.helpView()
	}
	if m.game.Over() {
		return m.gameOverView()
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render(" 2048") + "\n")
	bestLabel := fmt.Sprintf("BEST %d", m.bestScore)
	if m.newBestScore {
		bestLabel = styleBest.Render(bestLabel)
	} else {
		bestLabel = styleFaint.Render(bestLabel)
	}
	hud := fmt.Sprintf(" SCORE %-6d  %s", m.game.Score(), bestLabel)
	b.WriteString(hud + "\n\n")
	board := m.game.Board()
	for r := 0; r < engine.Size; r++ {
		b.WriteString("  ")
		for c := 0; c < engine.Size; c++ {
			b.WriteString(formatCell(board[r][c]))
			b.WriteString(" ")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleFaint.Render(" "+m.statusLine()) + "\n")
	b.WriteString(styleFaint.Render(fitLine(m.termW,
		" ←↑↓→ / hjkl move · n new · m menu · ? help · q quit",
		" arrows move · n new · m menu",
	)) + "\n")
	return b.String()
}

func (m Model) gameOverView() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(" 2048") + "\n")
	bestLabel := fmt.Sprintf("BEST %d", m.bestScore)
	if m.newBestScore {
		bestLabel = styleBest.Render(bestLabel)
	} else {
		bestLabel = styleFaint.Render(bestLabel)
	}
	b.WriteString(fmt.Sprintf(" SCORE %-6d  %s", m.game.Score(), bestLabel) + "\n\n")

	b.WriteString(styleErr.Render("      G A M E   O V E R") + "\n\n")

	star := "  " + styleBest.Render("★ new best")
	scoreLine := fmt.Sprintf("      score  %6d", m.game.Score())
	if m.newBestScore {
		scoreLine += star
	}
	tileLine := fmt.Sprintf("      tile   %6d", m.game.HighestTile())
	if m.newBestTile {
		tileLine += star
	}
	b.WriteString(scoreLine + "\n")
	b.WriteString(tileLine + "\n")
	b.WriteString(fmt.Sprintf("      best   %6d", m.bestScore) + "\n\n")

	if m.saveErr {
		b.WriteString(styleErr.Render(" could not save record") + "\n")
	}
	b.WriteString(styleFaint.Render(fitLine(m.termW,
		" n new game · m menu · q quit",
		" n new · m menu",
	)) + "\n")
	return b.String()
}

func (m Model) statusLine() string {
	if m.saveErr {
		return "could not save record"
	}
	if m.game.Over() {
		return "GAME OVER"
	}
	if m.showWin {
		return "YOU WIN — keep going"
	}
	if m.newBestScore || m.newBestTile {
		return "new best!"
	}
	return m.message
}

func (m Model) helpView() string {
	lines := []string{
		" 2048",
		"",
		" Slide tiles with arrows or hjkl.",
		" Equal tiles merge into their sum.",
		" Reach 2048 to win — you can keep playing.",
		" No moves left is game over.",
		"",
		" ←↑↓→ / hjkl  move",
		" n  new game",
		" m / esc  menu",
		" ?  help",
		" q  quit",
		"",
		" any key to close help",
	}
	for i, l := range lines {
		lines[i] = styleFaint.Render(clampLine(l, m.termW))
	}
	return strings.Join(lines, "\n")
}
