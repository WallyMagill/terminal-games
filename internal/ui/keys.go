package ui

import tea "github.com/charmbracelet/bubbletea"

// typedKeys expands a key message into the key strings to dispatch, in order.
// Bubble Tea's input parser coalesces the longest run of printable runes from
// one terminal read into a single KeyRunes message — fast typing, key repeat
// under render lag, and SSH byte batching all deliver e.g. "jl" as ONE
// KeyMsg — so a multi-rune typed message must behave exactly as if each
// character were typed (a rune with no binding is ignored individually by the
// caller's dispatch). Pasted text is different: executing a pasted string as
// commands is dangerous, so Paste messages expand to nothing outside the
// custom form, which filters msg.Runes itself. Special keys, single runes,
// and alt-modified runes pass through as their usual Key.String().
func typedKeys(msg tea.KeyMsg) []string {
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
