package main

// Hint represents a single logical deduction or step to progress the puzzle.
type Hint struct {
	row    int
	col    int
	val    int
	reason string
}

// getCandidates returns a bitmask (bits 1..9) of legal numbers for board[r][c].
func getCandidates(b *Board, r, c int) uint16 {
	if b[r][c] != 0 {
		return 0
	}
	var mask uint16 = 0x3FE // bits 1..9 set (1111111110)
	for i := 0; i < 9; i++ {
		if v := b[r][i]; v != 0 {
			mask &^= (1 << v)
		}
		if v := b[i][c]; v != 0 {
			mask &^= (1 << v)
		}
	}
	br, bc := (r/3)*3, (c/3)*3
	for dr := 0; dr < 3; dr++ {
		for dc := 0; dc < 3; dc++ {
			if v := b[br+dr][bc+dc]; v != 0 {
				mask &^= (1 << v)
			}
		}
	}
	return mask
}

func countBits(mask uint16) int {
	cnt := 0
	for num := 1; num <= 9; num++ {
		if (mask & (1 << num)) != 0 {
			cnt++
		}
	}
	return cnt
}

func singleVal(mask uint16) int {
	for num := 1; num <= 9; num++ {
		if (mask & (1 << num)) != 0 {
			return num
		}
	}
	return 0
}

// findHint searches for the easiest logical deduction to progress the puzzle.
// It prioritizes:
// 1. Naked Singles (cells where only one digit can possibly fit).
// 2. Hidden Singles in Rows (a digit can only go in one cell in a row).
// 3. Hidden Singles in Columns (a digit can only go in one cell in a column).
// 4. Hidden Singles in 3x3 Boxes (a digit can only go in one cell in a 3x3 box).
// 5. If no singles exist (e.g. advanced puzzle requiring pairs/backtracking),
//    it falls back to computing the full solution via solveSudoku and filling
//    the cell with the fewest candidates (most constrained).
func findHint(b *Board) (Hint, bool) {
	// First check if the puzzle is already full
	hasEmpty := false
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if b[r][c] == 0 {
				hasEmpty = true
				break
			}
		}
		if hasEmpty {
			break
		}
	}
	if !hasEmpty {
		return Hint{}, false
	}

	// Compute candidate masks for all empty cells
	var cand [9][9]uint16
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if b[r][c] == 0 {
				cands := getCandidates(b, r, c)
				if cands == 0 {
					// Puzzle has a cell with 0 valid candidates, board is dead/unsolvable
					return Hint{}, false
				}
				cand[r][c] = cands
			}
		}
	}

	// Priority 1: Naked Singles (only 1 candidate in cell)
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if b[r][c] == 0 && countBits(cand[r][c]) == 1 {
				v := singleVal(cand[r][c])
				return Hint{
					row:    r,
					col:    c,
					val:    v,
					reason: "Naked Single - only possible number for this cell",
				}, true
			}
		}
	}

	// Priority 2: Hidden Singles in 3x3 Boxes (often easiest to spot visually)
	for box := 0; box < 9; box++ {
		br := (box / 3) * 3
		bc := (box % 3) * 3
		for num := 1; num <= 9; num++ {
			targetBit := uint16(1 << num)
			placeCount := 0
			lastR, lastC := -1, -1
			for dr := 0; dr < 3; dr++ {
				for dc := 0; dc < 3; dc++ {
					r := br + dr
					c := bc + dc
					if b[r][c] == 0 && (cand[r][c]&targetBit) != 0 {
						placeCount++
						lastR, lastC = r, c
					}
				}
			}
			if placeCount == 1 {
				return Hint{
					row:    lastR,
					col:    lastC,
					val:    num,
					reason: "Hidden Single - only place in 3x3 block",
				}, true
			}
		}
	}

	// Priority 3: Hidden Singles in Rows
	for r := 0; r < 9; r++ {
		for num := 1; num <= 9; num++ {
			targetBit := uint16(1 << num)
			placeCount := 0
			lastC := -1
			for c := 0; c < 9; c++ {
				if b[r][c] == 0 && (cand[r][c]&targetBit) != 0 {
					placeCount++
					lastC = c
				}
			}
			if placeCount == 1 {
				return Hint{
					row:    r,
					col:    lastC,
					val:    num,
					reason: "Hidden Single - only place in row",
				}, true
			}
		}
	}

	// Priority 4: Hidden Singles in Columns
	for c := 0; c < 9; c++ {
		for num := 1; num <= 9; num++ {
			targetBit := uint16(1 << num)
			placeCount := 0
			lastR := -1
			for r := 0; r < 9; r++ {
				if b[r][c] == 0 && (cand[r][c]&targetBit) != 0 {
					placeCount++
					lastR = r
				}
			}
			if placeCount == 1 {
				return Hint{
					row:    lastR,
					col:    c,
					val:    num,
					reason: "Hidden Single - only place in column",
				}, true
			}
		}
	}

	// Priority 5: Fallback for hard puzzles where human singles aren't exposed.
	// Find the cell with the fewest candidates, solve the board to guarantee correctness.
	solved := *b
	if !solveSudoku(&solved) {
		return Hint{}, false
	}

	minCands := 10
	bestR, bestC := -1, -1
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if b[r][c] == 0 {
				cnt := countBits(cand[r][c])
				if cnt < minCands {
					minCands = cnt
					bestR, bestC = r, c
				}
			}
		}
	}

	if bestR >= 0 && bestC >= 0 {
		return Hint{
			row:    bestR,
			col:    bestC,
			val:    solved[bestR][bestC],
			reason: "Deduction - minimum remaining possibilities",
		}, true
	}

	return Hint{}, false
}
