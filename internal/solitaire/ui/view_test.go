package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/wallymagill/mines/internal/solitaire/engine"
)

func TestViewRendersCoreRegions(t *testing.T) {
	m := testModel(t)
	v := m.View()
	for _, want := range []string{"SOLITAIRE DRAW 3", "Stock", "Waste", "Foundations", "Tableau"} {
		if !strings.Contains(v, want) {
			t.Fatalf("view missing %q:\n%s", want, v)
		}
	}
}

func TestHelpViewRendersControls(t *testing.T) {
	m := testModel(t)
	m.showHelp = true
	v := m.View()
	for _, want := range []string{"Solitaire Draw 3", "d            draw 3", "spc/enter"} {
		if !strings.Contains(v, want) {
			t.Fatalf("help missing %q:\n%s", want, v)
		}
	}
}

func TestDrawUpdatesWasteInView(t *testing.T) {
	m := testModel(t)
	before := m.View()
	m, _ = m.Update(key("d"))
	after := m.View()
	if before == after {
		t.Fatal("view should change after drawing waste cards")
	}
	if !strings.Contains(after, "moves 001") {
		t.Fatalf("view should show one move after draw:\n%s", after)
	}
}

func TestViewShowsTooSmallGuardWhenWidthCannotFitLayout(t *testing.T) {
	m := testModel(t)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 24, Height: 24})
	m = model
	v := m.View()
	if !strings.Contains(v, "too small") {
		t.Fatalf("narrow view should show the too-small guard:\n%s", v)
	}
	checkSolitaireLinesFit(t, "too-small guard", v, 24)
}

func TestViewHeightGuardAccountsForTallTableau(t *testing.T) {
	m := testModel(t)
	m.termW = solitaireMinCols
	m.termH = solitaireMinRows

	initial := m.View()
	if strings.Contains(initial, "too small") {
		t.Fatalf("initial minimum-size view should render the normal board:\n%s", initial)
	}
	if !strings.Contains(initial, "Tableau") {
		t.Fatalf("initial minimum-size view should include tableau:\n%s", initial)
	}

	setTableauRows(t, m.game, 8)

	tall := m.View()
	if !strings.Contains(tall, "too small") {
		t.Fatalf("tall tableau at minimum height should show the too-small guard:\n%s", tall)
	}
}

func TestStatusLineUsesCompactHintAtMinimumWidth(t *testing.T) {
	m := testModel(t)
	m.termW = solitaireMinCols
	if s := m.statusLine(); !strings.Contains(s, "spc select") || strings.Contains(s, "foundation") {
		t.Fatalf("minimum-width status line should use compact controls, got %q", s)
	}
	checkSolitaireLinesFit(t, "minimum status", m.View(), solitaireMinCols)
}

func TestSelectedStatusLineTakesPriorityOverMessage(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	v := m.View()
	if !strings.Contains(v, "selected · choose destination") {
		t.Fatalf("selected view should show destination guidance:\n%s", v)
	}
	if strings.Contains(v, "\n Selected") {
		t.Fatalf("selected view should not show the generic selected message:\n%s", v)
	}
}

func TestWonCompactStatusKeepsNavigationHints(t *testing.T) {
	m := testModel(t)
	m.game = oneMoveFromWin(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m.elapsed = 999
	m.termW = solitaireMinCols
	m, _ = m.Update(key("a"))
	status := m.statusLine()
	for _, want := range []string{"BEST", "n new", "m menu"} {
		if !strings.Contains(status, want) {
			t.Fatalf("won compact status missing %q in %q", want, status)
		}
	}
	if w := lipgloss.Width(status); w > solitaireMinCols {
		t.Fatalf("won compact status is %d cols, max %d: %q", w, solitaireMinCols, status)
	}
}

func TestWonCompactSaveErrorStatusFitsMinimumWidth(t *testing.T) {
	m := testModel(t)
	m.game = oneMoveFromWin(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m.elapsed = 999
	m.termW = solitaireMinCols
	m, _ = m.Update(key("a"))
	m.saveErr = true
	status := m.statusLine()
	for _, want := range []string{"save failed", "n new", "m menu"} {
		if !strings.Contains(status, want) {
			t.Fatalf("save-error status missing %q in %q", want, status)
		}
	}
	if w := lipgloss.Width(status); w > solitaireMinCols {
		t.Fatalf("save-error status is %d cols, max %d: %q", w, solitaireMinCols, status)
	}
}

func TestWonWideSaveErrorStatusDoesNotCelebrateUnsavedBest(t *testing.T) {
	m := testModel(t)
	m.game = oneMoveFromWin(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m.elapsed = 999
	m.termW = 120
	m, _ = m.Update(key("a"))
	m.saveErr = true
	status := m.statusLine()
	for _, want := range []string{"Solved in", "save failed", "n new", "m menu"} {
		if !strings.Contains(status, want) {
			t.Fatalf("wide save-error status missing %q in %q", want, status)
		}
	}
	if strings.Contains(status, "NEW BEST") {
		t.Fatalf("wide save-error status should not celebrate an unsaved best: %q", status)
	}
}

func TestViewDecoratesFocusedStock(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	m := testModel(t)
	v := m.View()
	if !strings.Contains(v, "\x1b[7m") {
		t.Fatalf("focused stock should be reverse-styled:\n%q", v)
	}
}

func TestViewDecoratesSelectedTableauStack(t *testing.T) {
	m := testModel(t)
	col, row, card := firstFaceUpTableauCard(t, m)
	m.selected = &engine.Location{Kind: engine.Tableau, Pile: col, Index: row}
	lipgloss.SetColorProfile(termenv.TrueColor)
	v := m.View()
	want := styleSelected.Render(styledCard(card))
	if !strings.Contains(v, want) {
		t.Fatalf("selected tableau card missing %q:\n%q", want, v)
	}
}

func firstFaceUpTableauCard(t *testing.T, m Model) (int, int, engine.Card) {
	t.Helper()
	for col := range 7 {
		for row, card := range m.game.Tableau(col) {
			if card.FaceUp {
				return col, row, card.Card
			}
		}
	}
	t.Fatal("test deal has no face-up tableau cards")
	return 0, 0, engine.Card{}
}

func setTableauRows(t *testing.T, g *engine.Game, rows int) {
	t.Helper()
	var tableau [7][]engine.TableauCard
	for row := range rows {
		tableau[0] = append(tableau[0], engine.TableauCard{
			Card:   engine.Card{Rank: engine.Rank(row%int(engine.King) + 1), Suit: engine.Suit(row % 4)},
			FaceUp: true,
		})
	}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), tableau)
}

func checkSolitaireLinesFit(t *testing.T, ctx, view string, termW int) {
	t.Helper()
	for i, line := range strings.Split(view, "\n") {
		if w := lipgloss.Width(line); w > termW {
			t.Errorf("%s: line %d is %d cols (limit %d): %q", ctx, i, w, termW, line)
		}
	}
}
