package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wallymagill/mines/internal/mines/engine"
	"github.com/wallymagill/mines/internal/store"
)

type tickMsg struct{ gen int }

func tickCmd(gen int) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{gen: gen} })
}

type gameModel struct {
	game     *engine.Game
	preset   Preset
	scores   *store.Store
	rng      *rand.Rand
	cursorX  int
	cursorY  int
	elapsed  int
	tickGen  int
	termW    int // last known terminal width; 0 = unknown, render full chrome
	running  bool
	recorded bool
	newBest  bool
	saveErr  bool
	showHelp bool
}

func newGameModel(p Preset, scores *store.Store, rng *rand.Rand) (gameModel, error) {
	g, err := engine.New(p.Width, p.Height, p.Mines, rng)
	if err != nil {
		return gameModel{}, err
	}
	return gameModel{game: g, preset: p, scores: scores, rng: rng}, nil
}

func (m gameModel) Update(msg tea.Msg) (gameModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if msg.gen != m.tickGen {
			return m, nil // stale tick loop from a previous round dies here
		}
		if m.running && m.game.Status() == engine.Playing {
			if m.elapsed < 999 {
				m.elapsed++
			}
			return m, tickCmd(m.tickGen)
		}
		m.running = false
		return m, nil
	case tea.KeyMsg:
		// Coalesced typing dispatches per rune; pastes expand to nothing.
		var cmds []tea.Cmd
		for _, k := range typedKeys(msg) {
			var cmd tea.Cmd
			m, cmd = m.handleKey(k)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m gameModel) handleKey(k string) (gameModel, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}
	switch k {
	case "up", "k":
		if m.cursorY > 0 {
			m.cursorY--
		}
	case "down", "j":
		if m.cursorY < m.game.Height()-1 {
			m.cursorY++
		}
	case "left", "h":
		if m.cursorX > 0 {
			m.cursorX--
		}
	case "right", "l":
		if m.cursorX < m.game.Width()-1 {
			m.cursorX++
		}
	case " ", "enter":
		return m.activate(false)
	case "c":
		return m.activate(true)
	case "f":
		m.game.CycleFlag(m.cursorX, m.cursorY)
	case "n":
		fresh, err := newGameModel(m.preset, m.scores, m.rng)
		if err != nil {
			return m, nil // preset was already valid; keep current game
		}
		fresh.tickGen = m.tickGen + 1 // invalidate any in-flight tick
		fresh.termW = m.termW         // keep adapting chrome to the terminal
		return fresh, nil
	case "?":
		m.showHelp = true
	}
	return m, nil
}

// activate runs reveal-or-chord at the cursor, starts the timer on the first
// reveal, and records a best time on a win. chordOnly forces chord (the "c"
// key); otherwise a revealed numbered cell chords and anything else reveals.
func (m gameModel) activate(chordOnly bool) (gameModel, tea.Cmd) {
	wasNotStarted := m.game.Status() == engine.NotStarted
	cell, ok := m.game.CellAt(m.cursorX, m.cursorY)
	if !ok {
		return m, nil
	}
	if chordOnly || cell.State == engine.Revealed {
		m.game.Chord(m.cursorX, m.cursorY)
	} else {
		m.game.Reveal(m.cursorX, m.cursorY)
	}
	var cmd tea.Cmd
	if wasNotStarted && m.game.Status() == engine.Playing && !m.running {
		m.running = true
		cmd = tickCmd(m.tickGen)
	}
	m = m.recordIfWon()
	return m, cmd
}

func (m gameModel) recordIfWon() gameModel {
	if m.game.Status() != engine.Won || m.preset.Key == "" || m.recorded {
		return m
	}
	m.recorded = true
	isBest, err := m.scores.RecordIfBest(m.preset.Key, m.elapsed)
	if err != nil {
		// Deliberate: an unsaved record is not celebrated as a new best; the
		// HUD shows the save warning instead.
		m.saveErr = true
		return m
	}
	m.newBest = isBest
	return m
}

func (m gameModel) View() string {
	var b strings.Builder
	b.WriteString(m.hudView())
	b.WriteString("\n\n")
	b.WriteString(m.boardView())
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	return b.String()
}

func (m gameModel) hudView() string {
	face := "🙂"
	switch m.game.Status() {
	case engine.Won:
		face = "😎"
	case engine.Lost:
		face = "💀"
	}
	// The displayed counter clamps at -99 (classic minesweeper): %03d grows
	// to four characters at -100, shifting the face and timer mid-game. The
	// real count keeps going below; only the display saturates. The positive
	// side never overflows: presets and the 3-digit custom form cap mines at
	// 999.
	return styleHUD.Render(fmt.Sprintf(" 🚩 %03d   %s   ⏱ %03d",
		max(m.game.MinesRemaining(), -99), face, m.elapsed))
}

func (m gameModel) boardView() string {
	var b strings.Builder
	over := m.game.Status() == engine.Won || m.game.Status() == engine.Lost
	for y := 0; y < m.game.Height(); y++ {
		for x := 0; x < m.game.Width(); x++ {
			onCursor := x == m.cursorX && y == m.cursorY && !over
			// Cursor brackets live in the separator columns.
			switch {
			case onCursor:
				b.WriteString("[")
			case x > 0 && m.cursorY == y && m.cursorX == x-1 && !over:
				b.WriteString("]")
			default:
				b.WriteString(" ")
			}
			b.WriteString(m.cellView(x, y, onCursor))
		}
		if m.cursorY == y && m.cursorX == m.game.Width()-1 && !over {
			b.WriteString("]")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m gameModel) cellView(x, y int, onCursor bool) string {
	cell, _ := m.game.CellAt(x, y)
	lost := m.game.Status() == engine.Lost
	glyph, style := m.cellGlyph(cell, lost)
	if onCursor {
		// Reverse the cell's own style so the number keeps its color.
		return style.Reverse(true).Render(glyph)
	}
	return style.Render(glyph)
}

func (m gameModel) cellGlyph(cell engine.Cell, lost bool) (string, lipgloss.Style) {
	switch {
	case lost && cell.Mine && cell.State != engine.Flagged:
		return glyphMine, styleMine
	case lost && !cell.Mine && cell.State == engine.Flagged:
		return glyphWrong, styleMine
	case cell.State == engine.Flagged:
		return glyphFlag, styleFlag
	case cell.State == engine.Questioned:
		return glyphQuestion, styleFlag
	case cell.State != engine.Revealed:
		return glyphHidden, styleHidden
	case cell.Adjacent == 0:
		return glyphEmpty, styleFaint
	default:
		return fmt.Sprintf("%d ", cell.Adjacent), numberStyles[cell.Adjacent]
	}
}

func (m gameModel) statusLine() string {
	if m.showHelp {
		// Style each line individually: a multi-line Render pads every line
		// to the widest, leaving trailing whitespace. Every line stays
		// within chromeCols so the overlay never clips in width on a
		// terminal the guard accepted.
		lines := []string{
			"",
			" hjkl/arrows  move",
			" spc/enter    reveal / chord",
			" f            flag → ? → clear",
			" c            chord",
			" n new · m menu · q quit",
			"",
			" any key to close help",
		}
		for i, l := range lines {
			lines[i] = styleFaint.Render(l)
		}
		return strings.Join(lines, "\n")
	}
	// Full and compact variants: the compact ones all fit chromeCols, the
	// narrowest terminal the too-small guard accepts.
	switch m.game.Status() {
	case engine.Won:
		full := fmt.Sprintf(" 😎 Cleared in %ds", m.elapsed)
		compact := fmt.Sprintf(" 😎 Cleared %ds", m.elapsed)
		if m.newBest {
			full += " — NEW BEST!"
			compact += " — NEW BEST!"
		}
		if m.saveErr {
			full += styleErr.Render("  (could not save score)")
			compact += styleErr.Render(" (save failed)")
		}
		full += styleFaint.Render("  · n new · m menu")
		return fitLine(m.termW, full, compact)
	case engine.Lost:
		return " 💀 Boom." + styleFaint.Render("  · n new · m menu")
	default:
		return fitLine(m.termW,
			styleFaint.Render(" hjkl move · spc reveal · f flag · c chord · n new · ? help · m menu · q quit"),
			styleFaint.Render(" spc reveal · f flag · ? help"))
	}
}
