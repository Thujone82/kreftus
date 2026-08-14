package main

type board struct {
	givens   [81]byte
	solution [81]byte
	grid     [81]byte
	cursor   int
	mistakes int
}

func newBoard(givens, solution, grid string) board {
	var b board
	copy81(&b.givens, givens)
	copy81(&b.solution, solution)
	if grid == "" {
		b.grid = b.givens
	} else {
		copy81(&b.grid, grid)
		for i := 0; i < 81; i++ {
			if b.givens[i] != '0' {
				b.grid[i] = b.givens[i]
			}
		}
	}
	b.cursor = firstEmpty(&b)
	return b
}

func copy81(dst *[81]byte, s string) {
	n := len(s)
	if n > 81 {
		n = 81
	}
	for i := 0; i < n; i++ {
		c := s[i]
		if c == '.' {
			c = '0'
		}
		dst[i] = c
	}
	for i := n; i < 81; i++ {
		dst[i] = '0'
	}
}

func firstEmpty(b *board) int {
	for i := 0; i < 81; i++ {
		if !b.isLocked(i) {
			return i
		}
	}
	return 0
}

func (b *board) gridString() string {
	return string(b.grid[:])
}

func (b *board) isGiven(i int) bool {
	return b.givens[i] != '0'
}

func (b *board) isLocked(i int) bool {
	if b.isGiven(i) {
		return true
	}
	v := b.grid[i]
	return v != '0' && v == b.solution[i]
}

func (b *board) isWrong(i int) bool {
	v := b.grid[i]
	return v != '0' && v != b.solution[i]
}

func (b *board) isComplete() bool {
	for i := 0; i < 81; i++ {
		if b.grid[i] != b.solution[i] {
			return false
		}
	}
	return true
}

func (b *board) place(digit byte) bool {
	i := b.cursor
	if b.isLocked(i) {
		return false
	}
	if digit < '1' || digit > '9' {
		return false
	}
	if b.grid[i] == digit {
		return false
	}
	b.grid[i] = digit
	if digit != b.solution[i] {
		b.mistakes++
	}
	return true
}

func (b *board) clear() bool {
	i := b.cursor
	if b.isLocked(i) {
		return false
	}
	if b.grid[i] == '0' {
		return false
	}
	b.grid[i] = '0'
	return true
}

func (b *board) move(dx, dy int) {
	x, y := b.cursor%9, b.cursor/9
	x = (x + dx + 9) % 9
	y = (y + dy + 9) % 9
	b.cursor = y*9 + x
}
