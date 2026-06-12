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

// A non-paste multi-rune KeyRunes stringifies to its raw runes, so a typed
// batch spelling a special key's NAME — "esc", "tab", "enter", "ctrl+c" — is
// byte-identical to that key's Key.String(). Whole-string dispatch sites
// must never interpret a batch as a key name: per the project rule a
// multi-rune typed message behaves exactly as if each character were typed.
// Real special keys are never KeyRunes, and a single rune cannot spell a
// multi-character name, so guarding batches changes nothing legitimate.

func TestCoalescedEscLettersDoNotAbandonGame(t *testing.T) {
	a := testApp(t)
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]})
	a, _ = updApp(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")})
	if a.view != viewGame {
		t.Errorf("coalesced letters e,s,c abandoned the game (view = %v); per rune: e,s ignored, c chords",
			a.view)
	}
}

func TestCoalescedCtrlCLettersDoNotQuit(t *testing.T) {
	a := testApp(t)
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]})
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+c")})
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("letters c,t,r,l,+,c quit the app; per rune they are ordinary input")
		}
	}
}

// Batches containing the bindable single letters m/q still fall through to
// the focused view per rune, exactly as the round-2 amendment promises: the
// bound runes around them act, and the game view ignores m/q individually.
func TestCoalescedBoundLettersStillDispatchPerRune(t *testing.T) {
	a := testApp(t)
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]})
	a, _ = updApp(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jm")})
	if a.view != viewGame {
		t.Errorf("batched m abandoned the game (view = %v); batches dispatch per rune", a.view)
	}
	if a.game.cursorY != 1 {
		t.Errorf("cursorY = %d, want 1 — the j in a batch must still act", a.game.cursorY)
	}
	var cmd tea.Cmd
	a, cmd = updApp(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("qq")})
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("batched q quit the app; the game view ignores q per rune")
		}
	}
	if a.view != viewGame {
		t.Errorf("batched qq left the game view (view = %v)", a.view)
	}
}

// In the custom form, letter runs spelling key names must hit the per-rune
// digit filter (non-digits ignored) — not navigate, submit, or close — while
// the real enter/tab/esc keys keep their bindings.
func TestCoalescedNameLettersIgnoredInForm(t *testing.T) {
	m := openForm(t, testMenu())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})
	if m.form.active != 0 {
		t.Errorf("letters t,a,b advanced the form field to %d; non-digits must be ignored", m.form.active)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})
	if m.form.active != 0 {
		t.Errorf("letters e,n,t,e,r advanced the form field to %d; non-digits must be ignored", m.form.active)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")})
	if !m.showForm {
		t.Fatal("letters e,s,c closed the form; non-digits must be ignored")
	}
	// The real keys are not KeyRunes and keep working.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.form.active != 1 {
		t.Errorf("real enter key: active = %d, want 1", m.form.active)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.form.active != 2 {
		t.Errorf("real tab key: active = %d, want 2", m.form.active)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.showForm {
		t.Error("real esc key must close the form")
	}
}
