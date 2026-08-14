package main

import "testing"

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
