package main

import (
	"strings"
)

func isCellRune(ch rune) (val int, isCell bool) {
	if ch >= '1' && ch <= '9' {
		return int(ch - '0'), true
	}
	switch ch {
	case '0', '.', '_', '-', '*', '~', 'x', 'X', '?', 'o', 'O':
		return 0, true
	}
	return 0, false
}

// tryParseContiguous81 looks for an 81-character continuous puzzle string.
// Supports unspaced strings like "60005...009" or "6...5...4...".
func tryParseContiguous81(s string) (Board, bool) {
	trimmed := strings.Trim(s, "\"'` \t\r\n")
	runes := []rune(trimmed)

	// Direct match if the entire input is 81 cell characters
	if len(runes) == 81 {
		var board Board
		count := 0
		valid := true
		for j, ch := range runes {
			val, ok := isCellRune(ch)
			if !ok {
				valid = false
				break
			}
			board[j/9][j%9] = val
			if val != 0 {
				count++
			}
		}
		if valid && count >= 5 {
			return board, true
		}
	}

	// Sliding match for 81-character run embedded in other text/URLs
	fullRunes := []rune(s)
	for i := 0; i+81 <= len(fullRunes); i++ {
		candidate := fullRunes[i : i+81]
		var board Board
		count := 0
		valid := true
		for j, ch := range candidate {
			val, ok := isCellRune(ch)
			if !ok {
				valid = false
				break
			}
			board[j/9][j%9] = val
			if val != 0 {
				count++
			}
		}
		if valid && count >= 5 && validateInitialBoard(&board) {
			return board, true
		}
	}
	return Board{}, false
}

func isHorizontalBorder(line string) bool {
	hasBorderChar := false
	for _, ch := range line {
		if ch >= '1' && ch <= '9' {
			return false
		}
		switch ch {
		case '+', '=', '-', '#', '*', '─', '┌', '┬', '┐', '├', '┼', '┤', '└', '┴', '┘', '━', '┏', '┳', '┓', '┣', '╋', '┫', '┗', '┻', '┛':
			hasBorderChar = true
		}
	}
	borderCount := 0
	for _, ch := range line {
		switch ch {
		case '+', '=', '-', '#', '*', '─', '┌', '┬', '┐', '├', '┼', '┤', '└', '┴', '┘', '━', '┏', '┳', '┓', '┣', '╋', '┫', '┗', '┻', '┛':
			borderCount++
		}
	}
	return hasBorderChar && borderCount >= 4
}

func stripRowPrefix(line string) string {
	trimmed := strings.TrimSpace(line)
	lower := strings.ToLower(trimmed)

	// e.g. "Row 1:", "Row 1 -", "Row 1: ", "Row 1 "
	if strings.HasPrefix(lower, "row") {
		rest := strings.TrimSpace(trimmed[3:])
		i := 0
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i > 0 {
			afterNum := strings.TrimSpace(rest[i:])
			if len(afterNum) > 0 && (afterNum[0] == ':' || afterNum[0] == '-' || afterNum[0] == '.' || afterNum[0] == '|') {
				return strings.TrimSpace(afterNum[1:])
			}
			return afterNum
		}
	}

	// e.g. "[1] "
	if len(trimmed) >= 3 && trimmed[0] == '[' && trimmed[1] >= '1' && trimmed[1] <= '9' && trimmed[2] == ']' {
		return strings.TrimSpace(trimmed[3:])
	}

	// e.g. "1: " or "1. " (must be followed by whitespace so it is not a grid string like "3.621.8..")
	if len(trimmed) >= 3 && trimmed[0] >= '1' && trimmed[0] <= '9' && (trimmed[1] == ':' || trimmed[1] == '.') && (trimmed[2] == ' ' || trimmed[2] == '\t') {
		return strings.TrimSpace(trimmed[2:])
	}

	return trimmed
}

func trimPipeDecorations(line string) string {
	firstPipe := strings.IndexAny(line, "|│║")
	lastPipe := strings.LastIndexAny(line, "|│║")
	if firstPipe != -1 && lastPipe != -1 && firstPipe < lastPipe {
		return line[firstPipe+1 : lastPipe]
	}
	return line
}

func extractLineCells(line string) []int {
	line = trimPipeDecorations(line)
	line = stripRowPrefix(line)
	var cells []int
	for _, ch := range line {
		switch ch {
		case '|', '│', '║', '!', '[', ']', '{', '}', '(', ')', ',', ';', '+', '"', '\'', ' ', '\t', '\r':
			continue
		}
		if val, ok := isCellRune(ch); ok {
			cells = append(cells, val)
		}
	}
	return cells
}

func tryParseMultiLineGrid(s string) (Board, bool) {
	lines := strings.Split(s, "\n")
	var rows [][]int
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || isHorizontalBorder(line) {
			continue
		}
		cells := extractLineCells(line)
		if len(cells) == 9 {
			rows = append(rows, cells)
		}
	}
	if len(rows) == 9 {
		var board Board
		count := 0
		for r := 0; r < 9; r++ {
			for c := 0; c < 9; c++ {
				board[r][c] = rows[r][c]
				if board[r][c] != 0 {
					count++
				}
			}
		}
		if count >= 5 {
			return board, true
		}
	}
	return Board{}, false
}

func tryParseTokenStream(s string) (Board, bool) {
	lines := strings.Split(s, "\n")
	var allCells []int
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if isHorizontalBorder(line) {
			continue
		}
		cells := extractLineCells(line)
		allCells = append(allCells, cells...)
	}

	if len(allCells) == 81 {
		var board Board
		count := 0
		for i := 0; i < 81; i++ {
			board[i/9][i%9] = allCells[i]
			if allCells[i] != 0 {
				count++
			}
		}
		if count >= 5 {
			return board, true
		}
	}

	if len(allCells) > 81 {
		for i := 0; i+81 <= len(allCells); i++ {
			var board Board
			count := 0
			for j := 0; j < 81; j++ {
				board[j/9][j%9] = allCells[i+j]
				if allCells[i+j] != 0 {
					count++
				}
			}
			if count >= 5 && validateInitialBoard(&board) {
				return board, true
			}
		}
	}

	return Board{}, false
}

// decodeTextGrid attempts to parse a Sudoku board from arbitrary text.
func decodeTextGrid(s string) (Board, bool) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return Board{}, false
	}

	// Strategy 1: Contiguous 81-character run (e.g. "60005...009")
	if b, ok := tryParseContiguous81(s); ok {
		return b, true
	}

	// Strategy 2: Multi-line grid rows (handles ASCII borders, spacing, CSV, JSON rows)
	if b, ok := tryParseMultiLineGrid(s); ok {
		return b, true
	}

	// Strategy 3: Token stream extraction (81 separated values across text or 1D array)
	if b, ok := tryParseTokenStream(s); ok {
		return b, true
	}

	return Board{}, false
}
