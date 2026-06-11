package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wallymagill/mines/internal/store"
)

// TestAppTooSmallByHeightOnly pins the height arm of the too-small guard:
// a terminal exactly one row too short shows the too-small screen, one
// exactly tall enough plays. Kills the mutant that drops the chrome-rows
// term from the needed height.
func TestAppTooSmallByHeightOnly(t *testing.T) {
	needH := frameRows(presets[0].Height) // Beginner: 9 board rows + chrome
	a := testApp(t)
	model, _ := a.Update(startGameMsg{preset: presets[0]})
	a = model.(App)
	model, _ = a.Update(tea.WindowSizeMsg{Width: 100, Height: needH - 1})
	a = model.(App)
	if !strings.Contains(a.View(), "too small") {
		t.Errorf("%d rows is one short of %d; must show the too-small screen", needH-1, needH)
	}
	model, _ = a.Update(tea.WindowSizeMsg{Width: 100, Height: needH})
	a = model.(App)
	if strings.Contains(a.View(), "too small") {
		t.Errorf("%d rows is exactly tall enough; the game must play", needH)
	}
}

// TestAppFrameFitsWheneverGuardAccepts is the guard's contract as a
// property: across board shapes and terminal sizes straddling the guard
// boundary, an accepted terminal renders no line wider or frame taller
// than itself, and a rejected one gets a too-small screen that satisfies
// the same property.
func TestAppFrameFitsWheneverGuardAccepts(t *testing.T) {
	boards := []Preset{
		{Name: "2x2", Width: 2, Height: 2, Mines: 1},
		presets[0], // Beginner 9x9
		presets[1], // Intermediate 16x16
		presets[2], // Expert 30x16
		{Name: "5x12", Width: 5, Height: 12, Mines: 8},
		{Name: "12x5", Width: 12, Height: 5, Mines: 8},
	}
	for _, p := range boards {
		needW, needH := frameCols(p.Width), frameRows(p.Height)
		sizes := [][2]int{{80, 24}, {100, 35}}
		for _, w := range []int{needW - 1, needW, needW + 3} {
			for _, h := range []int{needH - 1, needH, needH + 3} {
				sizes = append(sizes, [2]int{w, h})
			}
		}
		for _, sz := range sizes {
			termW, termH := sz[0], sz[1]
			t.Run(fmt.Sprintf("%s_at_%dx%d", p.Name, termW, termH), func(t *testing.T) {
				a := NewApp(store.New(t.TempDir() + "/scores.json"))
				model, _ := a.Update(tea.WindowSizeMsg{Width: termW, Height: termH})
				a = model.(App)
				model, _ = a.Update(startGameMsg{preset: p})
				a = model.(App)
				if a.view != viewGame {
					t.Fatalf("could not start game: %q", a.menu.err)
				}
				v := a.View()
				rejected := strings.Contains(v, "too small")
				if want := termW < needW || termH < needH; rejected != want {
					t.Errorf("guard rejected=%v, want %v (need %d×%d, have %d×%d)",
						rejected, want, needW, needH, termW, termH)
				}
				lines := strings.Split(v, "\n")
				if len(lines) > termH {
					t.Errorf("view is %d lines on a %d-row terminal:\n%s", len(lines), termH, v)
				}
				for _, line := range lines {
					if lw := lipgloss.Width(line); lw > termW {
						t.Errorf("line is %d cols on a %d-col terminal: %q", lw, termW, line)
					}
				}
			})
		}
	}
}
