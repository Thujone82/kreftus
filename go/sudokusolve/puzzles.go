package main

import (
	_ "embed"
	"fmt"
	"strings"

	"image/color"
)

//go:embed puzzles.txt
var puzzleText string

type dieLevel int

const (
	dieEasy dieLevel = iota
	dieMedium
	dieHard
	dieDiabolical
	dieLevelCount
)

const (
	dieSize = 22
	dieX    = 800 - 16 - dieSize
	dieY    = (34 - dieSize) / 2
)

var puzzleBank [dieLevelCount][]string

func init() {
	level := dieLevel(-1)
	for _, line := range strings.Split(puzzleText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			switch strings.TrimSpace(strings.TrimPrefix(line, "#")) {
			case "easy":
				level = dieEasy
			case "medium":
				level = dieMedium
			case "hard":
				level = dieHard
			case "diabolical":
				level = dieDiabolical
			}
			continue
		}
		if level < 0 || level >= dieLevelCount || len(line) != 81 {
			continue
		}
		puzzleBank[level] = append(puzzleBank[level], line)
	}
}

func (d dieLevel) name() string {
	switch d {
	case dieEasy:
		return "Easy"
	case dieMedium:
		return "Medium"
	case dieHard:
		return "Hard"
	default:
		return "Diabolical"
	}
}

func (d dieLevel) helpLines() (string, string) {
	return fmt.Sprintf("Click for %s Puzzle", d.name()), "Right Click Change Difficulty"
}

func (d dieLevel) helpText() string {
	l1, l2 := d.helpLines()
	return l1 + "\n" + l2
}

func (d dieLevel) color() color.RGBA {
	switch d {
	case dieEasy:
		return color.RGBA{46, 180, 96, 255}
	case dieMedium:
		return color.RGBA{232, 196, 60, 255}
	case dieHard:
		return color.RGBA{214, 72, 78, 255}
	default:
		return color.RGBA{168, 96, 214, 255}
	}
}

func (d dieLevel) next() dieLevel {
	return (d + 1) % dieLevelCount
}

func boardFromGivens(s string) (Board, [9][9]bool, bool) {
	var board Board
	var initial [9][9]bool
	if len(s) != 81 {
		return board, initial, false
	}
	for i := 0; i < 81; i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return Board{}, [9][9]bool{}, false
		}
		r, col := i/9, i%9
		board[r][col] = int(c - '0')
		initial[r][col] = c != '0'
	}
	return board, initial, true
}

func inDie(mx, my int) bool {
	return mx >= dieX-6 && mx < dieX+dieSize+6 && my >= 0 && my < 34
}
