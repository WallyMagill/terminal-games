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
	game          *engine.Game
	scores        *store.Store
	rng           *legacyrand.Rand
	cursor        cursor
	selected      *engine.Location
	selectedMoves []engine.Move
	elapsed       int
	running       bool
	tickGen       int
	tickSession   uint64
	termW         int
	termH         int
	message       string
	showHelp      bool
	newBestTime   bool
	newBestMoves  bool
	saveErr       bool
	recordedWin   bool
	bestSeconds   *int
	fewestMoves   *int
}

func NewModel(scores *store.Store, rng *legacyrand.Rand) (Model, error) {
	game, err := engine.New(rng)
	if err != nil {
		return Model{}, err
	}
	bestSeconds, fewestMoves := loadDraw3Record(scores)
	return Model{
		game:        game,
		scores:      scores,
		rng:         rng,
		tickSession: nextTickSession.Add(1),
		bestSeconds: bestSeconds,
		fewestMoves: fewestMoves,
	}, nil
}

func loadDraw3Record(scores *store.Store) (*int, *int) {
	if scores == nil {
		return nil, nil
	}
	rec := scores.LoadRecords().Solitaire["draw3"]
	var bestSeconds *int
	if rec.BestSeconds != nil {
		bestSeconds = new(*rec.BestSeconds)
	}
	var fewestMoves *int
	if rec.FewestMoves != nil {
		fewestMoves = new(*rec.FewestMoves)
	}
	return bestSeconds, fewestMoves
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
		m.termW, m.termH = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.tooSmall() {
			if !uiutil.IsRuneBatch(msg) {
				switch msg.String() {
				case "m", "esc":
					m.showHelp = false
					m = m.clearSelection()
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
	case "d":
		return m.draw()
	case " ", "enter":
		return m.activate()
	case "a":
		return m.autoFoundation()
	case "tab":
		if m.selected != nil {
			return m.cycleSelectedDestination(1), nil
		}
		return m.cyclePlayableSource(1), nil
	case "shift+tab", "backtab":
		if m.selected != nil {
			return m.cycleSelectedDestination(-1), nil
		}
		return m.cyclePlayableSource(-1), nil
	case "1", "2", "3", "4", "5", "6", "7":
		m.cursor = cursor{region: regionTableau, col: int(k[0] - '1')}
		m.clampTableauRowToPlayable()
	case "esc":
		if m.selected != nil {
			m = m.clearSelection()
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
		fresh.termH = m.termH
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
		m = m.clearSelection()
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
		if loc.Kind == engine.Stock {
			return m.draw()
		}
		if !m.selectable(loc) {
			m.message = "No card selected"
			return m, nil
		}
		moves := m.game.LegalMovesFrom(loc)
		switch len(moves) {
		case 0:
			m.selected = &loc
			m.selectedMoves = nil
			m.message = "No legal destination"
			return m, nil
		case 1:
			changed := m.game.ApplyMove(moves[0])
			if changed {
				m.message = "Moved"
			} else {
				m.message = "Invalid move"
			}
			m.recordWinIfNeeded()
			return m.startTimerIfNeeded(changed)
		default:
			m.selected = &loc
			m.selectedMoves = append([]engine.Move(nil), moves...)
			m = m.focusDestination(m.selectedMoves[0].Destination)
			m.message = "Selected"
			return m, nil
		}
	}
	move, ok := m.selectedMoveTo(loc)
	if !ok {
		m = m.clearSelection()
		m.message = "Invalid move"
		return m, nil
	}
	changed := m.game.ApplyMove(move)
	m = m.clearSelection()
	if changed {
		m.message = "Moved"
	} else {
		m.message = "Invalid move"
	}
	m.recordWinIfNeeded()
	return m.startTimerIfNeeded(changed)
}

func (m Model) selectable(loc engine.Location) bool {
	switch loc.Kind {
	case engine.Waste:
		return len(m.game.WasteVisible()) > 0
	case engine.Foundation:
		_, ok := m.game.FoundationTop(engine.Suit(loc.Pile))
		return ok
	case engine.Tableau:
		cards := m.game.Tableau(loc.Pile)
		return loc.Index >= 0 && loc.Index < len(cards) && cards[loc.Index].FaceUp
	default:
		return false
	}
}

func (m Model) autoFoundation() (Model, tea.Cmd) {
	loc := m.cursor.location()
	if m.selected != nil {
		loc = *m.selected
	}
	changed := m.game.AutoFoundation(loc)
	m = m.clearSelection()
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
	if err == nil {
		if newBestTime {
			m.bestSeconds = new(m.elapsed)
		}
		if newBestMoves {
			m.fewestMoves = new(m.game.Moves())
		}
	}
}

func (m Model) recordSummary() string {
	return fmt.Sprintf("best %s/%s", formatRecordSeconds(m.bestSeconds), formatRecordMoves(m.fewestMoves))
}

func formatRecordSeconds(value *int) string {
	if value == nil {
		return "—"
	}
	return fmt.Sprintf("%ds", *value)
}

func formatRecordMoves(value *int) string {
	if value == nil {
		return "—"
	}
	return fmt.Sprintf("%dm", *value)
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
	if m.tooSmall() {
		return m.tooSmallView()
	}
	if m.showHelp {
		return m.helpView()
	}
	var b strings.Builder
	record := m.recordSummary()
	b.WriteString(styleTitle.Render(fitLine(m.termW,
		fmt.Sprintf(" SOLITAIRE DRAW 3      t %03d   moves %03d   %s", m.elapsed, m.game.Moves(), record),
		fmt.Sprintf(" SOLITAIRE DRAW 3   t %03d m %03d %s", m.elapsed, m.game.Moves(), record),
	)))
	b.WriteString("\n\n")
	b.WriteString(m.topView())
	b.WriteString("\n\n")
	b.WriteString(m.tableauView())
	b.WriteString("\n\n")
	b.WriteString(m.statusLine())
	return b.String()
}

func (m Model) tooSmall() bool {
	requiredRows := m.requiredRows()
	return (m.termW > 0 && m.termW < solitaireMinCols) || (m.termH > 0 && m.termH < requiredRows)
}

func (m Model) TooSmall() bool { return m.tooSmall() }

func (m Model) requiredRows() int {
	extraTableauRows := m.maxTableauRows() - 7
	if extraTableauRows <= 0 {
		return solitaireMinRows
	}
	return solitaireMinRows + extraTableauRows
}

func (m Model) maxTableauRows() int {
	maxRows := 0
	for col := range 7 {
		if rows := m.game.TableauLen(col); rows > maxRows {
			maxRows = rows
		}
	}
	return maxRows
}

func (m Model) tooSmallView() string {
	full := []string{
		"",
		" Terminal too small.",
		fmt.Sprintf(" need %d×%d", solitaireMinCols, m.requiredRows()),
		fmt.Sprintf(" have %d×%d", m.termW, m.termH),
		" press m for menu.",
		"",
	}
	compact := []string{
		"too small",
		fmt.Sprintf("need %d×%d", solitaireMinCols, m.requiredRows()),
		fmt.Sprintf("have %d×%d", m.termW, m.termH),
		"m: menu",
	}
	var lines []string
	switch {
	case allFit(full, m.termW) && (m.termH <= 0 || len(full) <= m.termH):
		lines = full
	case allFit(full[1:5], m.termW) && (m.termH <= 0 || len(full)-2 <= m.termH):
		lines = full[1:5]
	case allFit(compact, m.termW) && (m.termH <= 0 || len(compact) <= m.termH):
		lines = compact
	case m.termH != 1:
		lines = []string{compact[0], compact[3]}
	default:
		lines = compact[:1]
	}
	for i, l := range lines {
		lines[i] = clampLine(l, m.termW)
	}
	return strings.Join(lines, "\n")
}

func (m Model) topView() string {
	var b strings.Builder
	b.WriteString(" Stock        Waste             Foundations\n ")
	stock := emptyToken(" · ")
	if m.game.StockCount() > 0 {
		stock = hiddenToken()
	}
	b.WriteString(m.decorate(regionStock, 0, 0, stock))
	b.WriteString(" ")
	waste := m.game.WasteVisible()
	for i := range 3 - len(waste) {
		b.WriteString(emptyToken(""))
		if i < 2 || len(waste) > 0 {
			b.WriteString(" ")
		}
	}
	for i, card := range waste {
		token := styledCard(card)
		if i == len(waste)-1 {
			token = m.decorate(regionWaste, 0, 0, token)
		}
		b.WriteString(token)
		if i < len(waste)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString(" ")
	for suit := engine.Spades; suit <= engine.Clubs; suit++ {
		token := emptyToken("")
		if card, ok := m.game.FoundationTop(suit); ok {
			token = styledCard(card)
		}
		b.WriteString(m.decorate(regionFoundation, int(suit), 0, token))
		if suit < engine.Clubs {
			b.WriteString(" ")
		}
	}
	return b.String()
}

func (m Model) tableauView() string {
	var b strings.Builder
	b.WriteString(" Tableau\n")
	b.WriteString("   1     2     3     4     5     6     7\n")
	maxRows := 0
	cols := make([][]engine.TableauCard, 7)
	for col := range 7 {
		cols[col] = m.game.Tableau(col)
		if len(cols[col]) > maxRows {
			maxRows = len(cols[col])
		}
	}
	for row := range maxRows {
		for col := range 7 {
			token := "     "
			if row < len(cols[col]) {
				card := cols[col][row]
				if card.FaceUp {
					token = styledCard(card.Card)
				} else {
					token = hiddenToken()
				}
				token = m.decorate(regionTableau, col, row, token)
			}
			b.WriteString(" ")
			b.WriteString(token)
		}
		if row < maxRows-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) statusLine() string {
	if m.game.Status() == engine.Won {
		full := fmt.Sprintf(" Solved in %ds · %d moves", m.elapsed, m.game.Moves())
		compact := fmt.Sprintf(" %ds · %d moves", m.elapsed, m.game.Moves())
		if m.saveErr {
			compact = fmt.Sprintf(" %ds · %dm", m.elapsed, m.game.Moves())
		}
		if (m.newBestTime || m.newBestMoves) && !m.saveErr {
			full += " — NEW BEST!"
			compact += " — BEST!"
		}
		if m.saveErr {
			full += styleErr.Render(" (save failed)")
			compact += styleErr.Render(" (save failed)")
		}
		nav := styleFaint.Render(" · n new · m menu")
		full += nav
		compact += nav
		return fitLine(m.termW, full, compact)
	}
	if m.selected != nil {
		description := m.selectedDescription()
		if len(m.selectedMoves) == 0 {
			return fitLine(m.termW,
				styleFaint.Render(fmt.Sprintf(" Selected %s · no legal destination · esc cancel", description)),
				styleFaint.Render(fmt.Sprintf(" Selected %s · no destination · esc", describeLocation(*m.selected))),
			)
		}
		return fitLine(m.termW,
			styleFaint.Render(fmt.Sprintf(" Selected %s · choose destination · tab destinations · enter move · esc cancel", description)),
			styleFaint.Render(fmt.Sprintf(" Selected %s · tab/enter · esc", describeLocation(*m.selected))),
		)
	}
	if m.message != "" {
		return styleErr.Render(" " + clampLine(m.message, m.termW-1))
	}
	return fitLine(m.termW,
		styleFaint.Render(" spc select/move · d draw · a foundation · n new · m menu · ? help"),
		styleFaint.Render(" spc select · d draw · ? help"),
	)
}

func (m Model) helpView() string {
	lines := []string{
		" Solitaire Draw 3",
		"",
		" hjkl/arrows  move",
		" spc/enter    select card or move selection",
		" d            draw 3 / recycle stock",
		" a            move focused/selected card to foundation",
		" esc          cancel selection / menu",
		" n            new deal",
		" m            menu",
		" q            quit",
		"",
		" any key to close help",
	}
	for i, l := range lines {
		lines[i] = styleFaint.Render(clampLine(l, m.termW))
	}
	return strings.Join(lines, "\n")
}

func (m Model) decorate(r region, col, row int, token string) string {
	if m.cursor.region == r && m.cursor.col == col && m.cursor.row == row {
		return styleCursor.Render(token)
	}
	if m.selected != nil && r == regionTableau && m.selected.Kind == engine.Tableau &&
		m.selected.Pile == col && row >= m.selected.Index {
		return styleSelected.Render(token)
	}
	if m.selected != nil && r == regionWaste && m.selected.Kind == engine.Waste {
		return styleSelected.Render(token)
	}
	if m.selected != nil && r == regionFoundation && m.selected.Kind == engine.Foundation && m.selected.Pile == col {
		return styleSelected.Render(token)
	}
	return token
}

func suitGlyph(suit engine.Suit) string {
	switch suit {
	case engine.Spades:
		return "♠"
	case engine.Hearts:
		return "♥"
	case engine.Diamonds:
		return "♦"
	default:
		return "♣"
	}
}
