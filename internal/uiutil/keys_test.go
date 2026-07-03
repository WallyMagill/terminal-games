package uiutil

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTypedKeysExpandsCoalescedRunes(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("jl")}
	got := TypedKeys(msg)
	want := []string{"j", "l"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TypedKeys() = %#v, want %#v", got, want)
	}
}

func TestTypedKeysDropsPaste(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ddd"), Paste: true}
	if got := TypedKeys(msg); got != nil {
		t.Fatalf("TypedKeys(paste) = %#v, want nil", got)
	}
}

func TestIsRuneBatch(t *testing.T) {
	batch := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")}
	if !IsRuneBatch(batch) {
		t.Fatal("multi-rune KeyRunes must be a rune batch")
	}
	single := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	if IsRuneBatch(single) {
		t.Fatal("single rune must not be a rune batch")
	}
	esc := tea.KeyMsg{Type: tea.KeyEsc}
	if IsRuneBatch(esc) {
		t.Fatal("real escape key must not be a rune batch")
	}
}
