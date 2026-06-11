package ui

import (
	"strings"
	"testing"

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
	if m.form == nil {
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
			if m.form == nil {
				t.Error("form should stay open on validation error")
			}
		})
	}
}

func TestCustomFormEscReturnsToList(t *testing.T) {
	m := testMenu()
	for i := 0; i < 3; i++ {
		m, _ = m.Update(key("j"))
	}
	m, _ = m.Update(key("enter"))
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.form != nil {
		t.Error("esc should close the form")
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
