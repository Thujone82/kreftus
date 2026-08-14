package main

import (
	"os"
	"testing"
	"time"
)

const testPuzzleID = "testhash12ab"

func classicSolved() (givens, sol string) {
	givens = "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	sol = solveSudoku(givens)
	return
}

func chdirTemp(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}

func TestFinishSuccessWipesContinue(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, sol),
		view:       viewPlay,
		elapsed:    45 * time.Second,
	}
	g.save.Continue = &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: sol,
	}
	g.finishSuccess()
	if g.save.Continue != nil {
		t.Fatal("in-memory continue should be wiped")
	}
	if g.puzzle.ID != "" {
		t.Fatal("puzzle identity should be cleared immediately")
	}
	if g.shouldPersistContinue() {
		t.Fatal("must not persist continue after success")
	}
	loaded := loadSave()
	if loaded.Continue != nil {
		t.Fatal("save file continue should be wiped")
	}
	if _, ok := loaded.completedSet(diffEasy)[testPuzzleID]; !ok {
		t.Fatal("completed id missing from save")
	}
	if loaded.statsFor(diffEasy).Successes != 1 {
		t.Fatalf("successes=%d", loaded.statsFor(diffEasy).Successes)
	}
}

func TestPersistPlaySkipsCompletedBoard(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, sol),
		view:       viewPlay,
	}
	g.persistPlay()
	if g.save.Continue != nil {
		t.Fatal("must not write continue for a completed board")
	}
	if _, err := os.Stat(saveFileName); !os.IsNotExist(err) {
		t.Fatal("must not write save file for a completed board")
	}
}

func TestScrubCompletedContinueOnLoad(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	s := newSaveData()
	s.Continue = &continueGame{
		ID:         testPuzzleID,
		Difficulty: diffEasy,
		Givens:     givens,
		Solution:   sol,
		Grid:       sol,
		ElapsedMs:  12000,
		Mistakes:   2,
	}
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	loaded := loadSave()
	if loaded.Continue != nil {
		t.Fatal("load should wipe a completed continue")
	}
	if loaded.statsFor(diffEasy).Successes != 1 {
		t.Fatalf("successes=%d", loaded.statsFor(diffEasy).Successes)
	}
	if _, ok := loaded.completedSet(diffEasy)[testPuzzleID]; !ok {
		t.Fatal("completed id missing after scrub")
	}
	again := loadSave()
	if again.statsFor(diffEasy).Successes != 1 {
		t.Fatalf("must not double-count success, got %d", again.statsFor(diffEasy).Successes)
	}
	if again.Continue != nil {
		t.Fatal("continue must stay wiped")
	}
}

func TestShouldPersistContinue(t *testing.T) {
	givens, sol := classicSolved()
	g := &game{
		puzzle: puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:  newBoard(givens, sol, givens),
		view:   viewPlay,
	}
	if !g.shouldPersistContinue() {
		t.Fatal("in-progress board should persist")
	}
	g.board = newBoard(givens, sol, sol)
	if g.shouldPersistContinue() {
		t.Fatal("completed board must not persist")
	}
	g.board = newBoard(givens, sol, givens)
	g.pendingSolved = true
	if g.shouldPersistContinue() {
		t.Fatal("pending solved must not persist")
	}
}
