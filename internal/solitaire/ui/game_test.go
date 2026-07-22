package ui

import (
	legacyrand "math/rand"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wallymagill/terminal-games/internal/solitaire/engine"
	"github.com/wallymagill/terminal-games/internal/store"
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
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab", "backtab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
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

func testModelWithStore(t *testing.T, scores *store.Store) Model {
	t.Helper()
	m, err := NewModel(scores, legacyrand.New(legacyrand.NewSource(1)))
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

func TestActivateIgnoresEmptyWasteSelection(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionWaste}
	m, cmd := m.Update(key(" "))
	if cmd != nil {
		t.Fatal("empty waste selection should not return command")
	}
	if m.selected != nil {
		t.Fatalf("empty waste should not become selected: %+v", *m.selected)
	}
	_, cmd = m.Update(key("esc"))
	if cmd == nil {
		t.Fatal("esc after ignored empty selection should return to launcher")
	}
}

func TestActivateIgnoresEmptyFoundationSelection(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionFoundation, col: int(engine.Spades)}
	m, _ = m.Update(key(" "))
	if m.selected != nil {
		t.Fatalf("empty foundation should not become selected: %+v", *m.selected)
	}
}

func TestActivateIgnoresFaceDownTableauSelection(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionTableau, col: 6, row: 0}
	m, _ = m.Update(key(" "))
	if m.selected != nil {
		t.Fatalf("face-down tableau card should not become selected: %+v", *m.selected)
	}
}

func TestSpaceAppliesOnlyLegalDestinationWithoutManualTargeting(t *testing.T) {
	m := modelWithTopTableauAce(t, engine.Spades)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	if top, ok := m.game.FoundationTop(engine.Spades); !ok || top.Rank != engine.Ace {
		t.Fatalf("space should move sole legal ace destination to foundation, top=%+v ok=%v", top, ok)
	}
	if m.selected != nil {
		t.Fatalf("sole-destination smart move should not leave selection active: %+v", *m.selected)
	}
}

func TestSpaceKeepsSelectionWhenMultipleDestinationsExist(t *testing.T) {
	m := modelWithMovableKingAndTwoEmptyColumns(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	if m.selected == nil {
		t.Fatal("multiple destinations should keep source selected")
	}
	if len(m.selectedMoves) < 2 {
		t.Fatalf("expected multiple destination candidates, got %+v", m.selectedMoves)
	}
	if got, want := m.cursor.location(), m.selectedMoves[0].Destination; got != want {
		t.Fatalf("multiple destinations should highlight first candidate, got %+v want %+v", got, want)
	}
}

func TestDestinationCandidatesFocusNonEmptyTableauTopRows(t *testing.T) {
	m := modelWithMovableFourAndNonEmptyTableauDestinations(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}

	m, _ = m.Update(key(" "))
	if m.selected == nil {
		t.Fatal("source with multiple destinations should stay selected")
	}
	if len(m.selectedMoves) < 2 {
		t.Fatalf("expected multiple destination candidates, got %+v", m.selectedMoves)
	}

	firstPile := m.selectedMoves[0].Destination.Pile
	firstCards := m.game.Tableau(firstPile)
	firstCursor := cursor{region: regionTableau, col: firstPile, row: len(firstCards) - 1}
	if m.cursor != firstCursor {
		t.Errorf("initial destination focus = %+v, want top row %+v", m.cursor, firstCursor)
	}

	m, _ = m.Update(key("tab"))
	secondPile := m.selectedMoves[1].Destination.Pile
	secondCards := m.game.Tableau(secondPile)
	secondCursor := cursor{region: regionTableau, col: secondPile, row: len(secondCards) - 1}
	if m.cursor != secondCursor {
		t.Errorf("cycled destination focus = %+v, want top row %+v", m.cursor, secondCursor)
	}
}

func TestTabCyclesSelectedDestinationCandidates(t *testing.T) {
	m := modelWithMovableKingAndTwoEmptyColumns(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	m, _ = m.Update(key("tab"))
	if got, want := m.cursor.location(), m.selectedMoves[1].Destination; got != want {
		t.Fatalf("tab focused %+v, want next destination %+v", got, want)
	}
}

func TestEnterAppliesHighlightedDestinationCandidate(t *testing.T) {
	m := modelWithMovableKingAndTwoEmptyColumns(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	dst := m.cursor.location()
	m, cmd := m.Update(key("enter"))
	if cmd == nil || !m.running {
		t.Fatal("confirmed destination move should start timer")
	}
	if m.selected != nil {
		t.Fatalf("confirmed destination should clear selection: %+v", *m.selected)
	}
	cards := m.game.Tableau(dst.Pile)
	if len(cards) != 1 || cards[0].Card.Rank != engine.King {
		t.Fatalf("destination tableau = %+v, want moved king", cards)
	}
}

func TestEnterAppliesManualNonEmptyTableauDestination(t *testing.T) {
	m := modelWithMovableFourAndNonEmptyTableauDestinations(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m, _ = m.Update(key(" "))
	if m.selected == nil {
		t.Fatal("source with multiple destinations should stay selected")
	}

	dstPile := 1
	dstCards := m.game.Tableau(dstPile)
	m.cursor = cursor{region: regionTableau, col: dstPile, row: len(dstCards) - 1}
	m, cmd := m.Update(key("enter"))
	if cmd == nil || !m.running {
		t.Fatal("manual tableau destination move should start timer")
	}
	if m.selected != nil {
		t.Fatalf("manual tableau destination should clear selection after moving: %+v", *m.selected)
	}
	dstCards = m.game.Tableau(dstPile)
	if got := dstCards[len(dstCards)-1].Card; got != (engine.Card{Rank: engine.Four, Suit: engine.Spades}) {
		t.Fatalf("destination top = %+v, want moved four of spades", got)
	}
	if m.game.Moves() != 1 {
		t.Fatalf("moves = %d, want 1", m.game.Moves())
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
	m := modelWithTopTableauAce(t, engine.Spades)
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
	m := modelWithTopTableauAce(t, engine.Spades)
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

func TestSafeAutoFoundationMovesAllSafeCardsWithoutAssistedFlag(t *testing.T) {
	m := modelWithSafeFoundationCards(t)
	m, _ = m.Update(key("A"))
	top, ok := m.game.FoundationTop(engine.Spades)
	if !ok || top.Rank != engine.Two {
		t.Fatalf("A should move all currently safe spade cards to foundation, got %+v/%v", top, ok)
	}
	if m.assisted {
		t.Fatal("safe auto-foundation should not mark game assisted")
	}
}

func TestHintKeyRemoved(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("h"))
	if strings.Contains(m.message, "Hint") {
		t.Fatalf("h must no longer produce hints, got %q", m.message)
	}
	if m.assisted {
		t.Fatal("h must no longer mark the game assisted")
	}
}

func TestNoProgressWarningAfterStockPassWithoutProgress(t *testing.T) {
	m := modelWithOnlyStockCycle(t)
	for range 10 {
		m, _ = m.Update(key("d"))
	}
	if !strings.Contains(strings.ToLower(m.message), "no progress") {
		t.Fatalf("expected no-progress warning after stock pass, got %q", m.message)
	}
	if m.assisted {
		t.Fatal("no-progress warning should not mark assisted")
	}
}

func TestSafeAutoFoundationDoesNotMoveUnsafeHigherCard(t *testing.T) {
	m := modelWithUnsafeHeartFive(t)
	m, _ = m.Update(key("A"))
	if top, ok := m.game.FoundationTop(engine.Hearts); ok && top.Rank == engine.Five {
		t.Fatal("A should not move unsafe 5♥ to foundation")
	}
}

func TestSafeAutoFoundationUndoRestoresWholeBatch(t *testing.T) {
	m := modelWithSafeFoundationCards(t)
	m, _ = m.Update(key("A"))
	if top, ok := m.game.FoundationTop(engine.Spades); !ok || top.Rank != engine.Two {
		t.Fatalf("A should move through 2♠ before undo, got %+v/%v", top, ok)
	}
	m, _ = m.Update(key("u"))
	if _, ok := m.game.FoundationTop(engine.Spades); ok {
		t.Fatal("one undo should restore the whole safe auto-foundation batch")
	}
	if m.game.Moves() != 0 {
		t.Fatalf("undo should restore move count to 0, got %d", m.game.Moves())
	}
}

func TestAutoFoundationKeepsSelectionWhenSelectedCardIsUnsafe(t *testing.T) {
	m := modelWithUnsafeHeartFive(t)
	m.selected = &engine.Location{Kind: engine.Tableau, Pile: 0, Index: 0}
	m.cursor = cursor{region: regionStock}
	m, cmd := m.Update(key("a"))
	if cmd != nil {
		t.Fatal("failed safe auto-foundation should not start timer")
	}
	if m.selected == nil {
		t.Fatal("failed safe auto-foundation should keep selection")
	}
	if top, ok := m.game.FoundationTop(engine.Hearts); ok && top.Rank == engine.Five {
		t.Fatal("a should not move unsafe 5♥ to foundation")
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

func TestUndoRestoresPreviousGameAndMarksAssisted(t *testing.T) {
	m := testModel(t)
	beforeStock := m.game.StockCount()
	m, _ = m.Update(key("d"))
	if m.game.StockCount() == beforeStock {
		t.Fatal("draw should change stock before undo")
	}
	m, _ = m.Update(key("u"))
	if m.game.StockCount() != beforeStock {
		t.Fatalf("undo should restore stock count = %d, got %d", beforeStock, m.game.StockCount())
	}
	if !m.assisted {
		t.Fatal("undo should mark game assisted")
	}
	if m.assistReason != "undo" {
		t.Fatalf("assist reason = %q, want undo", m.assistReason)
	}
	if m.game.Moves() != 0 {
		t.Fatalf("undo should not add another move, got %d", m.game.Moves())
	}
}

func TestAssistedWinRecordsToAssistedBucket(t *testing.T) {
	scores := store.New(t.TempDir() + "/scores.json")
	m := oneMoveFromWinModel(t, scores)
	m.assisted = true
	m, _ = m.Update(key("a"))
	record := scores.LoadSolitaireDraw3(store.SolitaireAssisted)
	if record.BestSeconds == nil || record.FewestMoves == nil {
		t.Fatalf("assisted win should write assisted record, got %+v", record)
	}
	normal := scores.LoadSolitaireDraw3(store.SolitaireNormal)
	if normal.BestSeconds != nil || normal.FewestMoves != nil {
		t.Fatalf("assisted win should not write normal record, got %+v", normal)
	}
}

func TestUndoAfterWinAllowsAssistedRecordOnReplay(t *testing.T) {
	scores := store.New(t.TempDir() + "/scores.json")
	m := oneMoveFromWinModel(t, scores)
	m, _ = m.Update(key("a"))
	if normal := scores.LoadSolitaireDraw3(store.SolitaireNormal); normal.BestSeconds == nil {
		t.Fatalf("initial win should write normal record, got %+v", normal)
	}

	m, _ = m.Update(key("u"))
	if !m.assisted {
		t.Fatal("undoing a win should mark the replay assisted")
	}
	if m.game.Status() != engine.Playing {
		t.Fatalf("undoing a win should restore a playing game, got %v", m.game.Status())
	}

	m, _ = m.Update(key("a"))
	record := scores.LoadSolitaireDraw3(store.SolitaireAssisted)
	if record.BestSeconds == nil || record.FewestMoves == nil {
		t.Fatalf("assisted replay should write assisted record, got %+v", record)
	}
}

func TestUndoAfterStoppedWinRestartsTimerForRestoredPlayingGame(t *testing.T) {
	m := oneMoveFromWinModel(t, store.New(t.TempDir()+"/scores.json"))
	m.running = true

	m, _ = m.Update(key("a"))
	if m.game.Status() != engine.Won {
		t.Fatalf("status = %v, want won", m.game.Status())
	}

	m, _ = m.Update(tickMsg{session: m.tickSession, gen: m.tickGen})
	if m.running {
		t.Fatal("winning tick should stop the timer")
	}

	m, cmd := m.Update(key("u"))
	if m.game.Status() != engine.Playing {
		t.Fatalf("undoing a win should restore a playing game, got %v", m.game.Status())
	}
	if !m.running {
		t.Fatal("undoing a win should restart timer for restored playing game")
	}
	if cmd == nil {
		t.Fatal("undoing a win should schedule a timer tick for restored playing game")
	}
}

func oneMoveFromWinModel(t *testing.T, scores *store.Store) Model {
	t.Helper()
	m := testModelWithStore(t, scores)
	m.game = oneMoveFromWin(t)
	m.cursor = cursor{region: regionTableau, col: 0, row: 0}
	m.elapsed = 12
	return m
}

func modelWithTopTableauAce(t *testing.T, suit engine.Suit) Model {
	t.Helper()
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.Ace, Suit: suit}, FaceUp: true}},
	})
	// A drawable ace keeps the position live after the tested move, so
	// dead-end detection never fires inside these fixtures.
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card{{Rank: engine.Ace, Suit: engine.Diamonds}})
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card(nil))
	m.game = g
	return m
}

func modelWithMovableKingAndTwoEmptyColumns(t *testing.T) Model {
	t.Helper()
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.King, Suit: engine.Spades}, FaceUp: true}},
		nil,
		nil,
	})
	// Drawable ace: see modelWithTopTableauAce.
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card{{Rank: engine.Ace, Suit: engine.Diamonds}})
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card(nil))
	m.game = g
	return m
}

func modelWithMovableFourAndNonEmptyTableauDestinations(t *testing.T) Model {
	t.Helper()
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.Four, Suit: engine.Spades}, FaceUp: true}},
		{
			{Card: engine.Card{Rank: engine.Six, Suit: engine.Clubs}, FaceUp: true},
			{Card: engine.Card{Rank: engine.Five, Suit: engine.Hearts}, FaceUp: true},
		},
		{
			{Card: engine.Card{Rank: engine.Six, Suit: engine.Clubs}, FaceUp: true},
			{Card: engine.Card{Rank: engine.Five, Suit: engine.Diamonds}, FaceUp: true},
		},
	})
	// Drawable ace: see modelWithTopTableauAce.
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card{{Rank: engine.Ace, Suit: engine.Diamonds}})
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card(nil))
	m.game = g
	return m
}

func modelWithSafeFoundationCards(t *testing.T) Model {
	t.Helper()
	m := testModel(t)
	g := &engine.Game{}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.Ace, Suit: engine.Spades}, FaceUp: true}},
		{{Card: engine.Card{Rank: engine.Two, Suit: engine.Spades}, FaceUp: true}},
	})
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card(nil))
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card(nil))
	m.game = g
	return m
}

func modelWithUnsafeHeartFive(t *testing.T) Model {
	t.Helper()
	m := testModel(t)
	g := &engine.Game{}
	foundations := [4][]engine.Card{
		engine.Hearts: {
			{Rank: engine.Ace, Suit: engine.Hearts},
			{Rank: engine.Two, Suit: engine.Hearts},
			{Rank: engine.Three, Suit: engine.Hearts},
			{Rank: engine.Four, Suit: engine.Hearts},
		},
	}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("tableau"), [7][]engine.TableauCard{
		{{Card: engine.Card{Rank: engine.Five, Suit: engine.Hearts}, FaceUp: true}},
	})
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("foundations"), foundations)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card(nil))
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("waste"), []engine.Card(nil))
	m.game = g
	return m
}

func modelWithOnlyStockCycle(t *testing.T) Model {
	t.Helper()
	m := testModel(t)
	g := &engine.Game{}
	var stock []engine.Card
	for range 24 {
		stock = append(stock, engine.Card{Rank: engine.King, Suit: engine.Spades})
	}
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("status"), engine.Playing)
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), stock)
	m.game = g
	return m
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

func TestTabCyclesPlayableSourcesAndSkipsFaceDownCards(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionTableau, col: 6, row: 0}
	before := m.cursor
	m, _ = m.Update(key("tab"))
	if m.cursor == before {
		t.Fatalf("tab should move to a playable source, cursor remained %+v", m.cursor)
	}
	loc := m.cursor.location()
	if loc.Kind == engine.Tableau {
		cards := m.game.Tableau(loc.Pile)
		if loc.Index < 0 || loc.Index >= len(cards) || !cards[loc.Index].FaceUp {
			t.Fatalf("tab focused non-playable tableau location: cursor=%+v cards=%+v", m.cursor, cards)
		}
	}
}

func TestBacktabCyclesPlayableSourcesBackward(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("tab"))
	first := m.cursor
	m, _ = m.Update(key("shift+tab"))
	if m.cursor == first {
		t.Fatalf("shift+tab should cycle backward from first playable source, cursor remained %+v", m.cursor)
	}
	loc := m.cursor.location()
	if loc.Kind == engine.Tableau {
		cards := m.game.Tableau(loc.Pile)
		if loc.Index < 0 || loc.Index >= len(cards) || !cards[loc.Index].FaceUp {
			t.Fatalf("shift+tab focused non-playable tableau location: cursor=%+v cards=%+v", m.cursor, cards)
		}
	}
}

func TestNumberKeysJumpToTableauColumns(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("4"))
	if m.cursor.region != regionTableau || m.cursor.col != 3 {
		t.Fatalf("4 should jump to tableau column 4, cursor=%+v", m.cursor)
	}
}

func setUnexported(field reflect.Value, value any) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func TestHMovesCursorLeft(t *testing.T) {
	m := testModel(t)
	m.cursor = cursor{region: regionTableau, col: 3}
	m, _ = m.Update(key("h"))
	if m.cursor.region != regionTableau || m.cursor.col != 2 {
		t.Fatalf("h from tableau col 3 = %+v, want col 2", m.cursor)
	}
}

func TestJumpKeysReachRegions(t *testing.T) {
	m := testModel(t)
	m, _ = m.Update(key("w"))
	if m.cursor.region != regionWaste {
		t.Fatalf("w = %+v, want draw pile", m.cursor)
	}
	m, _ = m.Update(key("s"))
	if m.cursor.region != regionStock {
		t.Fatalf("s = %+v, want deck", m.cursor)
	}
	m, _ = m.Update(key("f"))
	if m.cursor.region != regionFoundation || m.cursor.col != 0 {
		t.Fatalf("f = %+v, want foundation 0", m.cursor)
	}
	m, _ = m.Update(key("f"))
	if m.cursor.col != 1 {
		t.Fatalf("second f = %+v, want foundation 1", m.cursor)
	}
	for range 3 {
		m, _ = m.Update(key("f"))
	}
	if m.cursor.col != 0 {
		t.Fatalf("f must wrap 4→1, got %+v", m.cursor)
	}
}

func TestVerticalNavigationSkipsFaceDownCards(t *testing.T) {
	m := testModel(t)
	// Column 3 deals rows 0-2 face-down, row 3 face-up.
	m.cursor = cursor{region: regionTableau, col: 3, row: 3}
	m, _ = m.Update(key("k"))
	if m.cursor.region == regionTableau && m.cursor.row < 3 && !m.game.Tableau(3)[m.cursor.row].FaceUp {
		t.Fatalf("up landed on face-down card: %+v", m.cursor)
	}
	if m.cursor.region != regionFoundation {
		t.Fatalf("up from only face-up row must exit to foundations, got %+v", m.cursor)
	}
	m, _ = m.Update(key("j"))
	if m.cursor.region != regionTableau || !m.game.Tableau(m.cursor.col)[m.cursor.row].FaceUp {
		t.Fatalf("down must land on a face-up card, got %+v", m.cursor)
	}
}

func TestSelectedDestinationsSortedByPriority(t *testing.T) {
	// RevealsCard is a source-level property, so all moves share it; the sort
	// must still order safe foundation > tableau > unsafe foundation.
	moves := []engine.Move{
		{Destination: engine.Location{Kind: engine.Foundation, Pile: 0}, RevealsCard: true},
		{Destination: engine.Location{Kind: engine.Tableau, Pile: 2}, RevealsCard: true},
		{Destination: engine.Location{Kind: engine.Foundation, Pile: 1}, Safe: true, RevealsCard: true},
	}
	sorted := sortMovesByPriority(moves)
	if !(sorted[0].Safe && sorted[0].Destination.Kind == engine.Foundation) {
		t.Fatalf("safe foundation must sort first: %+v", sorted[0])
	}
	if sorted[1].Destination.Kind != engine.Tableau {
		t.Fatalf("tableau must sort before unsafe foundation: %+v", sorted[1])
	}
	if len(sorted) != 3 || len(moves) != 3 {
		t.Fatal("sort must not add or drop moves")
	}
}

// forceDeadEnd rewrites the model's game into a two-queen position with no
// deck, draw pile, kings, or aces: no legal moves at all.
func forceDeadEnd(t *testing.T, m Model) Model {
	t.Helper()
	rv := reflect.ValueOf(m.game).Elem()
	setUnexported(rv.FieldByName("stock"), []engine.Card(nil))
	setUnexported(rv.FieldByName("waste"), []engine.Card(nil))
	var tableau [7][]engine.TableauCard
	tableau[0] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Spades, Rank: engine.Queen}, FaceUp: true}}
	tableau[1] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Hearts, Rank: engine.Queen}, FaceUp: true}}
	setUnexported(rv.FieldByName("tableau"), tableau)
	if !m.game.DeadEnd() {
		t.Fatal("constructed position must be a dead end")
	}
	return m
}

func TestDeadEndShowsGameOverScreen(t *testing.T) {
	m := forceDeadEnd(t, testModel(t))
	m, _ = m.Update(key("d")) // any action recomputes dead-end state
	if !m.gameOver {
		t.Fatal("gameOver must be set after action on dead board")
	}
	if m.running {
		t.Fatal("timer must stop on game over")
	}
	v := m.View()
	for _, want := range []string{"G A M E   O V E R", "no moves left", "n new game", "u undo"} {
		if !strings.Contains(v, want) {
			t.Fatalf("game-over view missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "Tableau") {
		t.Fatalf("board must be hidden on game over:\n%s", v)
	}
}

func TestGameOverGatesKeys(t *testing.T) {
	m := forceDeadEnd(t, testModel(t))
	m, _ = m.Update(key("d"))
	before := m.cursor
	m, _ = m.Update(key("1"))
	m, _ = m.Update(key("j"))
	if m.cursor != before {
		t.Fatal("cursor keys must be inert on game over")
	}
	_, cmd := m.Update(key("m"))
	if cmd == nil {
		t.Fatal("m must still return to launcher")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("m must produce BackMsg")
	}
}

func TestGameOverUndoReturnsToBoard(t *testing.T) {
	m := testModel(t)
	m.undoStack = append(m.undoStack, m.game.Clone()) // live position to restore
	m = forceDeadEnd(t, m)
	m, _ = m.Update(key("d"))
	if !m.gameOver {
		t.Fatal("expected game over")
	}
	m, _ = m.Update(key("u"))
	if m.gameOver {
		t.Fatal("undo into a live position must clear gameOver")
	}
	if !m.assisted {
		t.Fatal("undo must still mark the run assisted")
	}
	if !strings.Contains(m.View(), "Tableau") {
		t.Fatal("board must render again after undo")
	}
}

func TestGameOverNewDealResets(t *testing.T) {
	m := forceDeadEnd(t, testModel(t))
	m, _ = m.Update(key("d"))
	m, _ = m.Update(key("n"))
	if m.gameOver {
		t.Fatal("new deal must clear gameOver")
	}
	if m.game.Moves() != 0 {
		t.Fatalf("new deal must reset moves, got %d", m.game.Moves())
	}
}

// movesModel builds a position with exactly two productive sources:
// A♠ on tableau 0 (foundation move) and the draw-pile 8♠ (plays on 9♦).
func movesModel(t *testing.T) Model {
	t.Helper()
	m := testModel(t)
	rv := reflect.ValueOf(m.game).Elem()
	setUnexported(rv.FieldByName("stock"), []engine.Card(nil))
	setUnexported(rv.FieldByName("waste"), []engine.Card{{Suit: engine.Spades, Rank: engine.Eight}})
	var tableau [7][]engine.TableauCard
	tableau[0] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Spades, Rank: engine.Ace}, FaceUp: true}}
	tableau[1] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Diamonds, Rank: engine.Nine}, FaceUp: true}}
	setUnexported(rv.FieldByName("tableau"), tableau)
	return m
}

func TestShowMovesHighlightsProductiveSources(t *testing.T) {
	m := movesModel(t)
	m, _ = m.Update(key("v"))
	if !m.showMoves {
		t.Fatal("v must enable showMoves")
	}
	if !m.assisted || m.assistReason != "moves" {
		t.Fatalf("v must mark assisted/moves, got %v/%q", m.assisted, m.assistReason)
	}
	wantSources := []engine.Location{
		{Kind: engine.Tableau, Pile: 0, Index: 0},
		{Kind: engine.Waste},
	}
	for _, want := range wantSources {
		if !m.moveSources[want] {
			t.Fatalf("source %+v missing from moveSources %v", want, m.moveSources)
		}
	}
	if m.moveSources[engine.Location{Kind: engine.Tableau, Pile: 1, Index: 0}] {
		t.Fatal("9♦ has no productive move and must not highlight")
	}
	if !strings.Contains(m.message, "moves · any key to dismiss") {
		t.Fatalf("status = %q", m.message)
	}
}

func TestShowMovesKeepDrawingMessage(t *testing.T) {
	m := testModel(t)
	rv := reflect.ValueOf(m.game).Elem()
	setUnexported(rv.FieldByName("waste"), []engine.Card(nil))
	setUnexported(rv.FieldByName("stock"), []engine.Card{
		{Suit: engine.Clubs, Rank: engine.Five},
		{Suit: engine.Clubs, Rank: engine.Ace},
	})
	var tableau [7][]engine.TableauCard
	tableau[0] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Hearts, Rank: engine.Nine}, FaceUp: true}}
	setUnexported(rv.FieldByName("tableau"), tableau)
	if m.game.DeadEnd() {
		t.Fatal("fixture must not be a dead end (ace surfaces on draw)")
	}
	m, _ = m.Update(key("v"))
	if m.showMoves {
		t.Fatal("no board moves: must not enable highlights")
	}
	if m.message != "no moves on board — keep drawing" {
		t.Fatalf("status = %q", m.message)
	}
	if !m.assisted || m.assistReason != "moves" {
		t.Fatal("v marks assisted even in keep-drawing case")
	}
}

func TestShowMovesDismissedByNextKeyWhichStillActs(t *testing.T) {
	m := movesModel(t)
	m, _ = m.Update(key("v"))
	if !m.showMoves {
		t.Fatal("expected showMoves on")
	}
	m.cursor = cursor{region: regionTableau, col: 1}
	m, _ = m.Update(key("h"))
	if m.showMoves || m.moveSources != nil {
		t.Fatal("any key must clear highlights")
	}
	if m.cursor.col != 0 {
		t.Fatalf("the dismissing key must still act (h moves left), cursor %+v", m.cursor)
	}
}

func TestShowMovesInertOnGameOver(t *testing.T) {
	m := forceDeadEnd(t, testModel(t))
	m, _ = m.Update(key("d"))
	if !m.gameOver {
		t.Fatal("expected game over")
	}
	m, _ = m.Update(key("v"))
	if m.showMoves {
		t.Fatal("v must be inert on game over")
	}
}

func TestCursorReclampsAfterMoveEmptiesPile(t *testing.T) {
	m := testModel(t)
	rv := reflect.ValueOf(m.game).Elem()
	// Ace alone on pile 2; drawable card keeps the game live after the move.
	setUnexported(rv.FieldByName("waste"), []engine.Card(nil))
	setUnexported(rv.FieldByName("stock"), []engine.Card{{Suit: engine.Diamonds, Rank: engine.Ace}})
	var tableau [7][]engine.TableauCard
	tableau[2] = []engine.TableauCard{{Card: engine.Card{Suit: engine.Spades, Rank: engine.Ace}, FaceUp: true}}
	tableau[3] = []engine.TableauCard{
		{Card: engine.Card{Suit: engine.Hearts, Rank: engine.Three}, FaceUp: false},
		{Card: engine.Card{Suit: engine.Hearts, Rank: engine.Two}, FaceUp: true},
	}
	setUnexported(rv.FieldByName("tableau"), tableau)
	m.cursor = cursor{region: regionTableau, col: 2, row: 0}
	m, _ = m.Update(key("a")) // safe auto-foundation moves the ace, emptying pile 2
	if got := m.game.TableauLen(2); got != 0 {
		t.Fatalf("expected pile 2 emptied, len %d", got)
	}
	if m.cursor.region != regionTableau || m.cursor.col != 2 || m.cursor.row != 0 {
		t.Fatalf("cursor must re-clamp to the empty pile slot, got %+v", m.cursor)
	}
	// The very next frame must show the cursor: placeholder slot exists at col 2.
	if !strings.Contains(m.tableauView(), "[  ]") {
		t.Fatal("post-move view must contain the empty-column slot the cursor sits on")
	}
}
