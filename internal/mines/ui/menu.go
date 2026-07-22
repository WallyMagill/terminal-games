package ui

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/terminal-games/internal/mines/engine"
	"github.com/wallymagill/terminal-games/internal/store"
)

// startGameMsg is emitted by the menu and consumed by the root model.
type startGameMsg struct{ preset Preset }

func startGame(p Preset) tea.Cmd {
	return func() tea.Msg { return startGameMsg{preset: p} }
}

var (
	itemCustom = len(presets)
	itemQuit   = itemCustom + 1
)

type menuModel struct {
	cursor   int
	scores   store.Scores
	form     customForm
	showForm bool
	err      string
	termW    int
	termH    int
}

func newMenuModel(scores store.Scores, termW, termH int) menuModel {
	return menuModel{scores: scores, termW: termW, termH: termH}
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.showForm {
		return m.updateForm(keyMsg)
	}
	// Coalesced typing dispatches per rune; pastes expand to nothing. The
	// form cannot open mid-batch: only enter/space open it, and the parser
	// never coalesces those into a rune run.
	var cmds []tea.Cmd
	for _, k := range typedKeys(keyMsg) {
		var cmd tea.Cmd
		m, cmd = m.handleListKey(k)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m menuModel) handleListKey(k string) (menuModel, tea.Cmd) {
	switch k {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < itemQuit {
			m.cursor++
		}
	case "enter", " ":
		switch {
		case m.cursor < len(presets):
			return m, startGame(presets[m.cursor])
		case m.cursor == itemCustom:
			m.form = newCustomForm()
			m.showForm = true
			m.err = ""
		default:
			return m, tea.Quit
		}
	case "m", "esc":
		return m, backToLauncher()
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

type customForm struct {
	fields [3]string // width, height, mines
	active int
}

func newCustomForm() customForm { return customForm{} }

func (m menuModel) updateForm(keyMsg tea.KeyMsg) (menuModel, tea.Cmd) {
	// A coalesced rune run can spell a key NAME ("tab", "esc", "enter") and
	// would match the switch below (see isRuneBatch). Batches carry only
	// typed text or pastes, so route them straight to the digit filter.
	if isRuneBatch(keyMsg) {
		return m.formTyped(keyMsg), nil
	}
	switch keyMsg.String() {
	case "esc":
		m.showForm = false
		m.err = ""
	case "tab", "down":
		m.form.active = (m.form.active + 1) % 3
	case "shift+tab", "up":
		m.form.active = (m.form.active + 2) % 3
	case "backspace":
		if s := m.form.fields[m.form.active]; s != "" {
			m.form.fields[m.form.active] = s[:len(s)-1]
		}
		m.err = ""
	case "enter":
		if m.form.active < 2 {
			m.form.active++
			return m, nil
		}
		p, err := m.form.preset(m.termW, m.termH)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.showForm = false
		m.err = ""
		return m, startGame(p)
	default:
		return m.formTyped(keyMsg), nil
	}
	return m, nil
}

// formTyped applies typed or pasted input to the active form field. Both
// arrive as KeyRunes messages, and Key.String() wraps bracketed pastes in
// brackets ("[5]"), so filter msg.Runes directly: each rune is treated
// exactly as if typed — digits append up to the 3-char cap, non-digits are
// ignored.
func (m menuModel) formTyped(keyMsg tea.KeyMsg) menuModel {
	if keyMsg.Type != tea.KeyRunes || keyMsg.Alt {
		return m
	}
	for _, r := range keyMsg.Runes {
		if r >= '0' && r <= '9' && len(m.form.fields[m.form.active]) < 3 {
			m.form.fields[m.form.active] += string(r)
			m.err = ""
		}
	}
	return m
}

// preset validates the form. Rule checks are delegated to engine.New (DRY);
// the terminal-fit check is presentation-layer knowledge and lives here.
func (f customForm) preset(termW, termH int) (Preset, error) {
	w, err1 := strconv.Atoi(f.fields[0])
	h, err2 := strconv.Atoi(f.fields[1])
	mines, err3 := strconv.Atoi(f.fields[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return Preset{}, errors.New("all fields are required")
	}
	if _, err := engine.New(w, h, mines, rand.New(rand.NewSource(0))); err != nil {
		return Preset{}, err
	}
	if termW > 0 && termH > 0 && (frameCols(w) > termW || frameRows(h) > termH) {
		maxW := 0
		if termW >= chromeCols { // narrower terminals fit no board at all
			maxW = (termW - 1) / 3
		}
		return Preset{}, fmt.Errorf("board too large for terminal (max %d×%d)",
			maxW, max(termH-boardChromeRows, 0))
	}
	return Preset{Name: "Custom", Width: w, Height: h, Mines: mines}, nil
}

func (m menuModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render(" MINES") + "\n\n")
	if m.showForm {
		return b.String() + m.formView()
	}
	full := make([]string, len(presets))
	compact := make([]string, len(presets))
	for i, p := range presets {
		// Dims are padded to the widest preset ("30×16", 5 cols; × is one
		// rune and one column) so the best: column lines up across rows —
		// in both the full and the compact (chosen below whenever any
		// full row overflows the width budget) variants.
		dims := fmt.Sprintf("%d×%d", p.Width, p.Height)
		full[i] = fmt.Sprintf("%-13s %-5s · %d mines  best: %s",
			p.Name, dims, p.Mines, m.bestFor(p.Key))
		compact[i] = fmt.Sprintf("%-12s %-5s/%-2d %s",
			p.Name, dims, p.Mines, m.bestFor(p.Key))
	}
	// One variant for the whole block: a row's full width depends on its
	// best time ("best: 003" vs "best: —"), so deciding per row mixes
	// formats at widths where only some rows fit, breaking the alignment
	// the padding above establishes. If any full row overflows the budget
	// next to the 3-column cursor prefix, every row renders compact.
	labels := full
	if avail := m.termW - 3; avail > 0 && !allFit(full, avail) {
		labels = compact
	}
	for i := range presets {
		b.WriteString(m.menuRow(i, labels[i]))
	}
	b.WriteString(m.menuRow(itemCustom, "Custom…"))
	b.WriteString(m.menuRow(itemQuit, "Quit"))
	b.WriteString("\n" + styleFaint.Render(clampLine(" ↑↓ move · enter select", m.termW)))
	for _, l := range wrapText(m.err, m.termW) {
		b.WriteString("\n" + styleErr.Render(" "+l))
	}
	return b.String()
}

// menuRow renders one list row next to the 3-column cursor prefix,
// clamping the label as a last resort on terminals narrower than the
// variant chosen for it.
func (m menuModel) menuRow(i int, label string) string {
	if m.termW > 0 {
		label = clampLine(label, m.termW-3)
	}
	if m.cursor == i {
		return styleHUD.Render(" > "+label) + "\n"
	}
	return "   " + label + "\n"
}

func (m menuModel) bestFor(key string) string {
	if secs, ok := m.scores[key]; ok {
		return fmt.Sprintf("%03d", secs)
	}
	return "—"
}

func (m menuModel) formView() string {
	labels := []string{"width", "height", "mines"}
	var b strings.Builder
	b.WriteString(" Custom board\n\n")
	for i, l := range labels {
		cursor := " "
		if m.form.active == i {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf(" %s %-7s %s\n", cursor, l+":", m.form.fields[i]))
	}
	b.WriteString("\n" + styleFaint.Render(fitLine(m.termW,
		" tab/enter next · enter on mines starts · esc back",
		" tab next · enter start · esc")))
	for _, l := range wrapText(m.err, m.termW) {
		b.WriteString("\n" + styleErr.Render(" "+l))
	}
	return b.String()
}
