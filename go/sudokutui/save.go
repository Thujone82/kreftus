package main

import (
	"bytes"
	"encoding/json"
	"os"
	"unicode/utf8"
)

const (
	saveFileName  = "sudoku.json"
	sudokuVersion = "1.0"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type diffStats struct {
	Successes       int    `json:"successes"`
	Failed          int    `json:"failures"`
	FastestMs       *int64 `json:"fastestMs"`
	FastestMistakes *int   `json:"fastestMistakes"`
}

type continueGame struct {
	ID         string `json:"id"`
	Difficulty string `json:"difficulty"`
	Givens     string `json:"givens"`
	Solution   string `json:"solution"`
	Grid       string `json:"grid"`
	ElapsedMs  int64  `json:"elapsedMs"`
	Mistakes   int    `json:"mistakes"`
}

type saveData struct {
	Version   string                `json:"version"`
	Stats     map[string]*diffStats `json:"stats"`
	Completed map[string][]string   `json:"completed"`
	Continue  *continueGame         `json:"continue"`
}

func newSaveData() *saveData {
	s := &saveData{
		Version:   sudokuVersion,
		Stats:     map[string]*diffStats{},
		Completed: map[string][]string{},
	}
	for _, d := range difficultyOrder {
		s.Stats[d] = &diffStats{}
		s.Completed[d] = []string{}
	}
	return s
}

func loadSave() *saveData {
	data, err := os.ReadFile(saveFileName)
	if err != nil {
		return newSaveData()
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	s := newSaveData()
	if err := json.Unmarshal(data, s); err != nil {
		return newSaveData()
	}
	if s.Version == "" {
		s.Version = sudokuVersion
	}
	if s.Stats == nil {
		s.Stats = map[string]*diffStats{}
	}
	if s.Completed == nil {
		s.Completed = map[string][]string{}
	}
	for _, d := range difficultyOrder {
		if s.Stats[d] == nil {
			s.Stats[d] = &diffStats{}
		}
		if s.Completed[d] == nil {
			s.Completed[d] = []string{}
		}
	}
	if s.Continue != nil {
		if len(s.Continue.Solution) != 81 {
			s.Continue.Solution = solveSudoku(s.Continue.Givens)
		}
		if !validPuzzleStrings(s.Continue.Givens, s.Continue.Solution, s.Continue.Grid) {
			s.Continue = nil
		}
	}
	return s
}

func validPuzzleStrings(givens, solution, grid string) bool {
	if len(givens) != 81 || len(solution) != 81 || len(grid) != 81 {
		return false
	}
	for i := 0; i < 81; i++ {
		if !isDigitOrZero(givens[i]) || !isDigitOrZero(solution[i]) || !isDigitOrZero(grid[i]) {
			return false
		}
		if givens[i] != '0' && givens[i] != grid[i] {
			return false
		}
	}
	return utf8.ValidString(givens)
}

func isDigitOrZero(c byte) bool {
	return c >= '0' && c <= '9'
}

func (s *saveData) write() error {
	s.Version = sudokuVersion
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(saveFileName, append(utf8BOM, data...), 0644)
}

func (s *saveData) statsFor(d string) *diffStats {
	st := s.Stats[d]
	if st == nil {
		st = &diffStats{}
		s.Stats[d] = st
	}
	return st
}

func (s *saveData) completedSet(d string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, id := range s.Completed[d] {
		set[id] = struct{}{}
	}
	return set
}

func (s *saveData) markCompleted(d, id string) {
	set := s.completedSet(d)
	if _, ok := set[id]; ok {
		return
	}
	s.Completed[d] = append(s.Completed[d], id)
}

func (s *saveData) recordSuccess(d string, elapsedMs int64, mistakes int) {
	st := s.statsFor(d)
	st.Successes++
	if st.FastestMs == nil || elapsedMs < *st.FastestMs || (elapsedMs == *st.FastestMs && (st.FastestMistakes == nil || mistakes < *st.FastestMistakes)) {
		ms := elapsedMs
		m := mistakes
		st.FastestMs = &ms
		st.FastestMistakes = &m
	}
}

func (st *diffStats) UnmarshalJSON(b []byte) error {
	var raw struct {
		Successes       int    `json:"successes"`
		Abandonments    int    `json:"abandonments"`
		Failures        int    `json:"failures"`
		FastestMs       *int64 `json:"fastestMs"`
		FastestMistakes *int   `json:"fastestMistakes"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	st.Successes = raw.Successes
	st.Failed = raw.Failures
	if st.Failed == 0 && raw.Abandonments > 0 {
		st.Failed = raw.Abandonments
	}
	st.FastestMs = raw.FastestMs
	st.FastestMistakes = raw.FastestMistakes
	return nil
}

func (s *saveData) recordFailure(d string) {
	s.statsFor(d).Failed++
}
