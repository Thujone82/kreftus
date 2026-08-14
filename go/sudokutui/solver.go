package main

// solveSudoku fills a copy of givens (81 runes/bytes, '0' or '.' = empty, '1'-'9')
// and returns the unique solution as 81 digits, or "" if unsolvable.
func solveSudoku(givens string) string {
	var grid [81]int
	if !parseGrid(givens, &grid) {
		return ""
	}
	if !search(&grid) {
		return ""
	}
	out := make([]byte, 81)
	for i, v := range grid {
		out[i] = byte('0' + v)
	}
	return string(out)
}

func parseGrid(s string, grid *[81]int) bool {
	if len(s) != 81 {
		return false
	}
	for i := 0; i < 81; i++ {
		c := s[i]
		switch {
		case c == '0' || c == '.' || c == ' ':
			grid[i] = 0
		case c >= '1' && c <= '9':
			grid[i] = int(c - '0')
		default:
			return false
		}
	}
	return validPartial(grid)
}

func validPartial(grid *[81]int) bool {
	var rows, cols, boxes [9]int
	for i, v := range grid {
		if v == 0 {
			continue
		}
		bit := 1 << v
		r, c := i/9, i%9
		b := (r/3)*3 + c/3
		if rows[r]&bit != 0 || cols[c]&bit != 0 || boxes[b]&bit != 0 {
			return false
		}
		rows[r] |= bit
		cols[c] |= bit
		boxes[b] |= bit
	}
	return true
}

func search(grid *[81]int) bool {
	idx := -1
	best := 10
	var rows, cols, boxes [9]int
	for i, v := range grid {
		if v == 0 {
			continue
		}
		bit := 1 << v
		r, c := i/9, i%9
		b := (r/3)*3 + c/3
		rows[r] |= bit
		cols[c] |= bit
		boxes[b] |= bit
	}
	var mask [81]int
	for i, v := range grid {
		if v != 0 {
			continue
		}
		r, c := i/9, i%9
		b := (r/3)*3 + c/3
		used := rows[r] | cols[c] | boxes[b]
		m := 0
		n := 0
		for d := 1; d <= 9; d++ {
			if used&(1<<d) == 0 {
				m |= 1 << d
				n++
			}
		}
		if n == 0 {
			return false
		}
		mask[i] = m
		if n < best {
			best = n
			idx = i
			if n == 1 {
				break
			}
		}
	}
	if idx < 0 {
		return true
	}
	m := mask[idx]
	for d := 1; d <= 9; d++ {
		if m&(1<<d) == 0 {
			continue
		}
		grid[idx] = d
		if search(grid) {
			return true
		}
		grid[idx] = 0
	}
	return false
}
