package main

import (
	"math/rand/v2"
	"testing"
)

func TestBankCountsAndSampleSolvable(t *testing.T) {
	if err := loadPuzzleBank(); err != nil {
		t.Fatal(err)
	}
	for _, d := range difficultyOrder {
		list := puzzlesFor(d)
		want := bankSize[d]
		if len(list) != want {
			t.Errorf("%s: got %d puzzles, want %d", d, len(list), want)
		}
		seen := map[string]struct{}{}
		for i, p := range list {
			if p.ID == "" || len(p.Givens) != 81 {
				t.Errorf("%s: malformed entry %+v", d, p)
				continue
			}
			if _, dup := seen[p.ID]; dup {
				t.Errorf("%s: duplicate id %s", d, p.ID)
			}
			seen[p.ID] = struct{}{}
			if p.Solution != "" {
				t.Errorf("%s %s: bank must not bake a solution", d, p.ID)
			}
			if i < 5 {
				if got := solveSudoku(p.Givens); len(got) != 81 {
					t.Errorf("%s %s: unsolvable givens", d, p.ID)
				}
			}
		}
	}
}

func TestIncompletePoolSkipsCompleted(t *testing.T) {
	if err := loadPuzzleBank(); err != nil {
		t.Fatal(err)
	}
	all := puzzlesFor(diffEasy)
	done := map[string]struct{}{all[0].ID: {}, all[1].ID: {}}
	pool := incompletePool(diffEasy, done)
	if len(pool) != len(all)-2 {
		t.Fatalf("pool %d want %d", len(pool), len(all)-2)
	}
	if remainingCount(diffEasy, done) != len(all)-2 {
		t.Fatalf("remaining %d want %d", remainingCount(diffEasy, done), len(all)-2)
	}
	rng := rand.New(rand.NewPCG(1, 2))
	p, ok := pickIncomplete(diffEasy, done, rng)
	if !ok || p.ID == all[0].ID || p.ID == all[1].ID {
		t.Fatalf("pickIncomplete got %+v ok=%v", p, ok)
	}
	for _, p := range pool {
		if p.ID == all[0].ID || p.ID == all[1].ID {
			t.Fatalf("completed id %s still in pool", p.ID)
		}
	}
}

func TestBoardMistakeAndComplete(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	// First empty cell in this puzzle is index 2 (0-based), given 530...
	b.cursor = 2
	wrong := byte('1')
	if sol[2] == '1' {
		wrong = '2'
	}
	if !b.place(wrong) {
		t.Fatal("place wrong digit")
	}
	if !b.isWrong(2) || b.mistakes != 1 {
		t.Fatalf("wrong=%v mistakes=%d", b.isWrong(2), b.mistakes)
	}
	if !b.place(sol[2]) {
		t.Fatal("place correct digit")
	}
	if b.isWrong(2) {
		t.Fatal("corrected cell still marked wrong")
	}
	if b.mistakes != 1 {
		t.Fatalf("correcting should not add a mistake, got %d", b.mistakes)
	}
	if b.place(wrong) {
		t.Fatal("correct cell should be locked against overwrite")
	}
	if b.clear() {
		t.Fatal("correct cell should be locked against clear")
	}
	if b.grid[2] != sol[2] {
		t.Fatal("locked correct cell was changed")
	}
	b.cursor = 0 // given 5
	if b.place('1') || b.clear() {
		t.Fatal("given cell should be locked")
	}
	if b.grid[0] != '5' {
		t.Fatal("given cell was changed")
	}
	// Fill remaining empties with the solution.
	for i := 0; i < 81; i++ {
		if b.givens[i] == '0' {
			b.cursor = i
			b.grid[i] = sol[i]
		}
	}
	if !b.isComplete() {
		t.Fatal("board should be complete")
	}
}

func TestCompletedDigits(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	done := b.completedDigits()
	for d := 1; d <= 9; d++ {
		if done[d] {
			t.Errorf("digit %d should not be complete from givens alone", d)
		}
	}
	for i := 0; i < 81; i++ {
		if sol[i] == '5' {
			b.grid[i] = '5'
		}
	}
	b.recountCorrect()
	done = b.completedDigits()
	if !done[5] {
		t.Fatal("all correct 5s should mark digit 5 complete")
	}
	if done[3] {
		t.Fatal("unfinished digit 3 should not be complete")
	}
}

func TestActiveDigitsOmitsCompleted(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	if got := string(b.activeDigits()); got != "123456789" {
		t.Fatalf("all active: got %q", got)
	}
	for _, d := range []byte{'1', '2', '4', '5', '7', '9'} {
		for i := 0; i < 81; i++ {
			if sol[i] == d {
				b.grid[i] = d
			}
		}
	}
	b.recountCorrect()
	if got := string(b.activeDigits()); got != "368" {
		t.Fatalf("remaining active: got %q want 368", got)
	}
}

func TestEnsureSolvedFillsFromGivens(t *testing.T) {
	p := puzzleEntry{
		ID:     "wiki",
		Givens: "530070000600195000098000060800060003400803001700020006060000280000419005000080079",
	}
	if !p.ensureSolved() {
		t.Fatal("ensureSolved failed")
	}
	want := "534678912672195348198342567859761423426853791713924856961537284287419635345286179"
	if p.Solution != want {
		t.Fatalf("got %q want %q", p.Solution, want)
	}
	if !p.ensureSolved() || p.Solution != want {
		t.Fatal("second ensureSolved should keep the same solution")
	}
}

func TestCursorWrapsToroidally(t *testing.T) {
	var b board
	b.cursor = 0 // top-left
	b.move(-1, 0)
	if b.cursor != 8 {
		t.Fatalf("left wrap: got %d want 8", b.cursor)
	}
	b.move(1, 0)
	if b.cursor != 0 {
		t.Fatalf("right from wrapped: got %d want 0", b.cursor)
	}
	b.move(0, -1)
	if b.cursor != 72 {
		t.Fatalf("up wrap: got %d want 72", b.cursor)
	}
	b.move(0, 1)
	if b.cursor != 0 {
		t.Fatalf("down from wrapped: got %d want 0", b.cursor)
	}
	b.cursor = 8
	b.move(1, 0)
	if b.cursor != 0 {
		t.Fatalf("right wrap: got %d want 0", b.cursor)
	}
	b.cursor = 72
	b.move(0, 1)
	if b.cursor != 0 {
		t.Fatalf("down wrap: got %d want 0", b.cursor)
	}
}

func TestSelectionMarkPos(t *testing.T) {
	ox, oy := 4, 2
	digitX, midY := selectionMarkPos(ox, oy, 0, 0)
	if digitX != ox+2 || midY != oy+1 {
		t.Fatalf("top-left marks: digitX=%d midY=%d", digitX, midY)
	}
	digitX, midY = selectionMarkPos(ox, oy, 8, 8)
	if digitX != ox+8*4+2 || midY != oy+8*2+1 {
		t.Fatalf("bottom-right marks: digitX=%d midY=%d", digitX, midY)
	}
}

func TestPencilMarksCycleAndClear(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	b.cursor = 2
	if !b.markPencil('3') || b.pencil[2][0] != '3' || b.pencil[2][1] != 0 {
		t.Fatalf("first mark top: %+v", b.pencil[2])
	}
	if !b.markPencil('9') || b.pencil[2][1] != '9' {
		t.Fatalf("second mark bottom: %+v", b.pencil[2])
	}
	if !b.markPencil('1') || b.pencil[2][0] != '1' || b.pencil[2][1] != '9' {
		t.Fatalf("third overwrites top: %+v", b.pencil[2])
	}
	if !b.markPencil('6') || b.pencil[2][1] != '6' || b.pencil[2][0] != '1' {
		t.Fatalf("fourth overwrites bottom: %+v", b.pencil[2])
	}
	if !b.markPencil('1') || b.pencil[2][0] != 0 || b.pencil[2][1] != '6' || b.pencilSlot[2] != 0 {
		t.Fatalf("toggle 1 should free top: %+v slot %d", b.pencil[2], b.pencilSlot[2])
	}
	if !b.markPencil('6') || b.pencil[2][0] != 0 || b.pencil[2][1] != 0 || b.pencilSlot[2] != 0 {
		t.Fatalf("toggle last mark should clear cell: %+v slot %d", b.pencil[2], b.pencilSlot[2])
	}
	if !b.markPencil('3') || !b.markPencil('9') {
		t.Fatal("restore two marks")
	}
	if !b.clearPencil() || b.pencil[2][0] != 0 || b.pencil[2][1] != 0 {
		t.Fatal("0 should clear both marks")
	}
	if b.markPencil('4') && b.place('4') {
		if b.hasPencil(2) {
			t.Fatal("placing a digit should clear pencil marks")
		}
	}
}

func TestPencilToggleFreesThatSlot(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	b.cursor = 2
	if !b.markPencil('3') || !b.markPencil('9') {
		t.Fatal("setup top 3 / bottom 9")
	}
	if !b.markPencil('9') || b.pencil[2][0] != '3' || b.pencil[2][1] != 0 || b.pencilSlot[2] != 1 {
		t.Fatalf("toggle bottom should free that slot: %+v slot %d", b.pencil[2], b.pencilSlot[2])
	}
	if !b.markPencil('7') || b.pencil[2][0] != '3' || b.pencil[2][1] != '7' {
		t.Fatalf("next fill should be the freed bottom: %+v", b.pencil[2])
	}
}

func TestPencilSaveRoundTrip(t *testing.T) {
	var b board
	b.cursor = 4
	b.markPencil('2')
	b.markPencil('8')
	top, bot, slot := b.pencilsString()
	var b2 board
	b2.loadPencils(top, bot, slot)
	if b2.pencil[4][0] != '2' || b2.pencil[4][1] != '8' || b2.pencilSlot[4] != 0 {
		t.Fatalf("round trip %+v slot %d", b2.pencil[4], b2.pencilSlot[4])
	}
}

func TestPencilRejectsCompletedDigit(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	for i := 0; i < 81; i++ {
		if sol[i] == '5' {
			b.grid[i] = '5'
		}
	}
	b.recountCorrect()
	b.cursor = 2
	if b.markPencil('5') {
		t.Fatal("should ignore pencil mark for a completed (white) digit")
	}
	if !b.markPencil('4') {
		t.Fatal("unfinished digit should still mark")
	}
}

func TestPlaceRejectsCompletedDigit(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	for i := 0; i < 81; i++ {
		if sol[i] == '5' {
			b.grid[i] = '5'
		}
	}
	b.recountCorrect()
	b.cursor = 2
	if b.place('5') {
		t.Fatal("should ignore pen entry for a completed digit")
	}
	if b.grid[2] != '0' || b.mistakes != 0 {
		t.Fatal("completed digit must not place or count as a mistake")
	}
	if !b.place(sol[2]) {
		t.Fatal("unfinished digit should still place")
	}
}

func TestStripCompletedPencilKeepsTop(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	b.cursor = 2
	if !b.markPencil('4') || !b.markPencil('5') {
		t.Fatal("setup 4 top / 5 bottom")
	}
	only := -1
	for i := 0; i < 81; i++ {
		if emptyCell(b.grid[i]) && i != 2 {
			only = i
			break
		}
	}
	if only < 0 {
		t.Fatal("need a second empty cell")
	}
	b.cursor = only
	if !b.markPencil('5') {
		t.Fatal("setup lone 5 mark")
	}
	var last int
	for i := 0; i < 81; i++ {
		if sol[i] == '5' && b.grid[i] != '5' {
			last = i
		}
	}
	for i := 0; i < 81; i++ {
		if sol[i] == '5' && i != last {
			b.grid[i] = '5'
		}
	}
	b.recountCorrect()
	b.cursor = last
	if !b.place('5') {
		t.Fatal("place last 5")
	}
	if !b.completedDigits()[5] {
		t.Fatal("5 should be complete")
	}
	if b.pencil[2][0] != '4' || b.pencil[2][1] != 0 || b.pencilSlot[2] != 1 {
		t.Fatalf("remaining mark should stay on top, next fill bottom: %+v slot %d", b.pencil[2], b.pencilSlot[2])
	}
	if b.pencil[only][0] != 0 || b.pencil[only][1] != 0 {
		t.Fatalf("lone completed mark should be removed: %+v", b.pencil[only])
	}
	b.cursor = 2
	if !b.markPencil('3') || b.pencil[2][0] != '4' || b.pencil[2][1] != '3' {
		t.Fatalf("next fill should be the emptied bottom: %+v", b.pencil[2])
	}
}

func TestStripPencilPeersOnPlace(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	rowPeer, colPeer, boxPeer, distant := 5, 38, 10, 72
	b.cursor = rowPeer
	if !b.markPencil('4') || !b.markPencil('2') {
		t.Fatal("row peer 4 top / 2 bottom")
	}
	b.cursor = colPeer
	if !b.markPencil('4') {
		t.Fatal("col peer lone 4")
	}
	b.cursor = boxPeer
	if !b.markPencil('4') {
		t.Fatal("box peer lone 4")
	}
	b.cursor = distant
	if !b.markPencil('4') || !b.markPencil('1') {
		t.Fatal("distant 4 top / 1 bottom")
	}
	b.cursor = 2
	if !b.place('4') {
		t.Fatal("place 4")
	}
	if b.pencil[rowPeer][0] != 0 || b.pencil[rowPeer][1] != '2' || b.pencilSlot[rowPeer] != 0 {
		t.Fatalf("row peer leftover should be background: %+v slot %d", b.pencil[rowPeer], b.pencilSlot[rowPeer])
	}
	if b.pencil[colPeer][0] != 0 || b.pencil[colPeer][1] != 0 {
		t.Fatalf("col peer lone 4 should be cleared: %+v", b.pencil[colPeer])
	}
	if b.pencil[boxPeer][0] != 0 || b.pencil[boxPeer][1] != 0 {
		t.Fatalf("box peer lone 4 should be cleared: %+v", b.pencil[boxPeer])
	}
	if b.pencil[distant][0] != '4' || b.pencil[distant][1] != '1' {
		t.Fatalf("non-peer marks should stay: %+v", b.pencil[distant])
	}
}

func TestWrongPlaceDoesNotStripPeers(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	b.cursor = 5
	if !b.markPencil('1') {
		t.Fatal("mark 1 in row")
	}
	b.cursor = 2
	if !b.place('1') {
		t.Fatal("wrong 1")
	}
	if b.pencil[5][0] != '1' {
		t.Fatalf("wrong place should not strip peers: %+v", b.pencil[5])
	}
}

func TestStripImpossiblePencilsFromGivens(t *testing.T) {
	givens := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol := solveSudoku(givens)
	b := newBoard(givens, sol, givens)
	b.cursor = 2
	if !b.markPencil('5') || !b.markPencil('4') {
		t.Fatal("mark 5 (given in row) and 4")
	}
	b.stripImpossiblePencils()
	if b.pencil[2][0] != 0 || b.pencil[2][1] != '4' || b.pencilSlot[2] != 0 {
		t.Fatalf("given 5 should strip row marks: %+v slot %d", b.pencil[2], b.pencilSlot[2])
	}
}
