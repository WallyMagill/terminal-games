package ui

import (
	legacyrand "math/rand"
	"reflect"
	"strings"
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
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card(nil))
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
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card(nil))
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
	setUnexported(reflect.ValueOf(g).Elem().FieldByName("stock"), []engine.Card(nil))
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
