package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/engine"
)

// Bubble Tea's input parser collects the longest run of printable runes from
// one terminal read into a single KeyRunes message: fast typing, key repeat
// under render lag, and SSH byte batching all deliver e.g. "jl" as ONE
// KeyMsg. The project rule (set by the custom form's audit amendment) is
// that a multi-rune message behaves exactly as if each character were typed,
// in order; a rune with no binding is ignored individually.

func TestCoalescedRunesDispatchPerRuneInGame(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jl")})
	if m.cursorX != 1 || m.cursorY != 1 {
		t.Errorf("coalesced %q: cursor = (%d,%d), want (1,1) — both typed keys must act",
			"jl", m.cursorX, m.cursorY)
	}

	m = testGameModel(t)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ff")})
	c, _ := m.game.CellAt(0, 0)
	if c.State != engine.Questioned {
		t.Errorf("coalesced %q: cell state = %v, want Questioned (flag cycled twice)",
			"ff", c.State)
	}
}

func TestCoalescedRunesDispatchPerRuneOnMenu(t *testing.T) {
	m := testMenu()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jj")})
	if m.cursor != 2 {
		t.Errorf("coalesced %q: menu cursor = %d, want 2 — both typed keys must act",
			"jj", m.cursor)
	}
}

// Pasted text outside the custom form is fully inert: executing a pasted
// string as game commands is dangerous, so Paste:true messages are dropped
// without dispatching a single rune — they do not move the cursor and do not
// count as "any key" for the help overlay.

func TestPastedRunesInertInGame(t *testing.T) {
	m := testGameModel(t)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jl"), Paste: true})
	if m.cursorX != 0 || m.cursorY != 0 {
		t.Errorf("pasted %q: cursor = (%d,%d), want (0,0) — pasted text must not execute",
			"jl", m.cursorX, m.cursorY)
	}

	m.showHelp = true
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j"), Paste: true})
	if !m.showHelp {
		t.Error("paste closed the help overlay; pasted text must be fully inert")
	}
}

func TestPastedRunesInertOnMenu(t *testing.T) {
	m := testMenu()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jj"), Paste: true})
	if m.cursor != 0 {
		t.Errorf("pasted %q: menu cursor = %d, want 0 — pasted text must not execute",
			"jj", m.cursor)
	}
}
