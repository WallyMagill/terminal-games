package ui

import (
	"fmt"
	legacyrand "math/rand"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/solitaire/engine"
	"github.com/wallymagill/mines/internal/store"
	"github.com/wallymagill/mines/internal/uiutil"
)

type BackMsg struct{}

type tickMsg struct {
	session uint64
	gen     int
}

var nextTickSession atomic.Uint64

func tickCmd(session uint64, gen int) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{session: session, gen: gen} })
}

type Model struct {
	game         *engine.Game
	scores       *store.Store
	rng          *legacyrand.Rand
	cursor       cursor
	selected     *engine.Location
	elapsed      int
	running      bool
	tickGen      int
	tickSession  uint64
	termW        int
	message      string
	showHelp     bool
	newBestTime  bool
	newBestMoves bool
	saveErr      bool
	recordedWin  bool
}

func NewModel(scores *store.Store, rng *legacyrand.Rand) (Model, error) {
	game, err := engine.New(rng)
	if err != nil {
		return Model{}, err
	}
	return Model{game: game, scores: scores, rng: rng, tickSession: nextTickSession.Add(1)}, nil
}

func back() tea.Cmd { return func() tea.Msg { return BackMsg{} } }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if msg.session != m.tickSession || msg.gen != m.tickGen {
			return m, nil
		}
		if m.running && m.game.Status() == engine.Playing {
			if m.elapsed < 999 {
				m.elapsed++
			}
			return m, tickCmd(m.tickSession, m.tickGen)
		}
		m.running = false
		return m, nil
	case tea.WindowSizeMsg:
		m.termW = msg.Width
		return m, nil
	case tea.KeyMsg:
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
	case "d":
		return m.draw()
	case " ", "enter":
		return m.activate()
	case "a":
		return m.autoFoundation()
	case "esc":
		if m.selected != nil {
			m.selected = nil
			m.message = ""
			return m, nil
		}
		return m, back()
	case "m":
		return m, back()
	case "q":
		return m, tea.Quit
	case "n":
		fresh, err := NewModel(m.scores, m.rng)
		if err != nil {
			m.message = "Could not deal a new game"
			return m, nil
		}
		fresh.tickGen = m.tickGen + 1
		fresh.termW = m.termW
		return fresh, nil
	case "left", "h":
		m.moveCursorLeft()
	case "right", "l":
		m.moveCursorRight()
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	}
	return m, nil
}

func (m Model) draw() (Model, tea.Cmd) {
	changed := m.game.Draw()
	if changed {
		m.selected = nil
		m.message = "Drew cards"
	} else {
		m.message = "No cards to draw"
	}
	m.recordWinIfNeeded()
	return m.startTimerIfNeeded(changed)
}

func (m Model) activate() (Model, tea.Cmd) {
	loc := m.cursor.location()
	if m.selected == nil {
		m.selected = &loc
		m.message = "Selected"
		return m, nil
	}
	changed := m.game.Move(*m.selected, loc)
	m.selected = nil
	if changed {
		m.message = "Moved"
	} else {
		m.message = "Invalid move"
	}
	m.recordWinIfNeeded()
	return m.startTimerIfNeeded(changed)
}

func (m Model) autoFoundation() (Model, tea.Cmd) {
	changed := m.game.AutoFoundation(m.cursor.location())
	m.selected = nil
	if changed {
		m.message = "Moved to foundation"
	} else {
		m.message = "No foundation move"
	}
	m.recordWinIfNeeded()
	return m.startTimerIfNeeded(changed)
}

func (m Model) startTimerIfNeeded(changed bool) (Model, tea.Cmd) {
	if !changed || m.running || m.game.Status() != engine.Playing {
		return m, nil
	}
	m.running = true
	return m, tickCmd(m.tickSession, m.tickGen)
}

func (m *Model) recordWinIfNeeded() {
	if m.recordedWin || m.game.Status() != engine.Won {
		return
	}
	m.recordedWin = true
	if m.scores == nil {
		return
	}
	newBestTime, newBestMoves, err := m.scores.RecordSolitaireDraw3(m.elapsed, m.game.Moves())
	m.newBestTime = newBestTime
	m.newBestMoves = newBestMoves
	m.saveErr = err != nil
}

func (m *Model) moveCursorLeft() {
	switch m.cursor.region {
	case regionWaste:
		m.cursor = cursor{region: regionStock}
	case regionFoundation:
		if m.cursor.col > 0 {
			m.cursor.col--
		} else {
			m.cursor = cursor{region: regionWaste}
		}
	case regionTableau:
		if m.cursor.col > 0 {
			m.cursor.col--
			m.clampTableauRow()
		} else {
			m.cursor = cursor{region: regionFoundation, col: 0}
		}
	}
}

func (m *Model) moveCursorRight() {
	switch m.cursor.region {
	case regionStock:
		m.cursor = cursor{region: regionWaste}
	case regionWaste:
		m.cursor = cursor{region: regionFoundation}
	case regionFoundation:
		if m.cursor.col < 3 {
			m.cursor.col++
		} else {
			m.cursor = cursor{region: regionTableau, col: 3}
			m.clampTableauRow()
		}
	case regionTableau:
		if m.cursor.col < 6 {
			m.cursor.col++
			m.clampTableauRow()
		}
	}
}

func (m *Model) moveCursorUp() {
	switch m.cursor.region {
	case regionTableau:
		if m.cursor.row > 0 {
			m.cursor.row--
		} else if m.cursor.col < 4 {
			m.cursor = cursor{region: regionFoundation, col: m.cursor.col}
		} else {
			m.cursor = cursor{region: regionWaste}
		}
	}
}

func (m *Model) moveCursorDown() {
	switch m.cursor.region {
	case regionStock, regionWaste:
		m.cursor = cursor{region: regionTableau}
		m.clampTableauRow()
	case regionFoundation:
		m.cursor = cursor{region: regionTableau, col: m.cursor.col}
		m.clampTableauRow()
	case regionTableau:
		cards := m.game.Tableau(m.cursor.col)
		if m.cursor.row < len(cards)-1 {
			m.cursor.row++
		}
	}
}

func (m *Model) clampTableauRow() {
	cards := m.game.Tableau(m.cursor.col)
	if len(cards) == 0 {
		m.cursor.row = 0
		return
	}
	if m.cursor.row >= len(cards) {
		m.cursor.row = len(cards) - 1
	}
	if m.cursor.row < 0 {
		m.cursor.row = 0
	}
}

func (m Model) View() string {
	if m.showHelp {
		return "SOLITAIRE HELP\n\n←/h ↑/k ↓/j →/l move · d draw · enter select/move · a auto-foundation · n new deal · esc back"
	}
	var b strings.Builder
	b.WriteString("SOLITAIRE\n\n")
	b.WriteString(fmt.Sprintf("Time %03d · Moves %d\n", m.elapsed, m.game.Moves()))
	b.WriteString(fmt.Sprintf("Stock %d · Waste %d\n", m.game.StockCount(), len(m.game.WasteVisible())))
	b.WriteString("Foundations")
	for suit := engine.Spades; suit <= engine.Clubs; suit++ {
		card, ok := m.game.FoundationTop(suit)
		if ok {
			b.WriteString(fmt.Sprintf(" %s", cardText(card)))
		} else {
			b.WriteString(" --")
		}
	}
	b.WriteString("\nTableau")
	for col := range 7 {
		b.WriteString(fmt.Sprintf(" %d:%d", col+1, len(m.game.Tableau(col))))
	}
	if m.selected != nil {
		b.WriteString("\nSelected")
	}
	if m.message != "" {
		b.WriteString("\n" + m.message)
	}
	if m.game.Status() == engine.Won {
		b.WriteString("\nYou won")
		if m.newBestTime {
			b.WriteString(" · best time")
		}
		if m.newBestMoves {
			b.WriteString(" · fewest moves")
		}
		if m.saveErr {
			b.WriteString(" · score save failed")
		}
	}
	b.WriteString("\n\n? help · d draw · enter select · a auto · n new · esc menu")
	return b.String()
}

func cardText(card engine.Card) string {
	rank := fmt.Sprintf("%d", card.Rank)
	switch card.Rank {
	case engine.Ace:
		rank = "A"
	case engine.Jack:
		rank = "J"
	case engine.Queen:
		rank = "Q"
	case engine.King:
		rank = "K"
	}
	suit := "?"
	switch card.Suit {
	case engine.Spades:
		suit = "S"
	case engine.Hearts:
		suit = "H"
	case engine.Diamonds:
		suit = "D"
	case engine.Clubs:
		suit = "C"
	}
	return rank + suit
}
