package ui

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/engine"
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
		return m.handleKey(msg.String())
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
