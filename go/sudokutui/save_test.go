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
	if loaded.statsFor(diffEasy).Perfect != 1 {
		t.Fatalf("perfect=%d", loaded.statsFor(diffEasy).Perfect)
	}
	if loaded.statsFor(diffEasy).errorRate() != "0.00" {
		t.Fatalf("error rate=%s", loaded.statsFor(diffEasy).errorRate())
	}
	if n, ok := loaded.completedMistakes(diffEasy, testPuzzleID); !ok || n != 0 {
		t.Fatalf("completed mistakes=%d ok=%v want 0", n, ok)
	}
	if loaded.averageCompletion(diffEasy) != "0:45" {
		t.Fatalf("average=%s want 0:45", loaded.averageCompletion(diffEasy))
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
	if n, ok := loaded.completedMistakes(diffEasy, testPuzzleID); !ok || n != 2 {
		t.Fatalf("scrubbed continue mistakes=%d ok=%v want 2", n, ok)
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

func TestRecordSuccessPerfectAndErrorRate(t *testing.T) {
	s := newSaveData()
	if s.statsFor(diffEasy).errorRate() != "—" {
		t.Fatal("no successes should show em dash")
	}
	s.recordSuccess(diffEasy, 1000, 0)
	s.recordSuccess(diffEasy, 2000, 2)
	s.recordSuccess(diffEasy, 1500, 1)
	st := s.statsFor(diffEasy)
	if st.Perfect != 1 || st.Successes != 3 {
		t.Fatalf("perfect=%d successes=%d", st.Perfect, st.Successes)
	}
	if st.errorRate() != "1.00" {
		t.Fatalf("error rate=%s want 1.00", st.errorRate())
	}
	s.recordSuccess(diffEasy, 3000, 1)
	if st.errorRate() != "1.00" {
		t.Fatalf("error rate=%s want 1.00 after fourth", st.errorRate())
	}
	s.recordSuccess(diffMedium, 4000, 3)
	s.recordSuccess(diffMedium, 5000, 2)
	if s.statsFor(diffMedium).errorRate() != "2.50" {
		t.Fatalf("medium error rate=%s want 2.50", s.statsFor(diffMedium).errorRate())
	}
	if s.statsFor(diffMedium).Perfect != 0 {
		t.Fatal("medium should have no perfects")
	}
	st.RatedSuccesses = 20
	st.MistakeSum = 21
	if st.errorRate() != "1.05" {
		t.Fatalf("error rate=%s want 1.05", st.errorRate())
	}
}

func TestMarkCompletedStoresMistakes(t *testing.T) {
	s := newSaveData()
	if s.averageCompletion(diffEasy) != "—" {
		t.Fatal("no completions should show em dash")
	}
	s.markCompleted(diffEasy, "abc", 4, 60000)
	if n, ok := s.completedMistakes(diffEasy, "abc"); !ok || n != 4 {
		t.Fatalf("mistakes=%d ok=%v want 4", n, ok)
	}
	s.markCompleted(diffEasy, "abc", 9, 90000)
	if n, _ := s.completedMistakes(diffEasy, "abc"); n != 4 {
		t.Fatalf("duplicate id should keep first mistakes, got %d", n)
	}
	s.markCompleted(diffEasy, "def", 1, 120000)
	if s.averageCompletion(diffEasy) != "1:30" {
		t.Fatalf("average=%s want 1:30", s.averageCompletion(diffEasy))
	}
}

func TestContinueMenuLabel(t *testing.T) {
	if continueMenuLabel(nil) != "Continue Game" {
		t.Fatal("nil continue should be Continue Game")
	}
	c := &continueGame{Difficulty: diffMedium}
	if continueMenuLabel(c) != "Continue Game [Medium]" {
		t.Fatalf("got %q", continueMenuLabel(c))
	}
}

func TestResumeContinueRestoresPencilMode(t *testing.T) {
	givens, sol := classicSolved()
	g := &game{save: newSaveData()}
	g.save.Continue = &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: givens, Pencil: true,
	}
	g.resumeContinue()
	if !g.pencil {
		t.Fatal("continue with pencil true should open in pencil mode")
	}
	g.save.Continue.Pencil = false
	g.resumeContinue()
	if g.pencil {
		t.Fatal("continue with pencil false should open in pen mode")
	}
}
