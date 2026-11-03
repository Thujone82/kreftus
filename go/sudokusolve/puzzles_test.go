package main

import "testing"

func TestPuzzleBank(t *testing.T) {
	for d := dieLevel(0); d < dieLevelCount; d++ {
		bank := puzzleBank[d]
		if len(bank) != 250 {
			t.Fatalf("%s has %d puzzles", d.name(), len(bank))
		}
		board, _, ok := boardFromGivens(bank[0])
		if !ok || !validateInitialBoard(&board) {
			t.Fatalf("%s sample is not a legal board", d.name())
		}
		if !solveSudoku(&board) {
			t.Fatalf("%s sample did not solve", d.name())
		}
	}
	if dieEasy.next() != dieMedium || dieMedium.next() != dieHard || dieHard.next() != dieDiabolical || dieDiabolical.next() != dieEasy {
		t.Fatal("difficulty order")
	}
}

func TestDieHelpText(t *testing.T) {
	expectedLines := map[dieLevel][2]string{
		dieEasy:       {"Click for Easy Puzzle", "Right Click Change Difficulty"},
		dieMedium:     {"Click for Medium Puzzle", "Right Click Change Difficulty"},
		dieHard:       {"Click for Hard Puzzle", "Right Click Change Difficulty"},
		dieDiabolical: {"Click for Diabolical Puzzle", "Right Click Change Difficulty"},
	}
	titleRight := 35 + len("SUDOKU MASTER & OCR SOLVER")*7
	for d, want := range expectedLines {
		l1, l2 := d.helpLines()
		if l1 != want[0] || l2 != want[1] {
			t.Errorf("level %d: got (%q, %q), want (%q, %q)", d, l1, l2, want[0], want[1])
		}
		x1 := dieX - 12 - len(l1)*7
		x2 := dieX - 12 - len(l2)*7
		if x1 <= titleRight {
			t.Errorf("line1 overlaps header title for %s: x1=%d, titleRight=%d", d.name(), x1, titleRight)
		}
		if x2 <= titleRight {
			t.Errorf("line2 overlaps header title for %s: x2=%d, titleRight=%d", d.name(), x2, titleRight)
		}
	}
}

func TestLoadRandomPuzzle_UnfocusesGrid(t *testing.T) {
	app := NewApp("")
	if app.selectedRow != -1 || app.selectedCol != -1 {
		t.Fatalf("expected NewApp to have (-1, -1) focus, got (%d, %d)", app.selectedRow, app.selectedCol)
	}

	app.selectedRow = 4
	app.selectedCol = 5
	app.loadRandomPuzzle()
	if app.selectedRow != -1 || app.selectedCol != -1 {
		t.Fatalf("expected loadRandomPuzzle to unfocus grid (-1, -1), got (%d, %d)", app.selectedRow, app.selectedCol)
	}

	app.selectedRow = 2
	app.selectedCol = 3
	app.clear()
	if app.selectedRow != -1 || app.selectedCol != -1 {
		t.Fatalf("expected clear to unfocus grid (-1, -1), got (%d, %d)", app.selectedRow, app.selectedCol)
	}
}