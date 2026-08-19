package main

type board struct {
	givens     [81]byte
	solution   [81]byte
	grid       [81]byte
	cursor     int
	mistakes   int
	pencil     [81][2]byte // '1'-'9' or 0; [0] top half, [1] bottom half
	pencilSlot [81]byte    // next write: 0 top, 1 bottom
	digitCount [10]int     // correctly placed count for digits 1–9
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
	b.recountCorrect()
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
	return !emptyCell(b.givens[i])
}

func (b *board) isLocked(i int) bool {
	if b.isGiven(i) {
		return true
	}
	v := b.grid[i]
	if emptyCell(v) {
		return false
	}
	return v == b.solution[i]
}

func (b *board) isWrong(i int) bool {
	v := b.grid[i]
	if emptyCell(v) {
		return false
	}
	return v != b.solution[i]
}

func emptyCell(c byte) bool {
	return c == 0 || c == '0' || c == '.'
}

func (b *board) fillProgress() (filled, total int) {
	for i := 0; i < 81; i++ {
		if emptyCell(b.givens[i]) {
			total++
			if !emptyCell(b.grid[i]) {
				filled++
			}
		}
	}
	return filled, total
}

func (b *board) pencilMarkCount() int {
	n := 0
	for i := 0; i < 81; i++ {
		if b.pencil[i][0] != 0 {
			n++
		}
		if b.pencil[i][1] != 0 {
			n++
		}
	}
	return n
}

func (b *board) isComplete() bool {
	for i := 0; i < 81; i++ {
		if b.grid[i] != b.solution[i] {
			return false
		}
	}
	return true
}

func (b *board) recountCorrect() {
	var count [10]int
	for i := 0; i < 81; i++ {
		v := b.grid[i]
		if v >= '1' && v <= '9' && v == b.solution[i] {
			count[v-'0']++
		}
	}
	b.digitCount = count
}

func (b *board) completedDigits() [10]bool {
	var done [10]bool
	for d := 1; d <= 9; d++ {
		done[d] = b.digitCount[d] == 9
	}
	return done
}

func (b *board) digitComplete(digit byte) bool {
	return digit >= '1' && digit <= '9' && b.digitCount[digit-'0'] == 9
}

func (b *board) activeDigits() []byte {
	done := b.completedDigits()
	out := make([]byte, 0, 9)
	for d := byte(1); d <= 9; d++ {
		if !done[d] {
			out = append(out, '0'+d)
		}
	}
	return out
}

func (b *board) place(digit byte) bool {
	i := b.cursor
	if b.isLocked(i) {
		return false
	}
	if digit < '1' || digit > '9' {
		return false
	}
	if b.digitComplete(digit) {
		return false
	}
	if b.grid[i] == digit {
		return false
	}
	b.grid[i] = digit
	b.clearPencilsAt(i)
	if digit != b.solution[i] {
		b.mistakes++
		return true
	}
	b.digitCount[digit-'0']++
	b.stripPencilPeers(i, digit)
	if b.digitComplete(digit) {
		b.stripPencilDigit(digit)
	}
	return true
}

func (b *board) clear() bool {
	i := b.cursor
	if b.isLocked(i) {
		return false
	}
	if emptyCell(b.grid[i]) {
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

func (b *board) markPencil(digit byte) bool {
	i := b.cursor
	if b.isLocked(i) || !emptyCell(b.grid[i]) {
		return false
	}
	if digit < '1' || digit > '9' {
		return false
	}
	if b.digitComplete(digit) {
		return false
	}
	if b.pencil[i][0] == digit {
		b.pencil[i][0] = 0
		b.pencilSlot[i] = 0
		if b.pencil[i][1] == 0 {
			b.clearPencilsAt(i)
		}
		return true
	}
	if b.pencil[i][1] == digit {
		b.pencil[i][1] = 0
		b.pencilSlot[i] = 1
		if b.pencil[i][0] == 0 {
			b.clearPencilsAt(i)
		}
		return true
	}
	slot := b.pencilSlot[i]
	if slot > 1 {
		slot = 0
	}
	b.pencil[i][slot] = digit
	b.pencilSlot[i] = 1 - slot
	return true
}

func (b *board) clearPencil() bool {
	i := b.cursor
	if b.isLocked(i) {
		return false
	}
	if b.pencil[i][0] == 0 && b.pencil[i][1] == 0 {
		return false
	}
	b.clearPencilsAt(i)
	return true
}

func (b *board) clearPencilsAt(i int) {
	b.pencil[i][0] = 0
	b.pencil[i][1] = 0
	b.pencilSlot[i] = 0
}

func (b *board) stripPencilDigitAt(i int, digit byte) {
	top, bot := b.pencil[i][0], b.pencil[i][1]
	if top != digit && bot != digit {
		return
	}
	var remain byte
	if top != 0 && top != digit {
		remain = top
	}
	if bot != 0 && bot != digit {
		remain = bot
	}
	if remain == 0 {
		b.clearPencilsAt(i)
		return
	}
	b.pencil[i][0] = 0
	b.pencil[i][1] = remain
	b.pencilSlot[i] = 0
}

func (b *board) stripPencilDigit(digit byte) {
	for i := 0; i < 81; i++ {
		b.stripPencilDigitAt(i, digit)
	}
}

func (b *board) stripPencilPeers(at int, digit byte) {
	r, c := at/9, at%9
	br, bc := r/3*3, c/3*3
	for i := 0; i < 9; i++ {
		b.stripPencilDigitAt(r*9+i, digit)
		b.stripPencilDigitAt(i*9+c, digit)
		b.stripPencilDigitAt((br+i/3)*9+(bc+i%3), digit)
	}
}

func (b *board) stripCompletedPencils() {
	for d := byte(1); d <= 9; d++ {
		if b.digitComplete('0' + d) {
			b.stripPencilDigit('0' + d)
		}
	}
}

func (b *board) stripImpossiblePencils() {
	b.stripCompletedPencils()
	for i := 0; i < 81; i++ {
		if b.isLocked(i) {
			b.stripPencilPeers(i, b.grid[i])
		}
	}
}

func (b *board) hasPencil(i int) bool {
	return emptyCell(b.grid[i]) && (b.pencil[i][0] != 0 || b.pencil[i][1] != 0)
}

func (b *board) pencilsString() (top, bot, slot string) {
	var t, o, s [81]byte
	for i := 0; i < 81; i++ {
		t[i] = pencilDigit(b.pencil[i][0])
		o[i] = pencilDigit(b.pencil[i][1])
		if b.pencilSlot[i] == 1 {
			s[i] = '1'
		} else {
			s[i] = '0'
		}
	}
	return string(t[:]), string(o[:]), string(s[:])
}

func pencilDigit(d byte) byte {
	if d >= '1' && d <= '9' {
		return d
	}
	return '0'
}

func (b *board) loadPencils(top, bot, slot string) {
	for i := 0; i < 81; i++ {
		if i < len(top) && top[i] >= '1' && top[i] <= '9' {
			b.pencil[i][0] = top[i]
		}
		if i < len(bot) && bot[i] >= '1' && bot[i] <= '9' {
			b.pencil[i][1] = bot[i]
		}
		if i < len(slot) && slot[i] == '1' {
			b.pencilSlot[i] = 1
		}
	}
}
