package main

import (
	"bytes"
	"encoding/json"
	"math/rand/v2"
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
	g.save.setContinue(diffEasy, &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: sol,
	})
	g.save.setContinue(diffMedium, &continueGame{
		ID: "otherhash12ab", Difficulty: diffMedium, Givens: givens, Solution: sol, Grid: givens,
	})
	g.finishSuccess()
	if g.save.continueFor(diffEasy) != nil {
		t.Fatal("in-memory continue should be wiped")
	}
	if g.save.continueFor(diffMedium) == nil {
		t.Fatal("other difficulty continue should stay")
	}
	if g.puzzle.ID != "" {
		t.Fatal("puzzle identity should be cleared immediately")
	}
	if g.shouldPersistContinue() {
		t.Fatal("must not persist continue after success")
	}
	loaded := loadSave()
	if loaded.continueFor(diffEasy) != nil {
		t.Fatal("save file continue should be wiped")
	}
	if loaded.continueFor(diffMedium) == nil {
		t.Fatal("other difficulty continue should still be on disk")
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
	if g.save.continueFor(diffEasy) != nil {
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
	s.setContinue(diffEasy, &continueGame{
		ID:         testPuzzleID,
		Difficulty: diffEasy,
		Givens:     givens,
		Solution:   sol,
		Grid:       sol,
		ElapsedMs:  12000,
		Mistakes:   2,
	})
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	loaded := loadSave()
	if loaded.continueFor(diffEasy) != nil {
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
	if again.continueFor(diffEasy) != nil {
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

func TestResumeContinueRestoresPencilMode(t *testing.T) {
	givens, sol := classicSolved()
	g := &game{save: newSaveData()}
	g.save.setContinue(diffEasy, &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: givens, Pencil: true,
	})
	g.difficulty = diffEasy
	g.resumeContinue()
	if !g.pencil {
		t.Fatal("continue with pencil true should open in pencil mode")
	}
	g.save.continueFor(diffEasy).Pencil = false
	g.resumeContinue()
	if g.pencil {
		t.Fatal("continue with pencil false should open in pen mode")
	}
}

func TestPersistPlayDebouncesDisk(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, givens),
		view:       viewPlay,
		saveFlush:  make(chan struct{}, 1),
	}
	g.persistPlay()
	if g.save.continueFor(diffEasy) == nil {
		t.Fatal("continue should be in memory immediately")
	}
	if _, err := os.Stat(saveFileName); !os.IsNotExist(err) {
		t.Fatal("debounced persist should not write immediately")
	}
	g.flushSave()
	if _, err := os.Stat(saveFileName); err != nil {
		t.Fatal("flush should write the save file")
	}
}

func TestWriteAtomicCompactJSON(t *testing.T) {
	chdirTemp(t)
	s := newSaveData()
	s.markCompleted(diffEasy, "abc", 1, 1000)
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(saveFileName)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	if bytes.Contains(data, []byte("\n  ")) {
		t.Fatal("save should be compact JSON, not indented")
	}
	loaded := loadSave()
	if n, ok := loaded.completedMistakes(diffEasy, "abc"); !ok || n != 1 {
		t.Fatalf("round trip mistakes=%d ok=%v", n, ok)
	}
	s.markCompleted(diffEasy, "def", 2, 2000)
	if err := s.write(); err != nil {
		t.Fatal(err)
	}
	loaded = loadSave()
	if _, ok := loaded.completedSet(diffEasy)["def"]; !ok {
		t.Fatal("second atomic write should replace the file")
	}
}

func TestLegacyContinueObjectLoadsPerDifficulty(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	legacy := map[string]any{
		"version": "1.2",
		"continue": map[string]any{
			"id": testPuzzleID, "difficulty": diffMedium,
			"givens": givens, "solution": sol, "grid": givens,
			"elapsedMs": 1000, "mistakes": 0,
		},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(saveFileName, data, 0644); err != nil {
		t.Fatal(err)
	}
	loaded := loadSave()
	if loaded.continueFor(diffEasy) != nil {
		t.Fatal("legacy continue should not land on easy")
	}
	c := loaded.continueFor(diffMedium)
	if c == nil || c.ID != testPuzzleID {
		t.Fatal("legacy continue should map to medium")
	}
}

func TestReturnToMenuKeepsContinue(t *testing.T) {
	chdirTemp(t)
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		puzzle:     puzzleEntry{ID: testPuzzleID, Givens: givens, Solution: sol},
		board:      newBoard(givens, sol, givens),
		view:       viewPlay,
	}
	g.returnToMenu()
	if g.view != viewMenu {
		t.Fatalf("view=%d want menu", g.view)
	}
	if g.save.continueFor(diffEasy) == nil {
		t.Fatal("esc to menu should keep continue")
	}
	if g.save.statsFor(diffEasy).Failed != 0 {
		t.Fatal("esc to menu must not count as Failed")
	}
	if g.menuIndex != menuContinue {
		t.Fatal("menu should land on Continue")
	}
}

func TestNewGamePromptsOnlyForSameDifficulty(t *testing.T) {
	if err := loadPuzzleBank(); err != nil {
		t.Fatal(err)
	}
	givens, sol := classicSolved()
	g := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		menuIndex:  menuNewGame,
		rng:        rand.New(rand.NewPCG(1, 2)),
	}
	g.save.setContinue(diffMedium, &continueGame{
		ID: "medhash12ab", Difficulty: diffMedium, Givens: givens, Solution: sol, Grid: givens,
	})
	if g.activateMenu() {
		t.Fatal("new game should not quit")
	}
	if g.view != viewPlay {
		t.Fatalf("view=%d want play (no prompt for another difficulty's save)", g.view)
	}
	if g.save.continueFor(diffMedium) == nil {
		t.Fatal("medium continue should remain")
	}

	g2 := &game{
		save:       newSaveData(),
		difficulty: diffEasy,
		menuIndex:  menuNewGame,
		rng:        rand.New(rand.NewPCG(1, 2)),
	}
	g2.save.setContinue(diffEasy, &continueGame{
		ID: testPuzzleID, Difficulty: diffEasy, Givens: givens, Solution: sol, Grid: givens,
	})
	g2.activateMenu()
	if g2.view != viewConfirmNewGame {
		t.Fatalf("view=%d want confirm new game", g2.view)
	}
}
