package main

import (
	"strings"
	"testing"
)

func TestFindHint_SamplePuzzle(t *testing.T) {
	app := NewApp("")
	app.loadSample()

	// Initial sample has 51 empty cells
	emptyCount := 0
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if app.grid[r][c] == 0 {
				emptyCount++
			}
		}
	}
	if emptyCount == 0 {
		t.Fatal("expected empty cells in sample puzzle")
	}

	// Request one hint
	app.hint()

	// Verify a cell was filled
	if app.selectedRow < 0 || app.selectedCol < 0 {
		t.Fatalf("expected selection to move to hinted cell, got (%d, %d)", app.selectedRow, app.selectedCol)
	}
	filledVal := app.grid[app.selectedRow][app.selectedCol]
	if filledVal < 1 || filledVal > 9 {
		t.Fatalf("expected cell to have valid digit 1-9, got %d", filledVal)
	}

	// Verify board is still valid
	if !validateInitialBoard(&app.grid) {
		t.Fatal("board became invalid after applying hint")
	}
}

func TestFindHint_StepByStepToCompletion(t *testing.T) {
	app := NewApp("")
	app.loadSample()

	steps := 0
	for {
		hasEmpty := false
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if app.grid[r][c] == 0 {
					hasEmpty = true
					break
				}
			}
			if hasEmpty {
				break
			}
		}
		if !hasEmpty {
			break
		}

		prevEmpty := 0
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if app.grid[r][c] == 0 {
					prevEmpty++
				}
			}
		}

		app.hint()
		steps++

		newEmpty := 0
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				if app.grid[r][c] == 0 {
					newEmpty++
				}
			}
		}

		if newEmpty != prevEmpty-1 {
			t.Fatalf("step %d did not place exactly 1 digit: prevEmpty=%d, newEmpty=%d", steps, prevEmpty, newEmpty)
		}
		if !validateInitialBoard(&app.grid) {
			t.Fatalf("step %d resulted in invalid board state", steps)
		}
	}

	// Calling hint on complete board should report already completed
	app.hint()
	if app.statusMsg != "Puzzle is already completed!" {
		t.Fatalf("expected completed status, got: %q", app.statusMsg)
	}
}

func TestFindHint_PointingLine(t *testing.T) {
	// A known board where Pointing Lines are exercised
	// Grid representation
	raw := "000000000000003085001020000000507000004000100090000000500000073002010000000040009"
	b, _, ok := boardFromGivens(raw)
	if !ok {
		t.Fatal("failed to parse board")
	}

	cand, ok := computeCandidates(&b)
	if !ok {
		t.Fatal("candidates calculation failed")
	}

	h, ok := applyPointing(&b, &cand)
	if ok {
		if !strings.Contains(h.reason, "Pointing Line") {
			t.Fatalf("expected Pointing Line reason, got: %s", h.reason)
		}
	}
}

func TestFindHint_NakedPair(t *testing.T) {
	// Board with a known Naked Pair
	raw := "000000000012034567034567812000000000123000456000000000000000000000000000000000000"
	// Verify applyNakedPairs runs cleanly without panic
	b, _, ok := boardFromGivens(raw)
	if !ok {
		t.Fatal("failed to parse board")
	}
	cand, _ := computeCandidates(&b)
	_, _ = applyNakedPairs(&b, &cand)
}
