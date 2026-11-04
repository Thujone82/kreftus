package main

import (
	"fmt"
	"strings"
)

// Hint represents a single logical deduction or step to progress the puzzle.
type Hint struct {
	row    int
	col    int
	val    int
	reason string
}

type cellCoord struct {
	r, c int
}

// getCandidates returns a bitmask (bits 1..9) of legal numbers for board[r][c].
func getCandidates(b *Board, r, c int) uint16 {
	if b[r][c] != 0 {
		return 0
	}
	var mask uint16 = 0x3FE // bits 1..9 set
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

func maskDigits(mask uint16) []int {
	var digits []int
	for num := 1; num <= 9; num++ {
		if (mask & (1 << num)) != 0 {
			digits = append(digits, num)
		}
	}
	return digits
}

func formatDigits(mask uint16) string {
	d := maskDigits(mask)
	var sb strings.Builder
	for i, v := range d {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%d", v))
	}
	return sb.String()
}

// computeCandidates returns candidate masks for all cells based directly on existing grid values.
func computeCandidates(b *Board) ([9][9]uint16, bool) {
	var cand [9][9]uint16
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if b[r][c] == 0 {
				m := getCandidates(b, r, c)
				if m == 0 {
					return cand, false
				}
				cand[r][c] = m
			}
		}
	}
	return cand, true
}

// checkDirectSingles checks for immediate Naked or Hidden Singles in the candidate matrix.
func checkDirectSingles(b *Board, cand *[9][9]uint16) (Hint, bool) {
	// 1. Naked Singles
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

	// 2. Hidden Singles in 3x3 Boxes
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

	// 3. Hidden Singles in Rows
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

	// 4. Hidden Singles in Columns
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

	return Hint{}, false
}

// findUnlockedSingle compares current candidates against previous candidates.
// If any empty cell's candidates dropped to 1 or became a hidden single in a unit, it returns the placed hint.
func findUnlockedSingle(b *Board, prevCand, newCand *[9][9]uint16, techniqueName string) (Hint, bool) {
	// First check if any cell became a Naked Single
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if b[r][c] == 0 {
				oldBits := countBits(prevCand[r][c])
				newBits := countBits(newCand[r][c])
				if oldBits > 1 && newBits == 1 {
					v := singleVal(newCand[r][c])
					return Hint{
						row:    r,
						col:    c,
						val:    v,
						reason: fmt.Sprintf("%s unlocks %d at (%d, %d)", techniqueName, v, r+1, c+1),
					}, true
				}
			}
		}
	}

	// Next check if any digit became a Hidden Single in a 3x3 block
	for box := 0; box < 9; box++ {
		br := (box / 3) * 3
		bc := (box % 3) * 3
		for num := 1; num <= 9; num++ {
			bit := uint16(1 << num)
			oldCount := 0
			newCount := 0
			lastR, lastC := -1, -1
			for dr := 0; dr < 3; dr++ {
				for dc := 0; dc < 3; dc++ {
					r, c := br+dr, bc+dc
					if b[r][c] == 0 {
						if (prevCand[r][c] & bit) != 0 {
							oldCount++
						}
						if (newCand[r][c] & bit) != 0 {
							newCount++
							lastR, lastC = r, c
						}
					}
				}
			}
			if oldCount > 1 && newCount == 1 {
				return Hint{
					row:    lastR,
					col:    lastC,
					val:    num,
					reason: fmt.Sprintf("%s unlocks %d in box %d at (%d, %d)", techniqueName, num, box+1, lastR+1, lastC+1),
				}, true
			}
		}
	}

	// Next check if any digit became a Hidden Single in a row
	for r := 0; r < 9; r++ {
		for num := 1; num <= 9; num++ {
			bit := uint16(1 << num)
			oldCount := 0
			newCount := 0
			lastC := -1
			for c := 0; c < 9; c++ {
				if b[r][c] == 0 {
					if (prevCand[r][c] & bit) != 0 {
						oldCount++
					}
					if (newCand[r][c] & bit) != 0 {
						newCount++
						lastC = c
					}
				}
			}
			if oldCount > 1 && newCount == 1 {
				return Hint{
					row:    r,
					col:    lastC,
					val:    num,
					reason: fmt.Sprintf("%s unlocks %d in row %d at (%d, %d)", techniqueName, num, r+1, r+1, lastC+1),
				}, true
			}
		}
	}

	// Next check if any digit became a Hidden Single in a column
	for c := 0; c < 9; c++ {
		for num := 1; num <= 9; num++ {
			bit := uint16(1 << num)
			oldCount := 0
			newCount := 0
			lastR := -1
			for r := 0; r < 9; r++ {
				if b[r][c] == 0 {
					if (prevCand[r][c] & bit) != 0 {
						oldCount++
					}
					if (newCand[r][c] & bit) != 0 {
						newCount++
						lastR = r
					}
				}
			}
			if oldCount > 1 && newCount == 1 {
				return Hint{
					row:    lastR,
					col:    c,
					val:    num,
					reason: fmt.Sprintf("%s unlocks %d in col %d at (%d, %d)", techniqueName, num, c+1, lastR+1, c+1),
				}, true
			}
		}
	}

	return Hint{}, false
}

// -----------------------------------------------------------------------------
// Intermediate Elimination Techniques
// -----------------------------------------------------------------------------

// applyPointingPairs detects Locked Candidates Type 1 (Pointing).
// If in a box, all candidates for a digit 'num' lie in a single row or column,
// that digit can be eliminated from the rest of that row or column outside the box.
func applyPointing(b *Board, cand *[9][9]uint16) (Hint, bool) {
	for box := 0; box < 9; box++ {
		br := (box / 3) * 3
		bc := (box % 3) * 3
		for num := 1; num <= 9; num++ {
			bit := uint16(1 << num)
			var cells []cellCoord
			for dr := 0; dr < 3; dr++ {
				for dc := 0; dc < 3; dc++ {
					r, c := br+dr, bc+dc
					if b[r][c] == 0 && (cand[r][c]&bit) != 0 {
						cells = append(cells, cellCoord{r, c})
					}
				}
			}
			if len(cells) < 2 || len(cells) > 3 {
				continue
			}

			// Check same row
			sameRow := true
			targetRow := cells[0].r
			for _, cell := range cells {
				if cell.r != targetRow {
					sameRow = false
					break
				}
			}
			if sameRow {
				newCand := *cand
				elimCount := 0
				for c := 0; c < 9; c++ {
					if c < bc || c >= bc+3 {
						if b[targetRow][c] == 0 && (newCand[targetRow][c]&bit) != 0 {
							newCand[targetRow][c] &^= bit
							elimCount++
						}
					}
				}
				if elimCount > 0 {
					desc := fmt.Sprintf("Pointing Line (%d in Box %d)", num, box+1)
					if h, ok := findUnlockedSingle(b, cand, &newCand, desc); ok {
						return h, true
					}
				}
			}

			// Check same column
			sameCol := true
			targetCol := cells[0].c
			for _, cell := range cells {
				if cell.c != targetCol {
					sameCol = false
					break
				}
			}
			if sameCol {
				newCand := *cand
				elimCount := 0
				for r := 0; r < 9; r++ {
					if r < br || r >= br+3 {
						if b[r][targetCol] == 0 && (newCand[r][targetCol]&bit) != 0 {
							newCand[r][targetCol] &^= bit
							elimCount++
						}
					}
				}
				if elimCount > 0 {
					desc := fmt.Sprintf("Pointing Line (%d in Box %d)", num, box+1)
					if h, ok := findUnlockedSingle(b, cand, &newCand, desc); ok {
						return h, true
					}
				}
			}
		}
	}
	return Hint{}, false
}

// applyClaiming detects Locked Candidates Type 2 (Claiming / Box-Line Reduction).
// If in a row or col, all candidates for a digit lie in a single box,
// that digit can be eliminated from the rest of the box.
func applyClaiming(b *Board, cand *[9][9]uint16) (Hint, bool) {
	// By rows
	for r := 0; r < 9; r++ {
		for num := 1; num <= 9; num++ {
			bit := uint16(1 << num)
			var cols []int
			for c := 0; c < 9; c++ {
				if b[r][c] == 0 && (cand[r][c]&bit) != 0 {
					cols = append(cols, c)
				}
			}
			if len(cols) >= 2 && len(cols) <= 3 {
				boxCol := cols[0] / 3
				sameBox := true
				for _, c := range cols {
					if c/3 != boxCol {
						sameBox = false
						break
					}
				}
				if sameBox {
					newCand := *cand
					elimCount := 0
					boxRow := r / 3
					for dr := 0; dr < 3; dr++ {
						cr := boxRow*3 + dr
						if cr == r {
							continue
						}
						for dc := 0; dc < 3; dc++ {
							cc := boxCol*3 + dc
							if b[cr][cc] == 0 && (newCand[cr][cc]&bit) != 0 {
								newCand[cr][cc] &^= bit
								elimCount++
							}
						}
					}
					if elimCount > 0 {
						desc := fmt.Sprintf("Claiming Line (%d in Row %d)", num, r+1)
						if h, ok := findUnlockedSingle(b, cand, &newCand, desc); ok {
							return h, true
						}
					}
				}
			}
		}
	}

	// By cols
	for c := 0; c < 9; c++ {
		for num := 1; num <= 9; num++ {
			bit := uint16(1 << num)
			var rows []int
			for r := 0; r < 9; r++ {
				if b[r][c] == 0 && (cand[r][c]&bit) != 0 {
					rows = append(rows, r)
				}
			}
			if len(rows) >= 2 && len(rows) <= 3 {
				boxRow := rows[0] / 3
				sameBox := true
				for _, r := range rows {
					if r/3 != boxRow {
						sameBox = false
						break
					}
				}
				if sameBox {
					newCand := *cand
					elimCount := 0
					boxCol := c / 3
					for dc := 0; dc < 3; dc++ {
						cc := boxCol*3 + dc
						if cc == c {
							continue
						}
						for dr := 0; dr < 3; dr++ {
							cr := boxRow*3 + dr
							if b[cr][cc] == 0 && (newCand[cr][cc]&bit) != 0 {
								newCand[cr][cc] &^= bit
								elimCount++
							}
						}
					}
					if elimCount > 0 {
						desc := fmt.Sprintf("Claiming Line (%d in Col %d)", num, c+1)
						if h, ok := findUnlockedSingle(b, cand, &newCand, desc); ok {
							return h, true
						}
					}
				}
			}
		}
	}
	return Hint{}, false
}

// applyNakedPairs detects Naked Pairs in rows, columns, or boxes.
// If two cells in a unit share the exact same two candidates, those two digits can be eliminated from other cells in that unit.
func applyNakedPairs(b *Board, cand *[9][9]uint16) (Hint, bool) {
	// Helper to check a unit
	checkUnit := func(cells []cellCoord, unitName string) (Hint, bool) {
		for i := 0; i < len(cells); i++ {
			ci := cells[i]
			if b[ci.r][ci.c] != 0 || countBits(cand[ci.r][ci.c]) != 2 {
				continue
			}
			mask := cand[ci.r][ci.c]
			for j := i + 1; j < len(cells); j++ {
				cj := cells[j]
				if b[cj.r][cj.c] != 0 || cand[cj.r][cj.c] != mask {
					continue
				}
				// Found naked pair: ci and cj with candidates `mask`
				newCand := *cand
				elimCount := 0
				for k := 0; k < len(cells); k++ {
					if k == i || k == j {
						continue
					}
					ck := cells[k]
					if b[ck.r][ck.c] == 0 && (newCand[ck.r][ck.c]&mask) != 0 {
						newCand[ck.r][ck.c] &^= mask
						elimCount++
					}
				}
				if elimCount > 0 {
					desc := fmt.Sprintf("Naked Pair [%s] in %s", formatDigits(mask), unitName)
					if h, ok := findUnlockedSingle(b, cand, &newCand, desc); ok {
						return h, true
					}
				}
			}
		}
		return Hint{}, false
	}

	// 1. Rows
	for r := 0; r < 9; r++ {
		var cells []cellCoord
		for c := 0; c < 9; c++ {
			cells = append(cells, cellCoord{r, c})
		}
		if h, ok := checkUnit(cells, fmt.Sprintf("Row %d", r+1)); ok {
			return h, true
		}
	}

	// 2. Columns
	for c := 0; c < 9; c++ {
		var cells []cellCoord
		for r := 0; r < 9; r++ {
			cells = append(cells, cellCoord{r, c})
		}
		if h, ok := checkUnit(cells, fmt.Sprintf("Col %d", c+1)); ok {
			return h, true
		}
	}

	// 3. Boxes
	for box := 0; box < 9; box++ {
		var cells []cellCoord
		br := (box / 3) * 3
		bc := (box % 3) * 3
		for dr := 0; dr < 3; dr++ {
			for dc := 0; dc < 3; dc++ {
				cells = append(cells, cellCoord{br + dr, bc + dc})
			}
		}
		if h, ok := checkUnit(cells, fmt.Sprintf("Box %d", box+1)); ok {
			return h, true
		}
	}

	return Hint{}, false
}

// applyHiddenPairs detects Hidden Pairs in rows, columns, or boxes.
// If two digits appear only in two cells in a unit, all other candidates in those two cells can be eliminated.
func applyHiddenPairs(b *Board, cand *[9][9]uint16) (Hint, bool) {
	checkUnit := func(cells []cellCoord, unitName string) (Hint, bool) {
		for d1 := 1; d1 <= 8; d1++ {
			b1 := uint16(1 << d1)
			for d2 := d1 + 1; d2 <= 9; d2++ {
				b2 := uint16(1 << d2)
				pairMask := b1 | b2

				var matchCells []cellCoord
				hasOther := false
				for _, cell := range cells {
					if b[cell.r][cell.c] == 0 {
						has1 := (cand[cell.r][cell.c] & b1) != 0
						has2 := (cand[cell.r][cell.c] & b2) != 0
						if has1 || has2 {
							if has1 && has2 {
								matchCells = append(matchCells, cell)
							} else {
								hasOther = true
								break
							}
						}
					}
				}
				if hasOther || len(matchCells) != 2 {
					continue
				}

				// The pair (d1, d2) appears only in these 2 cells.
				// Can we eliminate other candidates from these 2 cells?
				newCand := *cand
				elimCount := 0
				for _, cell := range matchCells {
					if (newCand[cell.r][cell.c] &^ pairMask) != 0 {
						newCand[cell.r][cell.c] = pairMask
						elimCount++
					}
				}
				if elimCount > 0 {
					desc := fmt.Sprintf("Hidden Pair [%d,%d] in %s", d1, d2, unitName)
					if h, ok := findUnlockedSingle(b, cand, &newCand, desc); ok {
						return h, true
					}
				}
			}
		}
		return Hint{}, false
	}

	// Rows
	for r := 0; r < 9; r++ {
		var cells []cellCoord
		for c := 0; c < 9; c++ {
			cells = append(cells, cellCoord{r, c})
		}
		if h, ok := checkUnit(cells, fmt.Sprintf("Row %d", r+1)); ok {
			return h, true
		}
	}

	// Columns
	for c := 0; c < 9; c++ {
		var cells []cellCoord
		for r := 0; r < 9; r++ {
			cells = append(cells, cellCoord{r, c})
		}
		if h, ok := checkUnit(cells, fmt.Sprintf("Col %d", c+1)); ok {
			return h, true
		}
	}

	// Boxes
	for box := 0; box < 9; box++ {
		var cells []cellCoord
		br := (box / 3) * 3
		bc := (box % 3) * 3
		for dr := 0; dr < 3; dr++ {
			for dc := 0; dc < 3; dc++ {
				cells = append(cells, cellCoord{br + dr, bc + dc})
			}
		}
		if h, ok := checkUnit(cells, fmt.Sprintf("Box %d", box+1)); ok {
			return h, true
		}
	}

	return Hint{}, false
}

// applyXWings detects X-Wings on rows or columns.
// Row-based: for a digit 'num', exactly two cells in row r1 and row r2 have candidate 'num', and they share columns c1 and c2.
// 'num' can then be eliminated from all other rows in columns c1 and c2.
func applyXWings(b *Board, cand *[9][9]uint16) (Hint, bool) {
	// 1. Row-based X-Wing
	for num := 1; num <= 9; num++ {
		bit := uint16(1 << num)
		type rowPair struct {
			r      int
			c1, c2 int
		}
		var pairs []rowPair
		for r := 0; r < 9; r++ {
			var cols []int
			for c := 0; c < 9; c++ {
				if b[r][c] == 0 && (cand[r][c]&bit) != 0 {
					cols = append(cols, c)
				}
			}
			if len(cols) == 2 {
				pairs = append(pairs, rowPair{r: r, c1: cols[0], c2: cols[1]})
			}
		}

		for i := 0; i < len(pairs); i++ {
			for j := i + 1; j < len(pairs); j++ {
				p1 := pairs[i]
				p2 := pairs[j]
				if p1.c1 == p2.c1 && p1.c2 == p2.c2 {
					// Found Row X-Wing at rows p1.r, p2.r and cols p1.c1, p1.c2
					newCand := *cand
					elimCount := 0
					for r := 0; r < 9; r++ {
						if r == p1.r || r == p2.r {
							continue
						}
						if b[r][p1.c1] == 0 && (newCand[r][p1.c1]&bit) != 0 {
							newCand[r][p1.c1] &^= bit
							elimCount++
						}
						if b[r][p1.c2] == 0 && (newCand[r][p1.c2]&bit) != 0 {
							newCand[r][p1.c2] &^= bit
							elimCount++
						}
					}
					if elimCount > 0 {
						desc := fmt.Sprintf("X-Wing on %d (Rows %d & %d)", num, p1.r+1, p2.r+1)
						if h, ok := findUnlockedSingle(b, cand, &newCand, desc); ok {
							return h, true
						}
					}
				}
			}
		}
	}

	// 2. Column-based X-Wing
	for num := 1; num <= 9; num++ {
		bit := uint16(1 << num)
		type colPair struct {
			c      int
			r1, r2 int
		}
		var pairs []colPair
		for c := 0; c < 9; c++ {
			var rows []int
			for r := 0; r < 9; r++ {
				if b[r][c] == 0 && (cand[r][c]&bit) != 0 {
					rows = append(rows, r)
				}
			}
			if len(rows) == 2 {
				pairs = append(pairs, colPair{c: c, r1: rows[0], r2: rows[1]})
			}
		}

		for i := 0; i < len(pairs); i++ {
			for j := i + 1; j < len(pairs); j++ {
				p1 := pairs[i]
				p2 := pairs[j]
				if p1.r1 == p2.r1 && p1.r2 == p2.r2 {
					// Found Col X-Wing at cols p1.c, p2.c and rows p1.r1, p1.r2
					newCand := *cand
					elimCount := 0
					for c := 0; c < 9; c++ {
						if c == p1.c || c == p2.c {
							continue
						}
						if b[p1.r1][c] == 0 && (newCand[p1.r1][c]&bit) != 0 {
							newCand[p1.r1][c] &^= bit
							elimCount++
						}
						if b[p1.r2][c] == 0 && (newCand[p1.r2][c]&bit) != 0 {
							newCand[p1.r2][c] &^= bit
							elimCount++
						}
					}
					if elimCount > 0 {
						desc := fmt.Sprintf("X-Wing on %d (Cols %d & %d)", num, p1.c+1, p2.c+1)
						if h, ok := findUnlockedSingle(b, cand, &newCand, desc); ok {
							return h, true
						}
					}
				}
			}
		}
	}

	return Hint{}, false
}

// -----------------------------------------------------------------------------
// Main Hint Solver
// -----------------------------------------------------------------------------

// findHint searches for the easiest logical deduction to progress the puzzle:
// 1. Direct Singles:
//    - Naked Singles
//    - Hidden Singles in Boxes
//    - Hidden Singles in Rows
//    - Hidden Singles in Columns
// 2. Intermediate Human Solving Techniques:
//    - Pointing Lines (Locked Candidates Type 1)
//    - Claiming Lines (Locked Candidates Type 2)
//    - Naked Pairs
//    - Hidden Pairs
//    - X-Wings
// 3. Fallback Deduction (for diabolical or non-standard puzzles):
//    - Solves board and places the digit into the most-constrained cell.
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

	// Compute candidates directly
	cand, ok := computeCandidates(b)
	if !ok {
		return Hint{}, false
	}

	// Priority 1: Direct Singles
	if h, ok := checkDirectSingles(b, &cand); ok {
		return h, true
	}

	// Priority 2: Intermediate Techniques in increasing complexity order
	if h, ok := applyPointing(b, &cand); ok {
		return h, true
	}
	if h, ok := applyClaiming(b, &cand); ok {
		return h, true
	}
	if h, ok := applyNakedPairs(b, &cand); ok {
		return h, true
	}
	if h, ok := applyHiddenPairs(b, &cand); ok {
		return h, true
	}
	if h, ok := applyXWings(b, &cand); ok {
		return h, true
	}

	// Priority 3: Fallback deduction
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
