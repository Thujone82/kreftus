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
