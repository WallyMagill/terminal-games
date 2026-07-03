package uiutil

import tea "github.com/charmbracelet/bubbletea"

// TypedKeys expands a key message into key strings to dispatch, in order.
func TypedKeys(msg tea.KeyMsg) []string {
	if msg.Type != tea.KeyRunes || msg.Alt {
		return []string{msg.String()}
	}
	if msg.Paste {
		return nil
	}
	keys := make([]string, len(msg.Runes))
	for i, r := range msg.Runes {
		keys[i] = string(r)
	}
	return keys
}

// IsRuneBatch reports whether msg is a coalesced multi-rune KeyRunes message.
func IsRuneBatch(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && len(msg.Runes) > 1
}
