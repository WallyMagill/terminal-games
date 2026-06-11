package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/store"
)

func testMenu() menuModel {
	return newMenuModel(store.Scores{"beginner": 42}, 120, 40)
}

// drain runs a returned command and hands back the message it produces.
func drain(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestMenuNavigationClamps(t *testing.T) {
	m := testMenu()
	m, _ = m.Update(key("k"))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", m.cursor)
	}
	for i := 0; i < 10; i++ {
		m, _ = m.Update(key("j"))
	}
	if m.cursor != 4 {
		t.Errorf("cursor = %d, want 4 (Quit, clamped)", m.cursor)
	}
}

func TestMenuSelectPresetEmitsStartGame(t *testing.T) {
	m := testMenu()
	m, _ = m.Update(key("j")) // Intermediate
	_, cmd := m.Update(key("enter"))
	msg, ok := drain(cmd).(startGameMsg)
	if !ok {
		t.Fatalf("msg = %T, want startGameMsg", drain(cmd))
	}
	if msg.preset.Name != "Intermediate" {
		t.Errorf("preset = %s, want Intermediate", msg.preset.Name)
	}
}

func TestMenuQuit(t *testing.T) {
	m := testMenu()
	for i := 0; i < 4; i++ {
		m, _ = m.Update(key("j"))
	}
	_, cmd := m.Update(key("enter"))
	if msg := drain(cmd); msg == nil {
		t.Fatal("selecting Quit should return tea.Quit")
	} else if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("msg = %T, want tea.QuitMsg", msg)
	}
}

func TestCustomFormFlow(t *testing.T) {
	m := testMenu()
	for i := 0; i < 3; i++ {
		m, _ = m.Update(key("j")) // Custom
	}
	m, _ = m.Update(key("enter"))
	if !m.showForm {
		t.Fatal("enter on Custom should open the form")
	}
	// Type width 12, height 8, mines 20.
	for _, k := range []string{"1", "2", "enter", "8", "enter", "2", "0"} {
		m, _ = m.Update(key(k))
	}
	_, cmd := m.Update(key("enter"))
	msg, ok := drain(cmd).(startGameMsg)
	if !ok {
		t.Fatalf("msg = %T, want startGameMsg", drain(cmd))
	}
	p := msg.preset
	if p.Width != 12 || p.Height != 8 || p.Mines != 20 || p.Key != "" {
		t.Errorf("preset = %+v, want 12x8/20 with empty key", p)
	}
}

func TestCustomFormValidation(t *testing.T) {
	tests := []struct {
		name    string
		entries []string // width, height, mines
		wantErr string
	}{
		{"empty fields", []string{"", "", ""}, "required"},
		{"board too small", []string{"1", "9", "5"}, "at least 2x2"},
		{"too many mines", []string{"4", "4", "15"}, "mines must be"},
		{"too big for terminal", []string{"99", "9", "10"}, "too large"},
		{"too wide for 80 cols", []string{"30", "9", "10"}, "too large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMenuModel(store.Scores{}, 80, 24)
			for i := 0; i < 3; i++ {
				m, _ = m.Update(key("j"))
			}
			m, _ = m.Update(key("enter")) // open form
			for _, field := range tt.entries {
				for _, ch := range field {
					m, _ = m.Update(key(string(ch)))
				}
				m, _ = m.Update(key("enter"))
			}
			if m.err == "" || !strings.Contains(m.err, tt.wantErr) {
				t.Errorf("err = %q, want containing %q", m.err, tt.wantErr)
			}
			if !m.showForm {
				t.Error("form should stay open on validation error")
			}
		})
	}
}

// A terminal narrower than the game chrome fits no board at all: even a
// 2×2 board's frame (HUD, status line) would be truncated there.
func TestCustomFormRejectsBelowChromeWidth(t *testing.T) {
	m := openForm(t, newMenuModel(store.Scores{}, chromeCols-1, 24))
	for _, k := range []string{"2", "enter", "2", "enter", "1", "enter"} {
		m, _ = m.Update(key(k))
	}
	if !strings.Contains(m.err, "too large") {
		t.Errorf("err = %q, want a too-large error: a 29-col terminal cannot show the game chrome", m.err)
	}
	if !strings.Contains(m.err, "max 0×") {
		t.Errorf("err = %q, want max width 0 when no board fits", m.err)
	}
}

func TestCustomFormEscReturnsToList(t *testing.T) {
	m := testMenu()
	for i := 0; i < 3; i++ {
		m, _ = m.Update(key("j"))
	}
	m, _ = m.Update(key("enter"))
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.showForm {
		t.Error("esc should close the form")
	}
}

func openForm(t *testing.T, m menuModel) menuModel {
	t.Helper()
	for i := 0; i < 3; i++ {
		m, _ = m.Update(key("j"))
	}
	m, _ = m.Update(key("enter"))
	if !m.showForm {
		t.Fatal("form should be open")
	}
	return m
}

func TestCustomFormEditing(t *testing.T) {
	m := openForm(t, testMenu())
	// tab wraps 0→1→2→0; shift+tab wraps back 0→2
	for i, want := range []int{1, 2, 0} {
		m, _ = m.Update(key("tab"))
		if m.form.active != want {
			t.Fatalf("tab step %d: active = %d, want %d", i, m.form.active, want)
		}
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.form.active != 2 {
		t.Errorf("shift+tab from 0: active = %d, want 2", m.form.active)
	}
	// backspace on empty field is a no-op
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.form.fields[2] != "" {
		t.Errorf("backspace on empty = %q, want empty", m.form.fields[2])
	}
	// digits append with leading zeros preserved, capped at 3
	for _, d := range []string{"0", "1", "2", "9"} {
		m, _ = m.Update(key(d))
	}
	if m.form.fields[2] != "012" {
		t.Errorf("field = %q, want \"012\" (3-char cap, leading zero kept)", m.form.fields[2])
	}
	// editing clears a stale error
	m.err = "stale"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.err != "" {
		t.Error("backspace should clear stale validation error")
	}
}

func TestFormViewRendersFieldsAndCursor(t *testing.T) {
	m := openForm(t, testMenu())
	m, _ = m.Update(key("9"))
	v := m.View()
	for _, want := range []string{"Custom board", "width", "height", "mines", ">", "9", "esc back"} {
		if !strings.Contains(v, want) {
			t.Errorf("form view missing %q:\n%s", want, v)
		}
	}
}

func TestMenuViewShowsBestTimes(t *testing.T) {
	m := testMenu()
	v := m.View()
	if !strings.Contains(v, "042") {
		t.Error("menu should show the beginner best time")
	}
	if !strings.Contains(v, "—") {
		t.Error("menu should show an em dash for missing best times")
	}
}

// The "best:" fields of all preset rows must start at the same display
// column, whatever times are recorded: the dims segment is padded so the
// narrow 9×9 row lines up with 16×16 and 30×16.
func TestMenuBestColumnAligned(t *testing.T) {
	for _, scores := range []store.Scores{{}, {"beginner": 42, "intermediate": 187}} {
		m := newMenuModel(scores, 120, 40)
		var cols []int
		for _, line := range strings.Split(m.View(), "\n") {
			if idx := strings.Index(line, "best:"); idx >= 0 {
				cols = append(cols, lipgloss.Width(line[:idx]))
			}
		}
		if len(cols) != len(presets) {
			t.Fatalf("scores %v: found %d rows with best:, want %d\n%s",
				scores, len(cols), len(presets), m.View())
		}
		for i, c := range cols[1:] {
			if c != cols[0] {
				t.Errorf("scores %v: best: column ragged: row 0 starts it at col %d, row %d at col %d\n%s",
					scores, cols[0], i+1, c, m.View())
			}
		}
	}
}
