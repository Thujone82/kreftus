package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"io"
	"math/rand/v2"
)

//go:embed puzzles.json.gz
var puzzlesGZ []byte

const (
	diffEasy       = "easy"
	diffMedium     = "medium"
	diffHard       = "hard"
	diffDiabolical = "diabolical"
)

var difficultyOrder = []string{diffEasy, diffMedium, diffHard, diffDiabolical}

var difficultyLabel = map[string]string{
	diffEasy:       "Easy",
	diffMedium:     "Medium",
	diffHard:       "Hard",
	diffDiabolical: "Diabolical",
}

var bankSize = map[string]int{
	diffEasy:       2500,
	diffMedium:     2000,
	diffHard:       1000,
	diffDiabolical: 250,
}

type puzzleEntry struct {
	ID     string `json:"id"`
	Givens string `json:"givens"`
	// Solution is filled at play time by ensureSolved; not stored in puzzles.json.
	Solution string `json:"-"`
}

type puzzleBank struct {
	Source     string        `json:"source"`
	Easy       []puzzleEntry `json:"easy"`
	Medium     []puzzleEntry `json:"medium"`
	Hard       []puzzleEntry `json:"hard"`
	Diabolical []puzzleEntry `json:"diabolical"`
}

var bank puzzleBank

func loadPuzzleBank() error {
	zr, err := gzip.NewReader(bytes.NewReader(puzzlesGZ))
	if err != nil {
		return err
	}
	defer zr.Close()
	data, err := io.ReadAll(zr)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(stripBOM(data), &bank); err != nil {
		return err
	}
	return nil
}

func stripBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}

func puzzlesFor(d string) []puzzleEntry {
	switch d {
	case diffEasy:
		return bank.Easy
	case diffMedium:
		return bank.Medium
	case diffHard:
		return bank.Hard
	case diffDiabolical:
		return bank.Diabolical
	default:
		return nil
	}
}

func incompletePool(d string, completed map[string]struct{}) []puzzleEntry {
	all := puzzlesFor(d)
	out := make([]puzzleEntry, 0, len(all))
	for _, p := range all {
		if _, done := completed[p.ID]; done {
			continue
		}
		out = append(out, p)
	}
	return out
}

func remainingCount(d string, completed map[string]struct{}) int {
	n := 0
	for _, p := range puzzlesFor(d) {
		if _, done := completed[p.ID]; !done {
			n++
		}
	}
	return n
}

func pickIncomplete(d string, completed map[string]struct{}, rng *rand.Rand) (puzzleEntry, bool) {
	n := remainingCount(d, completed)
	if n == 0 {
		return puzzleEntry{}, false
	}
	k := rng.IntN(n)
	for _, p := range puzzlesFor(d) {
		if _, done := completed[p.ID]; done {
			continue
		}
		if k == 0 {
			return p, true
		}
		k--
	}
	return puzzleEntry{}, false
}

func pickRandom(pool []puzzleEntry, rng *rand.Rand) (puzzleEntry, bool) {
	if len(pool) == 0 {
		return puzzleEntry{}, false
	}
	return pool[rng.IntN(len(pool))], true
}

func nextDifficulty(d string, dir int) string {
	idx := 0
	for i, v := range difficultyOrder {
		if v == d {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(difficultyOrder)) % len(difficultyOrder)
	return difficultyOrder[idx]
}

func puzzleByID(d, id string) (puzzleEntry, bool) {
	for _, p := range puzzlesFor(d) {
		if p.ID == id {
			return p, true
		}
	}
	return puzzleEntry{}, false
}

func (p *puzzleEntry) ensureSolved() bool {
	if len(p.Solution) == 81 {
		return true
	}
	p.Solution = solveSudoku(p.Givens)
	return len(p.Solution) == 81
}
