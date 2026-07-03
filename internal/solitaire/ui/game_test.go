package ui

import (
	legacyrand "math/rand"
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/mines/internal/solitaire/engine"
	"github.com/wallymagill/mines/internal/store"
)

func key(s string) tea.KeyMsg {
	switch s {
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func testModel(t *testing.T) Model {
	t.Helper()
	m, err := NewModel(store.New(t.TempDir()+"/scores.json"), legacyrand.New(legacyrand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestDKeyDrawsAndStartsTimer(t *testing.T) {
	m := testModel(t)
	m, cmd := m.Update(key("d"))
	if m.game.StockCount() != 21 {
		t.Fatalf("stock = %d, want 21", m.game.StockCount())
	}
	if !m.running || cmd == nil {
		t.Fatal("first draw should start timer and return tick command")
	}
}

func TestActivateOnStockDrawsAndStartsTimer(t *testing.T) {
	m := testModel(t)
	m, cmd := m.Update(key(" "))
	if m.game.StockCount() != 21 {
		t.Fatalf("stock = %d, want 21", m.game.StockCount())
	}
	if m.selected != nil {
		t.Fatal("stock activation should draw, not select stock")
	}
	if !m.running || cmd == nil {
		t.Fatal("stock activation should start timer and return tick command")
	}
}

func TestActivateOnEmptyStockRecyclesWaste(t *testing.T) {
	m := testModel(t)
	for range 8 {
		m, _ = m.Update(key("d"))
	}
	if m.game.StockCount() != 0 {
		t.Fatalf("stock after drawing deck = %d, want 0", m.game.StockCount())
	}
	m.cursor = cursor{region: regionStock}
	m, _ = m.Update(key("enter"))
	if m.game.StockCount() != 24 {
		t.Fatalf("stock after activate recycle = %d, want 24", m.game.StockCount())
	}
}

func TestFreshModelIgnoresStaleTickFromPreviousModel(t *testing.T) {
	old := testModel(t)
	old, oldTickCmd := old.Update(key("d"))
	if oldTickCmd == nil {
		t.Fatal("old model draw should start timer")
	}
	staleTick := oldTickCmd()

	fresh := testModel(t)
	fresh, newTickCmd := fresh.Update(key("d"))
	if newTickCmd == nil {
		t.Fatal("fresh model draw should start timer")
	}

	fresh, cmd := fresh.Update(staleTick)
	if cmd != nil {
		t.Fatal("stale tick from old model should not schedule another tick")
	}
	if fresh.elapsed != 0 {
		t.Fatalf("elapsed = %d, want 0 after stale tick", fresh.elapsed)
	}

	currentTick := newTickCmd()
	fresh, cmd = fresh.Update(currentTick)
	if cmd == nil {
		t.Fatal("current model tick should schedule another tick")
	}
	if fresh.elapsed != 1 {
		t.Fatalf("elapsed = %d, want 1 after current tick", fresh.elapsed)
	}
}

func TestEscWithNoSelectionRequestsBack(t *testing.T) {
	m := testModel(t)
	_, cmd := m.Update(key("esc"))
	if cmd == nil {
		t.Fatal("esc with no selection should request launcher back")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("esc with no selection should produce BackMsg")
	}
}

func TestHelpDismissesWithoutMoving(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("?"))
	if !m.showHelp {
		t.Fatal("? should open help")
	}
	before := m.cursor
	m, _ = m.Update(key("l"))
	if m.showHelp {
		t.Fatal("any key should close help")
	}
	if m.cursor != before {
		t.Fatal("help-dismissing key must not also move cursor")
	}
}

func TestTooSmallGuardMenuBypassesHiddenHelp(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("?"))
	if !m.showHelp {
		t.Fatal("? should open help")
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: 24, Height: 8})
	_, cmd := m.Update(key("m"))
	if cmd == nil {
		t.Fatal("m on too-small guard should request launcher back even when help was open")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("m on too-small guard should produce BackMsg")
	}
}

func TestTooSmallGuardIgnoresGameplayKeys(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 24, Height: 8})
	beforeCursor := m.cursor
	m, cmd := m.Update(key("d"))
	if cmd != nil {
		t.Fatal("too-small guard should not start gameplay timers")
	}
	if m.game.StockCount() != 24 || m.game.Moves() != 0 {
		t.Fatalf("too-small guard allowed draw: stock=%d moves=%d", m.game.StockCount(), m.game.Moves())
	}
	m, _ = m.Update(key("l"))
	if m.cursor != beforeCursor {
		t.Fatalf("too-small guard allowed cursor movement: %+v -> %+v", beforeCursor, m.cursor)
	}
}

func TestCursorMovesAcrossRegions(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("right"))
	if m.cursor.region != regionWaste {
		t.Fatalf("right from stock region = %v, want waste", m.cursor.region)
	}
	m, _ = m.Update(key("l"))
	if m.cursor.region != regionFoundation || m.cursor.col != 0 {
		t.Fatalf("right from waste cursor = %+v, want first foundation", m.cursor)
	}
	m, _ = m.Update(key("down"))
	if m.cursor.region != regionTableau || m.cursor.col != 0 || m.cursor.row != 0 {
		t.Fatalf("down from foundation cursor = %+v, want tableau col 0 row 0", m.cursor)
	}
}

func TestSelectionEscCancelsWithoutBack(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key("enter"))
	if m.selected == nil {
		t.Fatal("enter should select current location")
	}
	m, cmd := m.Update(key("esc"))
	if cmd != nil {
		t.Fatal("esc with selection should cancel selection, not leave solitaire")
	}
	if m.selected != nil {
		t.Fatal("esc should clear selection")
	}
}

func TestDrawRecyclesEmptyStock(t *testing.T) {
	m := testModel(t)
	for range 8 {
		m, _ = m.Update(key("d"))
	}
	if m.game.StockCount() != 0 {
		t.Fatalf("stock after drawing deck = %d, want 0", m.game.StockCount())
	}
	m, _ = m.Update(key("d"))
	if m.game.StockCount() != 24 {
		t.Fatalf("stock after recycle = %d, want 24", m.game.StockCount())
	}
	if m.game.Moves() != 9 {
		t.Fatalf("moves = %d, want 9", m.game.Moves())
	}
}

func TestAutoFoundationMovesCursorCard(t *testing.T) {
	m := modelWithTopTableauAce(t)
	col, row := topTableauAce(t, m.game)
	ace := m.game.Tableau(col)[row].Card
	m.cursor = cursor{region: regionTableau, col: col, row: row}
	m, cmd := m.Update(key("a"))
	if cmd == nil || !m.running {
		t.Fatal("successful auto-foundation should start timer")
	}
	if m.game.Moves() != 1 {
		t.Fatalf("moves = %d, want 1", m.game.Moves())
	}
	top, ok := m.game.FoundationTop(ace.Suit)
	if !ok || top != ace {
		t.Fatalf("foundation top = %+v/%v, want moved ace %+v", top, ok, ace)
	}
}

func TestAutoFoundationMovesSelectedCard(t *testing.T) {
	m := modelWithTopTableauAce(t)
	col, row := topTableauAce(t, m.game)
	ace := m.game.Tableau(col)[row].Card
	m.selected = &engine.Location{Kind: engine.Tableau, Pile: col, Index: row}
	m.cursor = cursor{region: regionStock}
	m, cmd := m.Update(key("a"))
	if cmd == nil || !m.running {
		t.Fatal("successful selected auto-foundation should start timer")
	}
	if m.selected != nil {
		t.Fatal("auto-foundation should clear selection")
	}
	top, ok := m.game.FoundationTop(ace.Suit)
	if !ok || top != ace {
		t.Fatalf("foundation top = %+v/%v, want selected ace %+v", top, ok, ace)
	}
}

func TestNewDealResetsBoardAndTimer(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("d"))
	m.elapsed = 7
	m.tickGen = 3
	m, cmd := m.Update(key("n"))
	if cmd != nil {
		t.Fatal("new deal should not return a command")
	}
	if m.game.StockCount() != 24 || m.game.Moves() != 0 || m.elapsed != 0 || m.running {
		t.Fatalf("new deal did not reset state: stock=%d moves=%d elapsed=%d running=%v", m.game.StockCount(), m.game.Moves(), m.elapsed, m.running)
	}
	if m.tickGen != 4 {
		t.Fatalf("tickGen = %d, want 4", m.tickGen)
	}
}

func TestWinningMoveRecordsDraw3Best(t *testing.T) {
	scores := store.New(t.TempDir() + "/scores.json")
	m, err := NewModel(scores, legacyrand.New(legacyrand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	m.game = oneMoveFromWin(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m.elapsed = 12
	m, _ = m.Update(key("a"))
	if m.game.Status() != engine.Won {
		t.Fatalf("status = %v, want won", m.game.Status())
	}
	record := scores.LoadRecords().Solitaire["draw3"]
	if record.BestSeconds == nil || *record.BestSeconds != 12 {
		t.Fatalf("best seconds = %v, want 12", record.BestSeconds)
	}
	if record.FewestMoves == nil || *record.FewestMoves != 1 {
		t.Fatalf("fewest moves = %v, want 1", record.FewestMoves)
	}
	if !m.newBestTime || !m.newBestMoves || m.saveErr {
		t.Fatalf("record flags = time %v moves %v err %v, want true/true/false", m.newBestTime, m.newBestMoves, m.saveErr)
	}
}

func modelWithTopTableauAce(t *testing.T) Model {
	t.Helper()
	for seed := int64(1); seed < 500; seed++ {
		m, err := NewModel(store.New(t.TempDir()+"/scores.json"), legacyrand.New(legacyrand.NewSource(seed)))
		if err != nil {
			t.Fatal(err)
		}
		if hasTopTableauAce(m.game) {
			return m
		}
	}
	t.Fatal("no deterministic seed produced a top tableau ace")
	return Model{}
}

func hasTopTableauAce(g *engine.Game) bool {
	for col := range 7 {
		cards := g.Tableau(col)
		if len(cards) > 0 && cards[len(cards)-1].FaceUp && cards[len(cards)-1].Card.Rank == engine.Ace {
			return true
		}
	}
	return false
}

func topTableauAce(t *testing.T, g *engine.Game) (int, int) {
	t.Helper()
	for col := range 7 {
		cards := g.Tableau(col)
		if len(cards) > 0 && cards[len(cards)-1].FaceUp && cards[len(cards)-1].Card.Rank == engine.Ace {
			return col, len(cards) - 1
		}
	}
	t.Fatal("model has no top tableau ace")
	return 0, 0
}

func oneMoveFromWin(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.New(legacyrand.New(legacyrand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	var foundations [4][]engine.Card
	for suit := engine.Spades; suit <= engine.Clubs; suit++ {
		top := engine.King
		if suit == engine.Spades {
			top = engine.Queen
		}
		for rank := engine.Ace; rank <= top; rank++ {
			foundations[suit] = append(foundations[suit], engine.Card{Rank: rank, Suit: suit})
		}
	}
	var tableau [7][]engine.TableauCard
	tableau[0] = []engine.TableauCard{{Card: engine.Card{Rank: engine.King, Suit: engine.Spades}, FaceUp: true}}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card(nil))
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card(nil))
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), tableau)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("foundations"), foundations)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("moves"), 0)
	return g
}

func setUnexported(field reflect.Value, value any) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}
