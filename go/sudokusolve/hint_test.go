package main

import "testing"

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
