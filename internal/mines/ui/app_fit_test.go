package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

// TestAppTooSmallScreenFitsHostileTerminals: the too-small screen is the
// degraded-mode fallback — the screen shown BECAUSE the terminal is small
// must itself fit ANY terminal that triggers it, including terminals
// smaller than the screen's own full four-line footprint (20 cols, 6
// rows). The frame-fit property below only probes needW-1/needH-1, which
// never drops under 29×6, so the hostile range is pinned here.
func TestAppTooSmallScreenFitsHostileTerminals(t *testing.T) {
	for _, sz := range [][2]int{
		{10, 2}, {19, 6}, {20, 5}, {25, 4}, {28, 10},
		{10, 24}, {100, 2}, {9, 1},
	} {
		termW, termH := sz[0], sz[1]
		t.Run(fmt.Sprintf("%dx%d", termW, termH), func(t *testing.T) {
			a := NewApp(store.New(t.TempDir() + "/scores.json"))
			a, _ = updApp(t, a, tea.WindowSizeMsg{Width: termW, Height: termH})
			a, _ = updApp(t, a, startGameMsg{preset: presets[0]})
			v := a.View()
			if !strings.Contains(v, "too small") {
				t.Fatalf("%d×%d must trigger the guard for Beginner", termW, termH)
			}
			lines := strings.Split(v, "\n")
			if len(lines) > termH {
				t.Errorf("too-small screen is %d rows on a %d-row terminal:\n%s",
					len(lines), termH, v)
			}
			checkLinesFit(t, fmt.Sprintf("too-small screen at %d×%d", termW, termH), v, termW)
		})
	}
}

// Degradation must not shed payload while it still fits: at normal trigger
// sizes the screen is byte-identical to the original four-line form, a
// height-4 terminal keeps need/have by dropping only the blank margins,
// and the 10×2 floor still names the state and the way out.
func TestAppTooSmallDegradesByLineAndKeepsPayload(t *testing.T) {
	a := testApp(t)
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]}) // Beginner: frame 30×14

	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 28, Height: 10})
	if v, want := a.View(), "\n Terminal too small.\n need 30×14\n have 28×10\n press m for menu.\n"; v != want {
		t.Errorf("28×10 view = %q, want the full four-line form %q", v, want)
	}

	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 25, Height: 4})
	v := a.View()
	for _, want := range []string{"need 30×14", "have 25×4", "press m for menu."} {
		if !strings.Contains(v, want) {
			t.Errorf("25×4 view must keep %q:\n%s", want, v)
		}
	}

	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 10, Height: 2})
	if v, want := a.View(), "too small\nm: menu"; v != want {
		t.Errorf("10×2 view = %q, want %q", v, want)
	}
}

// TestAppTooSmallHeightThreeFallsToFloor pins the height bound of the
// four-line arm: height 3 is one row too short for the four payload
// lines, so the ladder must fall past them to a form that fits three
// rows. Kills the mutant relaxing `len(full)-2 <= termH` to
// `len(full)-3 <= termH`, which renders four lines on a 3-row terminal.
func TestAppTooSmallHeightThreeFallsToFloor(t *testing.T) {
	a := NewApp(store.New(t.TempDir() + "/scores.json"))
	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 25, Height: 3})
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]})
	v := a.View()
	if !strings.Contains(v, "too small") {
		t.Fatal("25×3 must trigger the too-small screen for Beginner")
	}
	if n := len(strings.Split(v, "\n")); n > 3 {
		t.Errorf("too-small screen is %d lines on a 3-row terminal:\n%s", n, v)
	}
}

// TestAppTooSmallFloorClampsBelowNineCols: on terminals narrower than the
// narrowest variant ("too small", 9 cols) no ladder arm fits whole, and
// only the final hard clamp keeps the floor lines inside the terminal.
// Kills the mutant that deletes tooSmallView's clamp loop.
func TestAppTooSmallFloorClampsBelowNineCols(t *testing.T) {
	a := NewApp(store.New(t.TempDir() + "/scores.json"))
	a, _ = updApp(t, a, tea.WindowSizeMsg{Width: 5, Height: 2})
	a, _ = updApp(t, a, startGameMsg{preset: presets[0]})
	checkLinesFit(t, "floor clamp on a 5-col terminal", a.View(), 5)
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
				checkLinesFit(t, "accepted-or-rejected frame", v, termW)
			})
		}
	}
}
